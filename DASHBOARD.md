# Module & Submodule Dashboard (Omni-Workspace)

## Version Snapshot
- **Root Version**: `11.10.0`
- **Workspace Root**: `bobtorrent/`
- **Primary Branch**: `master`
- **Build Status**: Go workspace compile validated with `go build -buildvcs=false ./...`

## Repository Structure
```text
bobtorrent/
├── bobcoin/                  # Submodule: Bobcoin economy, frontend, game-server
├── qbittorrent/              # Local reference client checkout / submodule target (remote currently broken)
├── cmd/
│   ├── dht-proxy/            # Go DHT privacy proxy binary
│   ├── lattice-go/           # Go block lattice node entrypoint
│   ├── supernode-go/         # Go supernode entrypoint
│   └── wasm/                 # Go WASM entrypoint for browser storage kernel
├── docs/
│   ├── UNIVERSAL_LLM_INSTRUCTIONS.md
│   └── WASM_STORAGE_BRIDGE.md
├── internal/
│   ├── bridges/              # Cross-chain / external-network bridge layer
│   ├── consensus/            # Go lattice engine + websocket server
│   ├── dhtproxy/             # DHT proxy crawler/db/API
│   ├── publish/              # Published shard + manifest persistence registry
│   ├── tracker/              # HTTP + UDP tracker implementation
│   ├── transport/            # DHT transport node wrapper
│   └── tui/                  # Bubble Tea operator dashboard
├── pkg/
│   ├── storage/              # Encryption + erasure coding
│   └── torrent/              # Block, crypto, GeoIP utilities
├── supernode-java/           # Legacy / reference Java supernode
└── web/
    └── storage-wasm-loader.js # Browser loader for Go storage.wasm
```

## Submodule / External Status
| Component | Location | Status | Notes |
|---|---|---:|---|
| Bobcoin | `bobcoin/` | Active | Frontend + game stack remains the primary UI reference; now includes Go storage WASM workbench, publication flow, browser-side retrieval/reconstruction, and signed Go-lattice manifest anchoring (`v8.10.0`). |
| qBittorrent fork | `qbittorrent/` | Blocked | Local files exist, but remote repo reference remains unreachable. |
| Forest research | `bobcoin/research/forest` | Blocked | Nested submodule metadata issue remains unresolved upstream/local. |
| Solana research | `bobcoin/research/solana` | Blocked | Nested submodule metadata issue remains unresolved upstream/local. |

## Go Service Matrix
| Service | Artifact | Status | Purpose |
|---|---|---:|---|
| DHT Proxy | `build/dht-proxy` | Buildable | Privacy-preserving peer discovery with GeoIP sorting |
| Lattice Node | `build/lattice-go` | Buildable | Go asynchronous block lattice + websocket event feed |
| Supernode | `build/supernode-go` | Buildable | Tracker, DHT, seeding, market automation, TUI, Bobcoin UI-compatible stats/control endpoints, WASM artifact serving, and manifest/shard publication registry |
| Storage WASM | `build/storage.wasm` | Buildable | Browser-side Go storage kernel |
| Go WASM Runtime | `build/wasm_exec.js` | Packaged | Required runtime bridge for browser execution |

## Current Go Port Capabilities
- HTTP and UDP tracker support
- Kademlia DHT wrapper via `anacrolix/dht`
- GeoIP-enriched DHT proxy responses
- Block lattice consensus in Go
- Governance / NFT / staking / swap block types in Go lattice engine
- WebSocket live block feed
- Terminal operations dashboard
- Browser-consumable storage WASM runtime

## Current Known Gaps
- Lattice state is still in-memory only
- Filecoin bridge is simulated, not production RPC-backed
- Manifest anchors are implemented, but broader reuse across storage-market / NFT / vault surfaces is not yet complete
- qBittorrent remote reference remains broken
