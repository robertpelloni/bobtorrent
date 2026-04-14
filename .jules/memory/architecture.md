# BobTorrent Project Memory & Architectural Summary

## 1. Project Vision & Evolution
**BobTorrent** is a decentralized, anonymity-focused peer-to-peer file sharing and predictive streaming platform. 
* **Legacy State (v2.2.0 and prior):** The project operated as a fragmented monorepo consisting of a Node.js Reference Client, a Java Supernode (`supernode-java`), and a heavily patched C++ qBittorrent submodule serving as the BitTorrent backend.
* **Current State (v3.0.0+):** The project is undergoing a massive architectural shift into a single, unified, natively-compiled Go "ultra-project" (`bobtorrent/`). All legacy components have been safely moved to `archive/`.

## 2. Core Architecture & Technologies (Go Port)
The new Go architecture is structured idiomatically (`cmd/`, `pkg/`, `internal/`) to handle the following core domains natively:

* **BitTorrent & DHT Engine (`pkg/dht`):**
  * We are replacing the C++ `libtorrent`/`qBittorrent` backend with pure Go implementations (e.g., `anacrolix/torrent`).
  * **InfoHash Mapping:** The project uses a unique mapping system. 32-byte SHA256 Blob IDs (Megatorrent keys) are mapped/truncated to 20-byte SHA1 InfoHashes to maintain compatibility with standard BitTorrent DHT `announce` and `lookup` operations.
* **Obfuscated Storage Protocol (`pkg/storage`):**
  * Files are split into chunks ("Blobs") and encrypted using **AES-256-GCM**.
  * The resulting blobs consist of a Nonce + Ciphertext + Auth Tag.
  * The system utilizes detached decryption keys (Manifest entries) to ensure data privacy and obfuscation against Deep Packet Inspection (DPI).
* **Solana Wallet & Identity (`internal/wallet`):**
  * Previously handled by `@solana/web3.js`, this is now powered by `github.com/gagliardetto/solana-go`.
  * Features include automatic local keypair generation/persistence (`wallet.json`), Devnet balance checking, and Airdrop requests.
* **Anonymity Network (`internal/i2p`):**
  * Deep integration with I2P via SAM v3.1 sessions. We have scaffolded the Go structures to eventually connect to local I2P routers for secure, anonymous peer routing.
* **Predictive Streaming (`internal/streaming`):**
  * Supports HTTP Range requests to allow seamless video playback directly from encrypted, decentralized blobs.
  * The system is designed to use "Predictive Readahead" algorithms to eagerly fetch subsequent chunks before the player requests them.
* **Web API & UI (`cmd/bobtorrent`, `internal/api`):**
  * A REST API replaces both the Java Netty and Node.js Express servers.
  * The rich HTML/JS/CSS Web UI from the legacy Node.js reference client has been ported over and is now served directly by the Go backend.

## 3. Design Patterns & Decisions
* **Predictive Resource Allocation:** A major theme of Phase 2/3 is optimizing how nodes pre-fetch and allocate bandwidth for streaming media.
* **Embedded/Standalone Deployment:** The goal of the Go port is to compile down into a single standalone binary that potentially embeds the Web UI assets, drastically simplifying deployment compared to the previous multi-language stack.
* **Strict Documentation Standards:** The project enforces rigorous documentation updates. Every architectural change must be reflected in `VISION.md`, `ROADMAP.md`, `TODO.md`, `IDEAS.md`, `AGENTS.md`, and `CHANGELOG.md`. Model-specific instruction files (like `CLAUDE.md`) must point to a `UNIVERSAL_LLM_INSTRUCTIONS.md`.
* **Version Control:** Version numbers are tightly controlled and prominently maintained in a global `VERSION` file and the `CHANGELOG.md`. The leap to the Go ultra-project triggered the major `3.0.0` version bump.

## 4. Current Status
* The legacy codebase is archived.
* The Go module (`github.com/bobtorrent/bobtorrent`) is initialized.
* Crypto, DHT InfoHash mapping, Solana wallet logic, and the Web UI server are implemented, compiling, and tested.
* **Next Implementation Steps:** Wiring the pure Go BitTorrent engine to the network, finalizing the I2P/SAM client library integration, and completing the streaming readahead HTTP handlers.