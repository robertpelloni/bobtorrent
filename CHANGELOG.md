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
