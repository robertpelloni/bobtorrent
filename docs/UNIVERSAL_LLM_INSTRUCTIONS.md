# Universal AI Agent Instructions

**Effective Date:** 2026-02-12
**Version:** 2.0 (Unified)

## 🎯 Core Directive
You are a highly skilled, autonomous software engineer. Your mission is to implement features for **Megatorrent**, a next-generation decentralized content distribution platform. You must strive for **comprehensive detail**, **robustness**, and **visual completeness**.

---

## 🛠 Project Structure (The Monorepo)

*   **`reference-client/`**: Node.js implementation (Web UI + Client).
    *   `web-server.js`: The API backend and static file server.
    *   `web-ui/`: Single Page Application (HTML/JS/CSS).
    *   `lib/`: Protocol logic.
*   **`supernode-java/`**: High-performance Java Supernode.
    *   `io.supernode.Supernode`: Main entry point.
    *   `io.supernode.api.WebController`: Netty-based HTTP API (Parity with Node.js).
    *   `io.supernode.network`: Unified networking stack (DHT, Blobs, Transports).
    *   `io.supernode.storage`: Erasure Coding and Storage logic.
*   **`qbittorrent/`**: C++ submodule (Reference patches).
*   **`docs/`**: Project documentation.

## 📜 Documentation Standards

1.  **Single Source of Truth**:
    *   **Version Number**: Must strictly adhere to the content of the `VERSION` file in the root.
    *   **Changelog**: All changes must be recorded in `CHANGELOG.md` under the new version header.
    *   **Instructions**: This file (`UNIVERSAL_LLM_INSTRUCTIONS.md`) supersedes individual model files.

2.  **Version Control**:
    *   When implementing *any* feature or fix, increment the version in `VERSION`.
    *   Create a matching entry in `CHANGELOG.md`.
    *   Reference the version in your commit message.

3.  **Code Comments**:
    *   Comment complex logic (Why, not just What).
    *   Leave simple code bare.

## 🚀 Deployment & Build

*   **Node.js**: `cd reference-client && npm install && npm start`
*   **Java**: `cd supernode-java && ./gradlew installDist && ./build/install/supernode/bin/supernode [PORT]`
*   **Web UI**: Accessible at `http://localhost:3000` (Node) or `http://localhost:8080` (Java).

## 🤖 Feature Implementation Guide

1.  **Analyze**: Understand the requirement deeply. Ask clarifying questions if unsure.
2.  **Plan**: Create a step-by-step plan using `set_plan`.
3.  **Implement**:
    *   **Backend First**: Implement core logic in Java/Node.
    *   **API Layer**: Expose functionality via HTTP endpoints (`WebController`/`web-server.js`).
    *   **Frontend**: Create rich, visual UI components to represent the feature. **Do not leave features invisible.**
4.  **Verify**:
    *   **Tests**: Run unit/integration tests (`./gradlew test`, `npm test`).
    *   **Visual**: Use Playwright scripts to generate screenshots of the UI.
5.  **Document**: Update `MANUAL.md`, `ROADMAP.md`, `TODO.md`.

## 🔄 Handoff Protocol

At the end of your session:
1.  Merge all feature branches into `main` (if acting as a git operator) or ensure your changes are ready to commit.
2.  Update `HANDOFF.md` with:
    *   **Session Summary**: What was achieved?
    *   **Current State**: What is working? What is broken?
    *   **Next Steps**: Clear, actionable items for the next agent.
    *   **Context**: Any specific quirks or design decisions made.

---

*Keep going. Don't stop. Proceed with excellence.*
