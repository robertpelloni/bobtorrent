## [11.60.75] - 2026-11-17
### Tray Stop Actually Stops (Windows)
- Tray `stop_services` claimed success while externally-started lattice-go kept running: Windows console servers refuse graceful `taskkill /T`. Now escalates to `/F` after the graceful attempt and verifies the port is free before printing 'Stopped' (prints FAILED otherwise). Live-verified both ways; unrelated processes on managed ports still left untouched.

## [11.60.74] - 2026-11-17
### Graceful Lattice Shutdown
- lattice-go now serves HTTP in a goroutine and waits on SIGINT/SIGTERM so its deferred `StopBackgroundSync` + store `Close` actually run (previously ListenAndServe blocked main and every stop was an unclean kill for WAL recovery). Verified serving 27 blocks after change; Windows console processes still require force termination (platform signal limits).

## [11.60.73] - 2026-11-17
### UDP Tracker (BEP 15) Live-Verified
- Connect + announce exercised over UDP :6881: connection-id handshake, interval 1800, seeder counting, and compact peer exchange (second peer receives the first as 6-byte compact). Both tracker transports (HTTP + UDP) now live-proven.

## [11.60.72] - 2026-11-17
### Tracker Announce Live-Verified
- HTTP `/announce` exercised live: first peer gets empty list, second peer receives the first in compact 6-byte form, re-announce returns the other peer; parameter-less announce correctly 400. Classic tracker semantics intact in the Go port.

## [11.60.71] - 2026-11-17
### Submit-Proof Design Finding (owner decision)
- `/submit-proof` negative paths verified live (bad payload 400, score<1000 400). FINDING: its `zkVerified` gate is only `score >= 1000` - no cryptographic proof is checked, yet it mints `score/100` economy balance. Anyone who can reach the endpoint can mint with a fabricated JSON score. Bobcoin proof-of-play compatibility endpoint; hardening (signed proof, FHE-oracle linkage, or removal) is an owner call - not changed unilaterally.

## [11.60.70] - 2026-11-17
### Attestation Verifiers Live-Checked
- GitHub + ORCID real verifiers exercised against live APIs with deterministic negatives (missing Gist -> 'Gist not found'; real ORCID iD without attestation string -> correctly unverified). Positive verification requires owner-published content containing `BOBTORRENT_IDENTITY:<account>` (a Gist or ORCID biography) - owner action, not automatable here.

## [11.60.69] - 2026-11-17
### Real Attestation Verifiers Wired
- ORCID and Custom-URL attestation kinds were registered to the MockVerifier, so those dashboard options silently fake-verified (any URL containing 'verify-me' passed). They now use the real `ORCIDVerifier` and SSRF-protected `URLVerifier`; only the explicit Mock kind uses mock. Regression test `TestVerifierRoutingUsesRealVerifiers`; live loopback probe now returns the real SSRF refusal.

## [11.60.68] - 2026-11-17
### Torrent Actions Live-Verified
- Multipart `/upload` (file -> infoHash + magnet, seeded), `/add-torrent` with a valid magnet (200; garbage magnet 400 with parse reason), `/remove-torrent`, and no-file upload 400 - the full dashboard torrent action set exercised live.

## [11.60.67] - 2026-11-17
### Matchmaking Signaling Live-Verified
- `/matchmaking` exercised live: two clients matched (initiator/responder), SIGNAL relayed intact, `/status` signaling telemetry exact (2 connections, 1 match, 1 relay, 2 disconnects). No code change needed.

## [11.60.66] - 2026-11-17
### Signaling Matchmaker Instance Capture
- `handleSignalingSocket` read the mutable `signalingMatchmaker` global at operation time; a late-closing connection's deferred disconnect could mutate a different (swapped) instance - the CI failure in `TestHandleServiceStatusIncludesSignalingSnapshot` (activeConnections 2->1, disconnects 1). The handler now captures one instance per connection. Local suite x3 green.

## [11.60.65] - 2026-11-17
### Live Chat Delivery Fix (same-node)
- Dashboard chat messages published on a node never reached other clients on the SAME node live (libp2p readLoop skips own-host messages; sender saw only the UI's optimistic echo, peers saw nothing until rejoin). `Messenger.PublishFromHandler` now fans out locally to registered topic handlers (excluding the originator); wired into `/ws-messenger` and `/mega-bridge`. Regression test `TestMessengerSocketDeliversToPeerClient` (the old test tolerated the missing echo); live two-client verified.

## [11.60.64] - 2026-11-17
### Mega-Bridge Live-Verified + Log Hygiene
- `/mega-bridge` WebSocket exercised live: join/publish ack, history returns persisted message, unknown action errors cleanly. Normal client closes (1000) no longer logged as errors.

## [11.60.63] - 2026-11-17
### Shard Round-Trip Verified
- `/upload-shard` -> `/shards/<hash>` byte-identical round-trip live-verified; hash-mismatch and bad-base64 correctly 400, missing shard 404, path-traversal probes correctly 400 (Go ServeMux/ServeFile). No code change needed.

## [11.60.62] - 2026-11-17
### Publishing Pipeline Live-Verified
- Ingest -> publish -> assets -> manifest retrieval exercised end-to-end over HTTP: file ingested to torrent (400 on path traversal, 404 on missing), manifest signed+published, `GET /assets` lists it, `GET /manifests/<id>` serves it (200). Port map: bobcoin now also owns :8001 (go-casino) - bobtorrent verification uses :8002.

## [11.60.61] - 2026-11-17
### Subscribe Key Validation
- `/subscribe` accepted any non-empty string as a publisher key. Now requires a 64-char hex Ed25519 public key (400 otherwise). Test: `subscribe_validation_test.go`; live-verified (zz -> 400, valid -> 200). Found in exhaustive dashboard sweep: 21/21 GET surfaces 200, negative controls (FHE empty, bad magnet, negative mint, bogus block, empty peer) all correctly reject.

## [11.60.60] - 2026-11-17
### Disaster Recovery Verified
- Encrypted backup-bundle -> restore -> cold-boot path proven live on the real 27-block chain (wrong passphrase 400; restored node 27/27, hash-identical) and locked in as `TestSecureBundleRestorePreservesSnapshotChain`.

## [11.60.59] - 2026-11-16
### JS Gate Restored: Lint 0 + Real Bug Fixed
- `standard --fix` + manual cleanup: 669 -> 0 findings on owned JS (`index.js server.js lib test`; lint script scoped to match — unscoped standard swept submodules).
- **Bug fixed**: vendored `server.js` `listen()` referenced out-of-scope `onListening` with WebTransport enabled (ReferenceError). Local completion callback added; regression test `test/server-webtransport.test.js`.
- Restored vendored tracker modules deleted from this fork (`lib/common.js`, `lib/common-node.js`, `lib/server/{swarm,parse-http,parse-udp}.js`) from upstream — root server.js is importable again.
- `npm test` (lint + tests) green: **114/114**.

## [11.60.58] - 2026-11-14
### Opt-In Signature Enforcement (lattice-go -require-signatures)
- `Lattice.RequireSignatures` + `-require-signatures` flag (default OFF — permissive behavior unchanged). Live unsigned blocks rejected ('signature required'), signed accepted; replay of existing history never gated so cold boot cannot brick. Covered by `require_signatures_test.go` + live test on a throwaway chain. Enabling it makes the mandatory-signature policy a deployment choice instead of a code change.

## [11.60.57] - 2026-11-14
### Ledger Truncation Fix
- Dashboard `/lattice` proxy capped at 25 blocks (chain had 27 — newest blocks invisible). Limit raised to 200; verified 27/27 via dashboard at v11.60.57. Found during the v11.60.56 handoff relaunch.

## [11.60.56] - 2026-11-14
### Raw Block Submission in Dashboard
- Wallet tab gains Submit Lattice Block (Advanced): full block JSON -> `POST /process` via supernode proxy, exact consensus rejection reasons surfaced. Live-verified through the dashboard origin (block accepted, chain 26->27). Branch audit: all local branches already merged into `master`.

## [11.60.55] - 2026-11-14
### Lattice Account in Wallet Tab
- Block Lattice card shows the wallet's on-chain account: balance (demurrage-decayed, tooltip explains the rate), frontier height, staked amount — via proxied `/balance` + `/frontier`. Live-verified: genesis account height 8, balance 971348.

## [11.60.54] - 2026-11-14
### Lattice Persistence in Services Panel
- Supernode proxies `/persistence/*`; Services & Network Health gains a Lattice Persistence row (healthy, blockCount, sequence, snapshotCount, quick_check from `/persistence/verify`). Live-verified via :8001 (healthy, 22 blocks, seq 22, snapshots 0). Note: supernode lattice-feed/proxy calls can 502 if fired in the same second lattice starts — warmup race, self-heals.

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

