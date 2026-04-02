# Megatorrent Deployment Guide

## Overview

Megatorrent consists of two main components:
1.  **Reference Client (Node.js)**: User-friendly client with Web UI.
2.  **Supernode (Java)**: High-performance backend node.

Both can be controlled via the same Web UI.

## 1. Reference Client (Node.js)

### Prerequisites
*   Node.js v16+
*   NPM

### Build & Run
```bash
cd reference-client
npm install
npm start
```

### Access
*   **Web UI**: `http://localhost:3000`
*   **API**: `http://localhost:3000/api`

### Environment Variables
*   `PORT`: UI/API Port (Default: 3000)
*   `STORAGE_DIR`: Path to storage (Default: `./storage`)

---

## 2. Supernode (Java)

### Prerequisites
*   JDK 21+
*   Gradle (Wrapper included)

### Build
```bash
cd supernode-java
./gradlew build
./gradlew installDist
```

### Run
```bash
# Run from distribution
./build/install/supernode/bin/supernode [PORT]

# Example (Port 8080)
./build/install/supernode/bin/supernode 8080
```

### Access
*   **Web UI**: `http://localhost:8080` (Using embedded Netty server)
*   **API**: `http://localhost:8080/api`

---

## 3. Remote Management

You can use the **Reference Client's Web UI** to manage a remote **Supernode**.
1.  Open `http://localhost:3000` (Node.js UI).
2.  In the top header, select "Remote Supernode" from the dropdown.
3.  Ensure the dropdown value points to your Java node (e.g., `http://localhost:8080`).
