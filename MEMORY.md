# Project Memory & Observations

## Core Architectural State
- The Bobtorrent Go port is now a real multi-binary platform rather than a thin prototype.
- The Go lattice has moved beyond a minimal proof-of-concept and now includes governance, NFT, staking, swap, market, websocket, and peer-broadcast capabilities.
- The storage layer exists in both native Go and WebAssembly form, which is strategically important because it reduces frontend/backend crypto drift.
- Bobcoin frontend integration is now partially live: the React app contains a browser-side Go WASM workbench for storage preprocessing, publication, retrieval, signed manifest anchoring, searchable trust-aware Vault-based archive browsing, archive reuse inside Market/Gallery flows, owner-level trust/reputation overlays, signed publisher alias/website/statement metadata, degraded recovery diagnostics, saved/grouped archive workflows, publisher avatar/profile/proof overlays, exportable recovery reports, shard failure/source attribution, portable preset/batch archive actions, host-level source reliability summaries, and typed proof semantics, while the Go supernode serves the required WASM runtime artifacts directly.

## Compatibility Findings
- The existing bobcoin frontend still speaks a partially older lattice dialect.
- Important compatibility expectations discovered during this session:
  - some pages POST wrapped blocks as `{ block: ... }`
  - some pages expect `/proposals` rather than `/governance/proposals`
  - some pages expect websocket upgrades at the lattice root URL
  - some pages still omit explicit `height` and `staked_balance`
  - NFT transfer UI currently uses `recipient` naming, while newer Go code preferred `newOwner`
- The Go lattice now includes compatibility handling for all of the above, but this is a temporary bridge, not the final state.

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
- Lattice state is still in-memory only.
- Filecoin archival is simulated rather than backed by Lotus or real RPC infrastructure.
- The Bobcoin WASM workbench can now publish to the Go supernode registry, restore published files in-browser, and anchor manifests on the Go lattice. Vault, Storage Market, and Gallery now reuse those anchors; Vault additionally provides searchable discovery, heuristic trust/reputation badging, signed publisher metadata, publisher avatar/profile/proof overlays, typed proof semantics, parity-aware degraded recovery diagnostics, saved/grouped operator workflows, exportable recovery reports, shard failure/source attribution, portable preset/batch actions, and host-level source reliability summaries. The next gap is deeper identity semantics plus richer long-horizon source reliability analysis.
- `qbittorrent` remote remains unreachable.
- Nested `bobcoin/research/*` submodule metadata still needs cleanup.
