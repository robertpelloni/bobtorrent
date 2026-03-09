# Project Memory

## 🧠 Architectural Insights
- **Bobtorrent Tracker**: Acts as the central nervous system, handling multi-protocol peer exchange (HTTP, UDP, WebSocket). The manifest validation uses `libsodium` (Ed25519 signatures, optional XSalsa20 encryption).
- **Supernode Java**: Operates as the heavy-lifting storage core. Erasure coding (Reed-Solomon 4+2 and 6+2) and AES-GCM are production-ready.
- **Submodules**: 
  - `qbittorrent` is integrated for C++ native tracking and DHT operations.
  - `bobcoin` acts as the decentralized ledger and ZK proof layer for storage rewards.

## ⚠️ Known Hazards & Fixes
- **Git Submodules**: Inner submodules (like `bobcoin/research/forest`) can cause recursive update failures if upstream URLs change. Fix entails updating to the latest tracked commit on their respective main branches.
- **Manifest.js Merge Conflicts**: Trackers had divergent `validateManifest` algorithms in `origin/master` vs feature branches. The converged solution uses `fast-json-stable-stringify` before validation, while supporting `secretbox` encryption for sensitive payloads.

## 💡 Future Implementation Notes
- **WebTransport (QUIC)**: Next major refactor for the tracker network should integrate HTTP/3 WebTransport for zero-latency peer operations within the browser.
- **Multi-node Cluster Management**: Java Supernode requires a proper peer-exchange protocol and distributed Kademlia manifest sync to hit 1.0.0. Node CLI configuration utility is a low-hanging fruit to implement next.
