# Universal LLM Instructions

**Project:** Megatorrent (Decentralized Content Distribution Protocol)
**Context:** Reference Implementation (Node.js) & Integration (qBittorrent C++).

## 1. Core Directives
*   **Version Control:** Every major functional change MUST increment the version number in `VERSION` and `package.json`.
*   **Changelog:** Every version bump MUST be recorded in `CHANGELOG.md`.
*   **Structure:** Maintain strict separation between the Node.js client (root) and the C++ reference (`cpp-reference/`).
*   **Submodules:** Do NOT commit large binary files or untracked changes inside submodules unless explicitly intended to update the submodule pointer.

## 2. Coding Standards
*   **Node.js:** Standard JS, no TypeScript transpilation steps (keep it simple). Use `sodium-native` for crypto.
*   **C++:** Qt 6 + C++17. Use `OpenSSL` for crypto (EVP APIs). Do not introduce new heavy dependencies (Boost/Libtorrent are already present).

## 3. Workflow
1.  **Plan:** Always set a plan using `set_plan`.
2.  **Verify:** Use `grep` or `read_file` to confirm edits.
3.  **Test:** Run `scripts/simulate_network.js` for Node.js logic.
4.  **Sync:** Ensure `cpp-reference/` is up to date with any experimental changes made in `qbittorrent/`.

## 4. Versioning Protocol
*   **File:** `VERSION` (Plain text, e.g., `1.2.0`).
*   **Format:** SemVer (`Major.Minor.Patch`).
*   **Bump Rule:**
    *   **Major:** Protocol breaking change.
    *   **Minor:** New feature (e.g., C++ Integration).
    *   **Patch:** Bug fix / Refactor.
