# Supernode Java & Bobtorrent Roadmap

## Overview
Bobtorrent and Supernode Java constitute the P2P supernode implementation and decentralized distribution layer for the Bobcoin/Filecoin network. 

## Current Status: v11.2.4 (Tracker) / v0.2.0-SNAPSHOT (Java Supernode)

### ✅ Completed Features (v0.1.0 to v0.2.0)
- Core Node.js Tracker implementation with UDP/HTTP/WebSocket.
- Extracted and validated `manifest.js` logic with deterministic JSON serialization and Ed25519 signatures.
- Storage layer with streaming, caching, chunking strategies.
- Multi-transport support (Clearnet, Tor, I2P, IPFS, Hyphanet, Zeronet).
- Erasure coding (4+2 configuration, moving to 6+2) and AES-GCM encryption.
- Event-driven architecture with comprehensive health monitoring.
- Filecoin blockchain integration via BobcoinBridge.
- Predictive JVM resource allocation.

### ✅ Current Short Term Focus (v0.3.0) — COMPLETED
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

### 🚀 Medium Term (v0.4.0)
- [ ] **Enhanced Transport Protocol Implementations**
  - Tor v3: Improve circuit rotation and stream multiplexing.
  - IPFS: Add full CAR (Content Addressable Archive) payload extraction.
  - Hyphanet: Enhanced splitfile recovery.
- [ ] **Consensus-Verified Tracker Ledger**
  - Hooking tracker peer states into a Solana/Stone.Ledger memo bridge to definitively ban bad actors.

### 🌍 Long Term (v1.0.0 "Universal Mesh")
- [ ] **1000+ Concurrent Multi-Swarm Peer Handling**
- [ ] **Integrated Bobzilla Client Protocol**
- [ ] **"Proof-of-Seeding" Native Bobcoin Smart Contracts**
- [ ] **Embedded Game Asset Streaming**
