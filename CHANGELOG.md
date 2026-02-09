# Changelog

All notable changes to this project will be documented in this file.

## [1.8.0] - 2024-05-24
### Added
-   **Full Supernode Web API Coverage**: Expanded `supernode-java` to include endpoints for Identity, Publishing, Subscriptions, Discovery, and Wallet.
-   **API Parity**: Java Supernode now fully mimics the Reference Client API, enabling seamless Web UI usage.
-   **Component Integration**: `UnifiedNetwork` now exposes `DHTDiscovery` and `ManifestDistributor` for API consumption.

## [1.7.0] - 2024-05-23
### Added
-   **Supernode-Java Web API**: Implemented a Netty-based HTTP Controller in `supernode-java` to support the Web UI via `WebController`.
-   **Cross-Client Compatibility**: Verified `ingest`, `retrieve`, `status`, and `files` endpoints work seamlessly between Node.js Web UI and Java Supernode backend.
-   **Java Streaming**: Added HTTP Range request support to Java Supernode for video playback.
-   **Standalone Supernode**: Created `io.supernode.Supernode` main class and updated Gradle build to produce a runnable application.

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
