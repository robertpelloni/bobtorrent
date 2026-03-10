package io.supernode.network;

import io.supernode.storage.BlobStore;
import io.supernode.storage.SupernodeStorage;

import java.nio.ByteBuffer;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.time.Duration;
import java.time.Instant;
import java.util.*;
import java.util.concurrent.*;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicLong;
import java.util.function.Consumer;

/**
 * Embedded Game Asset Streaming
 *
 * Real-time P2P game asset delivery optimized for interactive applications.
 * Supports Level-of-Detail (LOD) progressive loading, priority-based chunk
 * fetching, and adaptive bitrate based on network conditions.
 *
 * Design for game engines:
 *   - Assets are registered with LOD levels (0=highest, N=lowest detail)
 *   - Chunk fetch priority is based on camera distance + LOD level
 *   - Critical assets (shaders, textures in view) get CRITICAL priority
 *   - Background prefetch for likely-needed assets
 *   - Streaming stats feed back to the game engine for LOD decisions
 *
 * Integration:
 *   GameAssetStreamer → SupernodeStorage (encrypted chunk retrieval)
 *                    → BlobNetwork (P2P fetching)
 *                    → SwarmCoordinator (peer management)
 */
public class GameAssetStreamer {

    // ==================== Enums & Records ====================

    /** Level of Detail for progressive asset loading. */
    public enum LODLevel {
        LOD_0(0, "Ultra"),      // Full quality
        LOD_1(1, "High"),
        LOD_2(2, "Medium"),
        LOD_3(3, "Low"),
        LOD_4(4, "Placeholder"); // Placeholder/silhouette

        final int level;
        final String label;
        LODLevel(int level, String label) {
            this.level = level;
            this.label = label;
        }
    }

    /** Priority for chunk fetching. Lower = more urgent. */
    public enum FetchPriority {
        CRITICAL(0),    // Required NOW (shader, active texture)
        HIGH(1),        // Needed very soon (adjacent chunks)
        NORMAL(2),      // Standard prefetch
        LOW(3),         // Background prefetch
        SPECULATIVE(4); // Might be needed later

        final int level;
        FetchPriority(int level) { this.level = level; }
    }

    /** A registered game asset with LOD chain. */
    public record GameAsset(
        String assetId,
        String name,
        AssetType type,
        Map<LODLevel, String> lodFileIds,  // LOD → fileId in storage
        Map<LODLevel, Long> lodSizes,      // LOD → byte size
        long totalSize,
        Set<String> tags
    ) {}

    /** Type of game asset. */
    public enum AssetType {
        TEXTURE, MODEL, SHADER, AUDIO, ANIMATION, LEVEL_DATA, UI, VIDEO, SCRIPT, OTHER
    }

    /** A chunk fetch request in the priority queue. */
    public record FetchRequest(
        String assetId,
        String fileId,
        LODLevel lod,
        FetchPriority priority,
        Instant requestedAt,
        Consumer<byte[]> callback
    ) implements Comparable<FetchRequest> {
        @Override
        public int compareTo(FetchRequest other) {
            int cmp = Integer.compare(this.priority.level, other.priority.level);
            if (cmp != 0) return cmp;
            return Integer.compare(this.lod.level, other.lod.level);
        }
    }

    /** Streaming session statistics. */
    public record StreamingStats(
        long assetsRegistered,
        long assetsLoaded,
        long assetsStreaming,
        long bytesStreamed,
        long cacheHits,
        long cacheMisses,
        long fetchesPending,
        double avgFetchLatencyMs,
        double throughputBytesPerSec,
        Map<LODLevel, Long> lodLoadCounts
    ) {}

    /** Streaming session configuration. */
    public record StreamerConfig(
        int maxConcurrentFetches,
        int prefetchAheadCount,
        Duration fetchTimeout,
        long maxCacheSizeBytes,
        boolean enableAdaptiveLOD,
        double targetFrameRate,
        long minBandwidthForLOD0
    ) {
        public static StreamerConfig defaults() {
            return new StreamerConfig(
                8,                          // maxConcurrentFetches
                5,                          // prefetchAheadCount
                Duration.ofSeconds(10),     // fetchTimeout
                512 * 1024 * 1024L,         // maxCacheSizeBytes: 512MB
                true,                       // enableAdaptiveLOD
                60.0,                       // targetFrameRate
                10 * 1024 * 1024L           // minBandwidthForLOD0: 10MB/s
            );
        }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private int maxConcurrentFetches = 8;
            private int prefetchAheadCount = 5;
            private Duration fetchTimeout = Duration.ofSeconds(10);
            private long maxCacheSizeBytes = 512 * 1024 * 1024L;
            private boolean enableAdaptiveLOD = true;
            private double targetFrameRate = 60.0;
            private long minBandwidthForLOD0 = 10 * 1024 * 1024L;

            public Builder maxConcurrentFetches(int m) { this.maxConcurrentFetches = m; return this; }
            public Builder prefetchAheadCount(int p) { this.prefetchAheadCount = p; return this; }
            public Builder fetchTimeout(Duration t) { this.fetchTimeout = t; return this; }
            public Builder maxCacheSizeBytes(long s) { this.maxCacheSizeBytes = s; return this; }
            public Builder enableAdaptiveLOD(boolean e) { this.enableAdaptiveLOD = e; return this; }
            public Builder targetFrameRate(double r) { this.targetFrameRate = r; return this; }
            public Builder minBandwidthForLOD0(long b) { this.minBandwidthForLOD0 = b; return this; }

            public StreamerConfig build() {
                return new StreamerConfig(
                    maxConcurrentFetches, prefetchAheadCount, fetchTimeout,
                    maxCacheSizeBytes, enableAdaptiveLOD, targetFrameRate, minBandwidthForLOD0
                );
            }
        }
    }

    // ==================== Fields ====================

    private final StreamerConfig config;
    private final BlobStore blobStore;

    // Asset registry
    private final ConcurrentHashMap<String, GameAsset> assets = new ConcurrentHashMap<>();

    // Priority fetch queue
    private final PriorityBlockingQueue<FetchRequest> fetchQueue = new PriorityBlockingQueue<>();

    // Asset loading state
    private final ConcurrentHashMap<String, AssetLoadState> loadStates = new ConcurrentHashMap<>();

    // In-memory asset cache (assetId:lod → data)
    private final ConcurrentHashMap<String, byte[]> assetCache = new ConcurrentHashMap<>();
    private final AtomicLong cacheSizeBytes = new AtomicLong();

    // Fetch worker pool
    private final ExecutorService fetchPool;
    private final ScheduledExecutorService scheduler;
    private volatile boolean running = false;

    // Bandwidth estimation
    private final AtomicLong bytesStreamedLastSecond = new AtomicLong();
    private volatile long estimatedBandwidth = 0;

    // Stats
    private final AtomicLong totalBytesStreamed = new AtomicLong();
    private final AtomicLong cacheHits = new AtomicLong();
    private final AtomicLong cacheMisses = new AtomicLong();
    private final AtomicLong assetsLoaded = new AtomicLong();
    private final AtomicLong totalFetchLatencyMs = new AtomicLong();
    private final AtomicLong fetchCount = new AtomicLong();
    private final ConcurrentHashMap<LODLevel, AtomicLong> lodLoadCounts = new ConcurrentHashMap<>();

    // Events
    private Consumer<GameAsset> onAssetReady;
    private Consumer<String> onAssetFailed;

    // ==================== Constructor ====================

    public GameAssetStreamer(BlobStore blobStore) {
        this(blobStore, StreamerConfig.defaults());
    }

    public GameAssetStreamer(BlobStore blobStore, StreamerConfig config) {
        this.blobStore = blobStore;
        this.config = config;
        this.fetchPool = Executors.newFixedThreadPool(config.maxConcurrentFetches, r -> {
            Thread t = new Thread(r, "asset-fetch");
            t.setDaemon(true);
            return t;
        });
        this.scheduler = Executors.newScheduledThreadPool(1, r -> {
            Thread t = new Thread(r, "asset-scheduler");
            t.setDaemon(true);
            return t;
        });

        // Initialize LOD counters
        for (LODLevel lod : LODLevel.values()) {
            lodLoadCounts.put(lod, new AtomicLong());
        }
    }

    // ==================== Lifecycle ====================

    public void start() {
        running = true;

        // Start fetch workers
        for (int i = 0; i < config.maxConcurrentFetches; i++) {
            fetchPool.submit(this::fetchWorker);
        }

        // Bandwidth estimator — samples every second
        scheduler.scheduleAtFixedRate(() -> {
            estimatedBandwidth = bytesStreamedLastSecond.getAndSet(0);
        }, 1000, 1000, TimeUnit.MILLISECONDS);
    }

    public void stop() {
        running = false;
        fetchPool.shutdown();
        scheduler.shutdown();
    }

    // ==================== Asset Registration ====================

    /**
     * Register a game asset with its LOD chain.
     */
    public void registerAsset(String assetId, String name, AssetType type,
                               Map<LODLevel, String> lodFileIds,
                               Map<LODLevel, Long> lodSizes,
                               Set<String> tags) {
        long totalSize = lodSizes.values().stream().mapToLong(Long::longValue).sum();
        GameAsset asset = new GameAsset(assetId, name, type, lodFileIds, lodSizes, totalSize, tags);
        assets.put(assetId, asset);
        loadStates.put(assetId, new AssetLoadState(assetId));
    }

    /**
     * Register a simple asset (single LOD).
     */
    public void registerAsset(String assetId, String name, AssetType type,
                               String fileId, long sizeBytes) {
        registerAsset(assetId, name, type,
            Map.of(LODLevel.LOD_0, fileId),
            Map.of(LODLevel.LOD_0, sizeBytes),
            Set.of()
        );
    }

    public void unregisterAsset(String assetId) {
        assets.remove(assetId);
        loadStates.remove(assetId);
        // Clear cache entries for this asset
        assetCache.keySet().removeIf(key -> key.startsWith(assetId + ":"));
    }

    // ==================== Asset Fetching ====================

    /**
     * Request an asset at a specific LOD level.
     * Returns a Future that resolves when the asset data is available.
     */
    public CompletableFuture<byte[]> requestAsset(String assetId, LODLevel lod,
                                                    FetchPriority priority) {
        // Check cache first
        String cacheKey = assetId + ":" + lod.level;
        byte[] cached = assetCache.get(cacheKey);
        if (cached != null) {
            cacheHits.incrementAndGet();
            return CompletableFuture.completedFuture(cached);
        }
        cacheMisses.incrementAndGet();

        GameAsset asset = assets.get(assetId);
        if (asset == null) {
            return CompletableFuture.failedFuture(
                new IllegalArgumentException("Unknown asset: " + assetId));
        }

        String fileId = asset.lodFileIds().get(lod);
        if (fileId == null) {
            // Fall back to next available LOD
            LODLevel fallbackLod = findClosestLOD(asset, lod);
            if (fallbackLod == null) {
                return CompletableFuture.failedFuture(
                    new IllegalStateException("No LOD available for asset: " + assetId));
            }
            fileId = asset.lodFileIds().get(fallbackLod);
            lod = fallbackLod;
        }

        CompletableFuture<byte[]> future = new CompletableFuture<>();
        final LODLevel finalLod = lod;
        fetchQueue.offer(new FetchRequest(assetId, fileId, finalLod, priority,
            Instant.now(), data -> future.complete(data)));

        return future;
    }

    /**
     * Request an asset with automatic LOD selection based on bandwidth.
     */
    public CompletableFuture<byte[]> requestAssetAdaptive(String assetId,
                                                           FetchPriority priority) {
        LODLevel bestLod = selectBestLOD(assetId);
        return requestAsset(assetId, bestLod, priority);
    }

    /**
     * Prefetch assets that might be needed soon (e.g., adjacent game areas).
     */
    public void prefetch(List<String> assetIds) {
        for (String assetId : assetIds) {
            LODLevel lod = config.enableAdaptiveLOD ? selectBestLOD(assetId) : LODLevel.LOD_2;
            requestAsset(assetId, lod, FetchPriority.SPECULATIVE);
        }
    }

    /**
     * Prefetch all assets matching a tag (e.g., "level_2", "character_models").
     */
    public void prefetchByTag(String tag) {
        List<String> matching = assets.values().stream()
            .filter(a -> a.tags().contains(tag))
            .map(GameAsset::assetId)
            .toList();
        prefetch(matching);
    }

    // ==================== Cache Management ====================

    /**
     * Check if an asset is cached at a specific LOD.
     */
    public boolean isCached(String assetId, LODLevel lod) {
        return assetCache.containsKey(assetId + ":" + lod.level);
    }

    /**
     * Get current cache size in bytes.
     */
    public long getCacheSizeBytes() {
        return cacheSizeBytes.get();
    }

    /**
     * Clear all cached asset data.
     */
    public void clearCache() {
        assetCache.clear();
        cacheSizeBytes.set(0);
    }

    /**
     * Evict least-recently-used entries to fit within cache limit.
     */
    private void evictCacheIfNeeded(long newEntrySize) {
        while (cacheSizeBytes.get() + newEntrySize > config.maxCacheSizeBytes
               && !assetCache.isEmpty()) {
            // Simple eviction: remove first entry
            // In production, use an LRU order
            Iterator<Map.Entry<String, byte[]>> it = assetCache.entrySet().iterator();
            if (it.hasNext()) {
                Map.Entry<String, byte[]> entry = it.next();
                cacheSizeBytes.addAndGet(-entry.getValue().length);
                it.remove();
            }
        }
    }

    // ==================== LOD Selection ====================

    /**
     * Select the best LOD level for an asset based on current bandwidth.
     */
    private LODLevel selectBestLOD(String assetId) {
        if (!config.enableAdaptiveLOD) return LODLevel.LOD_0;

        GameAsset asset = assets.get(assetId);
        if (asset == null) return LODLevel.LOD_2;

        if (estimatedBandwidth >= config.minBandwidthForLOD0) {
            return LODLevel.LOD_0;
        } else if (estimatedBandwidth >= config.minBandwidthForLOD0 / 2) {
            return findClosestLOD(asset, LODLevel.LOD_1);
        } else if (estimatedBandwidth >= config.minBandwidthForLOD0 / 4) {
            return findClosestLOD(asset, LODLevel.LOD_2);
        } else {
            return findClosestLOD(asset, LODLevel.LOD_3);
        }
    }

    private LODLevel findClosestLOD(GameAsset asset, LODLevel target) {
        // Try target first, then go lower quality
        for (int i = target.level; i <= LODLevel.LOD_4.level; i++) {
            LODLevel lod = LODLevel.values()[i];
            if (asset.lodFileIds().containsKey(lod)) return lod;
        }
        // If nothing lower, try higher quality
        for (int i = target.level - 1; i >= 0; i--) {
            LODLevel lod = LODLevel.values()[i];
            if (asset.lodFileIds().containsKey(lod)) return lod;
        }
        return null;
    }

    // ==================== Stats & Events ====================

    public StreamingStats getStats() {
        Map<LODLevel, Long> lodCounts = new EnumMap<>(LODLevel.class);
        lodLoadCounts.forEach((lod, count) -> lodCounts.put(lod, count.get()));

        double avgLatency = fetchCount.get() > 0
            ? (double) totalFetchLatencyMs.get() / fetchCount.get() : 0.0;

        return new StreamingStats(
            assets.size(),
            assetsLoaded.get(),
            fetchQueue.size(),
            totalBytesStreamed.get(),
            cacheHits.get(),
            cacheMisses.get(),
            fetchQueue.size(),
            avgLatency,
            estimatedBandwidth,
            lodCounts
        );
    }

    public long getEstimatedBandwidth() {
        return estimatedBandwidth;
    }

    public void setOnAssetReady(Consumer<GameAsset> listener) { this.onAssetReady = listener; }
    public void setOnAssetFailed(Consumer<String> listener) { this.onAssetFailed = listener; }

    // ==================== Internal — Fetch Worker ====================

    private void fetchWorker() {
        while (running) {
            try {
                FetchRequest request = fetchQueue.poll(1, TimeUnit.SECONDS);
                if (request == null) continue;

                Instant fetchStart = Instant.now();

                try {
                    // Fetch from blob store (returns Optional<byte[]>)
                    byte[] data = blobStore.getAsync(request.fileId())
                        .get(config.fetchTimeout.toMillis(), TimeUnit.MILLISECONDS)
                        .orElse(null);

                    if (data != null) {
                        // Cache the result
                        String cacheKey = request.assetId() + ":" + request.lod().level;
                        evictCacheIfNeeded(data.length);
                        assetCache.put(cacheKey, data);
                        cacheSizeBytes.addAndGet(data.length);

                        // Update stats
                        long latencyMs = Duration.between(fetchStart, Instant.now()).toMillis();
                        totalBytesStreamed.addAndGet(data.length);
                        bytesStreamedLastSecond.addAndGet(data.length);
                        totalFetchLatencyMs.addAndGet(latencyMs);
                        fetchCount.incrementAndGet();
                        assetsLoaded.incrementAndGet();
                        lodLoadCounts.get(request.lod()).incrementAndGet();

                        // Update load state
                        AssetLoadState state = loadStates.get(request.assetId());
                        if (state != null) {
                            state.loadedLODs.add(request.lod());
                        }

                        // Notify callback
                        if (request.callback() != null) {
                            request.callback().accept(data);
                        }

                        // Fire event
                        if (onAssetReady != null) {
                            GameAsset asset = assets.get(request.assetId());
                            if (asset != null) onAssetReady.accept(asset);
                        }
                    }
                } catch (TimeoutException e) {
                    if (request.callback() != null) {
                        request.callback().accept(null);
                    }
                    if (onAssetFailed != null) {
                        onAssetFailed.accept(request.assetId());
                    }
                } catch (Exception e) {
                    if (onAssetFailed != null) {
                        onAssetFailed.accept(request.assetId());
                    }
                }
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                break;
            }
        }
    }

    // ==================== Inner Classes ====================

    private static class AssetLoadState {
        final String assetId;
        final Set<LODLevel> loadedLODs = ConcurrentHashMap.newKeySet();

        AssetLoadState(String assetId) {
            this.assetId = assetId;
        }
    }
}
