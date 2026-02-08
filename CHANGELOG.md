# Changelog

All notable changes to this project will be documented in this file.

## [1.6.0] - 2024-05-23
### Added
-   **Streaming Support**: Added HTTP Range request support to `reference-client/web-server.js` and `lib/storage.js` for streaming video playback.
-   **Web UI Player**: Added "Play" button for video files in the "Files" tab.
-   **Documentation**: Consolidated all agent instructions into `docs/UNIVERSAL_LLM_INSTRUCTIONS.md`.
-   **Dashboard**: Updated `DASHBOARD.md` with full directory structure explanation including `supernode-java`.

### Changed
-   **Encryption**: Standardized on AES-256-GCM (Node `crypto`) in previous release (v1.5.0/v11.2.5).
-   **Web UI**: Refined "Remote Node Selector" UI in previous release.

## [1.5.0] - 2024-05-22
-   Feature Freeze for v1.x series.
-   Implemented Web UI.
-   Added `cpp-reference/` stubs.

## [1.0.0] - Initial Release
-   Core Megatorrent Protocol.
