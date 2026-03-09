package io.supernode.network.routing;

import io.supernode.network.BlobNetwork;
import io.supernode.network.BlobNetwork.PeerConnection;

import java.util.*;
import java.util.concurrent.*;

/**
 * Coordinates BitSwarm logic, tracking swarm health for specific blobs 
 * and requesting chunks in parallel from multiple peers.
 */
public class BitSwarmCoordinator {

    private final BlobNetwork blobNetwork;
    private final ConcurrentHashMap<String, SwarmState> activeSwarms = new ConcurrentHashMap<>();

    public BitSwarmCoordinator(BlobNetwork blobNetwork) {
        this.blobNetwork = blobNetwork;
    }

    /**
     * Finds the list of peers that have the requested blob, and if multiple
     * are available, establishes a SwarmState for tracking concurrent retrieval.
     */
    public CompletableFuture<byte[]> requestFromSwarm(String blobId) {
        List<PeerConnection> availablePeers = blobNetwork.findPeersWithBlob(blobId);
        
        if (availablePeers.isEmpty()) {
            return CompletableFuture.failedFuture(new IllegalStateException("No peers found in swarm for blob: " + blobId));
        }

        SwarmState swarm = activeSwarms.computeIfAbsent(blobId, k -> new SwarmState(blobId, availablePeers));
        
        // Use the underlying BlobNetwork to request from the optimized list of peers.
        // In a true BitTorrent implementation, this would request sub-pieces of the blob 
        // across peers. For Bobtorrent Supernode blobs (which are themselves chunks/shards), 
        // we utilize BlobNetwork's built-in requesting logic but provide all swarm peers 
        // as targets for failover and parallel request selection.
        return blobNetwork.requestBlob(blobId, swarm.getPeers()).whenComplete((result, ex) -> {
            activeSwarms.remove(blobId);
            if (ex == null) {
                swarm.recordSuccess();
            } else {
                swarm.recordFailure();
            }
        });
    }

    public SwarmHealth getSwarmHealth(String blobId) {
        SwarmState state = activeSwarms.get(blobId);
        if (state == null) {
            List<PeerConnection> passivePeers = blobNetwork.findPeersWithBlob(blobId);
            return new SwarmHealth(blobId, passivePeers.size(), false);
        }
        return state.getHealth();
    }

    private static class SwarmState {
        private final String blobId;
        private final List<PeerConnection> peers;
        private int successes = 0;
        private int failures = 0;

        public SwarmState(String blobId, List<PeerConnection> peers) {
            this.blobId = blobId;
            this.peers = peers;
        }

        public List<PeerConnection> getPeers() {
            return peers;
        }

        public synchronized void recordSuccess() {
            successes++;
        }

        public synchronized void recordFailure() {
            failures++;
        }

        public synchronized SwarmHealth getHealth() {
            return new SwarmHealth(blobId, peers.size(), true);
        }
    }

    public record SwarmHealth(String blobId, int peerCount, boolean active) {}
}
