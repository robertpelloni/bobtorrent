package io.supernode.network.routing;

import io.supernode.network.DHTDiscovery.PeerInfo;
import java.util.List;
import java.util.concurrent.CompletableFuture;

/**
 * Interface for content routing (finding providers for a specific content key).
 * Abstraction over DHT, Blockchain, or other discovery mechanisms.
 */
public interface ContentRouting {
    
    /**
     * Advertise that this node provides the content identified by the key.
     * @param key Content identifier (CID, BlobID, etc.)
     */
    CompletableFuture<Void> provide(String key);
    
    /**
     * Find peers that provide the content identified by the key.
     * @param key Content identifier
     * @return List of peers
     */
    CompletableFuture<List<PeerInfo>> findProviders(String key);
    
    /**
     * Stop providing/advertising the content.
     * @param key Content identifier
     */
    default CompletableFuture<Void> unprovide(String key) {
        return CompletableFuture.completedFuture(null);
    }
}
