package io.supernode.storage;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.Optional;

import static org.junit.jupiter.api.Assertions.*;

class FileBlobStoreTest {

    @TempDir
    Path tempDir;

    private FileBlobStore store;

    @BeforeEach
    void setUp() {
        store = new FileBlobStore(tempDir);
    }

    @Test
    void testPutAndGet() {
        String data = "Hello, World!";
        String hash = store.computeHash(data.getBytes());

        store.put(hash, data.getBytes());

        assertTrue(store.has(hash));
        Optional<byte[]> retrieved = store.get(hash);
        assertTrue(retrieved.isPresent());
        assertEquals(data, new String(retrieved.get()));
    }

    @Test
    void testDeduplication() throws IOException {
        String data = "Duplicate Data";
        String hash = store.computeHash(data.getBytes());

        store.put(hash, data.getBytes());
        Path blobPath = findBlobPath(hash);
        long size1 = Files.size(blobPath);
        long modTime1 = Files.getLastModifiedTime(blobPath).toMillis();

        // Put same data again
        try {
            Thread.sleep(100);
        } catch (InterruptedException ignored) {
        } // Ensure time passes
        store.put(hash, data.getBytes());

        long modTime2 = Files.getLastModifiedTime(blobPath).toMillis();

        assertEquals(modTime1, modTime2, "File should not be modified on deduplication");
    }

    @Test
    void testDelete() {
        String data = "To be deleted";
        String hash = store.computeHash(data.getBytes());

        store.put(hash, data.getBytes());
        assertTrue(store.has(hash));

        boolean deleted = store.delete(hash);
        assertTrue(deleted);
        assertFalse(store.has(hash));
    }

    @Test
    void testPersistence() {
        String data = "Persistent Data";
        String hash = store.computeHash(data.getBytes());

        store.put(hash, data.getBytes());

        // Create new store instance on same directory
        FileBlobStore newStore = new FileBlobStore(tempDir);
        assertTrue(newStore.has(hash));
        assertEquals(data, new String(newStore.get(hash).get()));
    }

    private Path findBlobPath(String hash) {
        if (hash.length() < 4)
            return tempDir.resolve(hash);
        return tempDir.resolve(hash.substring(0, 2))
                .resolve(hash.substring(2, 4))
                .resolve(hash);
    }
}
