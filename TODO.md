# BobTorrent TODO

## Immediate Tasks
- [ ] Refine `anacrolix/torrent` instantiation to correctly manage our custom 1MB chunks mapping to standard BitTorrent piece sizes.
- [ ] Connect the `ReadaheadBuffer` output directly to `http.ServeContent` in `StreamHandler` to properly respond to HTTP Range requests from the video player.
- [ ] Ensure the Web UI interacts perfectly with the new Go backend (test Airdrop and streaming).

## Completed Tasks
- [x] Phase 1: Initialize Go module, archive legacy code.
- [x] Phase 2: DHT InfoHash mapping and Solana wallet porting.
- [x] Phase 3: Manifest parsing, Detached AES-256-GCM encryption/decryption, Readahead streaming logic.
- [x] Phase 3: Native Go I2P/SAM v3.1 integration framework.

## Ongoing Documentation Tasks
- [ ] Continue updating `IDEAS.md` with potential improvements.
- [ ] Log structural findings in `MEMORY.md`.
