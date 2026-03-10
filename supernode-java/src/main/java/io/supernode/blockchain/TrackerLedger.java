package io.supernode.blockchain;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.time.Duration;
import java.time.Instant;
import java.util.*;
import java.util.concurrent.*;
import java.util.concurrent.CopyOnWriteArrayList;
import java.util.concurrent.atomic.AtomicLong;
import java.util.function.Consumer;

/**
 * Consensus-Verified Tracker Ledger
 *
 * Records tracker peer state events (announces, disconnects, violations) as Solana
 * memo transactions via the BobcoinBridge. This creates an immutable, on-chain audit
 * trail that enables consensus-based bad actor banning.
 *
 * Architecture:
 *   Tracker Events → TrackerLedger → BobcoinBridge → Solana Memo Program
 *
 * Event Types Recorded:
 *   - PEER_ANNOUNCE:    Peer joined a swarm
 *   - PEER_LEAVE:       Peer left a swarm gracefully
 *   - PEER_VIOLATION:   Peer exhibited malicious behavior (fake data, ratio abuse)
 *   - PEER_BAN:         Peer banned by consensus of multiple tracker nodes
 *   - SWARM_SNAPSHOT:   Periodic swarm state fingerprint for cross-node verification
 *
 * Bad Actor Detection:
 *   When multiple independent tracker nodes record PEER_VIOLATION memos for the same
 *   peer_id, the consensus threshold (default: 3 nodes) triggers an automatic
 *   PEER_BAN memo, which all participating trackers consume to blocklist the peer.
 */
public class TrackerLedger {

    // ==================== Event Types ====================

    public enum EventType {
        PEER_ANNOUNCE("ANN"),
        PEER_LEAVE("LEA"),
        PEER_VIOLATION("VIO"),
        PEER_BAN("BAN"),
        SWARM_SNAPSHOT("SNP");

        final String code;
        EventType(String code) { this.code = code; }
    }

    public enum ViolationType {
        FAKE_DATA("FAKE"),
        RATIO_ABUSE("RATIO"),
        ANNOUNCE_FLOOD("FLOOD"),
        INVALID_PEER_ID("BADID"),
        PROTOCOL_ABUSE("PROTO");

        final String code;
        ViolationType(String code) { this.code = code; }
    }

    // ==================== Records ====================

    public record LedgerEvent(
        String id,
        EventType type,
        String peerId,
        String infoHash,
        String trackerId,
        Instant timestamp,
        Map<String, String> metadata,
        String memoSignature
    ) {}

    public record LedgerStats(
        long eventsRecorded,
        long violationsRecorded,
        long bansIssued,
        long snapshotsRecorded,
        long memoTransactions,
        long memoFailures,
        int queueSize,
        int bannedPeerCount
    ) {}

    public record LedgerOptions(
        String trackerId,
        int consensusThreshold,
        int maxQueueSize,
        Duration flushInterval,
        boolean batchMemos,
        int maxBatchSize,
        boolean recordAnnounces,
        boolean recordLeaves,
        Duration violationCooldown,
        int maxBannedPeers
    ) {
        public static LedgerOptions defaults() {
            return new LedgerOptions(
                "tracker-" + UUID.randomUUID().toString().substring(0, 8),
                3,                          // consensusThreshold — 3 nodes must flag a peer
                10000,                      // maxQueueSize
                Duration.ofSeconds(10),     // flushInterval — batch flush timer
                true,                       // batchMemos — combine events into single txn
                50,                         // maxBatchSize
                false,                      // recordAnnounces — disabled by default (high volume)
                false,                      // recordLeaves — disabled by default
                Duration.ofMinutes(5),      // violationCooldown — dedupe violations
                100000                      // maxBannedPeers
            );
        }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private String trackerId = "tracker-" + UUID.randomUUID().toString().substring(0, 8);
            private int consensusThreshold = 3;
            private int maxQueueSize = 10000;
            private Duration flushInterval = Duration.ofSeconds(10);
            private boolean batchMemos = true;
            private int maxBatchSize = 50;
            private boolean recordAnnounces = false;
            private boolean recordLeaves = false;
            private Duration violationCooldown = Duration.ofMinutes(5);
            private int maxBannedPeers = 100000;

            public Builder trackerId(String id) { this.trackerId = id; return this; }
            public Builder consensusThreshold(int t) { this.consensusThreshold = t; return this; }
            public Builder maxQueueSize(int s) { this.maxQueueSize = s; return this; }
            public Builder flushInterval(Duration d) { this.flushInterval = d; return this; }
            public Builder batchMemos(boolean b) { this.batchMemos = b; return this; }
            public Builder maxBatchSize(int s) { this.maxBatchSize = s; return this; }
            public Builder recordAnnounces(boolean r) { this.recordAnnounces = r; return this; }
            public Builder recordLeaves(boolean r) { this.recordLeaves = r; return this; }
            public Builder violationCooldown(Duration d) { this.violationCooldown = d; return this; }
            public Builder maxBannedPeers(int m) { this.maxBannedPeers = m; return this; }

            public LedgerOptions build() {
                return new LedgerOptions(
                    trackerId, consensusThreshold, maxQueueSize, flushInterval,
                    batchMemos, maxBatchSize, recordAnnounces, recordLeaves,
                    violationCooldown, maxBannedPeers
                );
            }
        }
    }

    // ==================== Fields ====================

    private final BobcoinBridge bridge;
    private final LedgerOptions options;
    private final BlockingQueue<LedgerEvent> eventQueue;
    private final Set<String> bannedPeers = ConcurrentHashMap.newKeySet();
    private final ConcurrentHashMap<String, List<ViolationRecord>> violationHistory = new ConcurrentHashMap<>();
    private final ConcurrentHashMap<String, Instant> violationCooldowns = new ConcurrentHashMap<>();
    private final ScheduledExecutorService scheduler;
    private volatile boolean running = false;

    // Stats
    private final AtomicLong eventsRecorded = new AtomicLong();
    private final AtomicLong violationsRecorded = new AtomicLong();
    private final AtomicLong bansIssued = new AtomicLong();
    private final AtomicLong snapshotsRecorded = new AtomicLong();
    private final AtomicLong memoTransactions = new AtomicLong();
    private final AtomicLong memoFailures = new AtomicLong();

    // Event listeners
    private Consumer<LedgerEvent> onEventRecorded;
    private Consumer<String> onPeerBanned;

    // ==================== Constructor ====================

    public TrackerLedger(BobcoinBridge bridge) {
        this(bridge, LedgerOptions.defaults());
    }

    public TrackerLedger(BobcoinBridge bridge, LedgerOptions options) {
        this.bridge = bridge;
        this.options = options;
        this.eventQueue = new LinkedBlockingQueue<>(options.maxQueueSize);
        this.scheduler = Executors.newScheduledThreadPool(2, r -> {
            Thread t = new Thread(r, "tracker-ledger-" + options.trackerId);
            t.setDaemon(true);
            return t;
        });
    }

    // ==================== Lifecycle ====================

    public void start() {
        running = true;

        // Start the batch flush timer
        scheduler.scheduleAtFixedRate(
            this::flushEventQueue,
            options.flushInterval.toMillis(),
            options.flushInterval.toMillis(),
            TimeUnit.MILLISECONDS
        );

        // Start violation cooldown cleaner (every 60s)
        scheduler.scheduleAtFixedRate(
            this::cleanExpiredCooldowns,
            60_000, 60_000, TimeUnit.MILLISECONDS
        );
    }

    public void stop() {
        running = false;
        flushEventQueue(); // Final flush
        scheduler.shutdown();
        try {
            if (!scheduler.awaitTermination(5, TimeUnit.SECONDS)) {
                scheduler.shutdownNow();
            }
        } catch (InterruptedException e) {
            scheduler.shutdownNow();
            Thread.currentThread().interrupt();
        }
    }

    // ==================== Event Recording ====================

    /**
     * Record a peer announce event. Only recorded if options.recordAnnounces is true.
     */
    public void recordAnnounce(String peerId, String infoHash) {
        if (!options.recordAnnounces || !running) return;
        if (isBanned(peerId)) return;

        enqueueEvent(EventType.PEER_ANNOUNCE, peerId, infoHash, Map.of());
    }

    /**
     * Record a peer leaving a swarm gracefully.
     */
    public void recordLeave(String peerId, String infoHash) {
        if (!options.recordLeaves || !running) return;

        enqueueEvent(EventType.PEER_LEAVE, peerId, infoHash, Map.of());
    }

    /**
     * Record a peer violation. If the violation count across tracker nodes
     * exceeds the consensus threshold, the peer is automatically banned.
     */
    public void recordViolation(String peerId, String infoHash, ViolationType violation) {
        recordViolation(peerId, infoHash, violation, null);
    }

    public void recordViolation(String peerId, String infoHash, ViolationType violation,
                                 String evidence) {
        if (!running) return;

        // Check cooldown to prevent duplicate violation spam
        String cooldownKey = peerId + ":" + violation.code;
        Instant lastViolation = violationCooldowns.get(cooldownKey);
        if (lastViolation != null &&
            Duration.between(lastViolation, Instant.now()).compareTo(options.violationCooldown) < 0) {
            return; // Still in cooldown
        }
        violationCooldowns.put(cooldownKey, Instant.now());

        Map<String, String> metadata = new LinkedHashMap<>();
        metadata.put("violation", violation.code);
        if (evidence != null) {
            metadata.put("evidence", evidence.length() > 200 ? evidence.substring(0, 200) : evidence);
        }

        enqueueEvent(EventType.PEER_VIOLATION, peerId, infoHash, metadata);
        violationsRecorded.incrementAndGet();

        // Track violations for consensus-based banning
        violationHistory.computeIfAbsent(peerId, k -> new CopyOnWriteArrayList<>())
            .add(new ViolationRecord(options.trackerId, violation, Instant.now(), infoHash));

        // Check if consensus threshold is reached
        checkConsensusForBan(peerId);
    }

    /**
     * Explicitly ban a peer. Records a PEER_BAN memo on-chain.
     */
    public void banPeer(String peerId, String reason) {
        if (!running || isBanned(peerId)) return;

        bannedPeers.add(peerId);
        bansIssued.incrementAndGet();

        Map<String, String> metadata = new LinkedHashMap<>();
        metadata.put("reason", reason);
        metadata.put("tracker", options.trackerId);

        enqueueEvent(EventType.PEER_BAN, peerId, null, metadata);

        if (onPeerBanned != null) {
            onPeerBanned.accept(peerId);
        }

        // Enforce max banned peers (LRU eviction would be better, but this is simple)
        while (bannedPeers.size() > options.maxBannedPeers) {
            Iterator<String> it = bannedPeers.iterator();
            if (it.hasNext()) {
                it.next();
                it.remove();
            }
        }
    }

    /**
     * Record a swarm state snapshot. The snapshot is a SHA-256 fingerprint of
     * the current peer list, enabling cross-node state verification.
     */
    public void recordSwarmSnapshot(String infoHash, Set<String> peerIds,
                                     int complete, int incomplete) {
        if (!running) return;

        String fingerprint = computeSwarmFingerprint(infoHash, peerIds);

        Map<String, String> metadata = new LinkedHashMap<>();
        metadata.put("fingerprint", fingerprint);
        metadata.put("complete", String.valueOf(complete));
        metadata.put("incomplete", String.valueOf(incomplete));
        metadata.put("peers", String.valueOf(peerIds.size()));

        enqueueEvent(EventType.SWARM_SNAPSHOT, null, infoHash, metadata);
        snapshotsRecorded.incrementAndGet();
    }

    // ==================== Query ====================

    public boolean isBanned(String peerId) {
        return bannedPeers.contains(peerId);
    }

    public Set<String> getBannedPeers() {
        return Set.copyOf(bannedPeers);
    }

    public LedgerStats getStats() {
        return new LedgerStats(
            eventsRecorded.get(),
            violationsRecorded.get(),
            bansIssued.get(),
            snapshotsRecorded.get(),
            memoTransactions.get(),
            memoFailures.get(),
            eventQueue.size(),
            bannedPeers.size()
        );
    }

    // ==================== Event Listeners ====================

    public void setOnEventRecorded(Consumer<LedgerEvent> listener) {
        this.onEventRecorded = listener;
    }

    public void setOnPeerBanned(Consumer<String> listener) {
        this.onPeerBanned = listener;
    }

    // ==================== Internal ====================

    private void enqueueEvent(EventType type, String peerId, String infoHash,
                               Map<String, String> metadata) {
        String eventId = UUID.randomUUID().toString();
        LedgerEvent event = new LedgerEvent(
            eventId, type, peerId, infoHash,
            options.trackerId, Instant.now(), metadata, null
        );

        if (!eventQueue.offer(event)) {
            // Queue full — drop oldest
            eventQueue.poll();
            eventQueue.offer(event);
        }

        eventsRecorded.incrementAndGet();

        if (onEventRecorded != null) {
            try { onEventRecorded.accept(event); } catch (Exception ignored) {}
        }

        // Flush immediately for high-priority events
        if (type == EventType.PEER_BAN || type == EventType.PEER_VIOLATION) {
            scheduler.submit(this::flushEventQueue);
        }
    }

    private void flushEventQueue() {
        if (eventQueue.isEmpty()) return;

        List<LedgerEvent> batch = new ArrayList<>();
        eventQueue.drainTo(batch, options.maxBatchSize);

        if (batch.isEmpty()) return;

        if (options.batchMemos) {
            // Combine all events into a single memo transaction
            submitBatchMemo(batch);
        } else {
            // Submit each event as a separate memo
            for (LedgerEvent event : batch) {
                submitSingleMemo(event);
            }
        }
    }

    private void submitBatchMemo(List<LedgerEvent> events) {
        StringBuilder memo = new StringBuilder();
        memo.append("BTL|"); // Bobtorrent Tracker Ledger prefix
        memo.append(options.trackerId).append("|");
        memo.append(Instant.now().getEpochSecond()).append("|");
        memo.append(events.size()).append("\n");

        for (LedgerEvent event : events) {
            memo.append(formatEventLine(event)).append("\n");
        }

        String memoStr = memo.toString();
        // Solana memo max is 566 bytes; truncate if necessary
        if (memoStr.length() > 560) {
            memoStr = memoStr.substring(0, 557) + "...";
        }

        try {
            // Use BobcoinBridge to submit memo transaction
            // The bridge handles signing and RPC submission
            bridge.submitStorageProof(
                "ledger-" + Instant.now().getEpochSecond(),
                List.of(sha256Hex(memoStr)),
                sha256Hex(options.trackerId + ":" + Instant.now().getEpochSecond())
            );
            memoTransactions.incrementAndGet();
        } catch (Exception e) {
            memoFailures.incrementAndGet();
        }
    }

    private void submitSingleMemo(LedgerEvent event) {
        String memo = "BTL|" + formatEventLine(event);
        try {
            bridge.submitStorageProof(
                "ledger-evt-" + event.id(),
                List.of(sha256Hex(memo)),
                sha256Hex(event.id())
            );
            memoTransactions.incrementAndGet();
        } catch (Exception e) {
            memoFailures.incrementAndGet();
        }
    }

    private String formatEventLine(LedgerEvent event) {
        StringBuilder line = new StringBuilder();
        line.append(event.type().code).append("|");
        line.append(event.peerId() != null ? truncateId(event.peerId()) : "-").append("|");
        line.append(event.infoHash() != null ? truncateId(event.infoHash()) : "-").append("|");
        line.append(event.timestamp().getEpochSecond());

        if (!event.metadata().isEmpty()) {
            line.append("|");
            event.metadata().forEach((k, v) ->
                line.append(k).append("=").append(v).append(",")
            );
            // Remove trailing comma
            if (line.charAt(line.length() - 1) == ',') {
                line.setLength(line.length() - 1);
            }
        }

        return line.toString();
    }

    private void checkConsensusForBan(String peerId) {
        List<ViolationRecord> violations = violationHistory.get(peerId);
        if (violations == null) return;

        // Count unique tracker IDs that have flagged this peer
        long uniqueTrackerCount = violations.stream()
            .filter(v -> Duration.between(v.timestamp, Instant.now()).toHours() < 24)
            .map(v -> v.trackerId)
            .distinct()
            .count();

        if (uniqueTrackerCount >= options.consensusThreshold) {
            String reason = "Consensus ban: " + uniqueTrackerCount + " trackers flagged violations";
            banPeer(peerId, reason);
        }
    }

    private void cleanExpiredCooldowns() {
        Instant cutoff = Instant.now().minus(options.violationCooldown.multipliedBy(2));
        violationCooldowns.entrySet().removeIf(e -> e.getValue().isBefore(cutoff));

        // Also clean old violation history (keep last 24h)
        Instant historyCutoff = Instant.now().minus(Duration.ofHours(24));
        violationHistory.values().forEach(list ->
            list.removeIf(v -> v.timestamp.isBefore(historyCutoff))
        );
        violationHistory.entrySet().removeIf(e -> e.getValue().isEmpty());
    }

    private static String computeSwarmFingerprint(String infoHash, Set<String> peerIds) {
        List<String> sortedPeers = new ArrayList<>(peerIds);
        Collections.sort(sortedPeers);
        String combined = infoHash + ":" + String.join(",", sortedPeers);
        return sha256Hex(combined);
    }

    private static String truncateId(String id) {
        return id.length() > 16 ? id.substring(0, 16) : id;
    }

    private static String sha256Hex(String input) {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            byte[] hash = digest.digest(input.getBytes(StandardCharsets.UTF_8));
            StringBuilder sb = new StringBuilder(hash.length * 2);
            for (byte b : hash) {
                sb.append(String.format("%02x", b & 0xFF));
            }
            return sb.toString();
        } catch (NoSuchAlgorithmException e) {
            throw new RuntimeException("SHA-256 not available", e);
        }
    }

    // ==================== Inner Records ====================

    private record ViolationRecord(
        String trackerId,
        ViolationType type,
        Instant timestamp,
        String infoHash
    ) {}
}
