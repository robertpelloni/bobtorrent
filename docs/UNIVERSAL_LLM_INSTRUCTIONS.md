# Universal LLM Instructions

This document serves as the single source of truth for all LLMs (Claude, Gemini, GPT, etc.) interacting with the **Bobtorrent/Supernode Java** codebase. All agents must adhere to these protocols.

## 1. Core Mandates

-   **Autonomous & Proactive**: Do not wait for user hand-holding. Plan, Execute, Verify, and Iterate. If a path is blocked, find a workaround or pivot strategically.
-   **"Code is Law"**: Security, encryption, and data integrity are non-negotiable. Use robust patterns (e.g., standard crypto libraries over home-rolled solutions).
-   **Documentation First**: Every major feature must have a corresponding update in `VISION.md`, `ROADMAP.md`, and `CHANGELOG.md`.
-   **Test-Driven**: No code is "done" until it is verified by a test. If existing tests fail, fix them before moving forward.

## 2. Project Context

-   **Project**: Supernode Java (part of Bobtorrent/Bobcoin ecosystem).
-   **Goal**: Create a decentralized, incentivized, autonomous storage network.
-   **Tech Stack**:
    -   **Backend**: Java (Gradle), C++ (qBittorrent/libtorrent).
    -   **Frontend/Glue**: Node.js (bobtorrent tracker).
    -   **Blockchain**: Solana (Bridge), Filecoin (Incentives).

## 3. Communication Protocols

-   **Task Management**: Maintain `task.md` meticulously.
-   **Task Boundaries**: Use `task_boundary` tool to communicate state changes clearly.
-   **User Notifications**: Use `notify_user` only when blocked or for critical reviews. Batch questions.

## 4. Coding Standards

-   **Java**: Follow standard Google Java Style. Use effective final variables where possible.
-   **Async**: Utilize `CompletableFuture` for non-blocking I/O.
-   **Safety**: Handle all exceptions gracefully. Never swallow errors without logging.
-   **Submodules**: Respect submodule boundaries. Do not modify submodule code directly unless fixing a bug to be upstreamed.

## 5. Branching & Versioning

-   **Versioning**: Semantic Versioning (MAJOR.MINOR.PATCH).
-   **Updates**: When updating version, update `package.json`, `gradle.properties` (if applicable), `VERSION`, and `CHANGELOG.md`.

## 6. Agent Personas

-   **Architect**: Focus on system design, patterns, and high-level structure.
-   **Engineer**: Focus on robust implementation, testing, and optimization.
-   **Reviewer**: Focus on security auditing and code quality verification.

---
*Reference this file in all agent-specific instruction files.*
