# BobTorrent Roadmap (v3.0.0+)

## Phase 1: Go Core Porting (Complete)
* [x] Project Reorganization (Archiving v2.2.0, Version Bump to 3.0.0)
* [x] Initialize Go module (`bobtorrent`)
* [x] Port DHT Discovery & Mapping (32-byte InfoHash to 20-byte `libtorrent` style)
* [x] Port Custom Binary Protocol v5 and AES-256-GCM Blob Storage
* [x] Port Solana Wallet & Identity Management natively in Go

## Phase 2/3/4/5/6: Advanced Network Features, Streaming, Testing & API (Complete)
* [x] Port I2P/SAM v3.1 Integration from C++ to native Go
* [x] Unify Manifests and Key Distribution
* [x] Re-implement Predictive Streaming and Readahead algorithms
* [x] Handle HTTP Range Requests (206 Partial Content) correctly using `io.Seeker`
* [x] Port Web UI & HTTP API from Java/Node.js to Go
* [x] Implement Ingestion endpoint with automatic encryption and Manifest generation
* [x] Embed Web UI directly into the Go binary (`go:embed`)
* [x] Wire up `anacrolix/torrent` BitTorrent logic for dynamically mapping and downloading AES Blobs
* [x] Local multi-node network End-to-End simulations

## Phase 7: Enhancement, Integration, and Polish (Next)
* [ ] Submodule updates and cross-branch testing
* [ ] Implement missing Web UI tooltips, labels, and fine details
* [ ] Performance profiling and concurrency tuning
