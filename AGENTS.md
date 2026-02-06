# Agents & AI Orchestration

This project utilizes a multi-agent architecture to achieve autonomous development.

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
    -   Running test suites (`gradle test`).
    -   Ensuring "Zero Data Loss" guarantees.
    -   Verifying fixes and regressions.

## 📜 Instructions

All agents must strictly follow the **Universal LLM Instructions**:
[UNIVERSAL_LLM_INSTRUCTIONS.md](docs/UNIVERSAL_LLM_INSTRUCTIONS.md)

## 🔄 Workflow

1.  **Plan**: Director creates a plan in `task.md`.
2.  **Act**: Builder implements the plan.
3.  **Verify**: QA runs tests and confirms stability.
4.  **Document**: All agents update documentation continuously.
