## [11.15.0] - 2026-04-04
### Go Port: Bobcoin Trust-Aware Archive Intelligence Sync
- **Bobcoin Intelligence Surface**: Updated the `bobcoin` submodule to `v8.16.0`, preserving upstream Go-parity hardening while adding owner trust scores, trust tiers, sorting modes, and a sovereign publisher leaderboard to the Vault archive surface.
- **Trust Surfacing**: The Bobcoin archive UI now exposes heuristic trust overlays and clearer provenance cues, making anchored content easier to evaluate at a glance.
- **Validation**: The Bobcoin frontend production build remained green after the merged trust/reputation overlay and root workspace sync.

## [11.39.0] - 2026-04-05
### Go Port: Comparative Source Diagnostics Sync
- **Bobcoin Diagnostics Export**: Updated the `bobcoin` submodule to `v8.67.0`, where Vault can now export comparative source diagnostics derived from retained recovery reports as portable JSON.
- **Operator Portability**: The reliability/trend system is now not only visible in-browser, but also exportable for offline review, incident handoff, and external analysis.
- **Validation**: Re-validated `cd bobcoin/frontend && npm run build` after integrating the comparative diagnostics export workflow.

## [11.38.0] - 2026-04-05
### Go Port: Signaling Session Hardening
- **Matchmaker Liveness Controls**: Hardened the Go websocket matchmaker with read/write deadlines, periodic ping frames, pong-driven activity refresh, and bounded websocket message size.
- **Stale Queue Eviction**: Added stale waiting-player eviction logic so abandoned matchmaking entries do not sit in the single waiting queue indefinitely.
- **Operational Telemetry**: `supernode-go` status/signaling surfaces now expose signaling metrics including active connections, active pairs, waiting state, total matches, relayed signals, disconnects, and stale-wait evictions.
- **Regression Coverage**: Added tests for stale waiting-peer eviction and signaling snapshot exposure alongside the existing websocket pairing/relay/disconnect tests.
- **Validation**: Re-validated `go test ./cmd/supernode-go ./internal/... -buildvcs=false` and `go build -buildvcs=false ./...` after the signaling hardening pass.

## [11.37.0] - 2026-04-05
### Go Port: WebRTC Signaling Matchmaker
- **Go Matchmaking WebSocket**: Added a native websocket matchmaking/signaling handler to `supernode-go`, compatible with the Bobcoin `FIND_MATCH` / `MATCH_FOUND` / `SIGNAL` / `OPPONENT_DISCONNECTED` contract.
- **Go Signaling Activation**: Updated the `bobcoin` submodule to `v8.66.0`, where matchmaking signaling now defaults to the Go supernode while preserving explicit overrides for specialized or legacy deployments.
- **Regression Coverage**: Extended `cmd/supernode-go/main_test.go` with websocket tests covering player pairing, signaling relay, and opponent-disconnect notification.
- **Validation**: Re-validated `go test ./cmd/supernode-go ./internal/... -buildvcs=false`, `go build -buildvcs=false ./...`, and `cd bobcoin/frontend && npm run build` after the Go signaling migration.

## [11.36.0] - 2026-04-05
### Go Port: Bobcoin Go-First HTTP Routing Sync
- **Bobcoin Runtime Alignment**: Updated the `bobcoin` submodule to `v8.65.0`, where the frontend now defaults migrated compatibility HTTP calls toward the Go supernode while keeping WebRTC signaling on a dedicated configurable legacy path.
- **Go Port Activation**: This makes the already-ported Go service endpoints more likely to be used by default instead of remaining available only behind manual environment retargeting.
- **Mixed Runtime Observability**: Bobcoin System Status now distinguishes the active HTTP compatibility target from the signaling WebSocket path, making the remaining Node-specific surface easier to reason about during migration.
- **Validation**: Re-validated `go test ./cmd/supernode-go ./internal/... -buildvcs=false`, `go build -buildvcs=false ./...`, and `cd bobcoin/frontend && npm run build` across the integrated workspace state.

## [11.35.0] - 2026-04-05
### Go Port: FHE Oracle Compatibility Bridge
- **Go FHE Oracle Endpoint**: Added `POST /fhe-oracle` to `supernode-go`, moving the frontend-facing homomorphic oracle HTTP surface into Go.
- **Specialized Oracle Helper Isolation**: The Go endpoint now orchestrates the server-side FHE compatibility flow and delegates the specialized SEAL arithmetic to an isolated helper bridge, preserving feature behavior without pretending a native Go BFV stack already exists in the workspace.
- **Validation Coverage**: Added targeted handler tests for missing ciphertext, successful ciphertext transformation, and oracle-failure behavior in `cmd/supernode-go/main_test.go`.
- **Frontend Routing Alignment**: Updated Bobcoin so migrated HTTP compatibility calls default toward the Go supernode while keeping WebRTC signaling on its own configurable legacy path.
- **Validation**: Re-validated `go test ./cmd/supernode-go ./internal/... -buildvcs=false`, `go build -buildvcs=false ./...`, and `cd bobcoin/frontend && npm run build` after the new oracle bridge and frontend routing split.

## [11.34.0] - 2026-04-05
### Go Port: Proof Submission Compatibility Endpoints
- **Go Proof Submission Path**: Ported the lightweight game-server `POST /submit-proof` compatibility behavior into `supernode-go`, including proof payload validation, deterministic mock verification, reward mint orchestration, and durable transaction recording.
- **Service Status Compatibility**: Added a simple `GET /status` compatibility endpoint in `supernode-go` so more health/orchestrator checks can point at Go rather than the legacy Node game-server.
- **Further Node-to-Go Migration**: This reduces another practical Node-only compatibility cluster by moving proof-submission orchestration and status reporting into Go.
- **Validation**: Re-validated `go test ./internal/... -buildvcs=false`, `go build -buildvcs=false ./...`, and `cd bobcoin/frontend && npm run build` after the new Go proof/status compatibility endpoints were added.

## [11.33.0] - 2026-04-05
### Go Port: Economic Orchestration Compatibility Endpoints
- **Go Economic Compatibility Layer**: Ported the practical game-server economic surface into `supernode-go` with Go-native `/bankroll`, `/transactions`, `/mint`, and `/burn` endpoints.
- **Durable Transaction History**: Added `internal/economy/database.go`, a small SQLite-backed transaction log so mint/burn compatibility events are preserved durably instead of remaining Node-only ephemeral behavior.
- **Node-to-Go Service Migration**: `supernode-go` can now expose the core bankroll visibility and transaction-history paths that previously lived only in the Node game-server, reducing the remaining service-side Node dependency footprint.
- **Validation**: Re-validated `go test ./internal/... -buildvcs=false`, `go build -buildvcs=false ./...`, and `cd bobcoin/frontend && npm run build` after the new Go economic endpoints were added.

## [11.32.0] - 2026-04-05
### Go Port: Structured Publisher Attestations
- **Consensus Attestation Enrichment**: Extended Go manifest anchors with structured attestation metadata, adding per-proof labels and issuers alongside proof kinds and proof URLs.
- **Bobcoin Identity UX**: Updated the `bobcoin` submodule to `v8.53.0`, where publisher proofs can now be authored as richer attestation records and Vault renders them as structured proof cards instead of only compact badges.
- **Searchable Identity Evidence**: Vault discovery now indexes attestation labels and issuers in addition to proof URLs and proof kinds.
- **Validation**: Re-validated `go test ./internal/consensus -buildvcs=false`, `go build -buildvcs=false ./...`, and the Bobcoin frontend production build after structured attestation integration.

## [11.31.0] - 2026-04-05
### Go Port: Persistence Import & Restore Controls
- **Portable Bundle Import**: Added a controlled import workflow that can materialize a fresh portable lattice database from the JSON persistence export bundle, preserving confirmed block sequences and the newest usable snapshot.
- **Backup Restore Workflow**: Added a restore workflow that can rehydrate a verified portable lattice database from a previously created SQLite backup copy.
- **Operator Endpoints**: Exposed `POST /persistence/import` and `POST /persistence/restore` so operators can create restored databases for the next node boot without hot-swapping the live store.
- **Validation Coverage**: Added consensus regression coverage proving imported bundle databases and restored backup databases reopen correctly as persistent lattices.
- **Validation**: Re-validated `go test ./internal/consensus -buildvcs=false`, `go build -buildvcs=false ./...`, and `cd bobcoin/frontend && npm run build` after the import/restore integration.

## [11.30.0] - 2026-04-05
### Go Port: Persistence Backup & Export Controls
- **Portable Persistence Export**: Added JSON export bundling for the lattice persistence layer, including integrity metadata, durable confirmed blocks, and the newest usable snapshot for operator inspection or manual archival.
- **Consistent Live SQLite Backup**: Added a backup workflow that checkpoints WAL state and uses SQLite `VACUUM INTO` to create a consistent backup copy of the live lattice database without shutting down the node.
- **Operator Endpoints**: Exposed `GET /persistence/export` and `POST /persistence/backup` so operators can export or back up the persistence layer through the running node.
- **Validation Coverage**: Added consensus regression coverage proving export bundles include durable history and that backup copies can be reopened as portable lattice databases.
- **Validation**: Re-validated `go test ./internal/consensus -buildvcs=false`, `go build -buildvcs=false ./...`, and `cd bobcoin/frontend && npm run build` after the backup/export integration.

## [11.29.0] - 2026-04-05
### Go Port: Persistence Integrity Verification & Repair
- **Durable Store Verification**: Added SQLite-backed persistence verification that checks `PRAGMA quick_check`, validates confirmed block JSON/hash integrity, and detects invalid or orphaned lattice snapshots.
- **Conservative Snapshot Repair**: Added a repair workflow that safely rebuilds the snapshot layer from the live in-memory lattice state while leaving the confirmed block log untouched as the correctness-critical source of truth.
- **Operator Endpoints**: Exposed `GET /persistence/verify` and `POST /persistence/repair` so operators can inspect and repair the snapshot layer without stopping the node.
- **Validation Coverage**: Added consensus regression coverage proving corrupt snapshot rows are detected and that repair rebuilds a healthy snapshot layer.
- **Validation**: Re-validated `go test ./internal/consensus -buildvcs=false`, `go build -buildvcs=false ./...`, and `cd bobcoin/frontend && npm run build` after the integrity-tooling integration.

## [11.28.0] - 2026-04-05
### Go Port: Snapshot-Accelerated Lattice Recovery
- **Materialized Snapshot Layer**: Extended `internal/consensus/store.go` with a snapshot table layered on top of the append-only confirmed block log, allowing the lattice to retain recent materialized state checkpoints in SQLite.
- **Tail-Replay Cold Boot**: `NewPersistentLattice` now restores the newest persisted snapshot first and replays only the newer confirmed blocks, reducing restart work on longer histories without changing the confirmed block log as the source of truth.
- **Automatic Snapshot Cadence**: The lattice now materializes snapshots automatically every 25 persisted blocks and retains the newest few snapshots for recovery acceleration.
- **Operational Visibility**: The lattice status endpoint now exposes persisted sequence, snapshot sequence, snapshot count, and snapshot interval so operators can confirm whether acceleration is active.
- **Validation**: Re-validated `go test ./internal/consensus -buildvcs=false`, `go build -buildvcs=false ./...`, and `cd bobcoin/frontend && npm run build` after snapshot integration.

## [11.27.0] - 2026-04-05
### Go Port: Long-Horizon Source Reliability Sync
- **Bobcoin Analytics Upgrade**: Updated the `bobcoin` submodule to `v8.43.0`, preserving the latest upstream replay/parity hardening while adding long-horizon source reliability analytics to Vault.
- **Trend-Aware Operator Diagnostics**: Vault source diagnostics now compare recent and prior-week behavior, score source reliability using successful and failed shard observations, and highlight degrading/improving/healthiest sources instead of only static failure totals.
- **Wider Local Recovery History**: Bobcoin recovery reports now retain a larger local history window and persist successful shard fetches, giving the reliability layer a stronger evidence base.
- **Validation**: Re-validated the Bobcoin frontend production build after the long-horizon analytics integration and synchronized the root workspace to the new submodule pointer.

## [11.26.0] - 2026-04-05
### Go Port: Durable Lattice Persistence & Cold-Boot Recovery
- **SQLite-Backed Consensus Durability**: Added a durable `internal/consensus/store.go` block log using `modernc.org/sqlite`, enabling confirmed lattice blocks to be appended transactionally instead of existing only in process memory.
- **Replay-Based Recovery**: Added `NewPersistentLattice` / `NewPersistentServer` so the lattice node now replays persisted blocks on startup to rebuild chains, pending transfers, proposals, swaps, NFT ownership, and manifest anchors after restart.
- **Atomic Persistence Guard**: `ProcessBlock` now snapshots in-memory state before mutating when persistence is enabled and rolls back cleanly if the SQLite append fails, keeping block commits atomic from the API's perspective.
- **Operational Visibility**: The lattice status endpoint now reports persistence enablement, database path, and persisted block count.
- **Validation**: Re-validated `go test ./internal/consensus -buildvcs=false`, `go build -buildvcs=false ./...`, and `cd bobcoin/frontend && npm run build` after the persistence integration.

## [11.25.0] - 2026-04-04
### Go Port: Typed Publisher Proof Semantics
- **Consensus Proof Typing**: Extended Go manifest anchors with `publisherProofKinds` alongside `publisherProofs`, allowing publisher attestations to carry explicit semantic hints instead of undifferentiated URLs.
- **Bobcoin Identity UX**: Updated the `bobcoin` submodule to `v8.35.0`, where the storage workbench accepts `kind|url` proof entries and Vault renders typed proof badges for publisher attestations.
- **Validation**: Re-validated `go test ./internal/consensus -buildvcs=false`, `go build -buildvcs=false ./...`, and the Bobcoin frontend production build after typed-proof integration.

## [11.24.0] - 2026-04-04
### Go Port: Source Reliability Dashboard Sync
- **Bobcoin Reliability Analytics**: Updated the `bobcoin` submodule to its latest archive-analytics state, adding a first-pass source reliability dashboard derived from persisted recovery reports.
- **Operator Visibility**: Vault can now summarize flaky shard sources across sessions using failure totals, success counts, and category rollups instead of only showing one-off restore diagnostics.
- **Validation**: Re-validated the Bobcoin frontend production build after source reliability dashboard integration and synchronized the root workspace to the new submodule pointer.

## [11.23.0] - 2026-04-04
### Go Port: Portable Archive Workspace Sync
- **Bobcoin Workspace Actions**: Updated the `bobcoin` submodule to its latest archive-operations state, preserving newer upstream replay/parity work while adding preset export/import and batch archive actions in Vault.
- **Operator Workflow Portability**: Vault archive workflows can now be carried between sessions and quickly exported/copied in bulk, making the archive more useful for repeat investigations.
- **Validation**: Re-validated the Bobcoin frontend production build after the portable-workspace integration and synchronized the root workspace to the new submodule pointer.

## [11.22.0] - 2026-04-04
### Go Port: Portable Presets & Batch Archive Actions
- **Bobcoin Workspace Ergonomics**: Updated the `bobcoin` submodule to its latest archive-operations state, adding preset export/import and batch actions for visible archive results.
- **Operator Workflow Portability**: Vault filter logic is now portable and reusable across sessions through preset export/import, and visible archive result sets can be exported or bulk-copied directly.
- **Validation**: Re-validated the Bobcoin frontend production build after preset-sharing and batch-action integration and synchronized the root workspace to the new submodule pointer.

## [11.21.0] - 2026-04-04
### Go Port: Failure Categorization & Source Attribution
- **Bobcoin Recovery Attribution**: Updated the `bobcoin` submodule to its latest diagnostics state, adding shard failure categories, source references, source hosts, and aggregated failure summaries to degraded restore analysis.
- **Operator Diagnostics**: Restore failures are now more actionable because the archive tooling distinguishes omission, corruption, and fetch-path failures instead of collapsing everything into opaque generic errors.
- **Validation**: Re-validated the Bobcoin frontend production build after source-attribution integration and synchronized the root workspace to the new submodule pointer.

## [11.20.0] - 2026-04-04
### Go Port: Exportable Recovery Diagnostics
- **Bobcoin Recovery Reporting**: Updated the `bobcoin` submodule to its latest recovery-reporting state, allowing operators to download structured JSON reports from the degraded-recovery diagnostics panel.
- **Operator Workflow Improvement**: Recovery evidence is no longer trapped in transient UI state; manifest identity, shard-failure reasons, parity status, and restored-file metadata can now be preserved for debugging and postmortem analysis.
- **Validation**: Re-validated the Bobcoin frontend production build after recovery-report export integration and synchronized the root workspace to the new submodule pointer.

## [11.19.0] - 2026-04-04
### Go Port: Publisher Profile Overlays & Linked Proofs
- **Publisher Identity Depth**: Updated the `bobcoin` submodule to the latest archive identity state, adding avatar/profile overlays and linked proof/attestation URLs to signed manifest-anchor metadata.
- **Vault Publisher Cards**: Bobcoin Vault now renders richer publisher profile cards and proof-link discovery, improving attributable archive inspection beyond plain text metadata.
- **Validation**: Re-validated the Go consensus tests and Bobcoin frontend production build after the publisher-profile overlay integration and synchronized the root workspace to the new submodule pointer.

## [11.18.0] - 2026-04-04
### Go Port: Saved Archive Presets & Grouped Inspection
- **Bobcoin Workflow Ergonomics**: Updated the `bobcoin` submodule to its latest archive-workflow state, adding saved Vault filter presets plus grouping by owner/type for repeatable archive investigations.
- **Archive Workspace Upgrade**: Bobcoin Vault now supports persistent operator workflows instead of only transient search and sorting interactions.
- **Validation**: Re-validated the Bobcoin frontend production build after preset/grouping integration and synchronized the root workspace to the new submodule pointer.

## [11.17.0] - 2026-04-04
### Go Port: Degraded Recovery Diagnostics & Parity Testing
- **Bobcoin Recovery UX**: Updated the `bobcoin` submodule to its latest restore-diagnostics state, adding parity sufficiency reporting, per-shard failure reasons, and manual shard-omission testing controls to the browser recovery flow.
- **Operator Visibility**: Restore success now explicitly distinguishes standard recovery from parity-assisted reconstruction, making storage restoration behavior far more diagnosable.
- **Validation**: Re-validated the Bobcoin frontend production build after degraded-recovery integration and synchronized the root workspace to the new submodule pointer.

## [11.16.0] - 2026-04-04
### Go Port: Signed Publisher Provenance Metadata
- **Publisher Provenance**: Updated the `bobcoin` submodule to `v8.17.0`, enabling manifest anchors to carry signed publisher alias, website, and statement metadata.
- **Vault Identity Surfacing**: Bobcoin Vault now displays and searches publisher identity metadata in addition to heuristic trust overlays, making archive provenance more attributable and human-readable.
- **Validation**: Re-validated the Go consensus test suite and Bobcoin frontend production build after publisher-provenance integration and synchronized the root workspace to the new submodule pointer.

## [11.14.0] - 2026-04-04
### Go Port: Archive Trust & Reputation Overlay
- **Bobcoin Archive Intelligence**: Updated the `bobcoin` submodule to the latest merged archive-intelligence state, adding owner trust scores, trust tiers, sorting modes, and a sovereign publisher leaderboard to the Vault surface.
- **Provenance Surfacing**: Anchored content is now easier to evaluate at a glance via trust badges, owner-level archive summaries, and richer provenance cues across the archive UI.
- **Validation**: Re-validated the Bobcoin frontend production build after trust/reputation overlay integration and synchronized the root workspace to the new submodule pointer.

## [11.13.0] - 2026-04-04
### Go Port: Archive Discovery & Provenance Surfacing
- **Bobcoin Archive Intelligence**: Updated the `bobcoin` submodule to `v8.14.0`, upgrading Vault into a searchable, filterable, provenance-aware archive surface with signed/unsigned and cloaked metadata cues.
- **Discovery UX**: Bobcoin Vault now supports search/filtering across anchor name, owner, locator, manifest ID, ciphertext hash, proof hash, and type, plus a searchable network archive stream.
- **Validation**: Re-validated the Bobcoin frontend production build after discovery/provenance integration and synchronized the root workspace to the new submodule pointer.

## [11.12.0] - 2026-04-04
### Go Port: Cross-Surface Archive Reuse in Bobcoin UI
- **Bobcoin Surface Expansion**: Updated the `bobcoin` submodule to its latest archive-reuse state, allowing anchored manifests to be selected directly from the Go-lattice archive inside both Storage Market and Gallery flows.
- **Archive-Backed UX**: Manifest anchors now act as reusable content sources across Vault, Market, and Gallery, turning the Go storage/archive system into a broader product substrate rather than a single-purpose workflow.
- **Validation**: Re-validated the Bobcoin frontend production build after cross-surface archive reuse integration and synchronized the root workspace to the new submodule pointer.

## [11.11.0] - 2026-04-03
### Go Port: Vault Archive Surface for Manifest Anchors
- **Bobcoin Surface Integration**: Updated the `bobcoin` submodule to `v8.11.0`, rebuilding the Vault page into a dedicated Go-lattice archive browser for manifest anchors.
- **Archive UX**: The Bobcoin Vault now exposes personal and network anchor views plus the embedded storage workbench, making manifest provenance a first-class archive experience instead of a hidden workflow.
- **Validation**: Re-validated the Bobcoin frontend production build after the Vault archive integration and synchronized the root workspace to the new submodule pointer.

## [11.10.0] - 2026-04-03
### Go Port: On-Chain Manifest Anchoring & Provenance
- **Consensus Anchors**: Added `publish_manifest` and `data_anchor` support to the Go lattice, including durable in-memory anchor indexing and `/anchors` query endpoints for all anchors or owner-filtered views.
- **Publication Provenance**: `publish_manifest` blocks can now carry an explicit signed `publicationProof`, verified against the submitting wallet account in the Go consensus engine.
- **Bobcoin Integration**: Updated the `bobcoin` submodule to `v8.10.0`, enabling the storage workbench to submit signed manifest-anchor blocks to the Go lattice and display recent wallet-owned anchors.
- **Validation**: Verified `go test ./internal/consensus ./internal/publish -buildvcs=false`, `go build -buildvcs=false ./...`, and a successful Bobcoin frontend production build after manifest-anchoring integration.

## [11.9.0] - 2026-04-03
### Go Port: Browser Round-Trip Retrieval & Reconstruction
- **Bobcoin Retrieval UX**: Updated the `bobcoin` submodule to `v8.9.0`, adding manifest loading by locator/ID/URL plus in-browser shard download, hash verification, Reed-Solomon reconstruction, Go WASM decryption, and restored-file download.
- **Round-Trip Milestone**: The storage flow now supports the full operator round-trip: preprocess → publish → fetch manifest → fetch shards → reconstruct → decrypt → download.
- **Validation**: Re-validated the Bobcoin frontend production build after retrieval-flow wiring while preserving the root Go workspace stability.

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
