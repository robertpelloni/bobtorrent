# Module & Submodule Dashboard (Omni-Workspace)

This dashboard provides a high-level view of the entire monorepo, its submodules, and their current versions.

## Project Structure Overview

```text
bobtorrent/ (Root)
├── .github/          # CI/CD Workflows
├── bobcoin/          # [SUBMODULE] Bobcoin economy, governance, and gaming
│   ├── frontend/     # React/Vite UI
│   ├── game-server/  # Node.js backend
│   └── ...
├── qbittorrent/      # [SUBMODULE] Reference C++ BitTorrent Client
├── cmd/              # [GO] CLI binaries (dht-proxy, supernode-go)
├── internal/         # [GO] Internal logic for the new Go port
├── pkg/              # [GO] Reusable Go packages (erasure, crypto)
├── supernode-java/   # Java implementation of the Supernode
├── webui-reference/  # Reference UI for the tracker
└── docs/             # Universal documentation and architectural specs
```

## Tracked Submodules

| Submodule | Location | Version | Last Sync | Purpose |
| :--- | :--- | :--- | :--- | :--- |
| **Bobcoin** | `bobcoin/` | v3.5.0 | 2026-04-03 | Economy and Gaming layer. Includes ZK-proofs and governance. |
| **qBittorrent**| `qbittorrent/` | v5.1.0-beta | 2026-03-09 | Reference BitTorrent implementation for testing and integration. |

## Internal Service Versions (Go Port)

| Service | Version | Status | Description |
| :--- | :--- | :--- | :--- |
| **DHT Proxy** | v11.3.1 | Active | Privacy proxy for BitTorrent DHT. SQLite backed. |
| **Core Storage** | v11.3.0 | WIP | Erasure coding and encrypted block storage in Go. |
| **Tracker (Go)** | v11.3.0 | WIP | High-performance Go implementation of the tracker. |

## External Projects (Nested)

| Project | Location | Purpose |
| :--- | :--- | :--- |
| **Forest** | `bobcoin/research/forest` | Rust implementation of Filecoin (reference). |
| **Solana** | `bobcoin/research/solana` | Solana core monorepo (reference). |
