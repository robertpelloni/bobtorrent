# Agents & AI Orchestration

This project utilizes a multi-agent architecture to achieve autonomous development.

## 📜 Universal Instructions

**ALL AGENTS MUST STRICTLY FOLLOW:**
[UNIVERSAL_LLM_INSTRUCTIONS.md](docs/UNIVERSAL_LLM_INSTRUCTIONS.md)

## 🤖 Active Agents

### 1. Director (The Architect)
-   **Role**: High-level planning, system design, and roadmap management.
-   **Responsibilities**:
    -   Maintaining `VISION.md` and `ROADMAP.md`.
    -   Breaking down complex features into tasks.
    -   Coordinating other agents.

### 2. Builder (The Engineer)
-   **Role**: Implementation and coding.
-   **Responsibilities**:
    -   Writing Java/JavaScript/C++ code.
    -   Implementing features defined by the Director.
    -   Refactoring and optimization.

### 3. QA (The Tester)
-   **Role**: Verification and Validation.
-   **Responsibilities**:
    -   Running test suites.
    -   Ensuring "Zero Data Loss" guarantees.
    -   Verifying fixes and regressions.
