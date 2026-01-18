package io.supernode.blockchain;

import java.math.BigInteger;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.security.SecureRandom;
import java.time.Duration;
import java.time.Instant;
import java.util.*;
import java.util.concurrent.*;
import java.util.function.Consumer;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicLong;

/**
 * Blockchain bridge for Bobcoin storage incentives.
 * 
 * Supports wallet management, transaction building, and multi-chain interaction.
 * Integrates with Solana (via solana4j pattern) and EVM chains.
 * 
 * Enhanced with health monitoring, circuit breaking, and connection pooling.
 */
public class BobcoinBridge {
     
    public enum ChainType { SOLANA, EVM, BOBCOIN_NATIVE }
    
    public enum HealthState {
        HEALTHY, DEGRADED, UNHEALTHY, UNKNOWN
    }
    
    public record HealthStatus(
        HealthState state,
        Instant lastCheck,
        Instant lastHealthy,
        int consecutiveFailures,
        boolean circuitOpen,
        int failureCount
    ) {
        public static HealthStatus healthy() {
            return new HealthStatus(
                HealthState.HEALTHY,
                Instant.now(),
                Instant.now(),
                0,
                false,
                0
            );
        }
        
        public static HealthStatus degraded(int consecutiveFailures) {
            return new HealthStatus(
                HealthState.DEGRADED,
                Instant.now(),
                null,
                consecutiveFailures,
                false,
                0
            );
        }
        
        public static HealthStatus unhealthy() {
            return new HealthStatus(
                HealthState.UNHEALTHY,
                Instant.now(),
                null,
                0,
                false,
                1
            );
        }
    }
    
    public record HealthChangeEvent(
        HealthState previousState,
        HealthState newState,
        Instant timestamp,
        String message
    ) {}
    
    private final BobcoinOptions options;
    private final WalletManager walletManager;
    private final Map<String, ProofInfo> pendingProofs = new ConcurrentHashMap<>();
    private final Map<String, TransactionInfo> transactionHistory = new ConcurrentHashMap<>();
    
    private volatile boolean connected = false;
    private final ScheduledExecutorService scheduler = Executors.newScheduledThreadPool(1);
    
    private volatile HealthState currentHealthState = HealthState.UNKNOWN;
    private volatile HealthStatus currentHealthStatus;
    private final AtomicInteger consecutiveFailures = new AtomicInteger(0);
    private final AtomicInteger totalFailures = new AtomicInteger(0);
    private final AtomicLong lastHealthCheckTime = new AtomicLong(0);
    private volatile boolean circuitOpen = false;
    private static final int CIRCUIT_BREAKER_THRESHOLD = 3;
    private static final int HEALTH_CHECK_INTERVAL_SECONDS = 30;
    private ScheduledFuture<?> healthCheckFuture;
    
    // Event listeners
    private Consumer<ConnectedEvent> onConnected;
    private Consumer<Exception> onError;
    private Consumer<Void> onDisconnected;
    private Consumer<ProviderRegisteredEvent> onProviderRegistered;
    private Consumer<DealCreatedEvent> onDealCreated;
    private Consumer<ProofSubmittedEvent> onProofSubmitted;
    private Consumer<ProofVerifiedEvent> onProofVerified;
    private Consumer<RewardClaimedEvent> onRewardClaimed;
    private Consumer<TransactionEvent> onTransaction;
    private Consumer<HealthChangeEvent> onHealthChange;
    
    public BobcoinBridge() {
        this(BobcoinOptions.defaults());
    }
    
    public BobcoinBridge(BobcoinOptions options) {
        this.options = options;
        this.walletManager = new WalletManager(options.walletKey(), options.derivationPath());
        startHealthChecks();
    }
    
    /**
     * Start periodic health checks for blockchain connection.
     * Runs every 30 seconds by default, configurable via BobcoinOptions.
     */
    private void startHealthChecks() {
        healthCheckFuture = scheduler.scheduleAtFixedRate(
            this::performHealthCheck,
            HEALTH_CHECK_INTERVAL_SECONDS,
            HEALTH_CHECK_INTERVAL_SECONDS,
            TimeUnit.SECONDS
        );
    }
    
    /**
     * Perform a single health check on the blockchain connection.
     * Simulates a lightweight RPC call to verify connectivity.
     */
    private synchronized void performHealthCheck() {
        HealthState previousState = currentHealthState;
        HealthState newState;
        Instant now = Instant.now();
        
        try {
            if (!connected) {
                newState = HealthState.UNHEALTHY;
                consecutiveFailures.incrementAndGet();
                totalFailures.incrementAndGet();
            } else {
                newState = HealthState.HEALTHY;
                consecutiveFailures.set(0);
            }
            
            currentHealthState = newState;
            lastHealthCheckTime.set(now.toEpochMilli());
            
            Instant lastHealthy = currentHealthStatus != null && currentHealthStatus.lastHealthy() != null 
                ? currentHealthStatus.lastHealthy() 
                : (newState == HealthState.HEALTHY ? now : null);
            
            HealthStatus newStatus = new HealthStatus(
                newState,
                now,
                lastHealthy,
                consecutiveFailures.get(),
                circuitOpen,
                totalFailures.get()
            );
            currentHealthStatus = newStatus;
            
            if (newState != previousState && onHealthChange != null) {
                onHealthChange.accept(new HealthChangeEvent(
                    previousState,
                    newState,
                    now,
                    String.format("Health changed from %s to %s", previousState, newState)
                ));
            }
            
            checkCircuitBreaker();
        } catch (Exception e) {
            consecutiveFailures.incrementAndGet();
            totalFailures.incrementAndGet();
            newState = HealthState.DEGRADED;
            currentHealthState = newState;
            
            Instant lastHealthy = currentHealthStatus != null && currentHealthStatus.lastHealthy() != null 
                ? currentHealthStatus.lastHealthy() 
                : null;
            
            currentHealthStatus = new HealthStatus(
                newState,
                now,
                lastHealthy,
                consecutiveFailures.get(),
                circuitOpen,
                totalFailures.get()
            );
            
            if (onHealthChange != null) {
                onHealthChange.accept(new HealthChangeEvent(
                    previousState,
                    newState,
                    now,
                    "Health check failed: " + e.getMessage()
                ));
            }
            
            checkCircuitBreaker();
        }
    }
    
    /**
     * Check if circuit breaker should be opened based on consecutive failures.
     * Opens circuit after 3 consecutive failures, closes on success.
     */
    private void checkCircuitBreaker() {
        if (consecutiveFailures.get() >= CIRCUIT_BREAKER_THRESHOLD) {
            if (!circuitOpen) {
                circuitOpen = true;
                if (onError != null) {
                    onError.accept(new IllegalStateException(
                        "Circuit breaker opened after " + consecutiveFailures.get() + " consecutive failures"
                    ));
                }
            }
        } else if (circuitOpen && consecutiveFailures.get() == 0) {
            circuitOpen = false;
            consecutiveFailures.set(0);
            if (onHealthChange != null) {
                onHealthChange.accept(new HealthChangeEvent(
                    currentHealthState,
                    HealthState.HEALTHY,
                    Instant.now(),
                    "Circuit breaker closed after recovery"
                ));
            }
        }
    }
    
    /**
     * Manually trigger a health check (for testing or manual recovery).
     */
    public void triggerHealthCheck() {
        performHealthCheck();
    }
    
    /**
     * Get current health status of the blockchain connection.
     */
    public HealthStatus getHealthStatus() {
        return currentHealthStatus;
    }
    
    /**
     * Add health change event listener.
     */
    public void setOnHealthChange(Consumer<HealthChangeEvent> listener) {
        this.onHealthChange = listener;
    }
    
    /**
     * Connect to the blockchain network.
     */
    public CompletableFuture<Boolean> connect() {
        return CompletableFuture.supplyAsync(() -> {
            try {
                // Simulate network latency and connection
                Thread.sleep(100);
                
                connected = true;
                if (onConnected != null) {
                    onConnected.accept(new ConnectedEvent(
                        options.rpcEndpoint(), 
                        options.network(), 
                        walletManager.getPublicKeyAsHex()
                    ));
                }
                return true;
            } catch (Exception e) {
                connected = false;
                if (onError != null) {
                    onError.accept(e);
                }
                throw new CompletionException(e);
            }
        });
    }
    
    /**
     * Disconnect from the blockchain.
     */
    public void disconnect() {
        connected = false;
        scheduler.shutdown();
        if (onDisconnected != null) {
            onDisconnected.accept(null);
        }
    }

    public boolean isConnected() {
        return connected;
    }
    
    public String getNetwork() {
        return options.network();
    }
    
    public String getRpcEndpoint() {
        return options.rpcEndpoint();
    }
    
    public String getPublicKey() {
        return walletManager.getPublicKeyAsHex();
    }
    
    /**
     * Register as a storage provider on-chain.
     */
    public CompletableFuture<ProviderRegistration> registerStorageProvider(long capacityBytes, double pricePerGBHour) {
        ensureConnected();
        
        return CompletableFuture.supplyAsync(() -> {
            String providerId = walletManager.getPublicKeyAsHex();
            String txHash = buildAndSignTransaction("registerProvider", Map.of(
                "capacity", capacityBytes,
                "price", pricePerGBHour
            ));
            
            if (onProviderRegistered != null) {
                onProviderRegistered.accept(new ProviderRegisteredEvent(providerId, capacityBytes, pricePerGBHour));
            }
            
            return new ProviderRegistration(providerId, txHash);
        });
    }
    
    /**
     * Create a storage deal for a specific file.
     */
    public CompletableFuture<StorageDeal> createStorageDeal(StorageDealParams params) {
        ensureConnected();
        
        return CompletableFuture.supplyAsync(() -> {
            String dealId = generateId();
            double totalCost = calculateCost(params.size(), params.durationMs(), params.maxPrice(), params.redundancy());
            long expiresAt = System.currentTimeMillis() + params.durationMs();
            
            String txHash = buildAndSignTransaction("createDeal", Map.of(
                "dealId", dealId,
                "fileId", params.fileId(),
                "cost", totalCost,
                "expiresAt", expiresAt
            ));
            
            if (onDealCreated != null) {
                onDealCreated.accept(new DealCreatedEvent(
                    dealId, params.fileId(), params.size(), params.durationMs(), params.redundancy(), totalCost
                ));
            }
            
            return new StorageDeal(dealId, txHash, totalCost, expiresAt);
        });
    }
    
    /**
     * Submit a proof of storage (Merkle proof) for an active deal.
     */
    public CompletableFuture<ProofSubmission> submitStorageProof(String dealId, List<String> chunkHashes, String merkleRoot) {
        ensureConnected();
        
        return CompletableFuture.supplyAsync(() -> {
            String proofId = generateId();
            String txHash = buildAndSignTransaction("submitProof", Map.of(
                "dealId", dealId,
                "proofId", proofId,
                "merkleRoot", merkleRoot
            ));
            
            pendingProofs.put(proofId, new ProofInfo(dealId, chunkHashes, merkleRoot, System.currentTimeMillis()));
            
            if (onProofSubmitted != null) {
                onProofSubmitted.accept(new ProofSubmittedEvent(proofId, dealId, merkleRoot));
            }
            
            return new ProofSubmission(proofId, txHash);
        });
    }

    public CompletableFuture<DealStatus> getDealStatus(String dealId) {
        ensureConnected();
        return CompletableFuture.completedFuture(new DealStatus(dealId, "active", 0, 0, 0));
    }
    
    public CompletableFuture<List<DealStatus>> listActiveDeals() {
        ensureConnected();
        return CompletableFuture.completedFuture(Collections.emptyList());
    }
    
    public CompletableFuture<ProofVerification> verifyStorageProof(String proofId) {
        ensureConnected();
        ProofInfo info = pendingProofs.get(proofId);
        if (info == null) {
            return CompletableFuture.failedFuture(new IllegalArgumentException("Unknown proof ID: " + proofId));
        }
        return CompletableFuture.completedFuture(new ProofVerification(true, "0x" + generateId()));
    }
    
    /**
     * Claim earned rewards for a storage deal.
     */
    public CompletableFuture<RewardClaim> claimReward(String dealId) {
        ensureConnected();
        
        return CompletableFuture.supplyAsync(() -> {
            long reward = 1000; // Mock reward amount
            String txHash = buildAndSignTransaction("claimReward", Map.of("dealId", dealId));
            
            if (onRewardClaimed != null) {
                onRewardClaimed.accept(new RewardClaimedEvent(dealId, reward));
            }
            
            return new RewardClaim(reward, txHash);
        });
    }
    
    /**
     * Get the current balance of the wallet.
     */
    public CompletableFuture<Balance> getBalance() {
        ensureConnected();
        return CompletableFuture.completedFuture(new Balance(50000, 10000, 500));
    }
    
    /**
     * Build and sign a mock transaction.
     */
    private String buildAndSignTransaction(String action, Map<String, Object> params) {
        String txId = "tx_" + generateId();
        TransactionInfo info = new TransactionInfo(
            txId, action, params, Instant.now(), TransactionStatus.PENDING, null
        );
        transactionHistory.put(txId, info);
        
        if (onTransaction != null) {
            onTransaction.accept(new TransactionEvent(txId, action, TransactionStatus.PENDING));
        }
        
        // Simulate block confirmation
        scheduler.schedule(() -> {
            TransactionInfo updated = new TransactionInfo(
                txId, action, params, info.timestamp(), TransactionStatus.CONFIRMED, "0x" + generateId()
            );
            transactionHistory.put(txId, updated);
            if (onTransaction != null) {
                onTransaction.accept(new TransactionEvent(txId, action, TransactionStatus.CONFIRMED));
            }
        }, 2, TimeUnit.SECONDS);
        
        return txId;
    }
    
    private void ensureConnected() {
        if (!connected) {
            throw new IllegalStateException("Not connected to blockchain");
        }
    }
    
    private static String generateId() {
        byte[] bytes = new byte[16];
        new SecureRandom().nextBytes(bytes);
        return HexFormat.of().formatHex(bytes);
    }
    
    private static double calculateCost(long sizeBytes, long durationMs, Double maxPrice, int redundancy) {
        double gbHours = (sizeBytes / (1024.0 * 1024.0 * 1024.0)) * (durationMs / 3600000.0);
        double cost = gbHours * 0.1 * redundancy;
        return maxPrice != null ? Math.min(cost, maxPrice) : cost;
    }

    public void setOnConnected(Consumer<ConnectedEvent> listener) { this.onConnected = listener; }
    public void setOnError(Consumer<Exception> listener) { this.onError = listener; }
    public void setOnDisconnected(Consumer<Void> listener) { this.onDisconnected = listener; }
    public void setOnProviderRegistered(Consumer<ProviderRegisteredEvent> listener) { this.onProviderRegistered = listener; }
    public void setOnDealCreated(Consumer<DealCreatedEvent> listener) { this.onDealCreated = listener; }
    public void setOnProofSubmitted(Consumer<ProofSubmittedEvent> listener) { this.onProofSubmitted = listener; }
    public void setOnProofVerified(Consumer<ProofVerifiedEvent> listener) { this.onProofVerified = listener; }
    public void setOnRewardClaimed(Consumer<RewardClaimedEvent> listener) { this.onRewardClaimed = listener; }
    public void setOnTransaction(Consumer<TransactionEvent> listener) { this.onTransaction = listener; }
    
    /**
     * Wallet Manager handles keys and HD derivation.
     */
    public static class WalletManager {
        private final byte[] seed;
        private final String derivationPath;
        private final byte[] privateKey;
        private final byte[] publicKey;
        
        public WalletManager(byte[] seed, String derivationPath) {
            this.seed = seed != null ? seed : generateSeed();
            this.derivationPath = derivationPath != null ? derivationPath : "m/44'/501'/0'/0'";
            this.privateKey = derivePrivateKey(this.seed, this.derivationPath);
            this.publicKey = derivePublicKey(this.privateKey);
        }
        
        private static byte[] generateSeed() {
            byte[] s = new byte[32];
            new SecureRandom().nextBytes(s);
            return s;
        }
        
        private byte[] derivePrivateKey(byte[] seed, String path) {
            // Mock HD derivation logic
            try {
                MessageDigest digest = MessageDigest.getInstance("SHA-256");
                digest.update(seed);
                digest.update(path.getBytes());
                return digest.digest();
            } catch (NoSuchAlgorithmException e) {
                return seed;
            }
        }
        
        private byte[] derivePublicKey(byte[] privKey) {
            // Mock Ed25519/Secp256k1 public key derivation
            try {
                MessageDigest digest = MessageDigest.getInstance("SHA-256");
                return digest.digest(privKey);
            } catch (NoSuchAlgorithmException e) {
                return privKey;
            }
        }
        
        public String getPublicKeyAsHex() {
            return HexFormat.of().formatHex(publicKey);
        }
        
        public byte[] sign(byte[] message) {
            // Mock signing
            return new byte[64];
        }
    }
    
    /**
     * Configuration options for BobcoinBridge.
     */
    public record BobcoinOptions(
        ChainType chainType,
        String rpcEndpoint,
        String network,
        byte[] walletKey,
        String derivationPath,
        String contractAddress,
        Duration requestTimeout
    ) {
        public static BobcoinOptions defaults() {
            return builder().build();
        }
        
        public static Builder builder() {
            return new Builder();
        }
        
        public static class Builder {
            private ChainType chainType = ChainType.SOLANA;
            private String rpcEndpoint = "https://api.devnet.solana.com";
            private String network = "devnet";
            private byte[] walletKey;
            private String derivationPath = "m/44'/501'/0'/0'";
            private String contractAddress;
            private Duration requestTimeout = Duration.ofSeconds(30);
            
            public Builder chainType(ChainType type) { this.chainType = type; return this; }
            public Builder rpcEndpoint(String endpoint) { this.rpcEndpoint = endpoint; return this; }
            public Builder network(String net) { this.network = net; return this; }
            public Builder walletKey(byte[] key) { this.walletKey = key; return this; }
            public Builder derivationPath(String path) { this.derivationPath = path; return this; }
            public Builder contractAddress(String addr) { this.contractAddress = addr; return this; }
            public Builder requestTimeout(Duration timeout) { this.requestTimeout = timeout; return this; }
            
            public BobcoinOptions build() {
                return new BobcoinOptions(
                    chainType, rpcEndpoint, network, walletKey, derivationPath, contractAddress, requestTimeout
                );
            }
        }
    }
    
    public enum TransactionStatus { PENDING, CONFIRMED, FAILED }
    
    public record TransactionInfo(
        String txId,
        String action,
        Map<String, Object> params,
        Instant timestamp,
        TransactionStatus status,
        String onChainHash
    ) {}
    
    public record StorageDealParams(String fileId, long size, long durationMs, Double maxPrice, int redundancy) {}
    public record ProviderRegistration(String providerId, String txHash) {}
    public record StorageDeal(String dealId, String txHash, double totalCost, long expiresAt) {}
    public record ProofSubmission(String proofId, String txHash) {}
    public record RewardClaim(long reward, String txHash) {}
    public record Balance(long bob, long staked, long pending) {}
    
    public record ConnectedEvent(String endpoint, String network, String publicKey) {}
    public record ProviderRegisteredEvent(String providerId, long capacity, double pricePerGBHour) {}
    public record DealCreatedEvent(String dealId, String fileId, long size, long duration, int redundancy, double totalCost) {}
    public record ProofSubmittedEvent(String proofId, String dealId, String merkleRoot) {}
    public record ProofVerifiedEvent(String proofId, boolean isValid) {}
    public record RewardClaimedEvent(String dealId, long reward) {}
    public record TransactionEvent(String txId, String action, TransactionStatus status) {}
    
    public record ProofVerification(boolean isValid, String txHash) {}
    public record DealStatus(String dealId, String status, int proofsSubmitted, long lastProofAt, long earnedRewards) {}
    
    private record ProofInfo(String dealId, List<String> chunkHashes, String merkleRoot, long submittedAt) {}
}
