# DHT Proxy Utility

## Concept
Based on [Janhouse's DHT Proxy](https://www.janhouse.lv/blog/it/dht-proxy-hiding-ip-from-bittorrent-dht-trackers/), this utility is being added to the Go port of Bobtorrent. It acts as a privacy shield between a user's BitTorrent client and the public DHT/tracker network.

## How It Works
When a user adds a torrent to their client, instead of the client querying the public DHT and trackers (which exposes the user's IP), the client queries the DHT Proxy.
1. The user submits a magnet link or `.torrent` file to the proxy.
2. The proxy rewrites the announce URL to point to its own private `/api/announce` endpoint and returns the modified `.torrent`.
3. In the background, the proxy crawls the public DHT and trackers, discovering peers.
4. The proxy enriches these peers with GeoIP data and stores them in its database.
5. When the user's client announces to the proxy, the proxy serves the closest discovered peers.
6. **Result**: The proxy's IP is exposed to the public DHT/trackers, not the user's IP.

## Go Implementation Architecture (`internal/dhtproxy`)

### Components
1.  **DHT Crawler**: Uses a Go BitTorrent DHT library (e.g., `github.com/nictuku/dht` or `github.com/anacrolix/dht`) to announce to the DHT and collect peers.
2.  **Tracker Announcer**: A background worker that sends HTTP/UDP announces to public trackers, records peers, and immediately sends a `stopped` event.
3.  **Peer Database**: SQLite (via `modernc.org/sqlite` or `mattn/go-sqlite3`) to store swarms, peers, and last-seen timestamps.
4.  **GeoIP Service**: Integration with MaxMind GeoLite2 to calculate distances.
5.  **API Server**:
    *   `/api/torrent/add`: Accepts magnets/torrents and returns rewritten `.torrent` files.
    *   `/api/announce`: The private tracker endpoint the user's client connects to.
    *   Admin UI/API for managing tracked torrents.

### Integration with Bobtorrent
The DHT Proxy will run as a standalone utility (`cmd/dht-proxy`) but will share the core tracking and DHT logic from `internal/tracker` and `internal/transport`. It aligns perfectly with Bobtorrent's "Privacy-First" (Tor, I2P, Mixnet) vision.
