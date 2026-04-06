# Bobtorrent Go Port Roadmap

## Overview
Bobtorrent is evolving from a mixed Node.js / Java / prototype stack into a unified Go-first distributed systems platform. The current roadmap centers on making the Go port production-credible across four domains:
- consensus
- storage
- transport
- operator experience

## Current Release Train
- **Current Version**: `11.50.0`
- **Primary Runtime Targets**:
  - `lattice-go` — block lattice consensus node
  - `supernode-go` — torrent seeding, market polling, TUI operations
  - `dht-proxy` — privacy-preserving peer discovery utility
  - `storage.wasm` — browser-side Go storage kernel

## ✅ Completed Through v11.50.0

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
- Added duplicate-aware block processing results, ordered confirmed-block catch-up, `GET /blocks`, `GET/POST /bootstrap`, and peer-registration-triggered late-join sync so new Go lattice nodes can bootstrap practical history from existing peers.
- Added peer-health telemetry plus bounded retry handling around bootstrap, block-page sync, peer-list sync, and fan-out delivery so multi-node operations expose lag/failure state instead of only raw peer counts.
- Added a stronger sync policy layer on top: peers can now enter cooldown after failures, broadcasts skip peers in cooldown, and missing-cursor cases on non-empty local chains are treated as divergence suspicion instead of silent full replay.
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

### 4. Go Supernode UX + Service Compatibility
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
  - `/status`
  - `/stats`
  - `/bankroll`
  - `/transactions`
  - `/mint`
  - `/burn`
  - `/fhe-oracle`
  - `/submit-proof`
  - `/add-torrent`
  - `/remove-torrent`
  - `/upload`
  - `/spora/:challenge`
- Added Go-native websocket matchmaking/signaling compatibility on `/` and `/matchmaking` for Bobcoin WebRTC flows.
- Hardened the Go signaling/session layer with liveness deadlines, ping/pong keepalive behavior, stale waiting-peer eviction, and operator-visible signaling telemetry.
- Added real multipart upload compatibility in the Go supernode: uploaded files are persisted locally, turned into real torrent metainfo/magnets, and registered with the active Go torrent client.
- Tightened the Go SPoRA compatibility surface so attestation now requires a valid challenge and a tracked Core Arcade anchor instead of returning an unconditional placeholder proof.
- Added a durable SQLite-backed economy transaction log for those compatibility endpoints.
- Ported the lightweight proof-submission orchestration path into Go using deterministic mock verification plus reward mint recording.
- Ported the homomorphic-oracle HTTP surface into Go while isolating the specialized SEAL arithmetic behind a dedicated helper bridge.
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
- Continue deepening publisher identity semantics beyond current profile overlays into even richer linked attestation models and external identity evidence.
- Extend the new week-over-week source reliability layer with even stronger comparative diagnostics, exports, and longer retained history.
- Add stronger batch/archive workspace operations beyond the initial export/copy actions.

### B. Persistence Hardening + Repair Tooling
- Added integrity checks and conservative snapshot-layer repair tooling for persistence corruption scenarios.
- Added portable JSON export, live SQLite backup, bundle import, and backup restore controls for operator-managed recovery workflows.
- Added signed/encrypted operator backup bundles layered on top of the safe SQLite backup flow, plus secure bundle restore into fresh verified databases.
- Expanded persistence-aware tests beyond manifest-anchor replay into a richer snapshot-tail mixed transition replay covering send/open/receive, governance, NFT, staking, and swap flows.
- Added operator-tunable snapshot cadence and retention controls via explicit persistence config and startup environment variables.
- Consider whether snapshot controls should remain startup-config-only or eventually gain runtime/API mutability.

### C. Continue Service-Side Go Migration + Multi-Node Networking
- Continue porting remaining practical Node-side service responsibilities into Go where feasible.
- Continue evolving the new lattice peer sync flow beyond its first practical bootstrap/catch-up version.
- The first health/retry/cooldown layer now exists; next focus is stronger peer gossip, richer backoff tuning, and heavier divergence reconciliation paths.
- Extend catch-up semantics beyond ordered block replay toward richer lag diagnostics and more explicit reconciliation tooling if multi-node deployments become heavier.

### D. Real Filecoin Ingestion
- Added a Lotus JSON-RPC integration path for Filecoin deal publication and verification in `internal/bridges/filecoin.go`.
- Persisted and exposed Filecoin deal IDs/state through bridge records and supernode endpoints.
- Future work can deepen the current integration beyond CID/deal orchestration into richer CAR/import pipelines and stronger deal lifecycle metadata.

## 🌍 Longer-Term Direction
- Full browser-integrated zero-trust storage uploads
- Production-grade Go tracker / DHT / supernode bundle
- Durable decentralized storage market
- Cross-chain storage archival and payout routing
- Native game engine ingestion for Bobcoin asset streaming
