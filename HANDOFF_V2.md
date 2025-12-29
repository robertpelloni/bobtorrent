# Handoff: Megatorrent v2.0.0-dev

## Session Summary
- **Submodule:** Added `bobcoin` submodule (placeholder for Solana/Monero hybrid).
- **Documentation:** Consolidated Agent instructions into `LLM_INSTRUCTIONS.md`. Created `DASHBOARD.md`.
- **Versioning:** Bumped to `2.0.0-dev`.
- **Protocol:** Defined `MSG_DHT_QUERY` (0x09) and `MSG_DHT_RESPONSE` (0x0A) for DHT-over-TCP.
- **Integration:** Added `BobcoinService` stub in Node.js client.

## Current State
- **Node.js Client:** v2.0.0-dev. Supports v5 protocol (fully) and v6 protocol constants (DHT-over-TCP). Includes Bobcoin service stub.
- **C++ Reference:** Matches Node.js protocol v5. Includes WebAPI publishing.
- **Bobcoin:** Initial scaffold (Node.js stub).

## Next Steps
1.  **Bobcoin:** Develop the actual Bobcoin node logic (or integrate the real C++/Rust implementation) in the `bobcoin` submodule.
2.  **DHT-over-TCP:** Implement the logic to encapsulate `bittorrent-dht` packets into `MSG_DHT_QUERY` frames in `lib/secure-transport.js`.
3.  **UI:** Build Qt widgets for Subscription Manager in `qbittorrent`.

## Notes for Next Agent
- **Submodules:** Remember to commit changes inside `bobcoin/` and `qbittorrent/` directories if you edit them directly.
- **Simulation:** `npm test` runs `scripts/simulate_network.js`. It currently fails at the Download step due to NAT/Public IP issues in the sandbox, but the Control Plane (DHT) works.
- **Source of Truth:** `cpp-reference/` is the canonical source for C++ integration. Sync it to `qbittorrent/src/base/` manually or via script.
