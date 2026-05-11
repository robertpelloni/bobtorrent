# Handoff Document

## Current Status (Phase 7 - Pub/Sub Identity)
The core Go port correctly manages BitTorrent InfoHash mapping, AES-256-GCM encryption/decryption, HTTP Range Readahead streaming, and local Devnet Solana wallets. 
The E2E tests have been stabilized and compilation errors in the API routing are resolved.

## What I Did
1. Re-analyzed the Node.js reference client, Java supernode, and C++ qBittorrent legacy ports.
2. Determined that the core blob/torrent operations are functional in the Go port, but the **Identity Generation, Subscriptions, and Publishing Pub/Sub mechanics** are currently stubbed or missing.
3. Updated the `ROADMAP.md` and `TODO.md` with explicit tasks to implement `/api/key/generate`, `/api/publish`, `/api/subscribe`, `/api/subscriptions`, and `/api/blobs`.
4. Fixed `TestE2E_IngestAndStream` EOF deadlocks in `readahead.go` and normalized BlobID vs InfoHash storage formats in `ingest.go`.
5. Created a unified `build.sh` and `-ldflags` approach to inject `VERSION` directly into the binary at build time.

## Immediate Next Steps for Next Session/Agent
- Start working down the new Immediate Tasks in `TODO.md`.
- **Wire BEP 44 Mutable DHT Items**: Connect the `dht.Engine` stub logic for `Put` and `Get` to correctly broadcast and poll for signed manifests.
- **Integrate Pub/Sub Tracking**: Implement continuous background polling or tracker updates to automatically sync subscribed channels.
- **Game Engine Asset Ingestion**: Investigate the ingestion path for specific game engine assets.
- **Global Network Launch**: Prepare the system for public network testing and launch.

