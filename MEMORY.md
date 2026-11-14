# Project Memory & Observations

## Core Architectural State
- The Bobtorrent Go port is now a real multi-binary platform rather than a thin prototype.
- The Go lattice has moved beyond a minimal proof-of-concept and now includes governance, NFT, staking, swap, market, websocket, peer-broadcast capabilities, duplicate-aware block ingestion, ordered confirmed-block catch-up via `/blocks`, bootstrap sync initiation via `/bootstrap`, replay-backed SQLite durability for confirmed blocks, materialized snapshots for faster cold boot, operator-visible persistence verification/repair tooling, backup/export controls, portable import/restore workflows for durable state, and a growing Go-side service compatibility surface around the supernode.
- The storage layer exists in both native Go and WebAssembly form, which is strategically important because it reduces frontend/backend crypto drift.
- Bobcoin frontend integration is now partially live: the React app contains a browser-side Go WASM workbench for storage preprocessing, publication, retrieval, signed manifest anchoring, searchable trust-aware Vault-based archive browsing, archive reuse inside Market/Gallery flows, owner-level trust/reputation overlays, signed publisher alias/website/statement metadata, degraded recovery diagnostics, saved/grouped archive workflows, publisher avatar/profile/proof overlays, exportable recovery reports, shard failure/source attribution, portable preset/batch archive actions, long-horizon source reliability trends, and structured publisher attestation semantics, while the Go supernode serves the required WASM runtime artifacts directly.

## Compatibility Findings
- The existing bobcoin frontend still speaks a partially older lattice dialect.
- Important compatibility expectations discovered during this session:
  - some pages POST wrapped blocks as `{ block: ... }`
  - some pages expect `/proposals` rather than `/governance/proposals`
  - some pages expect websocket upgrades at the lattice root URL
  - some pages still omit explicit `height` and `staked_balance`
  - NFT transfer UI currently uses `recipient` naming, while newer Go code preferred `newOwner`
- The Go lattice now includes compatibility handling for all of the above, but this is a temporary bridge, not the final state.
- Multi-node sync is no longer just best-effort fan-out: the lattice now preserves confirmed global block order for catch-up, exposes `GET /blocks`, and provides an autonomous background sync loop that periodically reconciles state with peers. The system can now automatically resolve lag and detect divergence without manual operator intervention. Every core consensus state transition is now verified by dedicated unit tests.

## Build / Toolchain Findings
- `anacrolix/dht` API drift required moving from an imagined `Addr` field to explicit `net.ListenPacket` wiring through `ServerConfig.Conn`.
- `reedsolomon.Encoder.Join` writes to an `io.Writer`, so buffer-based joining is required.
- `go build ./...` in this workspace requires `-buildvcs=false` because VCS stamping can fail under the current submodule / repo state.
- `cmd/wasm/main.go` must be gated behind `//go:build js && wasm` so host builds do not try to compile `syscall/js`.

## Operational Preferences
- Keep the supernode autonomous and observable.
- Preserve compatibility with the legacy bobcoin frontend until the frontend is explicitly migrated.
- Prefer shipping reusable bridge layers (like `web/storage-wasm-loader.js`) so future UI wiring is fast and low-risk.

## Technical Debt / Roadblocks
- Lattice state is no longer process-ephemeral: confirmed blocks are durably appended to SQLite, materialized snapshots accelerate cold boot, newer blocks are replayed on startup, the snapshot layer can be verified/rebuilt conservatively, and operators can now export, back up, import, or restore durable state, though configurable snapshot controls and signed/encrypted backup bundles are still outstanding.
- Filecoin archival is simulated rather than backed by Lotus or real RPC infrastructure.
- The Bobcoin WASM workbench can now publish to the Go supernode registry, restore published files in-browser, and anchor manifests on the Go lattice. Vault, Storage Market, and Gallery now reuse those anchors; Vault additionally provides searchable discovery, heuristic trust/reputation badging, signed publisher metadata, publisher avatar/profile/proof overlays, structured attestation semantics with live "Zero-Trust" verification (including real GitHub Gist checks), parity-aware degraded recovery diagnostics, saved/grouped operator workflows, exportable recovery reports, shard failure/source attribution, portable preset/batch actions, long-horizon source reliability trends, and exportable comparative source diagnostics. Bobcoin `v8.89.0` now also supports signed shareable diagnostics packages with in-browser verification plus local-vs-imported package comparison review, making reliability evidence attributable, portable, and easier to contextualize during operator handoff. The frontend bundle profile is highly optimized: page routes are lazy-loaded, heavy dependencies are manually chunked, and the heavy 3D topology visualization is aggressively deferred to a secondary load, resulting in a ~50kB main application bundle. The migrated Go compatibility surface now includes the FHE oracle endpoint, hardened Go-native websocket matchmaking, real multipart `/upload` torrent registration, stricter `/spora/:challenge` attestation behavior, a durable `torrents.json` seeding registry, a durable SQLite-backed publication registry with a `GET /assets` discovery API, and a production-ready identity `VerifierService` with real GitHub support. Bobcoin now defaults both compatibility HTTP traffic and signaling traffic toward the Go supernode while retaining explicit overrides for specialized legacy deployments. The signaling layer now has keepalive deadlines, ping/pong refresh, stale-wait eviction, and operator-visible telemetry. The replay-backed lattice persistence layer now also supports signed/encrypted operator backup bundles that wrap safe SQLite backups in `scrypt` + `ChaCha20-Poly1305` encryption with optional Ed25519 signatures, operator-tunable snapshot cadence/retention via explicit config/env, and a regression suite covering snapshot-tail replay of a mixed multi-account ledger spanning send/open/receive, governance, NFT, staking, and swap transitions. The Filecoin bridge now has a real Lotus JSON-RPC publication/verification path plus durable local deal records and explicit safe fallback behavior when Lotus is unconfigured. The next gap is deeper identity semantics and deciding how far to push the remaining specialized Node services into Go.
- `qbittorrent` remote remains unreachable.
- Nested `bobcoin/research/*` submodule metadata still needs cleanup.
- Mega-Messenger integration has moved into implementation: `libp2p` GossipSub is now scaffolded as the primary decentralized messaging backbone.
- I2P/SAM datagram support is scaffolded for low-latency anonymous signaling, complementing the stream-based SAM integration.

## Go Port Architecture Notes (v11.60.25)
- The legacy fragmented NodeJS, Java, and C++ repositories have been successfully unified into a single Go "ultra-project" under `bobtorrent/`.
- Legacy code resides in `archive/` for structural and functional reference.
- Custom Binary Protocol v5 has been natively mapped: AES-256-GCM Blobs (1MB) map 1:1 with standard BitTorrent pieces, mapped to 20-byte InfoHashes via SHA1 truncation.
- Identity and attestation verification now spans Ed25519 cryptographic pairs, Solana public keys, GitHub Gists, ORCID metadata, and custom signed-message endpoints.
- E2E testing uses an in-memory test network avoiding race conditions by enforcing correct InfoHash routing on ingestion.
- The HTTP APIs (`/api/ingest`, `/api/publish`, `/api/subscribe`, `/api/assets`) and Web UI are statically embedded and served concurrently without locking issues during HTTP streaming.

## Command Console & System Tray Consolidation (v11.60.44)
- Web UI has been consolidated from fractured tabs/subpages into a unified Sovereign Command Center with top-priority metrics and live transfers on the primary pane, backed by intuitive module tabs (Overview & Transfers, Publishing Studio & Registry, Decentralized Mesh Chat, Identity & Sybil Attestation, Bobcoin Wallet & Lattice Consensus).
- Enhanced with inline tooltip explanations, live status beacons, and direct real-time polling to eliminate stale state.
- System tray management consolidated into `scripts/system_tray_controller.py` providing complete daemon lifecycle controls (start, stop, restart, status, web console opening, graceful exit) across headless and GUI environments.

## Mobile Messenger Light Client & Integration (v11.60.46)
- Added `web/ui/mobile_preview.html`, providing a responsive light-node mobile chat interface wired to `cmd/supernode-go/mega_messenger_bridge.go` (`/mega-bridge`).
- Integrated a direct Mobile Preview launcher button in the Decentralized Mesh Chat header within `web/ui/index.html` to streamline side-by-side operator monitoring.
- Supports light client control plane messaging, topic switching (`#global`, `#announcements`, `#market`, `#support`), real-time libp2p GossipSub broadcasting, and SQLite message history retrieval.

## Session Findings (2026-11-14, v11.60.47-53)
- **Headless parity rule**: any new service init added to `realMain` (TUI) must also be added to `runHeadless` in `main_wrapper.go` — the tray launches `-headless` only. This divergence silently killed the dashboard once.
- **Version truth chain**: `VERSION` file -> `-X main.Version` -> `GET /status` -> dashboard footer. Never hardcode version strings in handlers or docs; DASHBOARD/DEPLOY now reference the VERSION file.
- **Windows toolchain**: local mingw gcc emits objects Go cannot parse; everything builds/tests with `CGO_ENABLED=0` (modernc sqlite stack). `go test -race` unavailable here. `build.bat` must build package paths, never single files.
- **Port map (this machine)**: bobcoin owns :8000 (go-supertorrent) + :4001 (go-lattice). Bobtorrent: supernode :8001 (`-port`), lattice :4000/:4002+ (`-port`, own DB per node). Never kill bobcoin processes.
- **Consensus live-math**: every balance-bearing block needs exact demurrage replication (0.0001/60000 per ms on frontier balance/timestamp) or the engine rejects with a precise expected-vs-got message. Costs: proposal 10, NFT mint 50 / transfer 1, data_anchor 10. Staking yield (5%/yr) truncates to 0 for small/short stakes. Signatures verified only when present (Ed25519 over block hash, base58).
- **Multi-node proven**: bootstrap catch-up (21 blocks/1 page), autonomous 30s sync, divergence -> analysis `divergent` + apply 409 refused, fail-closed. Persistence: `/persistence/verify` healthy on the 22-block DB; snapshotCount still 0 at sequence 22 (first snapshot threshold not yet reached).
- **Dashboard rule**: new backend endpoints are not done until surfaced same-origin in `web/ui` (proxy via `registerBobcoinProxies` for lattice state) with labels + tooltips; chain-supplied strings must be HTML-escaped in JS renderers.
