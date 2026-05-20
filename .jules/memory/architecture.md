# BobTorrent Project Memory & Architectural Summary

## 1. Project Overview & Omni-Workspace Concept
**BobTorrent** is a decentralized, anonymity-focused peer-to-peer file sharing and predictive streaming platform. The repository is structured as an **Omni-Workspace**, a centralized command center that orchestrates multiple independent components and submodules.

*   **Current State (v11.60+):** The architecture has fully pivoted to a native, high-performance Go ecosystem consisting of specialized, communicating binaries (`supernode-go`, `lattice-go`, `dht-proxy`) and WASM browser targets. 

## 2. Core Architecture & Services (Go Ecosystem)
*   **`supernode-go` (The Main Orchestrator):** Acts as the central tracker, hosts the comprehensive Web UI (`web/ui`), and provides high-level APIs for identity (`/key/generate`), ingestion (`/ingest`), publishing (`/publish`), and tracking subscriptions (`/subscribe`, `/subscriptions`).
*   **`lattice-go` (The Consensus Layer):** Manages an asynchronous, local block-lattice engine that governs economy (Bobcoin), NFTs, and metadata anchoring.
*   **`dht-proxy` (Privacy Routing):** Operates a standalone Kademlia DHT node, insulating end-users from direct IP exposure to public DHT networks.
*   **WASM Storage Bridge (`cmd/wasm`):** Compiles the core storage primitives into WebAssembly (`storage.wasm`) for zero-trust, browser-side file sharding and encryption.

## 3. Storage & Cryptography Patterns
*   Files are erasure-coded (data+parity shards) and encrypted with AES-256-GCM / ChaCha20-Poly1305.
*   Encrypted shards are uploaded to the supernode (`/upload-shard`) and durably persisted based on their SHA-256 hashes inside `internal/publish/registry.go`.
*   **Manifest Publication (BEP 44):** The system generates JSON manifests which are signed by Ed25519 identity keys and published via the BitTorrent Mainline DHT using BEP 44 mutable items.

## 4. UI & Pub/Sub Mechanics
*   The supernode holds active subscriptions in memory (`subscriptionStore`) and exposes a `/stats` endpoint that merges real-time metrics across storage utilization, active BitTorrent swarms, DHT routing health, Filecoin bridge status, and active Pub/Sub subscription counts. The Web UI continuously polls this to provide a live "Status" dashboard.

## 5. Session Accomplishments
*   Successfully fully wired the backend `getSubscriptionCount()` metric into the JSON `/stats` endpoint.
*   Patched the frontend dashboard (`web/ui/app.js`) to successfully display active Pub/Sub telemetry in real-time.
*   Updated all omnibus documentation (`ROADMAP.md`, `TODO.md`, `HANDOFF.md`, `CHANGELOG.md`, `VERSION`) to reflect the newly completed feature.
*   Successfully ran the test suite and submitted changes.

**Stopping Condition Met:** The next remaining tasks in the `TODO.md` backlog (e.g., "Game engine asset ingestion path", "Global decentralized storage network launch") are highly ambiguous and require major architectural decisions. Per the core directives, execution is halted here to avoid making unsupported assumptions.