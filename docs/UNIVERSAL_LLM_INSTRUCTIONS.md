# Universal LLM Instructions

This document serves as the single source of truth for all LLMs (Claude, Gemini, GPT, etc.) interacting with the **Megatorrent / Bobtorrent** codebase. All agents must adhere to these protocols.

## 1. Core Mandates

-   **Autonomous & Proactive**: Do not wait for user hand-holding. Plan, Execute, Verify, and Iterate. If a path is blocked, find a workaround or pivot strategically.
-   **"Code is Law"**: Security, encryption, and data integrity are non-negotiable. Use robust patterns (e.g., standard crypto libraries over home-rolled solutions).
-   **Documentation First**: Every major feature must have a corresponding update in `VISION.md`, `ROADMAP.md`, `CHANGELOG.md` and inline code documentation.
-   **Test-Driven**: No code is "done" until it is verified by a test. If existing tests fail, fix them before moving forward.
-   **Full Implementation**: Ensure every feature is 100% implemented, extremely robust, and well-documented with config and breadth of options. No hidden or underrepresented functionality.
-   **UI Representation**: Every feature must be well-represented in the UI with labels, descriptions, and tooltips.

## 2. Project Context

-   **Project**: Megatorrent (formerly Bobtorrent).
-   **Goal**: Create a decentralized, incentivized, autonomous storage network and immutable content distribution protocol.
-   **Tech Stack**:
    -   **Backend**: Node.js (Reference Client), Java (Supernode), C++ (qBittorrent/libtorrent integration).
    -   **Frontend**: HTML/JS (Web UI), Playwright (Verification).
    -   **Blockchain**: Solana (Bridge), Filecoin (Incentives).

## 3. Communication Protocols

-   **Task Management**: Maintain `task.md` meticulously if used.
-   **Task Boundaries**: Use `task_boundary` tool to communicate state changes clearly.
-   **User Notifications**: Use `notify_user` only when blocked or for critical reviews. Batch questions.
-   **Detailed Logging**: Document all findings, decisions, and changes in the conversation log and `HANDOFF.md`.

## 4. Coding Standards

-   **JavaScript/Node.js**: Modern ES6+ syntax, `import/export`, strict error handling.
-   **C++**: Qt-style for qBittorrent integration, RAII, safety.
-   **Java**: Standard Google Java Style.
-   **Safety**: Handle all exceptions gracefully. Never swallow errors without logging.
-   **Submodules**: Respect submodule boundaries. Do not modify submodule code directly unless fixing a bug to be upstreamed. Use patches or reference implementations.

## 5. Branching & Versioning

-   **Versioning**: Semantic Versioning (MAJOR.MINOR.PATCH).
-   **Global Versioning**: The project uses a single version number stored in `VERSION` text file.
-   **Updates**: When updating version, update `package.json`, `VERSION`, and `CHANGELOG.md`.
-   **Commit Messages**: Must include the new version number if bumped.

## 6. Agent Personas

-   **Architect**: Focus on system design, patterns, and high-level structure.
-   **Engineer**: Focus on robust implementation, testing, and optimization.
-   **Reviewer**: Focus on security auditing and code quality verification.

---
*Reference this file in all agent-specific instruction files.*
