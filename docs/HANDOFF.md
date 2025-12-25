# Megatorrent C++ Integration Status

This repository contains the C++ stubs and WebUI logic required to integrate the Megatorrent Protocol v2 (Decentralized + Encrypted) into qBittorrent.

## 📁 Preserved Assets
Since the `qbittorrent` submodule points to an external repository, changes made inside it are not persisted in this repo's git history unless moved out.
We have preserved the integration files here:

*   **C++ Core:** `cpp-reference/megatorrent/`
*   **WebUI Scripts:** `webui-reference/`

## 🛠 Integration Steps for C++ Developer

### 1. Copy C++ Source
Copy the contents of `cpp-reference/megatorrent/` to `qbittorrent/src/base/megatorrent/`.

### 2. Update CMakeLists
Modify `qbittorrent/src/base/CMakeLists.txt` to include the new files (as done in the reference implementation steps).

### 3. Copy WebUI Assets
*   Copy `webui-reference/megatorrent.js` to `qbittorrent/src/webui/www/private/scripts/`.
*   Apply the changes in `webui-reference/index.html.patched` to `qbittorrent/src/webui/www/private/index.html` (Add script tag + Tab link).

### 4. Link Dependencies
Ensure `libtorrent` and `OpenSSL` are correctly linked. The provided stubs use `Crypto::` namespace placeholders that must be backed by real OpenSSL EVP calls.

---

## 🧩 Component Details

### `dht_client.h/cpp` (Decentralized Control)
*   **Purpose:** Replaces the deprecated WebSocket Tracker.
*   **Functionality:** Handles `putManifest` (BEP 44), `getManifest`, `announceBlob`, and `findBlobPeers`.

### `secure_socket.h/cpp` (Encrypted Transport)
*   **Purpose:** Implements the custom Encrypted Transport Protocol (v5).
*   **Functionality:** Ephemeral ECDH Handshake, ChaCha20-Poly1305 Encryption, Binary Packet Parsing (`MSG_HELLO`, `MSG_DATA`, etc.).

### `manifest.h/cpp` (Data Structure)
*   **Purpose:** Parses and validates the JSON Manifest format and Ed25519 signatures.
