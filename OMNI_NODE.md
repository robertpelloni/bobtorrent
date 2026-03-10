# The Omni-Node: Supernetwork Server-Client

The Bobtorrent **Omni-Node** is the ultimate decentralized deployment stack. It runs a Bobtorrent Supernode alongside every major privacy and storage network, acting as a high-bandwidth bridge to host the "Library of Linux ISOs" and participate in cryptoeconomic consensus.

## Network Stack

The Omni-Node runs the following services via `docker-compose.supernode.yml`:

| Service | Port | Role |
|---------|------|------|
| **Bobtorrent Java Supernode** | `9090` | Core P2P engine, erasure coding, AES-GCM encryption |
| **Bobtorrent JS Tracker** | `8000` | WebTransport/UDP/HTTP/WS swarm tracking |
| **Tor Daemon** | `9050` (SOCKS) | Clearnet evasion and `.onion` routing |
| **I2P Router** | `4444` (HTTP) | Garlic routing for anonymous blobs |
| **IPFS Kubo** | `5001` (API) | CAR archive bridging and global pinning |
| **ZeroNet** | `43110` (Web) | Decentralized websites |
| **Hyphanet (Freenet)** | `8888` (FCP) | Plausible deniability splitfile storage |
| **Filecoin (Lotus)** | `1234` | Storage deals and FIL incentives |
| **Monero** | `18081` (RPC) | Private tracking transactions |
| **Bobcoin** | `9944` | Proof-of-Useful-Stake validation |

## Quickstart

1. Clone the repository to a high-bandwidth server (e.g., 10Gbps seedbox).
2. Create the auto-ingest directory: `mkdir -p /mnt/linux-isos`
3. Launch the stack:
   ```bash
   docker-compose -f docker-compose.supernode.yml up -d
   ```
4. Drop any `.iso` file into `/mnt/linux-isos`. The `ISODaemon` will automatically:
   - Chunk, encrypt (AES-GCM), and erasure-code (Reed-Solomon 4+2) the file.
   - Publish the manifest to the Kademlia DHT.
   - Announce the file across Tor, I2P, ZeroNet, and Hyphanet.
   - Pin the extracted blocks to IPFS.
   - Submit a Filecoin storage deal.
   - Submit a Bobcoin PoUS (Proof of Useful Stake) verification.

## Architecture: The ISODaemon

The `ISODaemon.java` module continuously monitors the mounted volume. When an ISO is detected:
1. It is ingested into `SupernodeStorage`.
2. A unique manifest ID is generated.
3. The `SupernodeNetwork` federates this ID across all configured transports via the `TransportManager`.
4. The `BobcoinBridge` and `TrackerLedger` submit cryptographic proofs (via `ProofOfSeeding`) to the Solana/EVM blockchains to earn rewards.

## Resource Requirements

Running the full Omni-Node requires significant hardware:
- **CPU**: 8+ Cores (for simultaneous Tor/I2P routing + Erasure Coding)
- **RAM**: 32GB+ (IPFS Kubo and Lotus are memory-intensive)
- **Storage**: 2TB+ NVMe SSD (for the Library of Linux ISOs + Blockchain state)
- **Bandwidth**: 1Gbps minimum (10Gbps recommended for Supernode Tier 1)
