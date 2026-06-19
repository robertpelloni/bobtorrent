# BobTorrent / OmniMesh Project Memory

## 1. Architectural Philosophy & Structure
- **Monorepo Evolution:** The project has recently undergone a major unification, flattening the directory structure by migrating all core Go logic out of the `bobtorrent/` submodule into the workspace root (`cmd/`, `internal/`, `pkg/`). The legacy `bobtorrent/` directory was removed to eliminate duplication.
- **The "Omni-Node" / Mega-Messenger Architecture:** The project has pivoted from being solely a BitTorrent tracker/supernode into a decentralized "mega-node" that provides chat (Matrix/Telegram clone), an anonymous blockchain (`bobcoin`), storefronts, and file sharing.
- **Control Plane Pattern:** To avoid a monolithic anti-pattern, the system separates concerns:
  - **Heavy Node (Go Supernode):** Acts as the Local Control Plane. It handles `go-libp2p-pubsub` for a decentralized gossip mesh, Tor routing, DHT queries, and `bobcoin` blockchain submissions.
  - **Light Node (UI/Frontend):** Mobile/Desktop clients handle the UI and local SQLite encryption (via SQLCipher), communicating with the Heavy Node via WebSockets (e.g., `/ws-messenger` bridge) passing blinded Protobuf envelopes (`envelope.proto`).

## 2. Core Technologies & Protocols
- **Storage Layer & Encryption:** Uses an "Obfuscated Storage Protocol" that splits files into AES-256-GCM encrypted chunks (Blobs) and generates manifest entries with detached decryption keys. The project provides WASM compilations (`storage.wasm`, `wasm_exec.js`) so the browser can execute the exact same Go encryption and erasure-coding logic.
- **Advanced Streaming:** The `ReadaheadBuffer` has been optimized to use memory-mapped files (`mmap`) to support efficient O(1) random access and reduce memory pressure during high-throughput tasks like 4K video playback via HTTP Range requests (`io.Seeker`).
- **Distributed Peer Discovery:** Implements `bittorrent-dht` with BEP 44 (Mutable DHT Items) to store and propagate signed market manifests.
- **Anonymity Transports:** Integrates native I2P/SAM datagrams for low-latency anonymous signaling alongside standard clear-net TCP/UDP trackers.
- **Consensus & Economy:** Contains a Block Lattice consensus engine (`lattice-go`) and a Filecoin bridge for long-term storage deals. The economy is heavily intertwined with `bobcoin` on the Solana devnet.

## 3. Identity & Trust Layer
- **Attestation System:** The network verifies user identities to link `bobcoin` public keys to external reputations.
- **Supported Verifiers:**
  - **GitHub (`GitHubVerifier`):** Validates ownership by checking for the user's public key inside a specific raw GitHub Gist URL.
  - **ORCID (`ORCIDVerifier`):** Validates ownership by fetching a user's `pub.orcid.org` or `orcid.org` profile and verifying the public key is present.
  - **URL (`URLVerifier`):** A generic verification mechanism allowing users to host their public key on an arbitrary website.
- **Security Constraints:** The verifiers enforce strict SSRF (Server-Side Request Forgery) protections. Local, loopback, unspecified, and private IP addresses (`127.0.0.0/8`, `10.0.0.0/8`, `192.168.0.0/16`, etc.) are explicitly blocked to prevent malicious users from probing internal networks via the Go Supernode.

## 4. Operational Guardrails & Developer Conventions
- **Continuous Autonomous Execution:** The primary operating directive is to maintain absolute autonomy for as long as possible, executing sequential features, and committing/pushing continuously without pausing for confirmation.
- **Documentation Governance:** A wide array of project state documentation must be strictly updated upon every major feature completion:
  - `CHANGELOG.md` (Detailed feature logs)
  - `VERSION` (Global version string, which must be referenced in the git commit)
  - `HANDOFF.md` (Session summaries for LLM successor context)
  - `ROADMAP.md` (Long-term architectural milestones)
  - `TODO.md` (Granular immediate tasks)
  - `IDEAS.md` (Creative pivoting and refactoring concepts)
- **UI Parity:** Every backend feature must be comprehensively wired to the frontend with robust interactive forms, clear labels, and detailed tooltips.
- **Build Configurations:** The Go runtime often requires `-buildvcs=false` during compilation due to the complex nested submodule state. Furthermore, a `-headless` flag exists for the `supernode-go` daemon to suppress the TUI during automated tests or CI pipelines.

## 5. Upcoming Trajectory (Phase 9+)
- Finalizing the Mobile Messenger frontend client.
- Bridging the `accept_bid` lattice logic with real Bobcoin rewards for long-term seeding incentives.
- Adding typing indicators and topic-specific rate limits to the libp2p gossip mesh.
- Continued synchronization of submodules (`bobcoin`, `element-web`) with the newly unified Go core.