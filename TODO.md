# Phase 8 (Current phase)
- [x] **Primary Torrent Download Module**: Implement the primary torrent download module, starting with the file verification routine (`Verifier`).
- [x] **Monorepo Unification & Core Cleanup**: Migrate all Go logic from the `bobtorrent/` submodule into the root workspace structure. Remove the `bobtorrent/` directory to eliminate code duplication and import path confusion.
- [x] **I2P Native Datagrams Refactoring**: Refactor the transport layer to fully integrate `github.com/eyedeekay/sam3` or similar for hybrid clear-net/dark-net peer connections.
- [x] **DHT Sub-Routing Optimization**: Optimize the Kademlia routing table implementation to better handle hybrid I2P/clearnet peer addressing and reduce lookup latency.
- [ ] **Mobile Client Implementation Phase 2**: Add background execution capabilities to the React Native `MobileMessenger` client to support receiving push notifications or silent network syncs.
