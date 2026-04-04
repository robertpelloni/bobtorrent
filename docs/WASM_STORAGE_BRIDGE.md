# Go Storage WASM Bridge

## Purpose
The Bobtorrent Go port now exports the storage kernel to WebAssembly so browser clients can use the exact same:
- ChaCha20-Poly1305 authenticated encryption
- Reed-Solomon erasure coding
- deterministic shard reconstruction flow

This eliminates drift between backend and frontend storage logic and enables a **zero-trust upload path** where encryption and sharding occur entirely in the browser before content ever leaves the client machine.

## Artifacts
The build pipeline now produces:
- `build/storage.wasm` — compiled Go storage kernel
- `build/wasm_exec.js` — official Go runtime bridge required by browser WASM execution
- `web/storage-wasm-loader.js` — ergonomic JS wrapper around the Go exports

## Exported WASM Functions
`cmd/wasm/main.go` exposes the following globals:
- `bobEncrypt(Uint8Array)`
- `bobDecrypt(Uint8Array, keyHex, nonceHex)`
- `bobEncodeErasure(Uint8Array)`
- `bobDecodeErasure(Array<Uint8Array|null>)`

## Recommended Frontend Integration
In a browser or React app, load the Go runtime and instantiate `storage.wasm` through the provided loader:

```js
import { createBobtorrentStorageClient } from '/web/storage-wasm-loader.js';

const storage = await createBobtorrentStorageClient({
  wasmExecUrl: '/wasm_exec.js',
  wasmBinaryUrl: '/storage.wasm'
});

const encrypted = await storage.encrypt(fileBytes);
const shards = await storage.encodeErasure(encrypted.blob);
```

## Suggested Upload Flow
1. Read a file into `Uint8Array`
2. Call `encrypt()` to produce `{ blob, key, nonce }`
3. Call `encodeErasure(blob)` to produce data+parity shards
4. Upload shards independently to storage peers / supernodes
5. Store metadata manifest containing:
   - shard IDs
   - encryption key
   - nonce
   - original file size
   - magnet / content identifier

## Compatibility Notes
### Browser Runtime
Go WASM requires `wasm_exec.js`. The updated `build.bat` copies this file automatically from the active Go toolchain.

### Current Lattice Compatibility Layer
The Go lattice server currently accepts both:
- raw block JSON
- `{ block: ... }` wrapped submissions

This preserves compatibility with both the Go supernode and the existing bobcoin frontend.

### Legacy Frontend Blocks
Some legacy frontend pages still omit explicit `height` and `staked_balance` fields. The lattice includes a temporary compatibility shim that infers these values from the frontier when possible.

## Security Notes
- The browser-side encryption path is now aligned with the Go backend implementation.
- Private encryption material is still returned to the caller, so the caller must store it securely.
- Future work should derive encryption keys from wallet-bound secrets or a per-upload KDF rather than returning raw random keys directly to the UI layer.

## Next Recommended Integration Tasks
1. Copy or import `web/storage-wasm-loader.js` into `bobcoin/frontend`
2. Add an upload page that uses `storage.wasm`
3. Persist encrypted shard manifests on the lattice as market/NFT payload metadata
4. Replace mock Filecoin archival with real Lotus RPC ingestion
