# Project Memory & Observations

## Architectural Transitions
-   **Go Port Progress**: The project is successfully transitioning to Go. We now have a unified suite of binaries (`supernode-go`, `lattice-go`, `dht-proxy`) that mirror and exceed the performance of the legacy Node.js/Java stack.
-   **Lattice Dominance**: The Bobcoin economy has fully migrated from SQLite-based persistence to a native, asynchronous Block Lattice consensus model. This is now implemented in both Node.js and Go.
-   **Storage Excellence**: The Go `pkg/storage` implementation provides 1:1 feature parity with the legacy stack while gaining SIMD-accelerated Reed-Solomon encoding and IETF-standard authenticated encryption.

## Design Preferences
-   **Privacy First**: Integration of GeoIP distance-based sorting was carefully implemented to ensure no identifying information is leaked to public trackers; the proxy acts as the sole point of contact.
-   **Autonomous Operations**: The Supernode now autonomously polls the market and accepts bids without human intervention, fulfilling the "Production-Grade Autonomous" vision.
-   **Developer Experience**: The introduction of the Bubble Tea TUI dashboard has significantly improved the observability of the Supernode.

## Technical Debt & Roadblocks
-   **Submodule Reachability**: `qbittorrent` submodule remote is currently unreachable on GitHub (`repository not found`), though local files are intact. This needs investigation in future sessions.
-   **Rust SP1 Integration**: The ZK-Service integration in the `game-server` currently uses an "AI Oracle" mock. Real SP1 verification requires a localized Rust environment.
