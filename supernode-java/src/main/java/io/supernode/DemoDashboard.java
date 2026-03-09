package io.supernode;

import io.supernode.network.UnifiedNetwork;

public class DemoDashboard {
    public static void main(String[] args) throws Exception {
        UnifiedNetwork.UnifiedNetworkOptions options = UnifiedNetwork.UnifiedNetworkOptions.allNetworks();
        options.enableDashboard = true;
        options.dashboardPort = 8080;
        
        System.out.println("Starting UnifiedNetwork with Dashboard on port 8080...");
        UnifiedNetwork network = new UnifiedNetwork(options);
        network.start().join();
        
        System.out.println("Started! Open http://localhost:8080 in your browser. Press Ctrl+C to stop.");
        Thread.currentThread().join();
    }
}
