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

## Phase 7: Publisher & Subscriber Identity and Network Propagation (Complete)
* [x] Setup Build Flags to inject Version into binary (Complete)
* [x] Implement Identity generation (`/api/key/generate`)
* [x] Implement Manifest Publishing (`/api/publish`) using Identity signatures
* [x] Implement Channel Subscriptions (`/api/subscribe`, `/api/subscriptions`)
* [x] Implement Local Blob listing (`/api/blobs`)
* [x] Port Tracker Pub/Sub or DHT Put/Get for decentralized manifest propagation
* [x] Update the UI to fully reflect active Pub/Sub status and real-time incoming manifests

## Phase 8: Enhancement, Integration, and Polish (Upcoming)
* [ ] Design and scaffold the Mega-Messenger architecture (Light Mobile Clients proxied to Heavy Go Nodes) utilizing the `element-web` reference frontend.
* [ ] Integrate native I2P/SAM Datagrams directly into the core networking layer
* [ ] Submodule updates and cross-branch testing
* [x] Implement missing Web UI tooltips, labels, and fine details
* [ ] Performance profiling and concurrency tuning

