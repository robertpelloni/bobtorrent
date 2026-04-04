## [11.8.0] - 2026-04-03
### Go Port: Real Shard Upload + Manifest Publication Flow
- **Publication Registry**: Added `internal/publish` with durable shard + manifest persistence for supernode-hosted Bobtorrent assets, including a tested content-addressed shard store and manifest registry.
- **Supernode Publish API**: Expanded `supernode-go` with `POST /upload-shard`, `POST /publish-manifest`, `GET /manifests/:id`, and `GET /shards/:hash`, plus permissive CORS for browser-based Bobcoin integration.
- **Bobcoin Workflow**: Updated the Bobcoin frontend workbench to upload WASM-prepared shards directly to the Go supernode and publish a retrievable manifest entry, upgrading the flow from preprocessing-only to actual publication.
- **Validation**: Verified `go test ./internal/publish`, `go build -buildvcs=false ./...`, and a successful Bobcoin frontend production build after publish-flow wiring.

## [11.7.0] - 2026-04-03
### Go Port: Bobcoin WASM Frontend Wiring & Supernode UI Compatibility
- **Bobcoin Integration**: Updated the `bobcoin` submodule to `v8.7.0`, integrating a browser-side Go storage WASM workbench into the frontend Supernode page and retargeting the default WASM asset origin to the Go supernode.
- **Supernode API Compatibility**: Expanded `supernode-go` with Bobcoin UI-friendly endpoints: `GET /stats`, `POST /add-torrent`, and `POST /remove-torrent`.
- **WASM Artifact Serving**: `supernode-go` now serves `GET /storage.wasm` and `GET /wasm_exec.js` directly from the generated build artifacts so frontend clients can fetch the Go runtime without manual copying.
- **Validation**: Rebuilt the root Go workspace successfully and validated the Bobcoin frontend production build after WASM integration and rebase onto the newer upstream Bobcoin mainline.

## [11.6.0] - 2026-04-03
### Go Port: Compatibility Hardening, Live Feed Integration, and WASM Packaging
- **Consensus Compatibility**: Hardened the Go lattice server to accept both raw block payloads and `{ "block": ... }` wrapped submissions, added `/pending/:account`, `/proposals`, and root WebSocket compatibility for the existing bobcoin frontend.
- **Consensus Features**: Expanded the Go lattice engine with governance, NFT, staking, and swap state transitions plus a temporary legacy compatibility shim for frontend blocks that still omit `height` and `staked_balance`.
- **WebSocket Feed**: Added a real-time lattice WebSocket hub emitting `NEW_BLOCK` events with compatibility-friendly `type`/`event` fields for both frontend and TUI consumers.
- **Supernode UX**: Upgraded `supernode-go` to subscribe to the lattice feed, publish richer TUI state, and operate against the repaired DHT/tracker/storage integrations.
- **WASM Packaging**: Added `web/storage-wasm-loader.js`, documented the bridge in `docs/WASM_STORAGE_BRIDGE.md`, and updated `build.bat` to package `storage.wasm` and `wasm_exec.js` automatically.
- **Build Validation**: Fixed compile issues caused by third-party API drift and verified `go build -buildvcs=false ./...` plus explicit native/WASM artifact builds.

## [11.5.1] - 2026-04-03
### Go Port: WASM Briding & Consensus Hardening
- **WASM**: Compiled the high-performance Go storage primitives (ChaCha20-Poly1305 and Reed-Solomon) to WebAssembly (`storage.wasm`), enabling browser-side zero-trust storage sharding.
- **P2P Consensus**: Implemented HTTP-based block broadcasting between `lattice-go` instances, hardening the consensus layer against single-node failures.
- **Bridges**: Developed `internal/bridges/filecoin.go` to provide a standardized interface for cross-chain metadata archival, integrated directly into the Supernode's autonomous polling loop.
- **Build**: Integrated WASM compilation into the main `build.bat` pipeline.

## [11.5.0] - 2026-04-03
### Go Port: Lattice Consensus Engine & Ecosystem Unification
- **Consensus**: Ported the entire asynchronous block lattice engine from Node.js to Go (`internal/consensus`). Implemented secure chain validation, demurrage calculations, and O(1) block indexing.
- **Server**: Developed a high-performance HTTP API for the Go lattice node, enabling full compatibility with existing frontend and supernode interactions.
- **Unification**: Structured the Go port into a suite of specialized binaries (`lattice-go`, `supernode-go`, `dht-proxy`) for maximum scalability and deployment flexibility.
- **Build System**: Updated `build.bat` to orchestrate the compilation of the entire unified Go ecosystem.

## [11.4.4] - 2026-04-03
### Go Port: Supernode TUI Dashboard
- **TUI**: Implemented a comprehensive terminal dashboard using `github.com/charmbracelet/bubbletea`, providing real-time visibility into account balances, lattice market bids, and node status.
- **Visuals**: Leveraged `lipgloss` for a high-fidelity cyberpunk terminal aesthetic, featuring styled tables and neon accents.
- **Event Driven**: Integrated the background poller with the TUI via thread-safe message passing, ensuring smooth UI updates during autonomous bid acceptance.

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
