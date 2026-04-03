# Bobtorrent Omni-Workspace Handoff (v11.4.0)

## Session Summary
-   **Submodule Sync**: Synchronized `bobcoin` to the latest `v3.6.0` (origin/main), which includes NFT protocols, atomic swaps, and the new asynchronous block lattice consensus. 
-   **GeoIP Enrichment**: Implemented `pkg/torrent/geoip.go` and updated `internal/dhtproxy` to enrich discovered BitTorrent peers with location data (country, lat, lon) using MaxMind GeoLite2.
-   **Instruction Unification**: Implemented `docs/UNIVERSAL_LLM_INSTRUCTIONS.md` as the monorepo-wide protocol. All `AGENTS.md`, `CLAUDE.md`, etc., now point to this single source.
-   **Build Process**: Updated `build.bat` to compile the new Go port (`dht-proxy`) into `build/`.
-   **Documentation**: Refreshed `DASHBOARD.md`, `ROADMAP.md`, and `TODO.md` to reflect the current state of the Go port and `bobcoin` integration.

## Key Observations
-   `bobcoin` has undergone massive development, moving from SQLite to a native block lattice for consensus and governance.
-   The Go Port is focusing on privacy-first utilities like the DHT Proxy before implementing the full tracker.
-   `qbittorrent` submodule remote is currently unreachable (`repository not found`), but local files are present and synced to `v5.1.0-beta`.

## Next Steps for Implementor
1.  **DHT Proxy Distance Sorting**: Update `internal/dhtproxy/database.go` and `cmd/dht-proxy/main.go` to sort peers returned in `/api/announce` based on proximity to the requester's IP (using `GeoIPService`).
2.  **Go Storage Layer**: Begin implementation of `pkg/storage` in Go, porting the erasure coding (Reed-Solomon) logic from the Node.js/Java implementations.
3.  **Bobcoin UI integration**: Verify that the latest `v3.6.0` features in `bobcoin/frontend` are correctly calling the `bobcoin-consensus` server.
4.  **Java Supernode Modernization**: Check if any logic from the Java Supernode should be migrated to the Go port next.
