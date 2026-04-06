# Deployment Instructions (Omni-Workspace)

## Current Release
- **Version**: `11.43.0`

## 1. Build All Go Artifacts
Use the Windows build helper:

```bat
build.bat
```

Or run manually:

```bash
go build -buildvcs=false -o build/dht-proxy cmd/dht-proxy/main.go
go build -buildvcs=false -o build/supernode-go cmd/supernode-go/main.go
go build -buildvcs=false -o build/lattice-go cmd/lattice-go/main.go
GOOS=js GOARCH=wasm go build -buildvcs=false -o build/storage.wasm cmd/wasm/main.go
```

Also ensure the Go runtime bridge is present beside the WASM binary:
- `build/wasm_exec.js`

## 2. Run the Go Lattice Node
```bash
./build/lattice-go
```
Default port: `4000`

Persistence:
- default SQLite path: `data/lattice/lattice.db`
- override with env: `BOBTORRENT_LATTICE_DB=/custom/path/lattice.db`
- snapshot cadence env: `BOBTORRENT_LATTICE_SNAPSHOT_INTERVAL=25` (set `0` to disable automatic snapshot creation)
- snapshot retention env: `BOBTORRENT_LATTICE_SNAPSHOT_RETENTION=3`
- materialized snapshots are created automatically according to the configured cadence to accelerate cold boot

Provides:
- HTTP consensus endpoints
- websocket live block feed
- market / NFT / proposal / swap / manifest-anchor query endpoints
- ordered block catch-up endpoint: `GET /blocks`
- bootstrap summary endpoint: `GET /bootstrap`
- bootstrap sync endpoint: `POST /bootstrap`
- peer list / health endpoints: `GET /peers`, `POST /peers`
- peer health telemetry is surfaced through `/status`, `/bootstrap`, and `/peers`
- peer sync now supports cooldown-skipped responses and explicit divergence suspicion when a remote peer lacks the local ordered-history cursor
- persistence verification endpoint: `GET /persistence/verify`
- persistence repair endpoint: `POST /persistence/repair`
- persistence export endpoint: `GET /persistence/export`
- persistence backup endpoint: `POST /persistence/backup`
- persistence secure backup bundle endpoint: `POST /persistence/backup-bundle`
- persistence import endpoint: `POST /persistence/import`
- persistence restore endpoint: `POST /persistence/restore`
- persistence secure restore bundle endpoint: `POST /persistence/restore-bundle`

## 3. Run the Go Supernode
```bash
./build/supernode-go
```
Default services:
- HTTP tracker / stats on `:8000`
- UDP tracker on `:6881`
- DHT node on `:6882`
- TUI in the foreground terminal session

The supernode expects the lattice node to be available at `http://localhost:4000`.

Optional FHE helper configuration:
- `BOBTORRENT_NODE_BIN=node` to override the Node binary used for the specialized SEAL helper
- `BOBTORRENT_FHE_ORACLE_HELPER=cmd/supernode-go/fhe_oracle_helper.mjs` to override the helper script path

Optional Filecoin/Lotus bridge configuration:
- `BOBTORRENT_FILECOIN_RPC_URL=http://127.0.0.1:1234/rpc/v0`
- `BOBTORRENT_FILECOIN_AUTH_TOKEN=...`
- `BOBTORRENT_FILECOIN_WALLET=f1...`
- `BOBTORRENT_FILECOIN_MINER=f0...`
- `BOBTORRENT_FILECOIN_RECORDS=data/filecoin/deals.json`

Additional frontend-facing endpoints now provided by `supernode-go`:
- `GET /status` (now includes signaling + Filecoin bridge telemetry)
- `GET /stats` (now includes signaling + Filecoin bridge telemetry)
- `GET /filecoin/status`
- `GET /filecoin/deals`
- `GET /bankroll`
- `GET /transactions`
- `POST /mint`
- `POST /burn`
- `POST /fhe-oracle`
- `POST /submit-proof`
- `POST /add-torrent`
- `POST /remove-torrent`
- `POST /upload`
- `GET /spora/:challenge`
- `POST /upload-shard`
- `POST /publish-manifest`
- `GET /manifests/:id`
- `GET /shards/:hash`
- `GET /storage.wasm`
- `GET /wasm_exec.js`

## 4. Run the DHT Proxy
```bash
./build/dht-proxy
```
Optional MaxMind database:
- `GeoLite2-City.mmdb`

Without the database, GeoIP enrichment degrades gracefully.

## 5. Browser / Frontend WASM Integration
The build now produces:
- `build/storage.wasm`
- `build/wasm_exec.js`

Recommended loader:
- `web/storage-wasm-loader.js`

The Go supernode now serves these artifacts directly, so a browser client can target:
- `http://localhost:8000/storage.wasm`
- `http://localhost:8000/wasm_exec.js`

Minimal browser integration flow:
1. Run `build.bat`
2. Start `./build/supernode-go`
3. Load the Go runtime from the supernode origin
4. Call `createBobtorrentStorageClient()`
5. Use `encrypt`, `encodeErasure`, `decrypt`, `decodeErasure`
6. Upload prepared shards via `POST /upload-shard`
7. Publish the final manifest via `POST /publish-manifest`
8. Restore the file later by loading the manifest, downloading shard URLs, reconstructing ciphertext, and decrypting in the browser via the Bobcoin workbench

See:
- `docs/WASM_STORAGE_BRIDGE.md`

## 6. Legacy Node.js Stack (Still Available)
### Game Server
```bash
cd bobcoin/game-server
npm install
npm start
```

### Frontend
```bash
cd bobcoin/frontend
npm install
npm run dev
```

Frontend runtime targeting notes:
- HTTP compatibility traffic defaults to `VITE_GAME_HTTP_URL || VITE_SUPERNODE_URL || http://localhost:8000`
- WebRTC signaling defaults to `VITE_GAME_SIGNALING_URL || VITE_SUPERNODE_URL || http://localhost:8000`
- Operators can still point signaling at a legacy websocket service explicitly via `VITE_GAME_SIGNALING_URL`

## Known Deployment Caveats
- `go build ./...` may fail in this repo without `-buildvcs=false` due to VCS/submodule state, so keep the flag in local build commands.
- qBittorrent remote sync is still broken upstream.
- Secure backup bundle restore still follows the project safety boundary: it creates a fresh verified database for the next boot/manual recovery rather than mutating the running node’s active persistence store.
