# Megatorrent Reference Client

A reference implementation of the **Megatorrent** protocol, featuring:

*   **Ingest**: Split files into encrypted blobs.
*   **Publish**: Sign manifests with Ed25519 keys.
*   **Subscribe**: Receive real-time updates from publishers.
*   **Web UI**: A complete interface for managing your node.
*   **CLI**: Command-line tools for scripting.

## Getting Started

### Prerequisites

*   Node.js (v16+)
*   NPM

### Installation

```bash
cd reference-client
npm install
```

### Running the Web UI

To start the graphical interface:

```bash
node web-server.js
```

Then open your browser at **http://localhost:3000**.

### Running the CLI

For headless operation:

```bash
# Generate identity
node index.js gen-key

# Ingest a file
node index.js ingest -i /path/to/file.mp4

# Publish (requires identity.json)
node index.js publish -i file_entry.json

# Subscribe
node index.js subscribe <public_key_hex>
```

See `MANUAL.md` for full documentation.
