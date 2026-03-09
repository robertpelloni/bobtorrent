# Project Dashboard

## 📂 Project Structure

-   `bobtorrent/` (Root) - JavaScript-based BitTorrent Tracker with Universal Swarm functionality.
    -   `supernode-java/` - Core Java implementation of the Supernode logic, Erasure Coding, and Blockchain integration.
    -   `qbittorrent/` - [Submodule] C++ qBittorrent client integration.
    -   `bobcoin/` - [Submodule] Bobcoin hybrid blockchain and ZK-based minting logic.
    -   `docs/` - Centralized universal documentation and Master Protocol definitions.
    -   `lib/` - Node.js core logic (including Manifest encryption and validation).

## 📦 Submodules

| Submodule | Path | Source Repository | Commit Status (Latest Build) |
| :--- | :--- | :--- | :--- |
| **qbittorrent** | `qbittorrent` | `https://github.com/robertpelloni/qbittorrent` | Synced |
| **bobcoin** | `bobcoin` | `https://github.com/robertpelloni/bobcoin` | Synced |

## 📊 Key Components

### Supernode Java (`supernode-java`)
-   **Cipher**: `AES/GCM` (MuxEngine)
-   **Erasure Coding**: Reed-Solomon (4+2, 6+2), Streaming Support
-   **Blockchain**: BobcoinBridge (Filecoin/Solana)
-   **Network**: DHTDiscovery (Kademlia), Multi-transport (Clearnet, Tor, I2P, IPFS, Hyphanet)

### Bobtorrent Tracker (`root`)
-   **Type**: HTTP/UDP/WebSocket Tracker
-   **Runtime**: Node.js (v18+)
-   **Manifest**: Signed and optionally encrypted JSON sequence blobs with Ed25519 & XSalsa20-Poly1305 (libsodium).

## 📅 Build Info
-   **Tracker Version**: 11.2.4
-   **Supernode Java Version**: 0.2.0-SNAPSHOT
-   **Last Build Date**: 2026-03-09
