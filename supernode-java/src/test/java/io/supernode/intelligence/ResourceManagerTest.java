package io.supernode.intelligence;

import io.supernode.network.SupernodeNetwork;
import io.supernode.storage.SupernodeStorage;
import io.supernode.network.BlobNetwork;
import io.supernode.network.DHTDiscovery;
import io.supernode.network.ManifestDistributor;
import io.supernode.network.UnifiedNetwork;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.mockito.Mockito;

import java.time.Duration;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

class ResourceManagerTest {

    private SupernodeNetwork mockNetwork;
    private SupernodeStorage.StorageStats mockStorageStats;
    private SupernodeNetwork.NetworkStats mockNetworkStats;
    private ResourceManager resourceManager;

    @BeforeEach
    void setUp() {
        mockNetwork = mock(SupernodeNetwork.class);
        mockStorageStats = mock(SupernodeStorage.StorageStats.class);
        
        // Mock the complex stats hierarchy
        mockNetworkStats = new SupernodeNetwork.NetworkStats(
            mockStorageStats,
            mock(BlobNetwork.BlobNetworkStats.class),
            mock(DHTDiscovery.DHTStats.class),
            mock(ManifestDistributor.ManifestDistributorStats.class),
            mock(UnifiedNetwork.NetworkStats.class)
        );

        when(mockNetwork.stats()).thenReturn(mockNetworkStats);
        
        resourceManager = new ResourceManager(mockNetwork);
    }

    @Test
    void testInitialState() {
        ResourceManager.ResourceState state = resourceManager.getCurrentState();
        assertEquals(ResourceManager.LoadLevel.LOW, state.loadLevel());
        assertEquals(ResourceManager.Recommendation.MAINTAIN, state.recommendation());
    }

    @Test
    void testAnalysisUpdate() throws InterruptedException {
        // Simulate high load
        when(mockStorageStats.activeOperations()).thenReturn(100);

        CountDownLatch latch = new CountDownLatch(1);
        resourceManager.setOnStateChange(state -> {
            if (state.loadLevel() == ResourceManager.LoadLevel.HIGH) {
                latch.countDown();
            }
        });

        resourceManager.start();
        resourceManager.stop();
    }
}
