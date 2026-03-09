package io.supernode.storage.mux;

import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import java.security.MessageDigest;
import java.util.HashMap;
import java.util.HexFormat;
import java.util.List;
import java.util.Map;
import java.util.Optional;

import static org.junit.jupiter.api.Assertions.*;

@DisplayName("Manifest Integrity")
class ManifestIntegrityTest {

    @Test
    @DisplayName("should pass verification when chunks are valid")
    void shouldPassVerificationWhenChunksValid() throws Exception {
        Map<String, byte[]> store = new HashMap<>();
        byte[] data1 = "chunk1".getBytes();
        byte[] data2 = "chunk2".getBytes();
        
        String hash1 = sha256Hex(data1);
        String hash2 = sha256Hex(data2);
        
        store.put(hash1, data1);
        store.put(hash2, data2);

        Manifest.ManifestOptions options = new Manifest.ManifestOptions(
            "file-1", "test.txt", 1024, "seed", 2048, null,
            List.of(
                createChunkSegment(hash1),
                createChunkSegment(hash2)
            )
        );
        Manifest manifest = Manifest.create(options);
        
        assertDoesNotThrow(() -> {
            manifest.verifyIntegrity(hash -> Optional.ofNullable(store.get(hash)));
        });
    }

    @Test
    @DisplayName("should fail verification when a chunk is missing")
    void shouldFailVerificationWhenChunkMissing() throws Exception {
        Map<String, byte[]> store = new HashMap<>();
        byte[] data1 = "chunk1".getBytes();
        String hash1 = sha256Hex(data1);
        String hash2 = sha256Hex("missing".getBytes());
        
        store.put(hash1, data1);
        // hash2 is missing

        Manifest.ManifestOptions options = new Manifest.ManifestOptions(
            "file-2", "test.txt", 1024, "seed", 2048, null,
            List.of(
                createChunkSegment(hash1),
                createChunkSegment(hash2)
            )
        );
        Manifest manifest = Manifest.create(options);
        
        IllegalStateException e = assertThrows(IllegalStateException.class, () -> {
            manifest.verifyIntegrity(hash -> Optional.ofNullable(store.get(hash)));
        });
        assertTrue(e.getMessage().contains("Integrity verification failed for: Chunk " + hash2));
    }

    @Test
    @DisplayName("should fail verification when chunk hash mismatches")
    void shouldFailVerificationWhenChunkCorrupted() throws Exception {
        Map<String, byte[]> store = new HashMap<>();
        byte[] data1 = "chunk1".getBytes();
        byte[] corruptedData = "corrupted".getBytes();
        
        String hash1 = sha256Hex(data1);
        String expectedHash2 = sha256Hex("original".getBytes());
        
        store.put(hash1, data1);
        store.put(expectedHash2, corruptedData);

        Manifest.ManifestOptions options = new Manifest.ManifestOptions(
            "file-3", "test.txt", 1024, "seed", 2048, null,
            List.of(
                createChunkSegment(hash1),
                createChunkSegment(expectedHash2)
            )
        );
        Manifest manifest = Manifest.create(options);
        
        IllegalStateException e = assertThrows(IllegalStateException.class, () -> {
            manifest.verifyIntegrity(hash -> Optional.ofNullable(store.get(hash)));
        });
        assertTrue(e.getMessage().contains("Integrity verification failed for: Chunk " + expectedHash2));
    }
    
    @Test
    @DisplayName("should pass verification when shards are valid")
    void shouldPassVerificationWhenShardsValid() throws Exception {
        Map<String, byte[]> store = new HashMap<>();
        byte[] shard1 = "shard1".getBytes();
        byte[] shard2 = "shard2".getBytes();
        
        String hash1 = sha256Hex(shard1);
        String hash2 = sha256Hex(shard2);
        
        store.put(hash1, shard1);
        store.put(hash2, shard2);

        Manifest.ManifestOptions options = new Manifest.ManifestOptions(
            "file-4", "test.txt", 1024, "seed", 2048, new Manifest.ErasureConfig(1, 1),
            List.of(
                createErasureSegment(List.of(
                    new Manifest.ShardInfo(0, hash1, shard1.length),
                    new Manifest.ShardInfo(1, hash2, shard2.length)
                ))
            )
        );
        Manifest manifest = Manifest.create(options);
        
        assertDoesNotThrow(() -> {
            manifest.verifyIntegrity(hash -> Optional.ofNullable(store.get(hash)));
        });
    }

    private Manifest.Segment createChunkSegment(String hash) {
        return new Manifest.Segment(hash, "key", "seed", 0, 1, 100, 100, null, null, null);
    }
    
    private Manifest.Segment createErasureSegment(List<Manifest.ShardInfo> shards) {
        return new Manifest.Segment(null, "key", "seed", 0, 1, 100, 100, shards, 100, 50);
    }

    private String sha256Hex(byte[] data) throws Exception {
        MessageDigest md = MessageDigest.getInstance("SHA-256");
        return HexFormat.of().formatHex(md.digest(data));
    }
}
