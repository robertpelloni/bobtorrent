package io.supernode;

import io.supernode.api.WebController;
import io.supernode.network.UnifiedNetwork;

public class Supernode {

    public static void main(String[] args) {
        System.out.println("Starting Megatorrent Supernode (Java)...");

        try {
            // parse args for port
            int port = 8080;
            if (args.length > 0) {
                port = Integer.parseInt(args[0]);
            }

            UnifiedNetwork.UnifiedNetworkOptions options = UnifiedNetwork.UnifiedNetworkOptions.allNetworks();

            // Enable persistence
            java.io.File storageDir = new java.io.File("supernode_storage");
            if (!storageDir.exists()) storageDir.mkdirs();
            options.blobStore = new io.supernode.storage.FileBlobStore(storageDir.toPath());

            UnifiedNetwork network = new UnifiedNetwork(options);
            network.start().join();
            System.out.println("Network started.");

            WebController web = new WebController(port, network);
            web.start(); // This blocks

        } catch (Exception e) {
            e.printStackTrace();
            System.exit(1);
        }
    }
}
