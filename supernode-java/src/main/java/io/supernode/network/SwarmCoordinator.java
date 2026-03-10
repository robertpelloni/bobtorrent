package io.supernode.network;

import io.supernode.network.transport.TransportAddress;
import io.supernode.network.transport.TransportType;

import java.time.Duration;
import java.time.Instant;
import java.util.*;
import java.util.concurrent.*;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicLong;
import java.util.function.Consumer;
import java.util.stream.Collectors;

/**
 * Multi-Swarm Peer Coordinator
 *
 * Manages 1000+ concurrent torrent swarms with efficient peer sharing, resource
 * allocation, and bandwidth throttling. Peers appearing in multiple swarms are
 * tracked using a unified peer registry to maximize connection reuse and minimize
 * overhead.
 *
 * Architecture:
 *   SwarmCoordinator → PeerRegistry (shared) → SwarmState (per-swarm)
 *   ↓                   ↓
 *   BandwidthAllocator  PeerScorer
 *
 * Features:
 *   - O(1) swarm lookup and peer insertion via ConcurrentHashMap
 *   - Cross-swarm peer sharing (peers in multiple swarms share connections)
 *   - Priority-based bandwidth allocation across swarms
 *   - Automatic garbage collection of inactive swarms
 *   - Per-swarm and aggregate statistics
 */
public class SwarmCoordinator {

    // ==================== Records ====================

    public enum SwarmPriority {
        CRITICAL(0),    // System files, boot assets
        HIGH(1),        // Active user downloads
        NORMAL(2),      // Standard torrents
        LOW(3),         // Background seeding
        IDLE(4);        // Rarely accessed

        final int level;
        SwarmPriority(int level) { this.level = level; }
    }

    public record SwarmInfo(
        String infoHash,
        SwarmPriority priority,
        int peerCount,
        int seeders,
        int leechers,
        long bytesUploaded,
        long bytesDownloaded,
        Instant createdAt,
        Instant lastActivity,
        boolean active
    ) {}

    public record PeerState(
        String peerId,
        TransportAddress address,
        Set<String> swarms,         // All swarms this peer participates in
        long bytesUploaded,
        long bytesDownloaded,
        double reliability,         // 0.0 - 1.0
        Instant connectedAt,
        Instant lastSeenAt,
        boolean choking,
        boolean interested
    ) {}

    public record CoordinatorStats(
        int activeSwarms,
        int totalPeers,
        int sharedPeers,          // Peers in 2+ swarms
        long totalBytesUp,
        long totalBytesDown,
        int peakSwarms,
        int peakPeers,
        double avgPeersPerSwarm,
        long evictedSwarms
    ) {}

    public record CoordinatorConfig(
        int maxSwarms,
        int maxPeersPerSwarm,
        int maxTotalPeers,
        Duration swarmIdleTimeout,
        Duration peerStaleTimeout,
        Duration cleanupInterval,
        long maxBandwidthBytesPerSecond,
        boolean enableCrossSwarmSharing,
        boolean enableBandwidthThrottling
    ) {
        public static CoordinatorConfig defaults() {
            return new CoordinatorConfig(
                5000,                       // maxSwarms
                200,                        // maxPeersPerSwarm
                50000,                      // maxTotalPeers
                Duration.ofHours(1),        // swarmIdleTimeout
                Duration.ofMinutes(30),     // peerStaleTimeout
                Duration.ofMinutes(5),      // cleanupInterval
                100 * 1024 * 1024L,         // maxBandwidth: 100 MB/s
                true,                       // enableCrossSwarmSharing
                true                        // enableBandwidthThrottling
            );
        }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private int maxSwarms = 5000;
            private int maxPeersPerSwarm = 200;
            private int maxTotalPeers = 50000;
            private Duration swarmIdleTimeout = Duration.ofHours(1);
            private Duration peerStaleTimeout = Duration.ofMinutes(30);
            private Duration cleanupInterval = Duration.ofMinutes(5);
            private long maxBandwidthBytesPerSecond = 100 * 1024 * 1024L;
            private boolean enableCrossSwarmSharing = true;
            private boolean enableBandwidthThrottling = true;

            public Builder maxSwarms(int m) { this.maxSwarms = m; return this; }
            public Builder maxPeersPerSwarm(int m) { this.maxPeersPerSwarm = m; return this; }
            public Builder maxTotalPeers(int m) { this.maxTotalPeers = m; return this; }
            public Builder swarmIdleTimeout(Duration d) { this.swarmIdleTimeout = d; return this; }
            public Builder peerStaleTimeout(Duration d) { this.peerStaleTimeout = d; return this; }
            public Builder cleanupInterval(Duration d) { this.cleanupInterval = d; return this; }
            public Builder maxBandwidthBytesPerSecond(long b) { this.maxBandwidthBytesPerSecond = b; return this; }
            public Builder enableCrossSwarmSharing(boolean e) { this.enableCrossSwarmSharing = e; return this; }
            public Builder enableBandwidthThrottling(boolean e) { this.enableBandwidthThrottling = e; return this; }

            public CoordinatorConfig build() {
                return new CoordinatorConfig(
                    maxSwarms, maxPeersPerSwarm, maxTotalPeers,
                    swarmIdleTimeout, peerStaleTimeout, cleanupInterval,
                    maxBandwidthBytesPerSecond, enableCrossSwarmSharing, enableBandwidthThrottling
                );
            }
        }
    }

    // ==================== Fields ====================

    private final CoordinatorConfig config;

    // Core data structures — all O(1) lookup
    private final ConcurrentHashMap<String, SwarmState> swarms = new ConcurrentHashMap<>();
    private final ConcurrentHashMap<String, PeerRecord> peers = new ConcurrentHashMap<>();

    // Bandwidth allocation
    private final ConcurrentHashMap<String, Long> swarmBandwidthAllocations = new ConcurrentHashMap<>();

    // Stats
    private final AtomicInteger peakSwarms = new AtomicInteger();
    private final AtomicInteger peakPeers = new AtomicInteger();
    private final AtomicLong totalBytesUp = new AtomicLong();
    private final AtomicLong totalBytesDown = new AtomicLong();
    private final AtomicLong evictedSwarms = new AtomicLong();

    // Scheduler
    private final ScheduledExecutorService scheduler;
    private volatile boolean running = false;

    // Event listeners
    private Consumer<SwarmInfo> onSwarmCreated;
    private Consumer<String> onSwarmEvicted;
    private Consumer<PeerState> onPeerJoined;
    private Consumer<PeerState> onPeerLeft;

    // ==================== Constructor ====================

    public SwarmCoordinator() {
        this(CoordinatorConfig.defaults());
    }

    public SwarmCoordinator(CoordinatorConfig config) {
        this.config = config;
        this.scheduler = Executors.newScheduledThreadPool(2, r -> {
            Thread t = new Thread(r, "swarm-coordinator");
            t.setDaemon(true);
            return t;
        });
    }

    // ==================== Lifecycle ====================

    public void start() {
        running = true;
        scheduler.scheduleAtFixedRate(
            this::cleanup,
            config.cleanupInterval.toMillis(),
            config.cleanupInterval.toMillis(),
            TimeUnit.MILLISECONDS
        );

        if (config.enableBandwidthThrottling) {
            scheduler.scheduleAtFixedRate(
                this::rebalanceBandwidth,
                10_000, 10_000, TimeUnit.MILLISECONDS
            );
        }
    }

    public void stop() {
        running = false;
        scheduler.shutdown();
        try {
            scheduler.awaitTermination(5, TimeUnit.SECONDS);
        } catch (InterruptedException e) {
            scheduler.shutdownNow();
            Thread.currentThread().interrupt();
        }
    }

    // ==================== Swarm Management ====================

    /**
     * Get or create a swarm for the given info hash.
     */
    public SwarmState getOrCreateSwarm(String infoHash) {
        return getOrCreateSwarm(infoHash, SwarmPriority.NORMAL);
    }

    public SwarmState getOrCreateSwarm(String infoHash, SwarmPriority priority) {
        return swarms.computeIfAbsent(infoHash, hash -> {
            if (swarms.size() >= config.maxSwarms) {
                evictLowestPrioritySwarm();
            }
            SwarmState newSwarm = new SwarmState(hash, priority);
            updatePeakSwarms();
            if (onSwarmCreated != null) {
                onSwarmCreated.accept(newSwarm.toInfo());
            }
            return newSwarm;
        });
    }

    /**
     * Get swarm info without creating it.
     */
    public Optional<SwarmInfo> getSwarm(String infoHash) {
        SwarmState state = swarms.get(infoHash);
        return state != null ? Optional.of(state.toInfo()) : Optional.empty();
    }

    /**
     * Remove a swarm entirely.
     */
    public void removeSwarm(String infoHash) {
        SwarmState removed = swarms.remove(infoHash);
        if (removed != null) {
            // Remove all peer-swarm associations
            for (String peerId : removed.getPeerIds()) {
                PeerRecord peer = peers.get(peerId);
                if (peer != null) {
                    peer.swarms.remove(infoHash);
                    if (peer.swarms.isEmpty()) {
                        peers.remove(peerId);
                    }
                }
            }
            if (onSwarmEvicted != null) {
                onSwarmEvicted.accept(infoHash);
            }
        }
    }

    /**
     * Get all active swarms, sorted by priority then activity.
     */
    public List<SwarmInfo> getActiveSwarms() {
        return swarms.values().stream()
            .map(SwarmState::toInfo)
            .sorted(Comparator.comparingInt((SwarmInfo s) -> s.priority().level)
                .thenComparing(SwarmInfo::lastActivity, Comparator.reverseOrder()))
            .toList();
    }

    /**
     * Get swarms ranked by peer count.
     */
    public List<SwarmInfo> getPopularSwarms(int limit) {
        return swarms.values().stream()
            .map(SwarmState::toInfo)
            .sorted(Comparator.comparingInt(SwarmInfo::peerCount).reversed())
            .limit(limit)
            .toList();
    }

    // ==================== Peer Management ====================

    /**
     * Add a peer to a swarm. If the peer already exists in other swarms,
     * their connection is shared (cross-swarm peer sharing).
     */
    public void addPeer(String infoHash, String peerId, TransportAddress address,
                         boolean isSeeder) {
        SwarmState swarm = getOrCreateSwarm(infoHash);

        // Get or create the peer record (shared across swarms)
        PeerRecord peer = peers.computeIfAbsent(peerId, id -> {
            updatePeakPeers();
            return new PeerRecord(id, address);
        });

        // Link peer to swarm
        peer.swarms.add(infoHash);
        peer.lastSeenAt = Instant.now();
        swarm.addPeer(peerId, isSeeder);

        if (onPeerJoined != null) {
            onPeerJoined.accept(peer.toState());
        }
    }

    /**
     * Remove a peer from a swarm. If the peer has no remaining swarms,
     * they're removed from the global registry.
     */
    public void removePeer(String infoHash, String peerId) {
        SwarmState swarm = swarms.get(infoHash);
        if (swarm != null) {
            swarm.removePeer(peerId);
        }

        PeerRecord peer = peers.get(peerId);
        if (peer != null) {
            peer.swarms.remove(infoHash);
            if (peer.swarms.isEmpty()) {
                PeerRecord removed = peers.remove(peerId);
                if (removed != null && onPeerLeft != null) {
                    onPeerLeft.accept(removed.toState());
                }
            }
        }
    }

    /**
     * Get peers shared between two or more swarms (useful for choking optimization).
     */
    public Set<String> getSharedPeers(String infoHash1, String infoHash2) {
        SwarmState s1 = swarms.get(infoHash1);
        SwarmState s2 = swarms.get(infoHash2);
        if (s1 == null || s2 == null) return Set.of();

        Set<String> common = new HashSet<>(s1.getPeerIds());
        common.retainAll(s2.getPeerIds());
        return common;
    }

    /**
     * Find all swarms a specific peer participates in.
     */
    public Set<String> getPeerSwarms(String peerId) {
        PeerRecord peer = peers.get(peerId);
        return peer != null ? Set.copyOf(peer.swarms) : Set.of();
    }

    /**
     * Get peer count for a specific swarm.
     */
    public int getPeerCount(String infoHash) {
        SwarmState swarm = swarms.get(infoHash);
        return swarm != null ? swarm.peerCount() : 0;
    }

    /**
     * Get the total number of unique peers across all swarms.
     */
    public int getTotalPeerCount() {
        return peers.size();
    }

    // ==================== Bandwidth Allocation ====================

    /**
     * Get the bandwidth allocation for a specific swarm (bytes/second).
     */
    public long getSwarmBandwidth(String infoHash) {
        return swarmBandwidthAllocations.getOrDefault(infoHash, 0L);
    }

    /**
     * Rebalance bandwidth across all active swarms based on priority.
     * Higher priority swarms get proportionally more bandwidth.
     */
    private void rebalanceBandwidth() {
        if (swarms.isEmpty()) return;

        long totalBandwidth = config.maxBandwidthBytesPerSecond;

        // Calculate total weight across all swarms
        double totalWeight = swarms.values().stream()
            .mapToDouble(s -> priorityWeight(s.priority))
            .sum();

        if (totalWeight == 0) return;

        // Allocate proportionally
        for (Map.Entry<String, SwarmState> entry : swarms.entrySet()) {
            double weight = priorityWeight(entry.getValue().priority);
            long allocation = (long) ((weight / totalWeight) * totalBandwidth);
            swarmBandwidthAllocations.put(entry.getKey(), allocation);
        }
    }

    private double priorityWeight(SwarmPriority priority) {
        return switch (priority) {
            case CRITICAL -> 8.0;
            case HIGH     -> 4.0;
            case NORMAL   -> 2.0;
            case LOW      -> 1.0;
            case IDLE     -> 0.25;
        };
    }

    // ==================== Data Tracking ====================

    /**
     * Record bytes uploaded to a peer in a specific swarm.
     */
    public void recordUpload(String infoHash, String peerId, long bytes) {
        SwarmState swarm = swarms.get(infoHash);
        if (swarm != null) {
            swarm.bytesUploaded.addAndGet(bytes);
            swarm.lastActivity = Instant.now();
        }
        PeerRecord peer = peers.get(peerId);
        if (peer != null) {
            peer.bytesUploaded.addAndGet(bytes);
        }
        totalBytesUp.addAndGet(bytes);
    }

    /**
     * Record bytes downloaded from a peer in a specific swarm.
     */
    public void recordDownload(String infoHash, String peerId, long bytes) {
        SwarmState swarm = swarms.get(infoHash);
        if (swarm != null) {
            swarm.bytesDownloaded.addAndGet(bytes);
            swarm.lastActivity = Instant.now();
        }
        PeerRecord peer = peers.get(peerId);
        if (peer != null) {
            peer.bytesDownloaded.addAndGet(bytes);
        }
        totalBytesDown.addAndGet(bytes);
    }

    // ==================== Stats ====================

    public CoordinatorStats getStats() {
        int sharedPeers = (int) peers.values().stream()
            .filter(p -> p.swarms.size() > 1)
            .count();

        double avgPeers = swarms.isEmpty() ? 0.0 :
            swarms.values().stream().mapToInt(SwarmState::peerCount).average().orElse(0.0);

        return new CoordinatorStats(
            swarms.size(),
            peers.size(),
            sharedPeers,
            totalBytesUp.get(),
            totalBytesDown.get(),
            peakSwarms.get(),
            peakPeers.get(),
            avgPeers,
            evictedSwarms.get()
        );
    }

    // ==================== Event Listeners ====================

    public void setOnSwarmCreated(Consumer<SwarmInfo> listener) { this.onSwarmCreated = listener; }
    public void setOnSwarmEvicted(Consumer<String> listener) { this.onSwarmEvicted = listener; }
    public void setOnPeerJoined(Consumer<PeerState> listener) { this.onPeerJoined = listener; }
    public void setOnPeerLeft(Consumer<PeerState> listener) { this.onPeerLeft = listener; }

    // ==================== Internal ====================

    private void cleanup() {
        Instant now = Instant.now();

        // Evict idle swarms
        List<String> idleSwarms = swarms.entrySet().stream()
            .filter(e -> Duration.between(e.getValue().lastActivity, now)
                .compareTo(config.swarmIdleTimeout) > 0)
            .map(Map.Entry::getKey)
            .toList();

        for (String hash : idleSwarms) {
            removeSwarm(hash);
            evictedSwarms.incrementAndGet();
        }

        // Remove stale peers
        List<String> stalePeers = peers.entrySet().stream()
            .filter(e -> Duration.between(e.getValue().lastSeenAt, now)
                .compareTo(config.peerStaleTimeout) > 0)
            .map(Map.Entry::getKey)
            .toList();

        for (String peerId : stalePeers) {
            PeerRecord peer = peers.remove(peerId);
            if (peer != null) {
                for (String swarmHash : peer.swarms) {
                    SwarmState swarm = swarms.get(swarmHash);
                    if (swarm != null) swarm.removePeer(peerId);
                }
            }
        }
    }

    private void evictLowestPrioritySwarm() {
        swarms.entrySet().stream()
            .min(Comparator.comparingInt((Map.Entry<String, SwarmState> e) ->
                    -e.getValue().priority.level) // lowest priority = highest level number
                .thenComparing(e -> e.getValue().lastActivity))
            .ifPresent(entry -> {
                removeSwarm(entry.getKey());
                evictedSwarms.incrementAndGet();
            });
    }

    private void updatePeakSwarms() {
        peakSwarms.updateAndGet(current -> Math.max(current, swarms.size()));
    }

    private void updatePeakPeers() {
        peakPeers.updateAndGet(current -> Math.max(current, peers.size()));
    }

    // ==================== Inner Classes ====================

    /**
     * Per-swarm state tracking.
     */
    static class SwarmState {
        final String infoHash;
        final SwarmPriority priority;
        final Set<String> seeders = ConcurrentHashMap.newKeySet();
        final Set<String> leechers = ConcurrentHashMap.newKeySet();
        final AtomicLong bytesUploaded = new AtomicLong();
        final AtomicLong bytesDownloaded = new AtomicLong();
        final Instant createdAt;
        volatile Instant lastActivity;

        SwarmState(String infoHash, SwarmPriority priority) {
            this.infoHash = infoHash;
            this.priority = priority;
            this.createdAt = Instant.now();
            this.lastActivity = Instant.now();
        }

        void addPeer(String peerId, boolean isSeeder) {
            if (isSeeder) {
                seeders.add(peerId);
                leechers.remove(peerId);
            } else {
                leechers.add(peerId);
                seeders.remove(peerId);
            }
            lastActivity = Instant.now();
        }

        void removePeer(String peerId) {
            seeders.remove(peerId);
            leechers.remove(peerId);
        }

        Set<String> getPeerIds() {
            Set<String> all = new HashSet<>(seeders);
            all.addAll(leechers);
            return all;
        }

        int peerCount() {
            return seeders.size() + leechers.size();
        }

        SwarmInfo toInfo() {
            return new SwarmInfo(
                infoHash, priority, peerCount(),
                seeders.size(), leechers.size(),
                bytesUploaded.get(), bytesDownloaded.get(),
                createdAt, lastActivity, true
            );
        }
    }

    /**
     * Global peer record — shared across all swarms the peer participates in.
     */
    private static class PeerRecord {
        final String peerId;
        final TransportAddress address;
        final Set<String> swarms = ConcurrentHashMap.newKeySet();
        final AtomicLong bytesUploaded = new AtomicLong();
        final AtomicLong bytesDownloaded = new AtomicLong();
        volatile Instant lastSeenAt;

        PeerRecord(String peerId, TransportAddress address) {
            this.peerId = peerId;
            this.address = address;
            this.lastSeenAt = Instant.now();
        }

        PeerState toState() {
            return new PeerState(
                peerId, address, Set.copyOf(swarms),
                bytesUploaded.get(), bytesDownloaded.get(),
                1.0, // Reliability computed elsewhere
                Instant.now(), lastSeenAt,
                false, false
            );
        }
    }
}
