# Project Vision: Universal Sovereign Distribution Mesh (Bobtorrent)
# Megatorrent Vision

## The Mission
To create a truly decentralized, censorship-resistant content distribution platform that incentivizes storage and bandwidth sharing through a novel "Proof-of-Storage" mechanism on the **Bobcoin** blockchain.

## Phase 1: Foundation (Completed)
*   **Protocol v1.0**: Stable Blob/Manifest format using AES-GCM encryption.
*   **Reference Client**: Functional Node.js client with Web UI.
*   **Supernode (Java)**: High-performance backend with Reed-Solomon Erasure Coding.
*   **Unified Interface**: Single Web UI controlling both Node.js and Java backends.
*   **Observability**: Real-time Network Topology, Peer Metrics, and System Resources.

## Phase 2: Optimization & Intelligence (Active)
*   **Smart Ingest**: User-configurable redundancy (Erasure Coding vs. Replication).
*   **Resource Management**: AI-driven capacity planning and load throttling.
*   **Blockchain Integration**:
    *   **Wallet**: Full integration with Bobcoin/Solana wallets.
    *   **Incentives**: Automated storage deals and proof submission.
    *   **Bridge**: Seamless interaction with Filecoin and EVM chains.
*   **Transport Intelligence**:
    *   **I2P/Tor**: Deep configuration and routing optimization.
    *   **Multiplexing**: Efficient connection reuse.

2.  **Hybrid Storage Layer**
    -   **MuxEngine**: An encryption-first storage engine using AES/GCM for confidentiality.
    -   **Erasure Coding**: Reed-Solomon (4+2, 6+2) sharding to ensure data availability even if 30% of nodes fail.
    -   **Content Addressing**: Deduplication and integrity verification using cryptographic hashes (SHA-256/CID).

3.  **Incentivized Participation**
    -   **Bobcoin Integration**: A bridge to the Bobcoin/Filecoin blockchain for automated storage deals, proofs of storage (PoS), and rewards.
    -   **"Proof-of-Seeding" Rewards**: Users earn Bobcoin for seeding critical ecosystem components.

4.  **Universal Connectivity (Zero-Latency)**
    -   **Multi-Protocol Support**: Seamless integration of BitTorrent, WebTorrent, HTTP, and WebSocket trackers.
    -   **WebTransport (Upcoming)**: High-frequency, low-latency UDP-like transport natively in modern browsers.
    -   **Privacy-First**: Native support for Tor, I2P, and Mixnet transports to protect user identity.
    -   **Consensus-Verified Trackers**: Tracker swarms validated by ledger state (Stone.Ledger) to prevent hijacking.

5.  **Game-Streaming Mesh Integration**
    -   Serve as the distribution layer for the upcoming Bobcoin gaming ecosystem, streaming assets directly to players via localized, incentivized peers instead of centralized CDNs.

## 🚀 Strategic Roadmap

### Phase 1: Foundation (Completed)
-   [x] Core Storage Engine with Erasure Coding.
-   [x] Secure AES/GCM Encryption.
-   [x] Basic P2P Transport (TCP/UDP/WebSocket).
-   [x] Blockchain Bridge Scaffolding.

### Phase 2: Intelligence & Optimization (Completed)
-   [x] Advanced Health Monitoring & Circuit Breakers.
-   [x] **Content-Addressed Storage (CAS)**: Implement `ContentStore` for automatic deduplication and content routing.
-   [x] **DHT Integration**: Bridge internal peer finding with Filecoin's content routing.
-   [x] **Streaming Erasure Coding**: Enable playback of large media files while they are being reconstructed.

### Phase 3: Production Scale & Sovereignty (Current Focus)
-   [ ] **Supernode CLI & Diagnostics**: Command-line interfaces for operating the Supernode daemon and monitoring network health.
-   [ ] **Global Supernode Clusters**: Automatic clustering of nodes for high availability and distributed manifest sync.
-   [ ] **WebTransport (QUIC) Trackers**: Sub-millisecond peer discovery for Bobzilla integration.
-   [ ] **Cross-Chain Interoperability**: Extend rewards to Solana and Ethereum networks.

## 🛠️ The Go Port Vision (v11.5.0+)

The project is currently undergoing a complete architectural modernization through a systematic port to Go. This "Go Port" is designed to achieve:

1.  **Extreme Concurrency**: Utilizing goroutines to handle 10,000+ simultaneous BitTorrent swarms and lattice block arrivals without bottlenecking.
2.  **Memory Safety**: Eliminating legacy vulnerabilities while maintaining high performance.
3.  **Unified Ecosystem**: Consolidating Tracker, DHT, Supernode, and Consensus into a single performant codebase with specialized binaries (`supernode-go`, `lattice-go`, `dht-proxy`).
4.  **Privacy-First Networking**: Deep integration of the DHT Proxy and multi-transport support directly into the core engine.

## 🧠 Design Philosophy

-   **"Code is Law, Performance is King"**: No compromise on security or speed.
-   **"Verify, Don't Trust"**: Every block, every peer, and every proof is cryptographically verified (libsodium Ed25519 signatures).
-   **"Autonomous by Default"**: The system should run for months without human intervention, automatically recovering from failures.
## Phase 3: Cluster & Scale (Future)
*   **Supernode Clusters**: Automated clustering of Java nodes for massive scale.
*   **Geo-Distribution**: Smart placement of shards based on latency and cost.
*   **Content Addressing**: Full IPFS compatibility layer.

## Core Values
1.  **Privacy**: Default encryption and optional anonymity (Tor/I2P).
2.  **Performance**: High-throughput I/O and low-latency peering.
3.  **Usability**: "It just works" UI for end-users, powerful APIs for devs.
