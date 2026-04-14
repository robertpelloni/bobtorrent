# Changelog
All notable changes to this project will be documented in this file.

## [3.0.0] - 2026-04-14
### Changed
- Massive architectural shift: Replaced the fragmented Node.js/Java/C++ monorepo with a single, unified, natively-compiled Go "ultra-project" (`bobtorrent/`).
- Moved all legacy code (v2.2.0 and prior) to `archive/` directory.

### Added
- Integrated `anacrolix/torrent` as the pure Go BitTorrent engine, natively replacing `libtorrent`.
- Implemented DHT InfoHash mapping translating 32-byte BobTorrent keys to 20-byte standard InfoHashes.
- Ported Solana Wallet management to Go using `github.com/gagliardetto/solana-go`.
- Implemented AES-256-GCM Detached Key Storage Protocol natively in Go.
- Added `storage.Manifest` struct for parsing JSON descriptors and detached decryption keys.
- Scaffolded I2P/SAM v3.1 integration using `github.com/go-i2p/sam3`.
- Implemented `ReadaheadBuffer` for predictive streaming and pre-fetching encrypted blobs via DHT.
- Hooked `io.Seeker` interface into the streaming buffer, allowing Go's native `http.ServeContent` to seamlessly handle 206 Partial Content Range requests for HTML5 video playback.
- Added `/api/ingest` multipart file endpoint which encrypts files into 1MB chunks and dynamically announces them to the torrent network on the fly.
- Embedded the React Web UI directly into the compiled Go binary utilizing the `go:embed` directive.
- Automated end-to-end multi-node HTTP and chunk prefetching test logic.
