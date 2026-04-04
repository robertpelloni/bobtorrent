# TODO (Autonomous Development Backlog)

## ✅ Completed Through v11.6.0
- Unified Go binaries for `dht-proxy`, `supernode-go`, and `lattice-go`
- Go-native block lattice consensus engine
- P2P lattice block broadcast
- WebSocket live block feed for frontend and TUI consumers
- Go supernode TUI with live market + block feed + network stats
- Go storage layer with ChaCha20-Poly1305 + Reed-Solomon
- WASM export of the Go storage kernel
- Browser-side loader for `storage.wasm`
- Frontend compatibility endpoints for existing bobcoin pages
- Build pipeline packaging of `storage.wasm` and `wasm_exec.js`
- Full repository compile validation with `go build -buildvcs=false ./...`

## Highest Priority Next Steps
- [ ] **Wire WASM into bobcoin frontend**
  - import/copy `web/storage-wasm-loader.js`
  - load `wasm_exec.js` + `storage.wasm`
  - switch upload preprocessing from JS to Go WASM
- [ ] **Persist lattice state**
  - durable storage for chains, blocks, pending txs, NFTs, proposals, swaps
  - startup replay / restore path
  - snapshot + rollback-safe state recovery
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
