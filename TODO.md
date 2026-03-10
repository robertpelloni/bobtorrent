# TODO (Autonomous Development Backlog)

## ✅ Completed (Reanalyzed State)
- **Merge Hell Resolved**: Re-aligned `lib/manifest.js` to correctly utilize `fast-json-stable-stringify` along with encryption payloads (XSalsa20).
- **Submodule Stabilization**: Corrected `bobcoin` detached references.
- **Tracker Integration**: Merged multiple UI and reference implementation branches flawlessly into core.
- **Documentation Overhaul**: Created `UNIVERSAL_LLM_INSTRUCTIONS.md`, generated DASHBOARD, MEMORY, DEPLOY, VISION, ROADMAP.
- **Storage Performance**: Concurrent erasure coding and parallel manifest validation are checked and integrated.

## ✅ v0.4.0 Features — COMPLETED

### 1. Enhanced Transport Protocol Implementations
- [x] Tor v3: MultiplexedCircuitPool — round-robin, failover, per-circuit rotation.
- [x] IPFS: CARExtractor — CAR v1 archive parsing with CID extraction.
- [x] Hyphanet: SplitfileRecoveryOptions — retry escalation with priority boosting.

### 2. Consensus-Verified Tracker Ledger
- [x] TrackerLedger — records peer violations as Solana memo txns, consensus-based banning.

## 🚀 Active Feature Backlog (v0.5.0+)

### 3. Code & Quality Improvements
- [ ] Convert `package.json` testing to use integrated modern runners instead of unmaintained endpoints.
- [ ] Ensure the Java ER test assertions perfectly mimic production byte layouts.
- [ ] Increase commenting density on all cryptographically sensitive components (`MuxEngine.java`, `Manifest.java`, `BobcoinBridge.java`).
