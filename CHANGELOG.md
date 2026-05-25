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

