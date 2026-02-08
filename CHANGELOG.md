# CHANGELOG

All notable changes to the Megatorrent (formerly Bobtorrent) project.

## [11.2.5] - 2026-02-08

### Added
- **Wallet Tab**: Web UI integration for Bobcoin earnings and transactions.
- **Remote Management**: Web UI can now connect to remote Supernodes via the node selector.
- **Encryption**: Standardized Reference Client on `AES-256-GCM` to match Java Supernode.

## [11.2.4] - 2026-02-08

### Added
- **Web UI**: Comprehensive Reference Client interface (`http://localhost:3000`).
  - **Dashboard**: Real-time system status, storage metrics, and version info.
  - **Discovery**: Hierarchical topic browser for Channels.
  - **Files**: List of ingested and downloading content.
  - **Identity**: Keypair generation and management.
- **Reference Client Backend**: Secure Node.js server (`web-server.js`) binding to localhost.
- **qBittorrent Integration**:
  - Added `MegatorrentController` (C++) stubs for API integration.
  - Added `megatorrent.html` and `megatorrent.js` for qBittorrent WebUI extension.
  - Created `cpp-reference/install_webui_patches.sh` for applying changes to the submodule.

### Fixed
- **Security**: Removed wildcard CORS headers from reference client to prevent CSRF.
- **Security**: Implemented path traversal protection in `web-server.js`.
- **Tracker**: Fixed `lru-cache` import issue (switched to `lru`).
- **Tracker**: Fixed crash when handling non-standard WebSocket messages (undefined `info_hash`).

### Documentation
- **Consolidated Instructions**: Created `docs/UNIVERSAL_LLM_INSTRUCTIONS.md`.
- **Manual**: Added `reference-client/MANUAL.md` with detailed usage guide.
- **Dashboard**: Updated `DASHBOARD.md` with submodule details and project structure.

## [11.2.3] - 2026-02-05

### Documentation
- **Major Update**: Added comprehensive project governance documentation (`VISION.md`, `AGENTS.md`, `DASHBOARD.md`).
- **Guidance**: Added `UNIVERSAL_LLM_INSTRUCTIONS.md` for autonomous agent coordination.
- **Roadmap**: Updated `ROADMAP.md` and project analysis.

## [11.2.1] (2025-01-19)

### Bug Fixes
* http announce no left ([#548](https://github.com/webtorrent/bittorrent-tracker/issues/548)) ([3cd77f3](https://github.com/webtorrent/bittorrent-tracker/commit/3cd77f3e6f5b52f5d58adaf004b333cd2061a4da))
