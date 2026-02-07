package io.supernode.intelligence;

import io.supernode.network.SupernodeNetwork;
import io.supernode.storage.SupernodeStorage;

import java.lang.management.ManagementFactory;
import java.lang.management.OperatingSystemMXBean;
import java.lang.management.MemoryMXBean;
import java.time.Instant;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicReference;
import java.util.function.Consumer;

/**
 * Predictive Resource Manager for Supernode.
 * Monitors system, network, and storage metrics to optimize performance.
 */
public class ResourceManager {

    private final SupernodeNetwork network;
    private final ScheduledExecutorService scheduler;
    private final OperatingSystemMXBean osBean;
    private final MemoryMXBean memoryBean;
    
    private final AtomicReference<ResourceState> currentState = new AtomicReference<>(ResourceState.idle());
    private Consumer<ResourceState> onStateChange;
    
    private volatile boolean running = false;

    public ResourceManager(SupernodeNetwork network) {
        this.network = network;
        this.scheduler = Executors.newSingleThreadScheduledExecutor(r -> {
            Thread t = new Thread(r, "resource-manager");
            t.setDaemon(true);
            return t;
        });
        this.osBean = ManagementFactory.getOperatingSystemMXBean();
        this.memoryBean = ManagementFactory.getMemoryMXBean();
    }

    public void start() {
        if (running) return;
        running = true;
        scheduler.scheduleAtFixedRate(this::analyze, 5, 5, TimeUnit.SECONDS);
    }

    public void stop() {
        running = false;
        scheduler.shutdown();
    }

    private void analyze() {
        if (!running) return;

        double systemLoad = osBean.getSystemLoadAverage();
        long usedMemory = memoryBean.getHeapMemoryUsage().getUsed();
        long maxMemory = memoryBean.getHeapMemoryUsage().getMax();
        double memoryUsage = (double) usedMemory / maxMemory;

        SupernodeNetwork.NetworkStats netStats = network.stats();
        SupernodeStorage.StorageStats storeStats = netStats.storage();
        
        int activeOps = storeStats.activeOperations();
        long totalIngested = storeStats.totalBytesIngested();

        LoadLevel loadLevel;
        if (memoryUsage > 0.85 || systemLoad > 0.8 * osBean.getAvailableProcessors()) {
            loadLevel = LoadLevel.CRITICAL;
        } else if (memoryUsage > 0.70 || activeOps > 50) {
            loadLevel = LoadLevel.HIGH;
        } else if (activeOps > 10) {
            loadLevel = LoadLevel.MODERATE;
        } else {
            loadLevel = LoadLevel.LOW;
        }

        Recommendation recommendation = generateRecommendation(loadLevel, memoryUsage);

        ResourceState newState = new ResourceState(
            Instant.now(),
            loadLevel,
            memoryUsage,
            systemLoad,
            activeOps,
            recommendation
        );

        ResourceState oldState = currentState.getAndSet(newState);
        
        if (onStateChange != null && !newState.equals(oldState)) {
            onStateChange.accept(newState);
        }
    }

    private Recommendation generateRecommendation(LoadLevel level, double memoryUsage) {
        switch (level) {
            case CRITICAL:
                return Recommendation.THROTTLE_INGEST;
            case HIGH:
                if (memoryUsage > 0.8) return Recommendation.GC_SUGGESTED;
                return Recommendation.MAINTAIN;
            case MODERATE:
                return Recommendation.MAINTAIN;
            case LOW:
            default:
                return Recommendation.INCREASE_CAPACITY;
        }
    }

    public void setOnStateChange(Consumer<ResourceState> listener) {
        this.onStateChange = listener;
    }

    public ResourceState getCurrentState() {
        return currentState.get();
    }

    public record ResourceState(
        Instant timestamp,
        LoadLevel loadLevel,
        double memoryUsage,
        double systemLoad,
        int activeOperations,
        Recommendation recommendation
    ) {
        public static ResourceState idle() {
            return new ResourceState(Instant.now(), LoadLevel.LOW, 0.0, 0.0, 0, Recommendation.MAINTAIN);
        }
    }

    public enum LoadLevel {
        LOW, MODERATE, HIGH, CRITICAL
    }

    public enum Recommendation {
        MAINTAIN,
        INCREASE_CAPACITY,
        GC_SUGGESTED,
        THROTTLE_INGEST,
        HALT_OPERATIONS
    }
}
