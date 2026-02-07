package io.supernode.storage.erasure;

import io.supernode.network.transport.Transport;
import java.io.*;
import java.nio.ByteBuffer;
import java.nio.channels.Channels;
import java.nio.channels.FileChannel;
import java.security.DigestOutputStream;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.time.Duration;
import java.time.Instant;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicLong;
import java.util.concurrent.locks.ReentrantReadWriteLock;
import java.util.function.Consumer;

public class ErasureCoder {
    private static final int GF_SIZE = 256;
    private static final int PRIMITIVE_POLY = 0x11d;
    private static final int DEFAULT_CHUNK_SIZE = 256 * 1024;
    private static final int OVERLAP_SIZE = 0;

    private final int dataShards;
    private final int parityShards;
    private final int totalShards;

    private final int[] expTable = new int[GF_SIZE * 2];
    private final int[] logTable = new int[GF_SIZE];
    private final int[][] encodeMatrix;

    private final ReentrantReadWriteLock lock = new ReentrantReadWriteLock();
    private final Map<Integer, byte[]> checksumCache = new ConcurrentHashMap<>();
    private final AtomicLong encodeCount = new AtomicLong();
    private final AtomicLong decodeCount = new AtomicLong();
    private final AtomicLong repairCount = new AtomicLong();

    private NetworkContext networkContext;
    private Consumer<EncodingEvent> onEncoding;
    private Consumer<DecodingEvent> onDecoding;
    private Consumer<RepairEvent> onRepair;

    public ErasureCoder() {
        this(4, 2);
    }

    public ErasureCoder(int dataShards, int parityShards) {
        this(dataShards, parityShards, null);
    }

    public ErasureCoder(int dataShards, int parityShards, NetworkContext networkContext) {
        if (dataShards < 1 || parityShards < 1) {
            throw new IllegalArgumentException("Must have at least 1 data and 1 parity shard");
        }
        if (dataShards + parityShards > 255) {
            throw new IllegalArgumentException("Total shards cannot exceed 255");
        }

        this.dataShards = dataShards;
        this.parityShards = parityShards;
        this.totalShards = dataShards + parityShards;
        this.networkContext = networkContext != null ? networkContext : new NetworkContext();

        initGaloisTables();
        this.encodeMatrix = buildVandermondeMatrix();
    }

    public EncodeResult encode(byte[] data) {
        lock.readLock().lock();
        try {
            encodeCount.incrementAndGet();
            int shardSize = (data.length + dataShards - 1) / dataShards;
            byte[] paddedData = Arrays.copyOf(data, shardSize * dataShards);

            byte[][] shards = new byte[totalShards][shardSize];

            for (int i = 0; i < dataShards; i++) {
                System.arraycopy(paddedData, i * shardSize, shards[i], 0, shardSize);
            }

            for (int i = dataShards; i < totalShards; i++) {
                for (int j = 0; j < shardSize; j++) {
                    int value = 0;
                    for (int k = 0; k < dataShards; k++) {
                        value ^= gfMul(encodeMatrix[i][k], shards[k][j] & 0xFF);
                    }
                    shards[i][j] = (byte) value;
                }
            }

            computeChecksums(shards);

            EncodeResult result = new EncodeResult(shards, shardSize, data.length);

            if (onEncoding != null) {
                EncodingEvent event = new EncodingEvent(
                    data.length,
                    dataShards,
                    parityShards,
                    shardSize,
                    Instant.now()
                );
                onEncoding.accept(event);
            }

            return result;
        } finally {
            lock.readLock().unlock();
        }
    }

    public byte[] decode(byte[][] shards, int[] presentIndices, int originalSize, int shardSize) {
        lock.readLock().lock();
        try {
            decodeCount.incrementAndGet();
            if (presentIndices.length < dataShards) {
                throw new IllegalArgumentException(
                    "Need at least " + dataShards + " shards, got " + presentIndices.length);
            }

            int[] selectedIndices = Arrays.copyOf(presentIndices, dataShards);
            byte[][] selectedShards = new byte[dataShards][];
            for (int i = 0; i < dataShards; i++) {
                selectedShards[i] = shards[selectedIndices[i]];
            }

            int[][] subMatrix = new int[dataShards][dataShards];
            for (int i = 0; i < dataShards; i++) {
                System.arraycopy(encodeMatrix[selectedIndices[i]], 0, subMatrix[i], 0, dataShards);
            }

            int[][] invMatrix = invertMatrix(subMatrix);

            byte[][] decoded = new byte[dataShards][shardSize];
            for (int i = 0; i < dataShards; i++) {
                for (int j = 0; j < shardSize; j++) {
                    int value = 0;
                    for (int k = 0; k < dataShards; k++) {
                        value ^= gfMul(invMatrix[i][k], selectedShards[k][j] & 0xFF);
                    }
                    decoded[i][j] = (byte) value;
                }
            }

            byte[] result = new byte[originalSize];
            int offset = 0;
            for (int i = 0; i < dataShards && offset < originalSize; i++) {
                int toCopy = Math.min(shardSize, originalSize - offset);
                System.arraycopy(decoded[i], 0, result, offset, toCopy);
                offset += toCopy;
            }

            if (onDecoding != null) {
                DecodingEvent event = new DecodingEvent(
                    originalSize,
                    presentIndices.length,
                    shards.length,
                    presentIndices.length < shards.length,
                    Instant.now()
                );
                onDecoding.accept(event);
            }

            return result;
        } finally {
            lock.readLock().unlock();
        }
    }

    public StreamEncodeResult encodeStream(InputStream input, long fileSize, OutputStream[] shardOutputs) 
            throws IOException {
        return encodeStream(input, fileSize, shardOutputs, DEFAULT_CHUNK_SIZE, OVERLAP_SIZE);
    }

    public StreamEncodeResult encodeStream(InputStream input, long fileSize, 
            OutputStream[] shardOutputs, int chunkSize, int overlapSize) 
            throws IOException {
        if (shardOutputs.length != totalShards) {
            throw new IllegalArgumentException(
                "Expected " + totalShards + " shard outputs, got " + shardOutputs.length);
        }

        lock.readLock().lock();
        try {
            encodeCount.incrementAndGet();
            long totalBytesRead = 0;
            byte[] carryOver = new byte[overlapSize];
            int carryOverLen = 0;
            int shardsProduced = 0;
            long totalShardSize = 0;

            while (true) {
                int readSize = chunkSize - carryOverLen;
                byte[] chunk = new byte[readSize];
                int bytesRead = input.read(chunk);

                if (bytesRead == -1 && carryOverLen == 0) {
                    break;
                }

                int actualRead = bytesRead == -1 ? 0 : bytesRead;
                int totalLen = carryOverLen + actualRead;

                if (totalLen == 0) {
                    break;
                }

                byte[] fullChunk = new byte[totalLen];
                System.arraycopy(carryOver, 0, fullChunk, 0, carryOverLen);
                if (actualRead > 0) {
                    System.arraycopy(chunk, 0, fullChunk, carryOverLen, actualRead);
                }

                int shardSize = (fullChunk.length + dataShards - 1) / dataShards;
                byte[][] shards = new byte[totalShards][shardSize];

                for (int i = 0; i < dataShards; i++) {
                    int start = i * shardSize;
                    int end = Math.min(start + shardSize, fullChunk.length);
                    int copyLen = end - start;
                    if (copyLen > 0) {
                        System.arraycopy(fullChunk, start, shards[i], 0, copyLen);
                    }
                }

                for (int i = dataShards; i < totalShards; i++) {
                    for (int j = 0; j < shardSize; j++) {
                        int value = 0;
                        for (int k = 0; k < dataShards; k++) {
                            value ^= gfMul(encodeMatrix[i][k], shards[k][j] & 0xFF);
                        }
                        shards[i][j] = (byte) value;
                    }
                }

                for (int i = 0; i < totalShards; i++) {
                    shardOutputs[i].write(shards[i], 0, shardSize);
                    totalShardSize += shardSize;
                }

                if (bytesRead == -1) {
                    totalBytesRead += actualRead;
                    shardsProduced++;
                    break;
                }

                carryOverLen = Math.min(overlapSize, totalLen);
                if (carryOverLen > 0) {
                    System.arraycopy(fullChunk, totalLen - carryOverLen, carryOver, 0, carryOverLen);
                }

                totalBytesRead += actualRead;
                shardsProduced++;

                if (onEncoding != null) {
                    double progress = fileSize > 0 ? (double) totalBytesRead / fileSize : 0;
                    EncodingEvent event = new EncodingEvent(
                        totalBytesRead,
                        dataShards,
                        parityShards,
                        shardSize,
                        Instant.now()
                    );
                    onEncoding.accept(event);
                }
            }

            computeChecksumsFromStreams(shardOutputs);

            return new StreamEncodeResult(
                totalBytesRead,
                shardsProduced,
                (int) (totalShardSize / totalShards),
                Instant.now()
            );
        } finally {
            lock.readLock().unlock();
        }
    }

    public byte[] decodeStream(InputStream[] shardInputs, int[] shardIndices, 
            long originalSize, int shardSize, OutputStream output) throws IOException {
        return decodeStream(shardInputs, shardIndices, originalSize, shardSize, output, 
                DEFAULT_CHUNK_SIZE, OVERLAP_SIZE);
    }

    public byte[] decodeStream(InputStream[] shardInputs, int[] shardIndices, 
            long originalSize, int shardSize, OutputStream output, 
            int chunkSize, int overlapSize) throws IOException {
        if (shardInputs.length < dataShards) {
            throw new IllegalArgumentException(
                "Need at least " + dataShards + " shard inputs, got " + shardInputs.length);
        }

        lock.readLock().lock();
        try {
            decodeCount.incrementAndGet();

            int[] selectedIndices = Arrays.copyOf(shardIndices, dataShards);
            int[][] subMatrix = new int[dataShards][dataShards];
            for (int i = 0; i < dataShards; i++) {
                System.arraycopy(encodeMatrix[selectedIndices[i]], 0, subMatrix[i], 0, dataShards);
            }

            int[][] invMatrix = invertMatrix(subMatrix);

            long bytesWritten = 0;
            int shardsProduced = 0;

            while (true) {
                byte[][] shardChunks = new byte[dataShards][chunkSize];
                int[] bytesReadPerShard = new int[dataShards];
                int maxBytesRead = 0;
                boolean anyData = false;

                for (int i = 0; i < dataShards; i++) {
                    bytesReadPerShard[i] = shardInputs[i].read(shardChunks[i]);
                    if (bytesReadPerShard[i] > 0) {
                        anyData = true;
                        maxBytesRead = Math.max(maxBytesRead, bytesReadPerShard[i]);
                    }
                }

                if (!anyData) {
                    break;
                }

                byte[][] decoded = new byte[dataShards][maxBytesRead];
                for (int i = 0; i < dataShards; i++) {
                    for (int j = 0; j < maxBytesRead; j++) {
                        int value = 0;
                        for (int k = 0; k < dataShards; k++) {
                            value ^= gfMul(invMatrix[i][k], shardChunks[k][j] & 0xFF);
                        }
                        decoded[i][j] = (byte) value;
                    }
                }

                long bytesToProcess = Math.min((long) maxBytesRead * dataShards, originalSize - bytesWritten);

                for (int i = 0; i < dataShards; i++) {
                    long shardStart = (long) i * maxBytesRead;
                    long shardRemaining = bytesToProcess - shardStart;
                    int shardWriteLen = (int) Math.min(maxBytesRead, Math.max(0, shardRemaining));
                    
                    if (shardWriteLen > 0) {
                        output.write(decoded[i], 0, shardWriteLen);
                    }
                }

                bytesWritten += bytesToProcess;
                shardsProduced++;

                if (onDecoding != null) {
                    double progress = originalSize > 0 ? (double) bytesWritten / originalSize : 0;
                    DecodingEvent event = new DecodingEvent(
                        bytesWritten,
                        shardInputs.length,
                        shardInputs.length,
                        false,
                        Instant.now()
                    );
                    onDecoding.accept(event);
                }
            }

            if (onDecoding != null) {
                DecodingEvent event = new DecodingEvent(
                    originalSize,
                    shardIndices.length,
                    shardIndices.length + (totalShards - shardIndices.length),
                    shardIndices.length < totalShards,
                    Instant.now()
                );
                onDecoding.accept(event);
            }

            byte[] result = output instanceof ByteArrayOutputStream 
                ? ((ByteArrayOutputStream) output).toByteArray() 
                : null;

            return result;
        } finally {
            lock.readLock().unlock();
        }
    }

    public VerificationResult verifyParity(byte[][] shards) {
        lock.readLock().lock();
        try {
            if (shards.length < totalShards) {
                throw new IllegalArgumentException(
                    "Need " + totalShards + " shards to verify parity, got " + shards.length);
            }

            int shardSize = shards[0].length;
            List<Integer> corruptedIndices = new ArrayList<>();

            for (int p = dataShards; p < totalShards; p++) {
                byte[] recomputedParity = new byte[shardSize];
                for (int j = 0; j < shardSize; j++) {
                    int value = 0;
                    for (int k = 0; k < dataShards; k++) {
                        value ^= gfMul(encodeMatrix[p][k], shards[k][j] & 0xFF);
                    }
                    recomputedParity[j] = (byte) value;
                }

                if (!Arrays.equals(shards[p], recomputedParity)) {
                    corruptedIndices.add(p);
                }
            }

            return new VerificationResult(
                corruptedIndices.isEmpty(),
                corruptedIndices,
                Instant.now()
            );
        } finally {
            lock.readLock().unlock();
        }
    }

    public RepairResult repairParity(byte[][] shards, int[] presentIndices, int originalSize, int shardSize) {
        lock.writeLock().lock();
        try {
            repairCount.incrementAndGet();
            int[] neededIndices = new int[parityShards];
            int neededCount = 0;

            for (int i = dataShards; i < totalShards; i++) {
                boolean present = false;
                for (int idx : presentIndices) {
                    if (idx == i) {
                        present = true;
                        break;
                    }
                }
                if (!present) {
                    neededIndices[neededCount++] = i;
                }
            }

            if (neededCount == 0) {
                return new RepairResult(
                    Collections.emptyList(),
                    0,
                    0,
                    0,
                    true,
                    Instant.now()
                );
            }

            for (int i = 0; i < neededCount; i++) {
                int missingIdx = neededIndices[i];
                shards[missingIdx] = new byte[shardSize];

                for (int j = 0; j < shardSize; j++) {
                    int value = 0;
                    for (int k = 0; k < dataShards; k++) {
                        value ^= gfMul(encodeMatrix[missingIdx][k], shards[k][j] & 0xFF);
                    }
                    shards[missingIdx][j] = (byte) value;
                }
            }

            if (onRepair != null) {
                RepairEvent event = new RepairEvent(
                    neededCount,
                    neededIndices,
                    shardSize,
                    Instant.now()
                );
                onRepair.accept(event);
            }

            List<Integer> repairedIndices = new ArrayList<>();
            for (int i = 0; i < neededCount; i++) {
                repairedIndices.add(neededIndices[i]);
            }

            return new RepairResult(
                repairedIndices,
                neededCount,
                (long) shardSize * neededCount,
                System.nanoTime(),
                true,
                Instant.now()
            );
        } finally {
            lock.writeLock().unlock();
        }
    }

    public int selectOptimalShardCount() {
        if (networkContext == null) {
            return parityShards;
        }

        Transport.HealthState networkState = networkContext.getNetworkHealth();
        double avgLatency = networkContext.getAverageLatency();
        double packetLoss = networkContext.getPacketLossRate();

        int optimalParity = parityShards;

        if (networkState == Transport.HealthState.UNHEALTHY) {
            optimalParity = Math.min(parityShards * 2, 8);
        } else if (networkState == Transport.HealthState.DEGRADED) {
            optimalParity = Math.min(parityShards + 1, 6);
        }

        if (avgLatency > 500) {
            optimalParity = Math.min(optimalParity + 1, 8);
        } else if (avgLatency < 100) {
            optimalParity = Math.max(optimalParity - 1, 2);
        }

        if (packetLoss > 0.1) {
            optimalParity = Math.min(optimalParity + 1, 8);
        } else if (packetLoss < 0.01) {
            optimalParity = Math.max(optimalParity - 1, 2);
        }

        return optimalParity;
    }

    public List<PeerScore> selectOptimalPeers(List<PeerScore> peers) {
        if (peers == null || peers.isEmpty()) {
            return Collections.emptyList();
        }

        int optimalShards = selectOptimalShardCount();
        peers.sort((a, b) -> {
            int scoreCompare = Double.compare(b.score(), a.score());
            if (scoreCompare != 0) return scoreCompare;
            return Long.compare(a.latency(), b.latency());
        });

        return peers.subList(0, Math.min(optimalShards, peers.size()));
    }

    private void computeChecksums(byte[][] shards) {
        checksumCache.clear();

        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");

            for (int i = 0; i < shards.length; i++) {
                byte[] hash = digest.digest(shards[i]);
                checksumCache.put(i, hash);
                digest.reset();
            }
        } catch (NoSuchAlgorithmException e) {
            throw new RuntimeException("SHA-256 not available", e);
        }
    }

    private void computeChecksumsFromStreams(OutputStream[] shardOutputs) throws IOException {
        checksumCache.clear();

        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");

            for (int i = 0; i < shardOutputs.length; i++) {
                if (shardOutputs[i] instanceof DigestOutputStream) {
                    DigestOutputStream dos = (DigestOutputStream) shardOutputs[i];
                    byte[] hash = dos.getMessageDigest().digest();
                    checksumCache.put(i, hash);
                }
            }
        } catch (NoSuchAlgorithmException e) {
            throw new RuntimeException("SHA-256 not available", e);
        }
    }

    public byte[] getChecksum(int shardIndex) {
        return checksumCache.get(shardIndex);
    }

    public Map<Integer, byte[]> getAllChecksums() {
        return new ConcurrentHashMap<>(checksumCache);
    }

    public boolean verifyChecksum(int shardIndex, byte[] expectedChecksum, byte[] shardData) {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            byte[] actualChecksum = digest.digest(shardData);
            return Arrays.equals(expectedChecksum, actualChecksum);
        } catch (NoSuchAlgorithmException e) {
            throw new RuntimeException("SHA-256 not available", e);
        }
    }

    private void initGaloisTables() {
        int x = 1;
        for (int i = 0; i < GF_SIZE - 1; i++) {
            expTable[i] = x;
            expTable[i + GF_SIZE - 1] = x;
            logTable[x] = i;
            x <<= 1;
            if (x >= GF_SIZE) {
                x ^= PRIMITIVE_POLY;
            }
        }
        logTable[0] = 0;
    }

    private int gfMul(int a, int b) {
        if (a == 0 || b == 0) return 0;
        return expTable[logTable[a] + logTable[b]];
    }

    private int gfDiv(int a, int b) {
        if (b == 0) throw new ArithmeticException("Division by zero in GF");
        if (a == 0) return 0;
        return expTable[(logTable[a] - logTable[b] + (GF_SIZE - 1)) % (GF_SIZE - 1)];
    }

    private int gfInv(int a) {
        if (a == 0) throw new ArithmeticException("Cannot invert zero in GF");
        return expTable[(GF_SIZE - 1) - logTable[a]];
    }

    private int gfPow(int base, int exp) {
        if (exp == 0) return 1;
        int result = 1;
        for (int i = 0; i < exp; i++) {
            result = gfMul(result, base);
        }
        return result;
    }

    private int[][] buildVandermondeMatrix() {
        int[][] matrix = new int[totalShards][dataShards];

        for (int i = 0; i < dataShards; i++) {
            matrix[i][i] = 1;
        }

        for (int i = dataShards; i < totalShards; i++) {
            for (int j = 0; j < dataShards; j++) {
                int base = i - dataShards + 1;
                matrix[i][j] = gfPow(base, j);
            }
        }

        return matrix;
    }

    private int[][] invertMatrix(int[][] matrix) {
        int n = matrix.length;
        int[][] work = new int[n][n * 2];

        for (int i = 0; i < n; i++) {
            System.arraycopy(matrix[i], 0, work[i], 0, n);
            work[i][n + i] = 1;
        }

        for (int i = 0; i < n; i++) {
            if (work[i][i] == 0) {
                int swapRow = -1;
                for (int j = i + 1; j < n; j++) {
                    if (work[j][i] != 0) {
                        swapRow = j;
                        break;
                    }
                }
                if (swapRow == -1) {
                    throw new IllegalArgumentException("Matrix is not invertible");
                }
                int[] temp = work[i];
                work[i] = work[swapRow];
                work[swapRow] = temp;
            }

            int inv = gfInv(work[i][i]);
            for (int j = 0; j < 2 * n; j++) {
                work[i][j] = gfMul(work[i][j], inv);
            }

            for (int j = 0; j < n; j++) {
                if (j != i && work[j][i] != 0) {
                    int factor = work[j][i];
                    for (int k = 0; k < 2 * n; k++) {
                        work[j][k] ^= gfMul(factor, work[i][k]);
                    }
                }
            }
        }

        int[][] result = new int[n][n];
        for (int i = 0; i < n; i++) {
            System.arraycopy(work[i], n, result[i], 0, n);
        }

        return result;
    }

    public static ErasureCoder createStandard() {
        return new ErasureCoder(4, 2);
    }

    public static ErasureCoder createHighRedundancy() {
        return new ErasureCoder(6, 2);
    }

    public static ErasureCoder createExtremeRedundancy() {
        return new ErasureCoder(8, 4);
    }

    public static ErasureCoder createWithContext(NetworkContext context) {
        int optimalParity = context.getNetworkHealth() == Transport.HealthState.UNHEALTHY ? 4 : 2;
        return new ErasureCoder(4, optimalParity, context);
    }

    public int getDataShards() { return dataShards; }
    public int getParityShards() { return parityShards; }
    public int getTotalShards() { return totalShards; }
    public long getEncodeCount() { return encodeCount.get(); }
    public long getDecodeCount() { return decodeCount.get(); }
    public long getRepairCount() { return repairCount.get(); }
    public NetworkContext getNetworkContext() { return networkContext; }
    public void setNetworkContext(NetworkContext context) { this.networkContext = context; }

    public void setOnEncoding(Consumer<EncodingEvent> listener) { this.onEncoding = listener; }
    public void setOnDecoding(Consumer<DecodingEvent> listener) { this.onDecoding = listener; }
    public void setOnRepair(Consumer<RepairEvent> listener) { this.onRepair = listener; }

    public record EncodeResult(byte[][] shards, int shardSize, int originalSize) {}

    public record StreamEncodeResult(
        long totalBytesProcessed,
        int chunksProcessed,
        int averageShardSize,
        Instant completedAt
    ) {}

    public record VerificationResult(
        boolean valid,
        List<Integer> corruptedShardIndices,
        Instant verifiedAt
    ) {
        public int corruptedCount() { return corruptedShardIndices.size(); }
    }

    public record RepairResult(
        List<Integer> repairedShardIndices,
        int repairedCount,
        long bytesRepaired,
        long nanosecondsTaken,
        boolean success,
        Instant repairedAt
    ) {}

    public record EncodingEvent(
        long bytesProcessed,
        int dataShards,
        int parityShards,
        int shardSize,
        Instant timestamp
    ) {}

    public record DecodingEvent(
        long bytesProcessed,
        int presentShards,
        int totalShards,
        boolean hadLoss,
        Instant timestamp
    ) {}

    public record RepairEvent(
        int repairedCount,
        int[] repairedIndices,
        int shardSize,
        Instant timestamp
    ) {}

    public static class NetworkContext {
        private volatile Transport.HealthState networkHealth = Transport.HealthState.UNKNOWN;
        private volatile double averageLatency = 0;
        private volatile double packetLossRate = 0;
        private volatile long lastUpdate = 0;
        private final Map<String, PeerMetrics> peerMetrics = new ConcurrentHashMap<>();

        public NetworkContext() {}

        public Transport.HealthState getNetworkHealth() { return networkHealth; }
        public double getAverageLatency() { return averageLatency; }
        public double getPacketLossRate() { return packetLossRate; }
        public long getLastUpdate() { return lastUpdate; }
        public Map<String, PeerMetrics> getPeerMetrics() { return new ConcurrentHashMap<>(peerMetrics); }

        public void setNetworkHealth(Transport.HealthState health) { 
            this.networkHealth = health;
            this.lastUpdate = System.currentTimeMillis();
        }
        public void setAverageLatency(double latency) { this.averageLatency = latency; }
        public void setPacketLossRate(double rate) { this.packetLossRate = rate; }

        public void recordPeerSuccess(String peerId, long latencyMs) {
            peerMetrics.compute(peerId, (k, v) -> {
                if (v == null) {
                    return new PeerMetrics(1, 0, latencyMs, latencyMs, latencyMs, 1.0);
                }
                long newSuccess = v.successCount() + 1;
                long newAvgLatency = (v.avgLatency() * v.successCount() + latencyMs) / newSuccess;
                long newMinLatency = Math.min(v.minLatency(), latencyMs);
                long newMaxLatency = Math.max(v.maxLatency(), latencyMs);
                double newSuccessRate = (double) newSuccess / (newSuccess + v.failureCount());
                return new PeerMetrics(newSuccess, v.failureCount(), newAvgLatency, 
                    newMinLatency, newMaxLatency, newSuccessRate);
            });
        }

        public void recordPeerFailure(String peerId) {
            peerMetrics.compute(peerId, (k, v) -> {
                if (v == null) {
                    return new PeerMetrics(0, 1, 0, 0, Long.MAX_VALUE, 0.0);
                }
                long newFailures = v.failureCount() + 1;
                double newSuccessRate = (double) v.successCount() / (v.successCount() + newFailures);
                return new PeerMetrics(v.successCount(), newFailures, v.avgLatency(), 
                    v.minLatency(), v.maxLatency(), newSuccessRate);
            });
        }

        public double calculatePeerScore(String peerId) {
            PeerMetrics metrics = peerMetrics.get(peerId);
            if (metrics == null) return 50.0;

            double successScore = metrics.successRate() * 40;
            double latencyScore = Math.max(0, (1.0 - metrics.avgLatency() / 1000.0)) * 30;
            double consistencyScore = (metrics.maxLatency() > 0 
                ? 1.0 - ((metrics.maxLatency() - metrics.minLatency()) / (double) metrics.maxLatency()) 
                : 1.0) * 20;
            double recencyScore = 10;

            return successScore + latencyScore + consistencyScore + recencyScore;
        }

        public record PeerMetrics(
            long successCount,
            long failureCount,
            long avgLatency,
            long minLatency,
            long maxLatency,
            double successRate
        ) {}
    }

    public record PeerScore(
        String peerId,
        double score,
        long latency,
        Transport.HealthState health
    ) {}
}
