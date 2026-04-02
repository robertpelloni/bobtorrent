# Go Port Architecture & Strategy

## Overview
This document outlines the strategy for porting the entire `bobtorrent` ecosystem (Node.js tracker, Java Supernode, and related utilities) to Go, achieving 100% 1:1 feature parity. Go was selected for its superior concurrency model (goroutines), memory safety, and performance, which are critical for a high-throughput, P2P storage supernode network.

## Core Components to Port

### 1. BitTorrent Tracker (from Node.js)
*   **Protocols**: HTTP, UDP (BEP 15), WebSocket (WebTorrent).
*   **Go Implementation**: 
    *   `net/http` for HTTP/WS endpoints.
    *   `net` for UDP listener.
    *   `github.com/gorilla/websocket` for WS tracker.
    *   In-memory and Redis-backed state for swarms (`internal/tracker`).

### 2. Storage & Cryptography Layer
*   **Features**: AES-GCM encryption, Reed-Solomon erasure coding (4+2, 6+2), streaming reconstruction.
*   **Go Implementation**:
    *   `crypto/aes` and `crypto/cipher` for GCM.
    *   `github.com/klauspost/reedsolomon` for SIMD-accelerated erasure coding (`pkg/erasure`).
    *   `io.Reader` and `io.Writer` interfaces for stream chunking (`internal/storage`).

### 3. Transport & P2P Mesh
*   **Features**: Clearnet, Tor, I2P, IPFS, WebTransport (QUIC), Kademlia DHT.
*   **Go Implementation**:
    *   `github.com/quic-go/quic-go` for WebTransport/QUIC.
    *   `golang.org/x/net/proxy` for Tor/I2P SOCKS5.
    *   `github.com/libp2p/go-libp2p` for IPFS and robust DHT networking (`internal/transport`).

### 4. Blockchain & Consensus Bridge
*   **Features**: BobcoinBridge (Filecoin), Solana Memo txns, Proof-of-Seeding verifier.
*   **Go Implementation**:
    *   Ed25519 signatures via `crypto/ed25519`.
    *   JSON-RPC clients for Solana/Filecoin interaction.

### 5. Java Supernode Migration
*   **Features**: Manifest synchronization, quotas, predictive resource allocation, Kademlia DHT broadcast.
*   **Go Implementation**:
    *   Replaced by the unified `cmd/supernode` binary.
    *   Boltdb or BadgerDB for local fast manifest/quota storage.

## Execution Plan
1.  **Phase 1**: Scaffold Go modules, set up CI/CD for Go, and implement the Core Crypto/Erasure packages.
2.  **Phase 2**: Implement the Multi-protocol Tracker (HTTP/UDP/WS).
3.  **Phase 3**: Build the DHT Proxy utility (see `DHT_PROXY_UTILITY.md`).
4.  **Phase 4**: Implement the Storage/Transport mesh and migrate the Java Supernode logic.
5.  **Phase 5**: Integrate Blockchain bridges and verify 1:1 parity with extensive integration tests.
