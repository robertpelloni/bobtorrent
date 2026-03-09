package io.supernode.storage;

import java.time.Instant;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Optional;
import java.util.concurrent.atomic.AtomicLong;
import java.util.concurrent.atomic.AtomicReference;
import java.util.concurrent.locks.ReadWriteLock;
import java.util.concurrent.locks.ReentrantReadWriteLock;

import io.supernode.storage.BlobStore.BlobCache;
import io.supernode.storage.BlobStore.CacheStats;

/**
 * Thread-safe LRU implementation of BlobCache.
 * Uses a size-bounded LinkedHashMap for underlying storage.
 */
public class LRUBlobCache implements BlobCache {

    private final ReadWriteLock lock = new ReentrantReadWriteLock();
    private volatile CacheOptions options;

    private LinkedHashMap<String, byte[]> cacheMap;
    
    // Stats
    private final AtomicLong hits = new AtomicLong();
    private final AtomicLong misses = new AtomicLong();
    private final AtomicLong evictions = new AtomicLong();
    private final AtomicLong currentBytes = new AtomicLong();
    private final AtomicReference<Instant> lastEviction = new AtomicReference<>(Instant.now());

    public LRUBlobCache() {
        this(CacheOptions.defaults());
    }

    public LRUBlobCache(CacheOptions options) {
        configure(options);
    }

    @Override
    public final void configure(CacheOptions options) {
        lock.writeLock().lock();
        try {
            this.options = options;
            
            // Re-initialize map with new capacity constraints
            LinkedHashMap<String, byte[]> oldMap = this.cacheMap;
            
            this.cacheMap = new LinkedHashMap<>(16, 0.75f, true);
            
            if (oldMap != null) {
                // Pre-fill with old data, which might trigger evictions naturally
                for (Map.Entry<String, byte[]> entry : oldMap.entrySet()) {
                    this.cacheMap.put(entry.getKey(), entry.getValue());
                }
            }
            evictIfNeeded();
        } finally {
            lock.writeLock().unlock();
        }
    }

    private void evictIfNeeded() {
        while (!cacheMap.isEmpty() && 
               (cacheMap.size() > options.maxEntries() || currentBytes.get() > options.maxBytes())) {
            var iterator = cacheMap.entrySet().iterator();
            if (iterator.hasNext()) {
                var eldest = iterator.next();
                iterator.remove();
                currentBytes.addAndGet(-eldest.getValue().length);
                evictions.incrementAndGet();
                lastEviction.set(Instant.now());
            } else {
                break;
            }
        }
    }

    @Override
    public Optional<byte[]> get(String hash) {
        lock.readLock().lock();
        try {
            byte[] data = cacheMap.get(hash);
            if (data != null) {
                hits.incrementAndGet();
                return Optional.of(data);
            } else {
                misses.incrementAndGet();
                return Optional.empty();
            }
        } finally {
            lock.readLock().unlock();
        }
    }

    @Override
    public void put(String hash, byte[] data) {
        if (data == null) return;
        
        lock.writeLock().lock();
        try {
            byte[] oldData = cacheMap.put(hash, data);
            
            if (oldData != null) {
                currentBytes.addAndGet(data.length - oldData.length);
            } else {
                currentBytes.addAndGet(data.length);
            }
            evictIfNeeded();
        } finally {
            lock.writeLock().unlock();
        }
    }

    @Override
    public boolean has(String hash) {
        lock.readLock().lock();
        try {
            return cacheMap.containsKey(hash);
        } finally {
            lock.readLock().unlock();
        }
    }

    @Override
    public void invalidate(String hash) {
        lock.writeLock().lock();
        try {
            byte[] removed = cacheMap.remove(hash);
            if (removed != null) {
                currentBytes.addAndGet(-removed.length);
            }
        } finally {
            lock.writeLock().unlock();
        }
    }

    @Override
    public void clear() {
        lock.writeLock().lock();
        try {
            cacheMap.clear();
            currentBytes.set(0);
        } finally {
            lock.writeLock().unlock();
        }
    }

    @Override
    public CacheStats stats() {
        long h = hits.get();
        long m = misses.get();
        double hitRate = (h + m > 0) ? (double) h / (h + m) : 0.0;
        
        lock.readLock().lock();
        int count = cacheMap.size();
        lock.readLock().unlock();

        return new CacheStats(
            h,
            m,
            evictions.get(),
            currentBytes.get(),
            count,
            hitRate,
            lastEviction.get()
        );
    }
}
