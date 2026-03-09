package io.supernode.cli;

import io.supernode.network.SupernodeNetwork;
import io.supernode.storage.FileBlobStore;
import io.supernode.storage.mux.Manifest;

import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.HexFormat;
import java.util.Optional;

public class NodeCLI {

    public static void main(String[] args) {
        if (args.length == 0) {
            printUsage();
            System.exit(1);
        }

        String command = args[0];

        try {
            switch (command) {
                case "status":
                    handleStatus();
                    break;
                case "manifest-inspect":
                    if (args.length < 2) {
                        System.err.println("Usage: node-cli manifest-inspect <cid> [masterKeyHex]");
                        System.exit(1);
                    }
                    String cid = args[1];
                    String masterKeyHex = args.length > 2 ? args[2] : null;
                    handleManifestInspect(cid, masterKeyHex);
                    break;
                default:
                    System.err.println("Unknown command: " + command);
                    printUsage();
                    System.exit(1);
            }
        } catch (Exception e) {
            System.err.println("Error executing command '" + command + "': " + e.getMessage());
            e.printStackTrace();
            System.exit(1);
        }
    }

    private static void printUsage() {
        System.out.println("Supernode CLI Configuration & Diagnostics");
        System.out.println("Usage:");
        System.out.println("  status                          - Output network peer counts, DHT state, and JVM memory metrics");
        System.out.println("  manifest-inspect <cid> [hexKey] - Parse, decrypt, and print a verified JSON manifest from storage");
    }

    private static void handleStatus() throws Exception {
        System.out.println("Bootstrapping minimal Supernode context for status check...");
        SupernodeNetwork network = new SupernodeNetwork();
        network.start().get();

        System.out.println("\n--- Supernode Status ---");
        
        // Peer Counts
        System.out.println("Total Peers Connected: " + network.getPeers().size());

        // Kademlia DHT State
        System.out.println("DHT Health State: " + network.getDht().getHealthStatus().message());
        System.out.println("DHT Known Nodes: " + network.getDht().getStats().knownPeers());
        
        // Storage Metrics
        io.supernode.storage.SupernodeStorage.StorageStats sStats = network.getStorageStats();
        long maxBytes = network.getStorage().getOptions().maxStorageBytes;
        System.out.println("Storage Usage: " + (sStats.totalBytes() / 1024 / 1024) + " MB");
        System.out.println("Storage Quota limit: " + (maxBytes == 0 ? "Unlimited" : (maxBytes / 1024 / 1024) + " MB"));
        System.out.println("Total Manifests: " + sStats.manifestCount());

        // Memory Metrics
        Runtime runtime = Runtime.getRuntime();
        long maxMemory = runtime.maxMemory();
        long allocatedMemory = runtime.totalMemory();
        long freeMemory = runtime.freeMemory();
        System.out.println("JVM Max Memory: " + (maxMemory / 1024 / 1024) + " MB");
        System.out.println("JVM Allocated Memory: " + (allocatedMemory / 1024 / 1024) + " MB");
        System.out.println("JVM Free Memory: " + (freeMemory / 1024 / 1024) + " MB");

        network.destroyAsync().get();
        System.exit(0);
    }

    private static void handleManifestInspect(String cid, String masterKeyHex) throws Exception {
        Path blobDir = Paths.get("data", "blobstore");
        System.out.println("Inspecting manifest storage at: " + blobDir.toAbsolutePath());
        
        FileBlobStore blobStore = new FileBlobStore(blobDir);
        Optional<byte[]> blobData = blobStore.get(cid);

        if (blobData.isEmpty()) {
            System.err.println("Manifest/Blob not found in local FileBlobStore for CID: " + cid);
            System.exit(1);
        }

        byte[] rawBytes = blobData.get();
        System.out.println("Found blob. Size: " + rawBytes.length + " bytes.");

        try {
            Manifest manifest;
            if (masterKeyHex != null) {
                System.out.println("Attempting decryption with provided master key...");
                byte[] masterKey = HexFormat.of().parseHex(masterKeyHex);
                byte[] manifestKey = Manifest.deriveManifestKey(masterKey, cid);
                manifest = Manifest.decrypt(rawBytes, manifestKey);
                System.out.println("Decryption successful.");
            } else {
                System.out.println("No key provided. Attempting to parse as plain JSON manifest...");
                com.fasterxml.jackson.databind.ObjectMapper mapper = new com.fasterxml.jackson.databind.ObjectMapper();
                manifest = mapper.readValue(rawBytes, Manifest.class);
                System.out.println("Parsing successful.");
            }

            System.out.println("\n--- Manifest Payload ---");
            System.out.println("File ID: " + manifest.getFileId());
            System.out.println("File Name: " + manifest.getFileName());
            System.out.println("File Size: " + manifest.getFileSize());
            System.out.println("Erasure Config: Data=" + manifest.getErasure().dataShards() + 
                               ", Parity=" + manifest.getErasure().parityShards());
            System.out.println("Segments: " + manifest.getSegments().size());

            System.out.println("\nVerifying integrity against local storage...");
            try {
                // To adapt to Manifest internal BlobStore interface.
                manifest.verifyIntegrity(hash -> blobStore.get(hash));
                System.out.println("Integrity Check: PASSED");
            } catch (IllegalStateException e) {
                System.err.println("Integrity Check: FAILED (" + e.getMessage() + ")");
            }

        } catch (Exception e) {
            System.err.println("Failed to parse/decrypt manifest. It may be encrypted and requires a key.");
            e.printStackTrace();
            System.exit(1);
        }
    }
}
