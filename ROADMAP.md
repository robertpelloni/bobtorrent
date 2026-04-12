# BobTorrent Roadmap (v3.0.0+)

## Phase 1: Go Core Porting (Complete)
* [x] Project Reorganization (Archiving v2.2.0, Version Bump to 3.0.0)
* [x] Initialize Go module (`bobtorrent`)
* [x] Port DHT Discovery & Mapping (32-byte InfoHash to 20-byte `libtorrent` style)
* [x] Port Custom Binary Protocol v5 and AES-256-GCM Blob Storage
* [x] Port Solana Wallet & Identity Management natively in Go

## Phase 2/3/4: Advanced Network Features & Streaming (Complete)
* [x] Port I2P/SAM v3.1 Integration from C++ to native Go
* [x] Unify Manifests and Key Distribution
* [x] Re-implement Predictive Streaming and Readahead algorithms
* [x] Handle HTTP Range Requests (206 Partial Content) correctly using `io.Seeker`
* [x] Port Web UI & HTTP API from Java/Node.js to Go (Wiring Complete)

## Phase 5: Enhancement, Integration, and Polish (In Progress)
* [ ] Deep `anacrolix/torrent` piece downloading mapping for Custom Blobs
* [ ] Implement missing Web UI tooltips, labels, and fine details
* [ ] Performance profiling and concurrency tuning
* [ ] End-to-end automated integration tests for the unified Go stack
