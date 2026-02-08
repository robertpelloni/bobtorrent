# Project Dashboard

## 📂 Project Structure

This monorepo contains the following components of the Megatorrent (formerly Bobtorrent) ecosystem:

-   `root` (Tracker/Client) - JavaScript-based BitTorrent Tracker & Reference Client.
    -   `reference-client/` - Reference implementation of the Megatorrent Client (Node.js).
    -   `supernode-java/` - Core Java implementation of the Supernode logic, Erasure Coding, and Blockchain integration.
    -   `qbittorrent/` - [Submodule] Native C++ qBittorrent client integration.
    -   `cpp-reference/` - Canonical source for C++ reference implementation files.
    -   `docs/` - Project documentation.

## 📦 Submodules

| Submodule | Path | Commit | Description |
| :--- | :--- | :--- | :--- |
| **qbittorrent** | `qbittorrent` | `5abf458e...` | Native C++ BitTorrent client core. Forked to implement Megatorrent protocol. |

*Note: `webtorrent-bittorrent-tracker` functionality has been merged into the root package.*

## 📊 Key Components

### Reference Client (`reference-client`)
-   **Runtime**: Node.js
-   **Protocol**: Megatorrent v1
-   **Web UI**: `http://localhost:3000` (Identity, Publish, Subscribe, Discovery)
-   **Features**: Encrypted Blobs, Manifest Signing, Channel Subscriptions.

### Supernode Java (`supernode-java`)
-   **Cipher**: `AES/GCM` (MuxEngine)
-   **Erasure Coding**: Reed-Solomon (4+2, 6+2)
-   **Blockchain**: BobcoinBridge (Filecoin/Solana)
-   **Network**: DHTDiscovery (Kademlia)

### Megatorrent Tracker (`root`)
-   **Type**: HTTP/UDP/WebSocket Tracker
-   **Runtime**: Node.js
-   **Role**: Coordinates peer discovery for blobs and channels.

## 📅 Build Info
-   **Version**: 11.2.3 (Tracker/Client)
-   **Last Updated**: 2026-02-08
