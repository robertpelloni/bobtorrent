# Deployment Instructions (Omni-Workspace)

## Current Release
- **Version**: `11.6.0`

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

Provides:
- HTTP consensus endpoints
- websocket live block feed
- market / NFT / proposal / swap query endpoints

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

Minimal browser integration flow:
1. Serve `storage.wasm`
2. Serve `wasm_exec.js`
3. Load `web/storage-wasm-loader.js`
4. Call `createBobtorrentStorageClient()`
5. Use `encrypt`, `encodeErasure`, `decrypt`, `decodeErasure`

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

## Known Deployment Caveats
- `go build ./...` may fail in this repo without `-buildvcs=false` due to VCS/submodule state, so keep the flag in local build commands.
- qBittorrent remote sync is still broken upstream.
- Lattice persistence is not yet durable across restarts.
