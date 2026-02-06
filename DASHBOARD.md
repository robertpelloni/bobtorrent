# Project Dashboard

## 📂 Project Structure

-   `bobtorrent/` (Root) - JavaScript-based BitTorrent Tracker (v11.2.2).
    -   `supernode-java/` - Core Java implementation of the Supernode logic, Erasure Coding, and Blockchain integration.
    -   `qbittorrent/` - [Submodule] C++ qBittorrent client integration.
    -   `webtorrent-bittorrent-tracker/` - [Submodule] Upstream WebTorrent tracker reference.
    -   `docs/` - Project documentation.

## 📦 Submodules

| Submodule | Path | Description |
| :--- | :--- | :--- |
| **qbittorrent** | `qbittorrent` | Native C++ BitTorrent client core. Used for high-performance peering. |
| **webtorrent-bittorrent-tracker** | `webtorrent-bittorrent-tracker` | Upstream reference for the tracker protocol. |

## 📊 Key Components

### Supernode Java (`supernode-java`)
-   **Cipher**: `AES/GCM` (MuxEngine)
-   **Erasure Coding**: Reed-Solomon (4+2, 6+2)
-   **Blockchain**: BobcoinBridge (Filecoin/Solana)
-   **Network**: DHTDiscovery (Kademlia)

### Bobtorrent Tracker (`root`)
-   **Type**: HTTP/UDP/WebSocket Tracker
-   **Runtime**: Node.js

## 📅 Build Info
-   **Version**: 11.2.2 (Tracker), 0.1.0-SNAPSHOT (Java)
-   **Last Updated**: 2026-02-05
