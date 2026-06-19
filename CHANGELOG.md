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

## [11.60.42] - 2026-06-19
### Primary Torrent Download Module
- **Verifier Routine**: Implemented `Verifier` in `pkg/torrent/downloader` to read and verify SHA256 hashes of downloaded file pieces against expected lists.

## [11.60.41] - 2026-06-19
### I2P Hybrid Networking
- **StreamSession Integration**: Refactored `I2PDatagramTransport` to include a reliable `sam3.StreamSession` alongside the existing unreliable `DatagramSession`.
- **Hybrid Connectivity**: Added `DialI2P` and `AcceptLoop` methods to facilitate hybrid clear-net/dark-net peer connections for large data transfers via I2P TCP-like streams.

## [11.60.40] - 2026-06-19
### Robust Message Dispatching
- **Topic History Pagination**: Updated the SQLite messenger store and WebSocket API (`FETCH_HISTORY`) to support offset-based pagination for chat history.
- **Offline Message Queueing**: Implemented an offline message queue for outgoing chat payloads. If the supernode is temporarily disconnected from the GossipSub mesh, messages are buffered to disk and automatically flushed once connectivity is restored.

## [11.60.39] - 2026-06-19
### Mobile Client Protocol Buffers
- **Protobuf Integration**: Upgraded the React Native `MobileMessenger` client to use the `envelope.proto` specification, replacing raw JSON payloads with serialized Protocol Buffers (base64 encoded) for all GossipSub chat messages.

## [11.60.38] - 2026-06-19
### Mobile Client Implementation
- **React Native UI**: Integrated the React Native client with the `/ws-messenger` bridge, creating a functional, real-time chat interface connected to the Heavy Node's GossipSub mesh.

## [11.60.37] - 2026-06-19
### Mobile Client Scaffolding
- **React Native Initialization**: Scaffolded the `MobileMessenger` React Native client to serve as the Light Node frontend bridging to the `supernode-go` WebSocket control plane.

## [11.60.36] - 2026-06-19
### Mega-Messenger Protocol Buffers
- **Envelope Specification**: Defined the Data Envelope Specification (Protocol Buffers) format in `internal/protobuf/envelope.proto` for Light UI to Heavy Node communication.

## [11.60.35] - 2026-06-19
### Mainnet Launch Preparation
- **Checklist Creation**: Created `MAINNET_LAUNCH.md` to track the DevOps, DNS, and Infrastructure readiness required for the Global Decentralized Storage Network launch.

## [11.60.34] - 2026-06-18
### Messenger Polish
- **Rate Limiting**: Implemented a token bucket rate limiter (5 req/sec, burst of 10) per GossipSub topic within `internal/transport/messenger.go` to prevent individual nodes from spamming the P2P chat network.
- **Typing Indicators**: The messenger now natively identifies transient `m.typing` events. These payloads are seamlessly broadcasted across the gossip mesh for real-time UI feedback but are intelligently omitted from the durable SQLite message store to prevent history bloat.

## [11.60.33] - 2026-06-18
### Seeding Incentives
- **Economy Bridge**: Bridged the `accept_bid` lattice logic with the node's local economy database. When the supernode accepts a storage market bid, the incoming Bobcoin bounty is now correctly recorded as an `ACCEPT_BID` transaction, making seeding rewards visible in the UI transaction history.

## [11.60.32] - 2026-06-18
### Verification Layer Expansion
- **ORCID Verifier**: Implemented an ORCID verifier that allows users to prove account ownership via their ORCID profile.
- **URL Verifier**: Implemented a generic URL verifier that allows users to prove account ownership by hosting their Bobcoin public key on an arbitrary website.
- **Removed Mocks**: Removed the `MockVerifier` implementations for ORCID and URL endpoints, replacing them with the new production-ready implementations in the Go Supernode registry.

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

