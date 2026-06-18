# BobTorrent TODO

## Immediate Tasks
- [x] Implement BEP 44 (Mutable DHT Items) in `dht.Engine` to store and retrieve signed manifests (target calculation logic stubbed).
- [x] Wire `/api/publish` to write the signed manifest using BEP 44 `Put`.
- [x] Wire `/api/subscribe` to continuously poll the DHT using BEP 44 `Get` for new manifests.
- [x] Implement Identity generation API (`/api/key/generate`)
- [x] Implement Publisher API (`/api/publish`)
- [x] Implement Subscriber API (`/api/subscribe`, `/api/subscriptions`)
- [x] Implement Identity Generation endpoint (`GET /api/key/generate`) using ed25519 or equivalent.
- [x] Implement `/api/publish` to sign and propagate manifests via Tracker/DHT.
- [x] Implement `/api/subscribe` and `/api/subscriptions` to track channels.
- [x] Implement `/api/blobs` endpoint for UI blob storage overview.
- [x] Integrate Pub/Sub tracking layer into the Go daemon to listen for newly published manifests.
- [x] Ensure all new API endpoints are fully integrated with the embedded Web UI.
- [x] Update the UI to fully reflect active Pub/Sub status and real-time incoming manifests

# BobTorrent TODO


## Completed Tasks
- [x] Phase 1: Go module initialization, legacy code archiving.
- [x] Phase 2: DHT InfoHash mapping and Solana wallet porting.
- [x] Phase 3: Manifest parsing, Detached AES-256-GCM encryption/decryption, Readahead streaming logic.
- [x] Phase 4: Full HTTP Range request processing via `io.Seeker` and API Server initialization.


## Ongoing Documentation Tasks
- [x] Continue updating `IDEAS.md` with potential improvements.
- [x] Log structural findings in `MEMORY.md`.

## Important Compatibility / Cleanup Tasks
- [x] **Remove temporary legacy block shim** once bobcoin frontend includes explicit `height` and `staked_balance`
- [x] **Unify block hashing rules** between Go and browser-side block construction
- [x] **Add tests** for consensus transitions:
  - send/receive
  - NFT mint/transfer
  - stake/unstake
  - swaps
  - proposals/votes
- [x] **Add integration tests** for websocket live feed and wrapped-vs-raw block submission formats

## Strategic Backlog
- [ ] **Mega-Messenger Integration**:
  - [x] Initialize `libp2p` host in `internal/transport/messenger.go`.
  - [x] Implement `GossipSub` topic management for rooms/channels.
  - [x] Add `WS_GOSSIP` handler to `supernode-go` for UI interaction.
  - [x] Scaffold `element-web` bridge logic to proxy Matrix-like events to libp2p.
  - [x] Implement event persistence for missed gossip.
- [ ] **I2P Native Datagrams**:
  - [x] Research `github.com/eyedeekay/sam3` datagram support.
  - [x] Implement `I2PDatagramTransport` in `internal/transport/i2p_datagram.go`.
  - [x] Wire I2P datagrams to `supernode-go` with PING/PONG.
- [x] **Go Supernode WebUI integration**: Ported `reference-client/web-ui` and implemented Go backend APIs for it.
- [x] **Durable market manifests + shard metadata registry**: Upgraded the publication registry with a SQLite-backed index and added `GET /assets` for durable asset discovery.
- [x] **Identity/Attestation verification**: Implemented a Go-native verifier service and integrated real-time verification badges into the Bobcoin Vault UI.
- [x] **Real identity verifiers**: Replaced the `MockVerifier` with a production-ready `GitHubVerifier` that validates Gist attestations via the GitHub API, and implemented `ORCIDVerifier` and `URLVerifier`.
- [x] **Seeding Incentives**: Bridged the `accept_bid` lattice logic with real Bobcoin rewards for long-term seeding.
- [ ] **Game engine asset ingestion path**
- [ ] **Global decentralized storage network launch**
- [ ] **Investigate unreachable `qbittorrent` remote**

