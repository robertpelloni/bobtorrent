# Bobtorrent Go Port Roadmap

## Overview
Bobtorrent is evolving from a mixed Node.js / Java / prototype stack into a unified Go-first distributed systems platform. The current roadmap centers on making the Go port production-credible across four domains:
- consensus
- storage
- transport
- operator experience

## Current Release Train
- **Current Version**: `11.28.0`
- **Primary Runtime Targets**:
  - `lattice-go` — block lattice consensus node
  - `supernode-go` — torrent seeding, market polling, TUI operations
  - `dht-proxy` — privacy-preserving peer discovery utility
  - `storage.wasm` — browser-side Go storage kernel

## ✅ Completed Through v11.28.0

### 1. Go Consensus Node
- Ported the Bobcoin asynchronous block lattice to Go.
- Implemented chain validation, frontier tracking, rolling state hashing, and in-memory state indexes.
- Added block categories for:
  - `open`
  - `send` / `receive`
  - `market_bid` / `accept_bid`
  - `proposal` / `vote`
  - `mint_nft` / `transfer_nft`
  - `stake` / `unstake`
  - `initiate_swap` / `claim_swap` / `refund_swap`
  - `publish_manifest` / `data_anchor`
- Added peer registration and HTTP-based P2P block broadcast between lattice nodes.
- Added wallet-attributed manifest anchor indexing and anchor query APIs.

### 2. Frontend Compatibility Layer
- The Go lattice now accepts both raw block payloads and wrapped payloads in the shape `{ "block": ... }`.
- Added compatibility endpoints expected by the existing bobcoin frontend:
  - `/pending/:account`
  - `/proposals`
  - `/chain/:account` returning both `chain` and `blocks`
  - WebSocket upgrade on `/` in addition to `/ws`
- Added a temporary compatibility shim for legacy frontend blocks that omit `height` and `staked_balance`.

### 3. Real-Time Eventing
- Added a WebSocket broadcast hub for live lattice block feed updates.
- Emitted compatibility-friendly websocket events using both `type` and `event` fields.
- Connected the Go supernode to the lattice feed for real-time TUI updates.

### 4. Go Supernode UX
- Upgraded the Bubble Tea terminal UI with:
  - live market bid table
  - live block feed
  - network statistics
  - balance/status bar
- Connected the supernode to:
  - the tracker
  - the DHT node
  - the lattice market poller
  - the lattice websocket feed
- Added simulated Filecoin archival during autonomous bid acceptance.
- Added frontend-facing HTTP compatibility endpoints:
  - `/stats`
  - `/add-torrent`
  - `/remove-torrent`
- Added static serving for:
  - `/storage.wasm`
  - `/wasm_exec.js`

### 5. Storage Kernel + WASM
- Implemented Go-native encrypted storage with ChaCha20-Poly1305.
- Implemented Reed-Solomon erasure coding and reconstruction.
- Exported the storage kernel to WebAssembly.
- Added a reusable browser-side loader at `web/storage-wasm-loader.js`.
- Added build pipeline packaging for `storage.wasm` and `wasm_exec.js`.
- Integrated the Go WASM preprocessing path into `bobcoin/frontend` via a Supernode workbench.
- Defaulted the Bobcoin frontend WASM client to the Go supernode origin so the browser can fetch artifacts directly from port `8000`.
- Added a supernode-hosted publication registry for uploaded shards and published manifests.
- Upgraded the Bobcoin workbench from preprocessing-only to real shard upload + manifest publication.
- Added Bobcoin browser-side restoration flow from published manifest back to reconstructed/decrypted file.

### 6. Durable Lattice Persistence
- Added an optional SQLite-backed confirmed block log for the Go lattice.
- Added replay-driven cold-boot recovery so restart rebuilds chains, pending transfers, governance state, swaps, NFTs, and anchors from persisted blocks.
- Added materialized SQLite snapshots so cold boot can restore a recent checkpoint and replay only the newer block tail.
- Updated `cmd/lattice-go` to boot the lattice in persistent mode by default using `data/lattice/lattice.db` (override with `BOBTORRENT_LATTICE_DB`).
- Added status reporting for persistence enablement, DB path, persisted block totals, snapshot count, and snapshot sequence.
- Added consensus tests proving restart/replay restores anchored manifest state and that snapshot restore + tail replay rebuild the latest frontier correctly.

### 7. Build + Toolchain Hardening
- Fixed third-party API drift in `anacrolix/dht` and `reedsolomon` integrations.
- Added `-buildvcs=false` to local build flows to avoid repo/submodule VCS stamping failures.
- Verified:
  - `go build -buildvcs=false ./...`
  - native binary builds
  - `GOOS=js GOARCH=wasm go build -buildvcs=false -o build/storage.wasm cmd/wasm/main.go`

## 🚧 Active Near-Term Focus

### A. Richer Attestation Semantics + Advanced Source Reliability Analysis
- Deepen publisher identity semantics beyond current profile overlays into richer linked attestation models.
- Extend the new week-over-week source reliability layer with even stronger comparative diagnostics, exports, and longer retained history.
- Add stronger batch/archive workspace operations beyond the initial export/copy actions.

### B. Persistence Hardening + Repair Tooling
- Add integrity checks / repair tooling for persistence corruption scenarios.
- Expand persistence-aware tests beyond manifest-anchor replay into broader consensus transitions.
- Consider operator controls for snapshot cadence and retention once the default behavior has proven stable.

### C. Multi-Node Consensus Networking
- Upgrade the current HTTP fan-out into more robust peer synchronization.
- Add peer gossip / bootstrap / duplicate suppression improvements.
- Introduce state sync and catch-up for late-joining nodes.

### D. Real Filecoin Ingestion
- Replace the simulated Filecoin bridge with Lotus RPC or equivalent.
- Persist returned deal IDs alongside Bobtorrent manifest metadata.

## 🌍 Longer-Term Direction
- Full browser-integrated zero-trust storage uploads
- Production-grade Go tracker / DHT / supernode bundle
- Durable decentralized storage market
- Cross-chain storage archival and payout routing
- Native game engine ingestion for Bobcoin asset streaming
