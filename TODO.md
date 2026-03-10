# TODO (Autonomous Development Backlog)

## ✅ Completed (Reanalyzed State)
- **Merge Hell Resolved**: Re-aligned `lib/manifest.js` to correctly utilize `fast-json-stable-stringify` along with encryption payloads (XSalsa20).
- **Submodule Stabilization**: Corrected `bobcoin` detached references.
- **Tracker Integration**: Merged multiple UI and reference implementation branches flawlessly into core.
- **Documentation Overhaul**: Created `UNIVERSAL_LLM_INSTRUCTIONS.md`, generated DASHBOARD, MEMORY, DEPLOY, VISION, ROADMAP.
- **Storage Performance**: Concurrent erasure coding and parallel manifest validation are checked and integrated.

## 🚀 Active Feature Backlog (Pending Implementation)

### 1. Enhanced Transport Protocol Implementations (v0.4.0)
- [ ] Tor v3: Improve circuit rotation and stream multiplexing.
- [ ] IPFS: Add full CAR (Content Addressable Archive) payload extraction.
- [ ] Hyphanet: Enhanced splitfile recovery.

### 2. Consensus-Verified Tracker Ledger (v0.4.0)
- [ ] Hook tracker peer states into a Solana/Stone.Ledger memo bridge to ban bad actors.

### 3. Code & Quality Improvements
- [ ] Convert `package.json` testing to use integrated modern runners instead of unmaintained endpoints.
- [ ] Ensure the Java ER test assertions perfectly mimic production byte layouts.
- [ ] Increase commenting density on all cryptographically sensitive components (`MuxEngine.java`, `Manifest.java`, `BobcoinBridge.java`).
