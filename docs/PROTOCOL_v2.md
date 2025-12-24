# Megatorrent Protocol v2

## 1. Decentralized Control Plane (Trackerless)

Instead of a central WebSocket tracker, Megatorrent v2 uses the BitTorrent DHT (BEP 44) for all control messages.

### 1.1. Identity & Manifests
*   **Identity:** Ed25519 Keypair.
*   **Manifest:** JSON document describing the collection (same as v1).
*   **Publishing:**
    *   The Author stores the Manifest as a **Mutable Item** in the DHT.
    *   `key`: Author's Public Key.
    *   `salt`: Optional (default empty).
    *   `seq`: Monotonically increasing sequence number (timestamp).
    *   `v`: Bencoded or raw JSON of the Manifest.
    *   `signature`: Ed25519 signature of the packet.
*   **Subscribing:**
    *   Subscribers periodically perform a `dht.get()` on the Author's Public Key.
    *   When a higher `seq` is found, the client downloads and verifies the new Manifest.

### 1.2. Blob Discovery
*   **Blobs:** Encrypted file chunks (same as v1).
*   **Announcement:**
    *   Nodes hosting a blob announce the **Blob Hash (SHA256)** to the DHT.
    *   This is a standard `announce_peer` operation where `info_hash = blob_id`.
*   **Lookup:**
    *   Clients perform `dht.lookup(blob_id)` to find IP:Port of peers.

## 2. Encrypted Data Plane (Anonymity)

To prevent traffic analysis and protect content privacy, the transport layer is fully encrypted.

### 2.1. Handshake (Opportunistic Encryption)
When Peer A connects to Peer B:
1.  **Peer A** generates ephemeral X25519 keypair (`A_pub`, `A_priv`). Sends `A_pub`.
2.  **Peer B** generates ephemeral X25519 keypair (`B_pub`, `B_priv`). Sends `B_pub`.
3.  Both compute shared secret `S = ECDH(Priv, RemotePub)`.
4.  Derive Session Keys:
    *   `Tx_Key_A = Rx_Key_B = BLAKE2b(S || 'C')`
    *   `Rx_Key_A = Tx_Key_B = BLAKE2b(S || 'S')`

### 2.2. Framing
All subsequent messages are framed:
*   `[Length (2 bytes)] [Encrypted Payload (N + 16 bytes)]`
*   Payload is encrypted with `ChaCha20-Poly1305`.
*   Nonce increments per packet.

### 2.3. Data Transfer
*   **Request:** `GET <BlobID>` (Encrypted)
*   **Response:** Raw Blob Data (Encrypted chunks)

## 3. Redundancy
*   Redundancy is achieved via the DHT's natural replication of announcements.
*   The more peers subscribe to a channel, the more replicas of the blobs exist in the swarm.

## 4. Security Considerations & Best Practices

### 4.1. Anonymity & DHT Leakage
While the Data Plane is encrypted and can be routed via Tor (SOCKS5), the **DHT traffic uses UDP**, which Tor does not support natively.
*   **Risk:** Performing DHT lookups exposes your IP address to the DHT swarm nodes.
*   **Mitigation (Current):** Use a **Gateway Node** or VPN to bootstrap into the DHT if anonymity is critical.
*   **Mitigation (Future):** Implement DHT-over-TCP or route DHT traffic through a specialized mixnet (e.g., I2P).

### 4.2. Sybil Resistance
*   The protocol relies on **Ed25519 Identity Keys**.
*   Spammers cannot forge updates for a channel they do not own.
*   However, a Sybil attacker could flood the DHT with fake peers for a blob. The Encrypted Handshake mitigates this by allowing clients to quickly disconnect from peers that fail the handshake, but traffic analysis of connection attempts is still possible.

### 4.3. Traffic Analysis
*   **Padding:** All blobs should ideally be padded to a fixed size (e.g., 2MB) to prevent fingerprinting files based on chunk sizes. The current implementation supports arbitrary blob sizes, but clients SHOULD enforce fixed-size chunking during ingestion.
