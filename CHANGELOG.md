## [11.60.53] - 2026-11-14
### Peer Sync Mesh Panel
- Wallet & Lattice tab gains Peer Sync Mesh: live `/peers` diagnostics (health, last sync, success/failure counts) + Add Peer & Sync form (`POST /peers`). Live-verified: peer :4002 registered through the supernode proxy, health `healthy`/`synced` 1/0. Dashboard backend coverage is now complete (all lattice + supernode surfaces).

## [11.60.52] - 2026-11-14
### Lattice Multi-Node: Port/Peer Flags + Live Bootstrap Sync
- lattice-go gains `-port` (default 4000) and `-peers` flags. **Live two-node proof**: fresh node on :4002 bootstrapped from :4000 via `POST /bootstrap` — 21/21 blocks applied in 1 fetched page, latest hash + state hash identical. (:4001 is occupied by bobcoin's go-lattice.exe; left untouched.)

## [11.60.51] - 2026-11-14
### Lattice State Panels in Dashboard
- Supernode now reverse-proxies lattice read routes (`/nfts`, `/swaps`, `/anchors`, `/market/bids`, `/frontier/`, `/balance/`, `/peers`) using the package `latticeURL` (was hardcoded + shadowed). Covered by `proxy_routes_test.go`.
- Wallet & Lattice tab renders Governance Proposals, NFT Registry, Atomic Swaps, Anchored Manifests (live chain data, HTML-escaped). Live-verified on the 20-block chain via :8001.

## [11.60.50] - 2026-11-14
### Real Lattice Ledger (Fabricated Blocks Removed)
- **`/lattice` no longer fabricates blocks**: it invented 5 synthetic blocks (heights 1024000-i, time-derived hashes) per call. Now proxies lattice-go `GET /blocks?limit=25` and maps hash/account/height/timestamp; empty or unreachable chain returns honest `[]` (dashboard reads 'No blocks confirmed yet'). Covered by `lattice_test.go` (proxy mapping + down-chain emptiness, both failed before the fix).
- **E2E verified**: lattice-go + supernode `-headless -port 8001` below-normal — lattice WS feed connected, `/lattice` `[]` on empty chain, `/status` v11.60.50.

## [11.60.49] - 2026-11-14
### Operator Tools in Command Center
- **Operator Tools** (Overview > Services & Network Health): SPoRA storage-proof runner (`GET /spora/{n}` — verified live, returns chunkHash), FHE oracle runner (`POST /fhe-oracle` — empty payload correctly 400s), pprof index/goroutine/heap links. Every backend endpoint on the supernode now has a dashboard surface.

## [11.60.48] - 2026-11-14
### Configurable HTTP Port + Lattice Feed Backoff
- **`-port` flag** on supernode-go (default 8000): dashboard/tracker HTTP bind is now configurable (`main.HTTPPort`), so bobtorrent coexists with bobcoin's go-supertorrent on :8000. Verified live on :8001 â€” dashboard HTML, `/status` (version 11.60.48), `/stats`, `/blobs`, `/assets`, `/bankroll`, `/transactions`, `/status/i2p`, `/filecoin/status`, `/channels/browse` all HTTP 200 while :8000 stayed untouched.
- **Tray**: honors `BOBTORRENT_PORT` env (default 8000), passes `-port` through, dashboard link follows it.
- **Lattice feed log spam**: reconnect backoff 5sâ†’60s exponential (was a flat 5s retry forever when lattice-go is down).

## [11.60.47] - 2026-11-14
### Headless Dashboard Fix, Tray on Windows, Full Backend Coverage in UI
- **Critical fix â€” headless mode served nothing**: `cmd/supernode-go/main_wrapper.go` `-headless` branch never called `startTrackerServices()`, so the tray-launched daemon had no HTTP dashboard/API on :8000. Headless now starts the identical service set as the TUI path (tracker+HTTP, DHT, messenger, I2P, ingestion, Filecoin bridge, market poller, lattice feed) and blocks on SIGINT/SIGTERM.
- **Version truth**: added `main.Version` (ldflags `-X main.Version=`, already passed by build scripts); `/status` reported hardcoded `11.40.0`, now reports the build version. Covered by `cmd/supernode-go/version_status_test.go`. Footer version is overwritten live from `/status`.
- **Tray controller now works on Windows**: was gated on `$DISPLAY` (never set on Windows, so no tray icon ever appeared), used `lsof` and extensionless `./build/supernode-go`. Now: `.exe` resolution, `build.bat` builds, `taskkill`/CIM port-owner lookup that only stops processes whose image matches our binaries, below-normal priority spawns, live status title, Quit stops services. GUI tray verified running with pystray+Pillow.
- **Dashboard coverage completed** (still one page, 5 value-ordered tabs): Overview gains Services & Network Health (`/status`, `/status/i2p`, `/filecoin/status`, `/channels/browse`) plus Add Magnet / Remove / Upload & Seed (`/add-torrent`, `/remove-torrent`, `/upload`); Wallet tab gains Node Economy (`/bankroll`, `/transactions`, `/mint`, `/burn`). Every control carries a descriptive label + tooltip naming its endpoint.
- **Honesty fix**: Devnet Airdrop button fabricated a confirmed 1.0 SOL credit when the (nonexistent) endpoint failed; it now shows UNAVAILABLE and credits nothing.
- **Build scripts**: `build.bat` rebuilt â€” package-path builds (single-file `go build main.go` cannot compile this package), no unrelated `npm install`, no blocking `pause`, `CGO_ENABLED=0` (local mingw emits unparseable objects); `build.sh` defaults `CGO_ENABLED=0` likewise.
- **Verified**: `cmd /c build.bat` (4 artifacts), `go test ./cmd/supernode-go/ ./internal/... ./pkg/...` all pass, headless + lattice-go launched below-normal and log-watched.

## [11.60.46] - 2026-06-29
### Mobile Messenger Launcher Integration in Sovereign Command Center
- **Direct Mobile Launcher**: Integrated an intuitive mobile light node launcher button (`ðŸ“± Mobile Preview`) directly into the Decentralized Mesh Chat header within the Sovereign Command Center (`web/ui/index.html`).
- **Streamlined Workflow**: Allows operators to open the mobile messenger light client (`/mobile_preview.html`) side-by-side with the heavy node command console.
- **Automated Verification**: Validated via Playwright screenshot capture (`verification/command_center_mesh_mobile_link.png`) and full Go test suite execution.

## [11.60.45] - 2026-06-28
### Mobile Messenger Light Node Web View & Bridge Integration
- **Mobile Messenger Light Client Preview**: Implemented `web/ui/mobile_preview.html`, providing a responsive mobile messenger interface designed for Light Node clients.
- **WebSocket Control Plane Wiring**: Connected directly to `cmd/supernode-go/mega_messenger_bridge.go` (`/mega-bridge`), supporting real-time chat publishing, topic switching across `#global`, `#announcements`, `#market`, and `#support`, and automatic history hydration.
- **Automated Frontend Verification**: Verified with Playwright headless test capturing live message transmission and rendering in `verification/mobile_messenger_preview.png`.

## [11.60.44] - 2026-06-28
### Sovereign Command Center UI Consolidation & Unified System Tray Controller
- **Unified Command Console**: Redesigned and consolidated the BobTorrent Web UI (`web/ui/index.html`, `style.css`, `app.js`) from fragmented and isolated subpages into an intuitive, high-value Sovereign Command Center.
- **Categorized High-Value Modules**:
  - *Overview & Live Swarm Transfers*: Real-time peer/DHT telemetry, live swarm active transfers, and local blob chunk persistence inspect.
  - *Publishing Studio & Asset Registry*: Asset ingestion pipeline with AES-256-GCM, BEP 44 DHT mutable item channel distribution, follower tracking, and durable local SQLite manifest registry.
  - *Decentralized Mesh Chat*: Direct integration with `libp2p` GossipSub (`/ws-messenger`), topic discovery, real-time message stream, and persistent history hydration.
  - *Cryptographic Identity & Sybil Attestation*: Ed25519 identity generation, key backup, and anti-sybil validation (GitHub Gist, ORCID, and signed URL with SSRF protection).
  - *Bobcoin Wallet & Block Lattice Ledger*: Solana Devnet wallet management, devnet airdrops, and live Block Lattice consensus stream (`ws://127.0.0.1:4000/ws`).
- **Comprehensive UI Guidance & Polish**: Added informative tooltips (`data-tooltip`), descriptive input labels, live status badges, and streamlined workflow navigation.
- **Unified System Tray Controller**: Created `scripts/system_tray_controller.py` consolidating background daemon lifecycle management (ports 8000 and 4000) supporting both Pillow/pystray system tray graphical icon and headless daemon CLI fallback with graceful shutdown.
- **Frontend Verification**: Completed Playwright end-to-end automated verification across all views and verified Go backend test suites.

## [11.60.52] - 2026-06-27
### Phase 8 Completion: Mega-Messenger Architecture
- **Element-Web Integration Planning**: Finalized Phase 8 by defining the architecture for integrating the `element-web` Matrix frontend with the native Go `libp2p` GossipSub backend (documented in `element-web/ARCHITECTURE_ELEMENT.md`).

## [11.60.51] - 2026-06-27
### Lattice Dialect Compatibility
- **Bobcoin Submodule**: Updated the `bobcoin` frontend to correctly request the `/governance/proposals` endpoint, resolving a dialect mismatch.
- **Consensus Bridge**: Verified that the Go supernode correctly handles both wrapped (`{"block": {...}}`) and raw block JSON payloads to maintain legacy frontend compatibility.

## [11.60.50] - 2026-06-25
### Phase 9: UI Enhancements & Bobcoin Compat
- **Bobcoin Compatibility**: Restored missing endpoints (`/blocks`, `/bootstrap`) to natively support the legacy bobcoin frontend lattice dialect.
- **Mega-Messenger Dispatcher**: Upgraded the `mega_messenger_bridge.go` WebSocket handler to support detailed client tracking, dynamic topic leaving, and robust history synchronization via the SQLite message store.

## [11.60.25] - 2026-05-19
### Mega-Messenger Bridge Scaffold
- **Control Plane API**: Created `cmd/supernode-go/mega_messenger_bridge.go` exposing the `/mega-bridge` WebSocket endpoint.
- **Light Node Architecture**: This bridge serves as the foundation for the Phase 8 decoupled UI layer, allowing a standalone React/Flutter frontend (derived from the new `element-web` submodule) to proxy blinded Protobuf envelopes to the Heavy Go Node without participating directly in P2P mesh logic.

## [11.60.24] - 2026-05-19
### Mega-Messenger Architecture & Element Submodule
- **Submodule Integration**: Added `element-web` as a git submodule to serve as the reference frontend and porting target for the decentralized chat/storefront platform.
- **Architectural Planning**: Updated `IDEAS.md`, `ROADMAP.md`, and `TODO.md` with the "Control Plane" architectural pattern, enforcing a strict separation between Light Mobile Nodes (UI/State) and Heavy Go Nodes (libp2p routing, Tor, Bobcoin).

## [11.60.23] - 2026-05-19
### Documentation & Phase 8 Planning
- **Roadmap Assessment**: Re-analyzed remaining TODO items, halting code execution due to the ambiguous nature of Phase 8 tasks (e.g. Game Engine Ingestion, Global Launch).
- **Ideation**: Expanded `IDEAS.md` with concrete architectural directions for tackling the remaining Phase 8 backlog.
- **Handoff**: Updated `HANDOFF.md` to clearly signal the completion of Phase 7 (Pub/Sub Identity) and outline the required design work for Phase 8.

## [11.60.22] - 2026-05-19
### UI Pub/Sub Tracking Integration
- **Subscriptions Tracking**: Added backend logic `getSubscriptionCount()` to surface the active count of tracking/publishing channels from the `subscriptionStore`.
- **Dashboard Telemetry**: Updated `handleStats` to embed `subscriptions` and correct `network.status` into its standard `/stats` endpoint.
- **WebUI Unification**: Patched `web/ui/app.js` to correctly point its main `updateStatus` polling routine toward `/stats` instead of the incomplete stub `/status`, successfully hydrating the main Dashboard UI with live active Pub/Sub telemetry.

## [11.60.21] - 2026-04-21
### Added
- Added /api/lattice endpoint to BobTorrent API
- Completed Web UI integration for Bobcoin wallet display and lattice visualization.

## [11.60.20] - 2026-04-20
### Added
- Added missing Web UI tooltips and labels to complete the embedded interface polish.

## [11.60.19] - 2026-04-20
### Added
- Added ORCID and URL verification interface logic to the Web UI.

## [11.60.18] - 2026-04-20
### Added
- Complete Web UI /api/status endpoint integration with real-time stats

## [11.60.17] - 2026-04-20
### Added
- Added ORCID and custom signed-message URL verifiers to the identity API endpoint to expand trust layer options.

## [11.60.16] - 2026-04-19
### Added
- Added Bobcoin wallet integration and lattice visualization mock to Web UI

## [11.60.15] - 2026-04-19
### Added
- Added Web UI integration for Bobcoin Solana wallet display, airdrop requests, and lattice visualization mock

## [11.60.14] - 2026-04-19
### Added
- Added BEP 44 Mutable DHT targets logic for subscribing to channels
- Wired up the backend API publisher loop

## [11.60.13] - 2026-04-19
### Added
- Complete Web UI UX polish, tooltips, and labels.
- Finalized BEP 44 implementation logic stubbing for Tracker interactions.

## [11.60.12] - 2026-04-19
### Added
- Upgraded asset discovery to use SQLite-backed durable market manifest registry

## [11.60.11] - 2026-04-18
### Added
- Added BEP 44 Mutable DHT target calculation stub for Subscriptions

## [11.60.10] - 2026-04-17
### Fixed
- Restored  logic completely, properly mapped the  endpoint without stripping critical logic.

## [11.60.9] - 2026-04-17
### Fixed
- Stripped legacy /api/ prefixes from Web UI client configuration for seamless native Go routing compatibility

## [11.60.8] - 2026-04-17
### Added
- Wired /api/publish and /api/subscribe to BEP 44 Mutable DHT Items engine implementations

## [11.60.7] - 2026-04-17
### Added
- Added ORCID Verifier and Custom Signed-Message URL Verifiers to Identity Trust Layer

## [11.60.6] - 2026-04-17
### Added
- Upgraded DHT engine with standalone server instance to support BEP44 mutable item broadcasting
- Unify block hashing rules and cleanup old TODO tasks

## [11.60.5] - 2026-04-17
### Added
- Added missing Web UI tooltips, clear labels, and UI polish

## [11.60.4] - 2026-04-17
### Added
- Added /api/assets endpoint to serve durable manifest registry
- Implemented missing Web UI tooltips, labels, and polish
- ReadaheadBuffer.Seek EOF exact test

# Changelog

## [11.60.31] - 2026-05-24
### Monorepo Unification & Core Cleanup
- **Directory Flattening**: Successfully migrated all Go logic from the `bobtorrent/` submodule into the root workspace structure (`cmd/`, `internal/`, `pkg/`).
- **Legacy Removal**: Removed the `bobtorrent/` directory to eliminate code duplication and import path confusion.
- **Repository Sanitization**: Performed a comprehensive sync and merge of `origin/megatorrent-reference-client-ui-8247358214956960041`, resolving unrelated history conflicts to unify the Java and Go supernode features.
- **Documentation Overhaul**: Updated `VERSION`, `ROADMAP.md`, and `TODO.md` to reflect the newly unified kernel state and Phase 8 priorities.
- **Web UI Enhancements**: Added external attestation verification form and real-time torrent download monitoring to the reference dashboard.

## [11.60.30] - 2026-05-23
### Integrated Decentralized Chat UI
- **Embedded Web UI**: Added a dedicated "Chat" tab to the reference client interface.
- **Messenger WebSocket Bridge**: Fully wired the frontend to the `/ws-messenger` GossipSub bridge.
- **Topic Management**: Implemented dynamic topic joining and switching within the UI.
- **Persistent History**: Enabled chat history hydration upon joining topics via the `FETCH_HISTORY` backend support.

## [11.60.29] - 2026-05-22
### Readahead Performance Optimization
- **mmap Backing**: Transitioned `ReadaheadBuffer` to use memory-mapped files (`mmap`) for backing reconstructed streams.
- **Efficient Seek/Read**: Improved random access performance and reduced memory pressure by writing decrypted chunks directly to `mmap`'ed disk-backed memory.
- **Synchronization**: Implemented `sync.Cond` based coordination between chunk fetching and reading.

## [11.60.28] - 2026-05-22
### TUI Gossip Visibility and Messenger Persistence
- **TUI Integration**: Added live GOSSIP MESH feed into the terminal dashboard.
- **Messenger Store**: Implemented SQLite-backed persistence for GossipSub messages.

## [11.60.27] - 2026-05-21
### Messenger Persistence and History
- **Durable Messenger**: Wired the `libp2p` messenger to a persistent SQLite store.

## [11.60.26] - 2026-05-21
### Dynamic Topics and I2P Datagrams
- **I2P Signaling**: Activated native I2P/SAM datagram transport for anonymous signaling.
- **Dynamic topics**: Support for dynamic GossipSub topic management.

## [11.60.25] - 2026-05-19
### Mega-Messenger Scaffolding
- **libp2p Messenger**: Scaffolded a `libp2p` host and GossipSub engine.
- **Control Plane API**: Created `mega_messenger_bridge.go` for Decoupled UI support.


## [11.60.32] - 2026-06-15
### Game Engine Asset Ingestion
- **Ingestion Service**: Created `internal/ingest` package to handle high-performance, asynchronous asset ingestion.
- **Asynchronous API**: Updated `POST /api/ingest` to support an `async` flag, returning a `jobId` for tracking.
- **Progress Tracking**: Added `GET /ingest/status?id=...` to monitor encryption and sharding progress in real-time.
- **Unified Logic**: Consolidated chunking, encryption, and registration into a single streaming pipeline.

## [11.60.33] - 2026-06-15
### Mega-Messenger Protocol and Integration
- **Envelope Protocol**: Defined `pkg/messenger/envelope.go` for metadata-blinded decentralized messaging.
- **Messenger Upgrade**: Integrated Envelope marshaling and signing into the `internal/transport` layer.
- **Bridge Support**: Added `PUBLISH_ENVELOPE` to the Mega-Bridge WebSocket for Light Node UI compatibility.
- **Integration Roadmap**: Documented the Element-Web to libp2p porting strategy.

## [11.60.34] - 2026-06-15
### Robust Messaging and Optimized Ingestion
- **Envelope Verification**: Implemented Ed25519 signature verification in `pkg/messenger/envelope.go` for end-to-end mesh integrity.
- **Mmap Ingestion**: Added `ProcessLargeFile` utilizing `github.com/edsrzf/mmap-go` for high-performance ingestion of multi-gigabyte game assets.
- **Submodule Cleanup**: Purged legacy metadata from `bobcoin/research` and resolved `qbittorrent` remote synchronization issues.

## [11.60.35] - 2026-06-17
### Performance Profiling and Anonymous Signaling
- **Observability**: Integrated `net/http/pprof` into `supernode-go` and `lattice-go` for production-grade profiling.
- **I2P Discovery**: Enhanced anonymous signaling with I2P Datagrams and wired `I2P_PING` to the Mega-Bridge.
- **Persistence Refinement**: Centralized ingestion registry storage within the torrent data directory using pure-Go SQLite.

