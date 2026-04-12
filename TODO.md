# BobTorrent TODO

## Immediate Tasks
- [ ] Build the deeper `anacrolix/torrent` integration layer to actively map 1MB encrypted blobs to standard BitTorrent piece boundaries.
- [ ] Connect the Web UI to a live Solana devnet (verify airdrops and balance polling from the Go backend).
- [ ] Verify that `ReadaheadBuffer.Seek` boundaries don't panic on exact EOF offsets during video playback.

## Completed Tasks
- [x] Phase 1: Go module initialization, legacy code archiving.
- [x] Phase 2: DHT InfoHash mapping and Solana wallet porting.
- [x] Phase 3: Manifest parsing, Detached AES-256-GCM encryption/decryption, Readahead streaming logic.
- [x] Phase 4: Full HTTP Range request processing via `io.Seeker` and API Server initialization.

## Ongoing Documentation Tasks
- [ ] Continue updating `IDEAS.md` with potential improvements.
- [ ] Log structural findings in `MEMORY.md`.
