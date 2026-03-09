# Universal LLM Instructions

**Project:** Megatorrent Monorepo (Megatorrent + Bobcoin + qBittorrent Integration)

## 1. Core Directives
*   **Version Control:** Every major functional change MUST increment the version number in `VERSION` and `package.json`.
*   **Changelog:** Every version bump MUST be recorded in `CHANGELOG.md`.
*   **Structure:** Maintain strict separation between:
    *   `reference-client/` (Root): Node.js Megatorrent Client.
    *   `cpp-reference/`: Canonical C++ Source.
    *   `qbittorrent/`: qBittorrent Submodule (C++ Integration Target).
    *   `bobcoin/`: Bobcoin Submodule (Economy/Token).
*   **Submodules:**
    *   Edit submodule code directly.
    *   **CRITICAL:** Always commit changes within the submodule and update the pointer in the main repo.
    *   Treat this as a "powerful monorepo" developing multiple projects.

## 2. Documentation & Analysis
*   **Document Inputs:** Always document input information in detail. Ask for clarification if needed to refine the vision.
*   **Research:** Research libs/submodules to infer selection reasons.
*   **Dashboard:** Maintain `DASHBOARD.md` listing all submodules, versions, and structure.

## 3. Coding Standards
*   **Node.js:** Standard JS, `sodium-native` for crypto.
*   **C++:** Qt 6 + C++17. Use `OpenSSL` for crypto.
*   **Versioning:** Single source of truth is `VERSION` (text file).

## 4. Vision
*   **Megatorrent:** Decentralized, anonymous, resilient content distribution.
*   **Bobcoin:** Solana+Monero hybrid, privacy-focused, high volume, mining-by-dancing.
*   **Integration:** Arcade machines double as miners/nodes.

## 5. Workflow
1.  **Plan:** Use `set_plan`.
2.  **Verify:** Check files.
3.  **Test:** Run `npm test` (Simulation).
4.  **Sync:** Ensure `cpp-reference` matches `qbittorrent` changes.
5.  **Commit:** Frequent commits with version bumps.
