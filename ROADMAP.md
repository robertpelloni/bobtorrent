# Megatorrent Project Roadmap & Status

**Current Version:** v2.0.0-alpha.1
**Protocol Version:** v6 (Draft)

## 🚧 In Progress: v2.0 (Anonymity++)

### 1. DHT-over-TCP
*   **Goal:** Allow nodes behind Tor/I2P (who cannot use UDP) to participate in the DHT and look up content.
*   **Mechanism:** Encapsulate KRPC messages inside the Encrypted TCP Transport (`MSG_DHT_QUERY`, `MSG_DHT_RESPONSE`).
*   **Status:** Protocol Definition.

### 2. Bobcoin Integration
*   **Goal:** Native economy for file sharing and arcade games.
*   **Mechanism:** Embedded Bobcoin Node (Solana/Monero Hybrid) with "Proof of Dance".
*   **Status:** Prototype Implemented (Node.js).

## ✅ Accomplished Features (v1.x)

### 1. Core Architecture
*   **Monorepo Structure:** Root Node.js project, `qbittorrent` submodule.
*   **Documentation:** `PROTOCOL_FINAL.md`, `HANDOFF.md`, `README.md`, `ROADMAP.md`, `DASHBOARD.md`.
*   **Docker:** `Dockerfile`, `docker-compose.yml` for swarm simulation.

### 2. Decentralized Control Plane (DHT)
*   **Library:** `bittorrent-dht` (Node.js) / `libtorrent` (C++).
*   **Feature:** Mutable Items (BEP 44) for Manifests.
*   **Feature:** Immutable Items for Peer Lists (Announce).

### 3. Data Plane & Anonymity
*   **Encryption:** ChaCha20-Poly1305 (IETF) + Ephemeral ECDH (X25519) Handshake.
*   **Transport:** Custom Binary Protocol v5 (`secure-transport.js`).
*   **Tor Support:** SOCKS5 client integration.
*   **Traffic Analysis Resistance:** Fixed-size 1MB padding.
*   **Safe Mode:** Auto-disables UDP DHT when Proxy is enabled to prevent leaks.

### 4. Subscription & Updates
*   **Manifest:** JSON format signed by Ed25519 keys.
*   **Gossip:** Automatic push updates to connected peers.
*   **Gateway:** Authenticated relaying of signed manifests to the DHT.

### 5. Client Implementation (Node.js)
*   **CLI:** `megatorrent-reference-client` (ingest, publish, subscribe, serve).
*   **WebUI:** qBittorrent WebUI integration (`megatorrent.js`).
*   **Streaming:** Memory-efficient ingestion and reassembly.

### 6. C++ Reference (qBittorrent Integration)
*   **Location:** `cpp-reference/megatorrent/` (Integrated into `qbittorrent/src/base/`).
*   **DHT Client:** `dht_client.h/cpp` (Wraps libtorrent).
*   **Secure Socket:** `secure_socket.h/cpp` (Transport Layer with OpenSSL Noise-IK).
*   **Manifest:** `manifest.h/cpp` (Parser & OpenSSL Ed25519 Validator).
*   **Blob Downloader:** `blob_downloader.h/cpp` (Peer Connection & Integrity).
*   **I2P Support:** `i2p_sam.h/cpp` (SAM v3.1 Client) & `SessionImpl` integration.
    *   *Details:* Implements `HELLO`, `SESSION CREATE`, `STREAM CONNECT`, and Accept Loop.
*   **WebAPI:** `MegatorrentController` exposing subscription management to WebUI.

---

## 🔮 Future Work

1.  **UI Implementation:** Build the actual Qt Widgets for the Subscription Manager (currently API-only).
2.  **Hybrid Mode:** Simultaneous UDP and TCP DHT participation for bridge nodes.
