# Bobtorrent Omni-Workspace Handoff (v11.8.0)

## Session Objective
Advance from the `v11.7.0` WASM/frontend wiring milestone into a real publish path: accept browser-prepared shards in the Go supernode, persist manifests durably on disk, wire the Bobcoin frontend to use those endpoints, validate both the Go workspace and React build, and push everything without stopping any active processes.

## What Was Implemented

### 1. Real Publication Registry in Go
New package:
- `internal/publish/registry.go`
- `internal/publish/registry_test.go`

Capabilities added:
- content-addressed shard storage using SHA-256
- deterministic manifest persistence on disk
- manifest ID derivation from `encryption.ciphertextHash` when present
- automatic shard URL injection into manifest entries
- local durable storage under `data/published/`
- idempotent shard uploads

Why this matters:
- the Bobcoin WASM workbench is no longer just a local cryptographic preview
- the Go supernode now has a concrete persistence layer for uploaded assets
- this creates the missing bridge between browser-side preprocessing and actual stored Bobtorrent artifacts

Validation:
- `go test ./internal/publish -buildvcs=false` ✅

### 2. Go Supernode Publication + Browser Access API
File:
- `cmd/supernode-go/main.go`

Added endpoints:
- `POST /upload-shard`
- `POST /publish-manifest`
- `GET /manifests/:id`
- `GET /shards/:hash`
- existing frontend-compatible endpoints retained:
  - `GET /stats`
  - `POST /add-torrent`
  - `POST /remove-torrent`
  - `GET /storage.wasm`
  - `GET /wasm_exec.js`

Additional hardening:
- added permissive CORS wrapper for the browser-facing endpoints
- publication registry initialized at startup
- `supernode-go` now acts as both:
  - torrent/storage operator
  - manifest/shard publication host

### 3. Bobcoin Frontend: Publish to Go Supernode
Submodule: `bobcoin`
Pushed latest submodule commit:
- `01499c4` — `feat(frontend): publish wasm-prepared storage to go supernode (v8.8.0)`

Updated files inside Bobcoin:
- `frontend/src/api.js`
  - added `uploadStorageShard()`
  - added `publishStorageManifest()`
- `frontend/src/components/StorageWasmWorkbench.jsx`
  - retains browser-side Go encryption + erasure coding
  - now uploads prepared shards sequentially to `supernode-go`
  - now publishes the final manifest to the supernode registry
  - shows publication status, locator, and manifest URL
- `frontend/src/lib/storageWasm.js`
  - already defaulted WASM runtime artifact URLs to the Go supernode origin

Net result:
- browser-side preprocessing → actual shard upload → manifest publication is now a real flow

### 4. Bobcoin Documentation / Versioning
Updated inside submodule:
- `VERSION.md` → `8.8.0`
- `CHANGELOG.md`
- `TODO.md`
- `HANDOFF.md`

### 5. Root Documentation / Versioning
Updated root docs to reflect the new state:
- `VERSION` → `11.8.0`
- `CHANGELOG.md`
- `ROADMAP.md`
- `TODO.md`
- `DASHBOARD.md`
- `DEPLOY.md`
- `MEMORY.md`
- `HANDOFF.md`

The backlog has been moved forward accordingly:
- "wire WASM frontend" is done
- "publish to supernode" is done
- next real items are now:
  - lattice anchoring
  - retrieval UX
  - durable consensus persistence

## Validation Performed
### Root repo
Executed successfully:
- `go test ./internal/publish -buildvcs=false`
- `go build -buildvcs=false ./...`

### Bobcoin frontend
Executed successfully:
- `cd bobcoin/frontend && npm run build`
- result: ✅ production Vite build succeeds
- warnings remain for chunk size and browser-externalized dependency modules, but build passes

## Git / Push Summary
### Bobcoin submodule
Successfully pushed:
- `01499c4` — publish WASM-prepared storage to Go supernode (`v8.8.0`)

### Root repo
Current session root-level changes include:
- new publication registry package
- expanded `supernode-go` API
- updated Bobcoin submodule pointer
- doc/version sync to `v11.8.0`

## Current State
- No processes were killed.
- The Go supernode can now:
  - serve WASM runtime assets
  - receive uploaded shards
  - persist and serve manifests
- The Bobcoin frontend can now:
  - preprocess files in-browser with the Go WASM kernel
  - upload the resulting shards
  - publish a retrievable manifest entry

## Remaining Gaps / Next Steps
1. **Anchor manifest metadata on the lattice**
   - create storage-market / NFT / manifest-specific block linkage
   - tie manifest IDs to wallet identities and on-chain records
2. **Add retrieval UX**
   - fetch published manifest JSON
   - retrieve shards by hash
   - reconstruct and decrypt the original file in-browser or via a helper endpoint
3. **Persist lattice state durably**
   - current consensus state still lacks robust restart-safe persistence in the main root Go path
4. **Replace mock Filecoin archival**
   - move from simulated bridge to real Lotus/Filecoin RPC

## Guidance for the Next Agent
- Do not undo the Bobcoin publish flow; it is validated and pushed.
- The next high-value step is **retrieval + lattice anchoring**, not more preprocessing UI.
- `qbittorrent` remains an unresolved untracked/broken remote situation at the root.
