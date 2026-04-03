## [11.4.3] - 2026-04-03
### Go Port: Autonomous Supernode & Torrent Seeding
- **Torrent**: Integrated `github.com/anacrolix/torrent` for native file seeding and data provisioning in Go.
- **Market**: Developed a background poller using `github.com/go-resty/resty/v2` to autonomously discover and accept storage bids on the Bobcoin Lattice.
- **Consensus**: Implemented `pkg/torrent/block.go` for Go-native Block Lattice operations, enabling the Supernode to sign and broadcast its own `accept_bid` blocks.
- **Unified Binary**: The `supernode-go` binary now orchestrates tracker, DHT, seeding, and lattice interaction in a single performant process.

## [11.4.2] - 2026-04-03
### Go Port: Tracker, DHT, and Supernode Core
- **Tracker**: Implemented multi-protocol support including BEP 3 (HTTP Bencoded) and BEP 15 (UDP), featuring compact peer list generation.
- **DHT**: Stand up a standalone Kademlia DHT node using `github.com/anacrolix/dht/v2` with full bootstrapping and search capabilities.
- **Supernode**: Initialized the unified `supernode-go` binary with Ed25519 wallet persistence and SPoRA (Succinct Proof of Random Access) challenge handlers.
- **Crypto**: Developed `pkg/torrent/crypto.go` providing Ed25519 signing/verification and SHA-256 hashing compatible with the Bobcoin lattice.

## [11.4.1] - 2026-04-03
### Go Port: Proximity Sorting & Erasure Storage
- **DHT Proxy**: Implemented Haversine distance calculation for discovered peers, sorting `/api/announce` results by proximity to the requester's IP.
- **Storage**: Developed `pkg/storage` in Go, implementing SIMD-accelerated 4+2 erasure coding and IETF ChaCha20-Poly1305 authenticated encryption for high-performance block storage.
- **Security**: Added secure random padding to encrypted blocks to mitigate size-based traffic analysis.

## [11.4.0] - 2026-04-03
### Submodule Synchronization & Documentation Synthesis
- **Bobcoin**: Synchronized `bobcoin` submodule to `v3.5.0`, including the latest NFT protocol, atomic swaps, and lattice consensus features.
- **Universal Instructions**: Implemented `docs/UNIVERSAL_LLM_INSTRUCTIONS.md` as the single source of truth for all AI agents across the monorepo.
- **Dashboard**: Refreshed the root-level `DASHBOARD.md` to reflect the latest project structure and submodule versions.
- **CI/CD**: Verified `bobcoin` build results and synchronized nested research repositories.

## [11.3.1] - 2026-04-02
### DHT Proxy Crawler & Database
- **Implementation**: Developed a SQLite-backed peer storage system and a DHT crawler for the DHT Proxy utility.
- **Features**: Added asynchronous DHT search triggering on torrent addition and a private announce API for peer discovery.
- **Dependencies**: Integrated `github.com/anacrolix/dht/v2` and `modernc.org/sqlite`.

## [11.3.0] - 2026-04-02
### Go Port & DHT Proxy Initialization
- **Architecture**: Planned the entire project's port to Go for enhanced performance, concurrency, and memory safety.
- **Utility**: Initialized the DHT Proxy utility to hide user IPs from the BitTorrent DHT and public trackers.
- **Scaffolding**: Created the `bobtorrent` Go module and initial structure for the DHT Proxy.

## [11.2.4] - 2026-03-09
### Omni-workspace Stabilization & Autonomous Refactoring
- **Documentation**: Consolidated Agent instructions into `UNIVERSAL_LLM_INSTRUCTIONS.md`. Rebuilt `VISION.md`, `ROADMAP.md`, `TODO.md`, `DASHBOARD.md`, `DEPLOY.md`, `MEMORY.md`. 
- **Merge Resolutions**: Intelligently merged `feature/megatorrent-reference` and `megatorrent-reference-client-ui`. Resolved critical conflicts in `lib/manifest.js`, retaining deterministic `fast-json-stable-stringify` validation while merging new XSalsa20 manifest encryption capabilities.
- **Submodules**: Synchronized and fixed detached HEADs in the `bobcoin` and `qbittorrent` submodules.

## [11.2.3] - 2026-02-05
### Tracker Polish
- **Dep Updates**: Bumped bittorrent-dht to ^11.0.11
- **UI Integrations**: Preliminary support for megatorrent client webui.

## [11.2.2] - 2025-11-20
### Java Supernode Erasure Coding & Fixes
- **Cipher Migration**: ChaCha20 → AES/GCM (MuxEngine.java).
- **Network**: Added freenet and ipfs transport schemes. Fixed WebSocket handshake timings.

## [11.2.1] - 2025-08-15
### Initial Supernode Beta Integration
- Integrated Java Supernode capabilities alongside standard Node.js tracker.
