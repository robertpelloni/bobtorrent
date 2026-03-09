# TODO (Autonomous Development Backlog)

## ✅ Completed (Reanalyzed State)
- **Merge Hell Resolved**: Re-aligned `lib/manifest.js` to correctly utilize `fast-json-stable-stringify` along with encryption payloads (XSalsa20).
- **Submodule Stabilization**: Corrected `bobcoin` detached references.
- **Tracker Integration**: Merged multiple UI and reference implementation branches flawlessly into core.
- **Documentation Overhaul**: Created `UNIVERSAL_LLM_INSTRUCTIONS.md`, generated DASHBOARD, MEMORY, DEPLOY, VISION, ROADMAP.
- **Storage Performance**: Concurrent erasure coding and parallel manifest validation are checked and integrated.

## 🚀 Active Feature Backlog (Pending Implementation)

### 1. Supernode CLI Configuration & Diagnostics (Next Target)
*Create a highly robust Command Line Interface (CLI) component for the Supernode:*
- [ ] Implement `io.supernode.cli.NodeCLI` taking command line arguments utilizing a robust parsing library or manual elegant parsing.
- [ ] Add `status` command to output current peer counts, Kademlia DHT state, and memory metrics.
- [ ] Add `manifest-inspect <cid>` command to parse, decrypt (if key provided), and print a verified JSON manifest.
- [ ] Make sure it runs as an entry point script / shell wrapper.

### 2. Distributed Manifest Synchronization
*Implement the swarm state sync capability:*
- [ ] Extend Kademlia routing tables (`DHTDiscovery`) to broadcast new manifests (`Manifest.java`) to at least 4 nearest neighbors automatically.
- [ ] Add deduplication checks to prevent broadcast storms.

### 3. WebTransport Tracker Support (Node.js)
*Upgrade the `bobtorrent` Javascript tracker:*
- [ ] Add a `lib/server/parse-webtransport.js` handler for incoming QUIC connections.
- [ ] Bind robust fallback logic ensuring peers on WS can share swarms with peers on WebTransport.

### 4. Code & Quality Improvements
- [ ] Convert `package.json` testing to use integrated modern runners instead of unmaintained endpoints.
- [ ] Ensure the Java ER test assertions perfectly mimic production byte layouts.
- [ ] Increase commenting density on all cryptographically sensitive components (`MuxEngine.java`, `Manifest.java`, `BobcoinBridge.java`).
