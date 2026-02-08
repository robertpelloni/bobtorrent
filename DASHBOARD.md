# Project Dashboard

## 📁 Directory Structure

*   **`reference-client/`**: The Node.js Reference Implementation.
    *   `web-server.js`: Web UI Backend.
    *   `web-ui/`: Single Page Application (HTML/JS).
    *   `lib/`: Core protocol logic (Storage, Manifests, Crypto).
*   **`supernode-java/`**: The High-Performance Java Supernode (Gradle project).
    *   Implements the MuxEngine (AES-GCM storage layer) and high-concurrency network stack.
*   **`qbittorrent/`**: Official qBittorrent submodule (C++).
    *   Contains the core BitTorrent client logic.
*   **`cpp-reference/`**: Canonical C++ Integration Stubs & Patches.
    *   `megatorrent/`: C++ implementation of Megatorrent protocol.
    *   `qbittorrent-patches/`: Patches to integrate Megatorrent into qBittorrent.
*   **`docs/`**: Project Documentation.
    *   `UNIVERSAL_LLM_INSTRUCTIONS.md`: Directives for all AI agents.
*   **`verification/`**: Test scripts and verification artifacts.

## 📊 Status

*   **Version**: 1.6.0 (Streaming & Polish)
*   **Core Protocol**: v1.0 (Stable)
*   **Reference Client**: Feature Complete (Web UI + Streaming)
*   **Supernode (Java)**: Active Development
*   **qBittorrent Integration**: Prototype/Stubs

## 🔗 Submodules

*   `qbittorrent` (tracked at specific commit)
*   `supernode-java` (monorepo component)

## 🛠 Build Status

*   **Node.js**: Passing (CI)
*   **Java**: Gradle Build (Manual)
*   **C++**: CMake Configuration (Manual)
