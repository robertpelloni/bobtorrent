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
            // Configure storage options here if needed (e.g. persistent blob store path)

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
