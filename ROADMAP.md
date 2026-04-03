# Supernode Java & Bobtorrent Roadmap

## Overview
Bobtorrent and Supernode Java constitute the P2P supernode implementation and decentralized distribution layer for the Bobcoin/Filecoin network. 

## Current Status: v11.2.4 (Tracker) / v0.4.0 (Java Supernode)

### ✅ Completed Features (v0.1.0 to v0.2.0)
- Core Node.js Tracker implementation with UDP/HTTP/WebSocket.
- Extracted and validated `manifest.js` logic with deterministic JSON serialization and Ed25519 signatures.
- Storage layer with streaming, caching, chunking strategies.
- Multi-transport support (Clearnet, Tor, I2P, IPFS, Hyphanet, Zeronet).
- Erasure coding (4+2 configuration, moving to 6+2) and AES-GCM encryption.
- Event-driven architecture with comprehensive health monitoring.
- Filecoin blockchain integration via BobcoinBridge.
- Predictive JVM resource allocation.

### ✅ Current Short Term Focus (v11.4.3) — COMPLETED
- [x] **Autonomous Go Supernode**:
  - Integrated `github.com/anacrolix/torrent` for real file seeding and data provisioning.
  - Implemented background market poller using `github.com/go-resty/resty/v2`.
  - Added automated bid acceptance logic: Supernode now discovers, seeds, and claims funds for storage bids autonomously.
- [x] **Lattice Protocol Port**:
  - Implemented `pkg/torrent/block.go` with hashing and signing logic compatible with the Bobcoin Block Lattice.
  - Ported wallet handling and SPoRA proof generation to Go.
- [x] **Node CLI and Diagnostics Tools**: 
  - Implementation of a terminal UI/CLI for node configuration, manifest inspection, and real-time swarm diagnostic monitoring.
- [x] **Distributed Manifest Synchronization**:
  - Kademlia DHT broadcast mechanisms to sync manifests across global clusters autonomously.
- [x] **Storage Quotas Enforcement**:
  - `maxStorageBytes` configurable limit with quota check in ingest pipeline.
- [x] **Streaming Reed-Solomon Parity Repair**:
  - On-the-fly re-encoding and persistence of missing shards during retrieval.
- [x] **WebTransport Integration**:
  - QUIC-based HTTP/3 transport for the Node.js tracker with graceful fallback.

### ✅ Medium Term (v0.4.0) — COMPLETED
- [x] **Enhanced Transport Protocol Implementations**
  - Tor v3: MultiplexedCircuitPool with round-robin, failover, per-circuit rotation.
  - IPFS: CARExtractor for CAR v1 archive parsing and block extraction.
  - Hyphanet: SplitfileRecoveryOptions with retry escalation and priority boosting.
- [x] **Consensus-Verified Tracker Ledger**
  - TrackerLedger records peer violations as Solana memo txns; consensus-based bad actor banning.

### ✅ Advanced Features (v0.5.0) — COMPLETED
- [x] **Proof-of-Seeding Verifier**
  - Cryptographic challenge-response with Merkle proofs, seeder reliability scoring, on-chain submission.
- [x] **Multi-Swarm Peer Coordinator**
  - O(1) swarm lookup, cross-swarm peer sharing, priority bandwidth allocation for 1000+ concurrent swarms.

### 🚀 Next (v0.6.0) — COMPLETED
- [x] **Embedded Game Asset Streaming**
  - Real-time game asset delivery via P2P with prioritized chunk fetching and LOD support.
- [x] **Bobzilla Client Protocol**
  - Native Bobzilla wire protocol for cross-client interoperability, capability negotiation, and CRC-32 integrity.

### 🌍 Long Term (v1.0.0 "Universal Mesh")
- [ ] **1000+ Concurrent Multi-Swarm Peer Handling**
- [ ] **Full Game Engine Integration**
- [ ] **Global Decentralized Storage Network Launch**
