package io.supernode.storage;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.time.Duration;
import java.util.Optional;

import static org.junit.jupiter.api.Assertions.*;

class LRUBlobCacheTest {

    private LRUBlobCache cache;

    @BeforeEach
    void setUp() {
        cache = new LRUBlobCache();
    }

    @Test
    void testPutAndGet() {
        BlobStore.BlobCache.CacheOptions options = new BlobStore.BlobCache.CacheOptions(
                1024,
                10,
                Duration.ofMinutes(1),
                BlobStore.BlobCache.EvictionPolicy.LRU,
                false
        );
        cache.configure(options);

        cache.put("hash1", new byte[]{1, 2, 3});
        
        Optional<byte[]> data = cache.get("hash1");
        assertTrue(data.isPresent());
        assertArrayEquals(new byte[]{1, 2, 3}, data.get());
    }

    @Test
    void testEvictionByMaxEntries() {
        BlobStore.BlobCache.CacheOptions options = new BlobStore.BlobCache.CacheOptions(
                1024,
                2,
                Duration.ofMinutes(1),
                BlobStore.BlobCache.EvictionPolicy.LRU,
                false
        );
        cache.configure(options);

        cache.put("hash1", new byte[]{1});
        cache.put("hash2", new byte[]{2});
        
        // This should evict hash1
        cache.put("hash3", new byte[]{3});

        assertFalse(cache.get("hash1").isPresent());
        assertTrue(cache.get("hash2").isPresent());
        assertTrue(cache.get("hash3").isPresent());
    }
    
    @Test
    void testEvictionByMaxSize() {
        BlobStore.BlobCache.CacheOptions options = new BlobStore.BlobCache.CacheOptions(
                20, // 20 bytes max
                10,
                Duration.ofMinutes(1),
                BlobStore.BlobCache.EvictionPolicy.LRU,
                false
        );
        cache.configure(options);

        cache.put("hash1", new byte[10]);
        cache.put("hash2", new byte[10]);
        
        // The cache is now full (20 bytes). Adding more should evict.
        // This should evict hash1
        cache.put("hash3", new byte[5]);

        assertFalse(cache.get("hash1").isPresent());
        assertTrue(cache.get("hash2").isPresent());
        assertTrue(cache.get("hash3").isPresent());
    }

    @Test
    void testLRUOrder() {
        BlobStore.BlobCache.CacheOptions options = new BlobStore.BlobCache.CacheOptions(
                1024,
                3,
                Duration.ofMinutes(1),
                BlobStore.BlobCache.EvictionPolicy.LRU,
                false
        );
        cache.configure(options);

        cache.put("hash1", new byte[]{1});
        cache.put("hash2", new byte[]{2});
        cache.put("hash3", new byte[]{3});
        
        // Access hash1 to make it recently used
        cache.get("hash1");
        
        // Adding hash4 should evict hash2, not hash1 (which would be evicted without the access)
        cache.put("hash4", new byte[]{4});

        assertTrue(cache.get("hash1").isPresent());
        assertFalse(cache.get("hash2").isPresent());
        assertTrue(cache.get("hash3").isPresent());
        assertTrue(cache.get("hash4").isPresent());
    }
    
    @Test
    void testInvalidate() {
        BlobStore.BlobCache.CacheOptions options = new BlobStore.BlobCache.CacheOptions(
                1024,
                10,
                Duration.ofMinutes(1),
                BlobStore.BlobCache.EvictionPolicy.LRU,
                false
        );
        cache.configure(options);

        cache.put("hash1", new byte[]{1, 2, 3});
        assertTrue(cache.get("hash1").isPresent());
        
        cache.invalidate("hash1");
        assertFalse(cache.get("hash1").isPresent());
    }
    
    @Test
    void testClear() {
        BlobStore.BlobCache.CacheOptions options = new BlobStore.BlobCache.CacheOptions(
                1024,
                10,
                Duration.ofMinutes(1),
                BlobStore.BlobCache.EvictionPolicy.LRU,
                false
        );
        cache.configure(options);

        cache.put("hash1", new byte[]{1});
        cache.put("hash2", new byte[]{2});
        
        cache.clear();
        
        assertFalse(cache.get("hash1").isPresent());
        assertFalse(cache.get("hash2").isPresent());
    }
}
