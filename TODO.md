# TODO (Autonomous Development Backlog)

## ✅ Completed Through v11.32.0
- Unified Go binaries for `dht-proxy`, `supernode-go`, and `lattice-go`
- Go-native block lattice consensus engine
- P2P lattice block broadcast
- WebSocket live block feed for frontend and TUI consumers
- Go supernode TUI with live market + block feed + network stats
- Go storage layer with ChaCha20-Poly1305 + Reed-Solomon
- WASM export of the Go storage kernel
- Browser-side loader for `storage.wasm`
- Bobcoin frontend Go WASM workbench for browser-side encryption + erasure preprocessing
- Bobcoin frontend upload/publish flow for WASM-prepared shards and manifests
- Bobcoin frontend retrieval/reconstruction/decryption flow for published manifests
- Bobcoin frontend signed manifest anchoring on the Go lattice
- Bobcoin Vault archive browser for personal and network manifest anchors
- Bobcoin archive reuse across Storage Market and Gallery flows
- Bobcoin Vault search/filter/discovery and provenance badging for archive intelligence
- Bobcoin Vault owner trust scores, tiers, sorting modes, and sovereign publisher leaderboard
- Bobcoin signed publisher alias/website/statement metadata on manifest anchors
- Bobcoin degraded recovery diagnostics, parity sufficiency reporting, and manual shard-omission testing controls
- Bobcoin saved archive presets and owner/type grouping modes in Vault
- Bobcoin publisher avatar/profile/proof-link overlays in Vault archive cards
- Bobcoin exportable JSON recovery reports from degraded recovery diagnostics
- Bobcoin shard failure categorization and source host attribution in restore diagnostics
- Bobcoin preset export/import and batch archive export/copy actions in Vault
- Bobcoin host-level source reliability summaries derived from persisted recovery reports
- Bobcoin long-horizon source reliability trends with week-over-week host comparisons
- Go manifest anchors with typed publisher proof-kind metadata
- Go manifest anchors with structured attestation labels and issuers
- SQLite-backed durable lattice block log with replay-based cold-boot recovery
- Materialized lattice snapshots with tail-replay recovery acceleration
- Lattice persistence verification and conservative snapshot repair workflow
- Lattice persistence JSON export and live SQLite backup workflow
- Lattice persistence bundle import and backup restore workflow
- Frontend compatibility endpoints for existing bobcoin pages
- Go supernode compatibility endpoints for Bobcoin UI (`/stats`, `/add-torrent`, `/remove-torrent`)
- Go supernode static serving for `storage.wasm` and `wasm_exec.js`
- Go supernode publication registry for uploaded shards and manifests
- Build pipeline packaging of `storage.wasm` and `wasm_exec.js`
- Full repository compile validation with `go build -buildvcs=false ./...`

## Highest Priority Next Steps
- [ ] **Expand publication provenance beyond current publisher profile overlays**
  - optional uploader profile / reputation layer
  - external identity/attestation integrations beyond the new structured proof cards
- [ ] **Expand per-source recovery analytics further**
  - exportable comparative source diagnostics
  - even longer-term source-path trend visibility
- [ ] **Deepen lattice persistence hardening**
  - broader persistence-aware consensus test coverage
  - configurable snapshot cadence / retention controls
  - signed/encrypted operator backup bundles
- [ ] **Real Filecoin bridge**
  - replace mock `internal/bridges/filecoin.go` behavior with Lotus RPC
  - persist and expose deal IDs
- [ ] **Consensus peer sync improvements**
  - initial peer bootstrap
  - duplicate suppression / loop prevention
  - late joiner state catch-up

## Important Compatibility / Cleanup Tasks
- [ ] **Remove temporary legacy block shim** once bobcoin frontend includes explicit `height` and `staked_balance`
- [ ] **Unify block hashing rules** between Go and browser-side block construction
- [ ] **Add tests** for consensus transitions:
  - send/receive
  - NFT mint/transfer
  - stake/unstake
  - swaps
  - proposals/votes
- [ ] **Add integration tests** for websocket live feed and wrapped-vs-raw block submission formats

## Strategic Backlog
- [ ] **Go Supernode WebUI integration**
- [ ] **Durable market manifests + shard metadata registry**
- [ ] **Game engine asset ingestion path**
- [ ] **Global decentralized storage network launch**
- [ ] **Investigate unreachable `qbittorrent` remote**
