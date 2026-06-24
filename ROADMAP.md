# Phase 8 (Complete)
- [x] **Primary Torrent Download Module**: Implement the primary torrent download module, starting with the file verification routine (`Verifier`).
- [x] **Monorepo Unification & Core Cleanup**: Migrate all Go logic from the `bobtorrent/` submodule into the root workspace structure. Remove the `bobtorrent/` directory to eliminate code duplication and import path confusion.
- [x] **I2P Native Datagrams Refactoring**: Refactor the transport layer to fully integrate `github.com/eyedeekay/sam3` or similar for hybrid clear-net/dark-net peer connections.
- [x] **DHT Sub-Routing Optimization**: Optimize the Kademlia routing table implementation to better handle hybrid I2P/clearnet peer addressing and reduce lookup latency.
- [x] **Mobile Client Implementation Phase 2**: Add background execution capabilities to the React Native `MobileMessenger` client to support receiving push notifications or silent network syncs.

# Phase 9 (Complete)
- [x] **Performance Profiling**: Implement `pprof` endpoints and analyze CPU/memory bottlenecks in the supernode.
- [x] **Advanced Anonymity**: Design and implement deep integration of I2P into the anacrolix k-buckets once the library supports non-IP krpc.NodeAddr types.
- [x] **Cross-Node Testing**: Perform cross-node consensus and messaging reliability tests under heavy load.
# Phase 10 (Current phase)
- [x] **Game Engine Asset Ingestion Path**: Implement a specialized asset ingestion pipeline for game engines to upload large textures and models directly into the swarm.
- [ ] **Swarm Discovery API**: Implement an endpoint to allow game engine clients to query the DHT directly for asset piece availability before beginning download.
# Phase 11: Jules Autopilot Orchestrator (Current phase)
- [x] **Shadow Pilot Git Diff Monitoring**: Scaffold the internal anomaly detection engine using `git status --porcelain`.
- [x] **System Status Integration**: Hook the `Shadow Pilot` state into the `/api/system/status` API endpoint.
- [ ] **Frontend Dashboard Integration**: Wire the Vite/React UI to display Shadow Pilot git anomaly data.
- [ ] **CI Pipeline Auto-Fix**: Automatically trigger anomaly fixes and commits when Shadow Pilot detects drift.
- [ ] **Submodule Status Check**: Extend Shadow Pilot to recursively check submodules for anomalies.
