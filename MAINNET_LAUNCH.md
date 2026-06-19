# OmniMesh (BobTorrent) Mainnet Launch Checklist

This document tracks the DevOps, DNS, and Infrastructure readiness required for the Global Decentralized Storage Network launch.

## 1. Network Infrastructure & Bootstrapping
- [ ] **Provision Bootstrapper Nodes**: Deploy at least 3 geographically distributed, high-availability Go supernodes (`supernode-go`) with static IP addresses.
- [ ] **Provision Tracker Nodes**: Deploy standalone BitTorrent trackers and/or WebTorrent trackers to supplement the DHT.
- [ ] **Hardcode Bootstrap IPs**: Update `pkg/torrent` and `internal/transport/dht.go` to default to the newly provisioned static bootstrapper IPs.
- [ ] **Federated Node Discovery**: Ensure the dynamic decentralized node list queried from the Bobcoin Solana smart contract is active and reliable.

## 2. DNS & Networking
- [ ] **Domain Configuration**: Configure `omni-mesh.net` (or equivalent) DNS records.
- [ ] **TLS/SSL**: Provision Let's Encrypt certificates for the public-facing WebUI and secure WebSockets (`wss://`).
- [ ] **DNS Seeds**: Setup round-robin DNS records (e.g., `seed.omni-mesh.net`) pointing to the bootstrapper nodes.

## 3. Economy & Incentives (Bobcoin)
- [ ] **Solana Mainnet Deployment**: Migrate the Bobcoin program from Solana Devnet to Mainnet Beta.
- [ ] **Initial Airdrop Campaign**: Execute a genesis airdrop to early contributors to bootstrap the storage market economy.
- [ ] **Seeding Incentives**: Verify that the `accept_bid` economy bridge is correctly crediting real Bobcoin rewards to mainnet wallets.
- [ ] **Filecoin Bridge Funding**: Ensure the `f1bobtorrentnode` Filecoin wallet is funded for archival fallback deals.

## 4. Security & Performance
- [ ] **Performance Profiling**: Complete a systematic profiling pass (`pprof`) under high simulated load (e.g., 10k concurrent simulated swarms).
- [ ] **Rate Limiting Verification**: Verify that the GossipSub token bucket rate limits (5 req/sec, burst 10) are effective against DDoS attempts.
- [ ] **Penetration Testing**: Audit the `/ingest` endpoints and WebSockets for directory traversal and SSRF vulnerabilities.

## 5. Client & Distribution
- [ ] **Binary Releases**: Compile reproducible, stripped binaries for Windows, macOS, and Linux.
- [ ] **WASM Deployment**: Ensure `storage.wasm` and `wasm_exec.js` are hosted on a fast, global CDN.
- [ ] **Mobile Messenger**: Publish the Light Node React Native/Flutter client to the App Store and Google Play.
- [ ] **Documentation**: Publish the final `MANUAL.md` and user-facing API documentation.

## 6. Launch Sequence
1. Deploy Bobcoin Mainnet Contract.
2. Spin up Bootstrapper and Tracker nodes.
3. Verify DNS Seeds.
4. Execute Genesis Airdrop.
5. Release binaries to the public.
