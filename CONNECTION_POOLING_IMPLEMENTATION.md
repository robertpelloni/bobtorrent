# Connection Pooling Implementation for BlobNetwork

## Overview
This document describes the connection pooling implementation added to `BlobNetwork.java` which provides thread-safe connection management with configurable pooling, metrics tracking, and lifecycle management.

## Features Implemented

### 1. Connection Pool Infrastructure
- **Thread-safe pool storage**: `ConcurrentHashMap<String, PeerConnection> connectionPool`
- **Access tracking**: `ConcurrentHashMap<String, Long> lastAccessTime` and `accessCount`
- **Keep-alive management**: `Set<String> keepAlivePeers` for frequently accessed peers
- **Atomic counters**: Thread-safe metrics with `AtomicInteger` and `AtomicLong`

### 2. Configuration (ConnectionPoolOptions)
```java
public static class ConnectionPoolOptions {
    public final int minPoolSize;           // Default: 1
    public final int maxPoolSize;           // Default: 10
    public final Duration idleTimeout;        // Default: 60 seconds
    public final int keepAliveCount;         // Default: 5
    public final Duration cleanupInterval;     // Default: 30 seconds
    public final boolean enableWarmup;       // Default: false
    public final Set<String> warmupPeers;   // Default: empty
}
```

### 3. Metrics Tracking (PoolMetrics)
```java
public record PoolMetrics(
    int poolSize,              // Current size of connection pool
    int activeConnections,       // Currently in-use connections
    int availableConnections,    // Available for reuse
    int waitingConnections,      // Threads waiting for connections
    long connectionsCreated,    // Total connections created
    long connectionsClosed,     // Total connections closed
    long connectionsReused,    // Total connections reused
    double hitRate,           // Hit rate (reused / total acquired)
    double reuseRate          // Reuse ratio (reused / created)
)
```

### 4. Public API Methods

#### connectWithPool(String address)
```java
public CompletableFuture<PeerConnection> connectWithPool(String address)
```
- Acquires a connection from pool or creates a new one
- Reuses existing connections when available
- Tracks access patterns for keep-alive optimization
- Returns failed future if pool is exhausted

#### releaseConnection(String peerId)
```java
public void releaseConnection(String peerId)
```
- Returns a connection to pool for reuse
- Updates last access time
- Decrements active connection count

#### releaseConnection(PeerConnection conn)
```java
public void releaseConnection(PeerConnection conn)
```
- Convenience overload for releasing by connection object
- Internally calls `releaseConnection(String peerId)`

#### getPoolMetrics()
```java
public PoolMetrics getPoolMetrics()
```
- Returns current pool metrics
- Calculates hit rate and reuse rate
- Thread-safe snapshot of pool state

### 5. Lifecycle Management

#### Initialization
- Pool initialized in constructor via `initializeConnectionPool()`
- Optional warmup of frequently used peers on startup
- Periodic cleanup task scheduled on initialization

#### Cleanup
```java
private void cleanup()
```
Runs every `cleanupInterval` (default 30s):
- Removes inactive connections beyond `idleTimeout` (default 60s)
- Enforces `maxPoolSize` limit
- Preserves keep-alive connections
- Updates connection counts

#### Destroy
```java
public void destroy()
```
- Cancels periodic cleanup tasks
- Shuts down cleanup executor gracefully
- Closes all pooled connections
- Clears all tracking maps

### 6. Connection Reuse Strategy

The pool implements a three-tier reuse strategy:

1. **First Priority**: Reuse existing active connection
   - Checks pool for existing connection to same peer
   - Validates connection is active before reuse
   - Updates access time and count

2. **Second Priority**: Create new connection (if pool has space)
   - Checks if `currentPoolSize < maxPoolSize`
   - Creates new connection via `connect(address)`
   - Adds to pool for future reuse

3. **Fallback**: No pooling for single connections
   - When `minPoolSize == 1 && currentPoolSize == 0`
   - Creates connection without pooling
   - Suitable for one-off connections

### 7. Keep-Alive Management

```java
private void updateKeepAlivePeers()
```
- Identifies top N frequently accessed peers
- Maintains `keepAliveCount` connections (default: 5)
- Protects these connections from idle timeout cleanup
- Updated on every connection reuse

### 8. Warmup Feature

```java
private void warmupConnections(Set<String> peerAddresses)
```
- Pre-connects to configured peers on startup
- Executes asynchronously to not block initialization
- Adds successful connections to pool
- Increments `connectionsCreated` counter

### 9. Thread-Safety Guarantees

All operations use thread-safe constructs:
- `ConcurrentHashMap` for storage (lock-free reads, fine-grained locking)
- `AtomicInteger` for connection counts
- `AtomicLong` for metrics
- `ConcurrentHashMap.newKeySet()` for keep-alive tracking
- `ScheduledExecutorService` with single-threaded executor for cleanup

## Usage Examples

### Basic Usage with Pooling

```java
BlobNetworkOptions options = BlobNetworkOptions.builder()
    .connectionOptions(
        ConnectionPoolOptions.builder()
            .minPoolSize(1)
            .maxPoolSize(10)
            .idleTimeout(Duration.ofSeconds(60))
            .keepAliveCount(5)
            .cleanupInterval(Duration.ofSeconds(30))
            .build()
    )
    .build();

BlobNetwork network = new BlobNetwork(blobStore, options);

// Use pooled connection
CompletableFuture<PeerConnection> future = network.connectWithPool("ws://peer.example.com:8080");
future.thenAccept(conn -> {
    // Use connection
    // ...
    
    // Return to pool when done
    network.releaseConnection(conn);
});
```

### With Warmup

```java
Set<String> warmupPeers = Set.of(
    "ws://peer1.example.com:8080",
    "ws://peer2.example.com:8080",
    "ws://peer3.example.com:8080"
);

BlobNetworkOptions options = BlobNetworkOptions.builder()
    .connectionOptions(
        ConnectionPoolOptions.builder()
            .enableWarmup(true)
            .warmupPeers(warmupPeers)
            .build()
    )
    .build();

BlobNetwork network = new BlobNetwork(blobStore, options);
// Prewarms connections in background
```

### Monitoring Pool Metrics

```java
PoolMetrics metrics = network.getPoolMetrics();
System.out.println("Pool size: " + metrics.poolSize());
System.out.println("Active: " + metrics.activeConnections());
System.out.println("Available: " + metrics.availableConnections());
System.out.println("Hit rate: " + String.format("%.2f%%", metrics.hitRate() * 100));
System.out.println("Reuse rate: " + String.format("%.2fx", metrics.reuseRate()));
```

## Configuration Guidelines

### Pool Sizing

| Environment | minPoolSize | maxPoolSize |
|-------------|--------------|--------------|
| Development | 1 | 5 |
| Testing | 1 | 10 |
| Production | 5 | 50 |

### Timeout Values

| Parameter | Default | Production Range |
|-----------|----------|------------------|
| idleTimeout | 60s | 30-300s |
| cleanupInterval | 30s | 15-60s |
| keepAliveCount | 5 | 3-10 |

### When to Use Pooling

**Use pooling when:**
- Multiple connections to same peers
- Connections are reused frequently
- Latency is a concern
- Want to limit connection count

**Don't use pooling when:**
- Single-use connections only
- Connection count is low (minPoolSize=1 disables pooling)
- Connection overhead is negligible

## Performance Characteristics

### Advantages
- **Reduced latency**: Reusing connections avoids TCP handshake
- **Resource efficiency**: Limits total connection count
- **Connection resilience**: Maintains keep-alive for frequent peers
- **Predictable behavior**: Enforces pool size limits

### Considerations
- **Memory overhead**: Each pooled connection maintains state
- **Cleanup overhead**: Periodic cleanup runs every 30s by default
- **Pool exhaustion**: Returns failed future when at maxPoolSize

## Implementation Details

### Best Practices Applied

Based on research from production-ready Java libraries (HikariCP, OkHttp, Apache HttpClient):

1. **Thread-safe collections**: `ConcurrentHashMap` for lock-free reads
2. **Atomic counters**: `AtomicInteger`/`AtomicLong` for metrics
3. **Periodic cleanup**: `ScheduledExecutorService` for idle connection removal
4. **Keep-alive management**: Track frequently accessed peers
5. **Graceful shutdown**: Cancel tasks, close connections, shutdown executors
6. **Connection validation**: Check `channel().isActive()` before reuse
7. **Access tracking**: Track both frequency and recency for keep-alive decisions

### Thread-Safety Pattern

```java
// Lock-free read from pool
PeerConnection existing = connectionPool.get(peerId);

// Atomic counter update
availableConnections.incrementAndGet();

// Concurrent map update
lastAccessTime.put(peerId, System.currentTimeMillis());

// Atomic compare-and-set logic
if (connectionPool.size() >= maxPoolSize) {
    return CompletableFuture.failedFuture(...);
}
```

## Error Handling

### Pool Exhausted
```java
return CompletableFuture.failedFuture(
    new IllegalStateException("Connection pool exhausted")
);
```
Callers should handle this exception by:
1. Waiting and retrying
2. Reducing concurrent connection usage
3. Increasing maxPoolSize

### Connection Invalidation
Connections are automatically removed from pool when:
- Connection is not active (`!channel().isActive()`)
- Exceeds idle timeout (60s default)
- Beyond pool size limit (10 default)
- Not in keep-alive set

## Testing Recommendations

1. **Pool Size Limits**: Verify pool respects maxPoolSize
2. **Idle Timeout**: Confirm idle connections are removed after timeout
3. **Keep-Alive**: Verify frequently accessed peers are preserved
4. **Thread Safety**: Test concurrent acquire/release operations
5. **Metrics Accuracy**: Validate hit rate and reuse rate calculations
6. **Warmup**: Confirm pre-connections are established on startup

## Migration from Non-Pooled Code

### Before
```java
network.connect(address).thenAccept(conn -> {
    // Use connection
    // Connection closed on completion
});
```

### After
```java
network.connectWithPool(address).thenAccept(conn -> {
    try {
        // Use connection
    } finally {
        network.releaseConnection(conn);  // Return to pool
    }
});
```

## Future Enhancements

Potential improvements:
1. Connection leak detection with stack traces
2. Configurable eviction policy (LRU, LFU)
3. Connection validation before reuse
4. Dynamic pool sizing based on load
5. Connection health checks in cleanup
6. Metrics export to Prometheus/Datadog
