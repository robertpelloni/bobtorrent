# Megatorrent Reference Client - User Manual

## Introduction

The Megatorrent Reference Client is a standalone Node.js application that implements the Megatorrent protocol. It allows you to ingest files into encrypted blobs, publish manifests to a tracker, and subscribe to channels to receive content updates.

It features a **Web User Interface** for easy management of your identity, publications, and subscriptions.

## Features

*   **Identity Management**: Generate and manage Ed25519 keypairs for signing content.
*   **Encrypted Storage**: Files are split into encrypted "blobs" (chunks) to ensure privacy and plausible deniability.
*   **Manifest Publishing**: Publish signed manifests containing file metadata and decryption keys.
*   **Channel Subscriptions**: Subscribe to other users' public keys to automatically receive their updates.
*   **DHT Integration**: Uses the BitTorrent DHT for decentralized peer discovery.

## Installation

### Prerequisites

*   **Node.js**: Version 16 or higher.
*   **NPM**: Installed with Node.js.

### Setup

1.  Navigate to the `reference-client` directory:
    ```bash
    cd reference-client
    ```

2.  Install dependencies (if not already done):
    ```bash
    npm install
    ```
    *Note: If you are in the monorepo root, dependencies might already be installed.*

## Running the Web UI

To start the client with the Web UI:

```bash
node web-server.js
```

By default, the UI will be available at:
**http://localhost:3000**

### Configuration

You can configure the server using environment variables:

*   `PORT`: The HTTP port for the UI (default: 3000).
*   `STORAGE_DIR`: Directory to store blobs and config (default: `./storage`).
*   `TRACKER_URL`: WebSocket URL of the tracker (default: `ws://localhost:8000`).

Example:
```bash
PORT=8080 STORAGE_DIR=/data/megatorrent node web-server.js
```

## Using the Interface

### 1. Identity

On the **Identity** tab:
*   Click **Generate New Identity** to create a fresh public/private keypair.
*   **Public Key**: Share this with others so they can subscribe to your channel.
*   **Private Key**: Keep this secret. It is used to sign your publications.

### 2. Publishing Content

On the **Publish** tab:
1.  **Ingest File**:
    *   Enter the **absolute path** to a file on your local machine (e.g., `/home/user/videos/my_video.mp4`).
    *   Click **Ingest File**.
    *   The system will split the file into encrypted blobs and store them in the `storage` directory.
    *   A `FileEntry` JSON object is generated containing the decryption keys.
2.  **Publish Manifest**:
    *   Once a file is ingested, click **Publish Manifest**.
    *   This signs the `FileEntry` with your private key and sends it to the tracker.
    *   Subscribers will immediately receive this update.

### 3. Discovery

The **Discovery** tab allows you to browse the hierarchical topic network (similar to a filesystem).
1.  Enter a topic path (e.g., `mp3/electronic`) or leave blank for root.
2.  Click **Browse**.
3.  Click on subtopics to navigate deeper.
4.  If a publisher is active in a topic, they will appear in the list. Click **Sub** to subscribe to their channel.

### 4. Subscribing

On the **Subscribe** tab:
1.  Enter the **Public Key** (64-character hex) of a publisher you want to follow (or use the **Discovery** tab to find one).
2.  Click **Subscribe**.
3.  The client will connect to the tracker and listen for updates from that key.
4.  When an update is received, it will appear in the "Active Subscriptions" list.

### 5. Files

The **Files** tab lists all content you have ingested or are downloading.
*   **Name**: File name.
*   **Size**: Total size.
*   **Progress**: Percentage of blobs available locally.
*   **Status**: `Downloading` or `Complete`.

### 6. Downloads

On the **Downloads** tab, you can view the status of active file transfers.
*   The client automatically attempts to download content from subscribed channels (in a full implementation).
*   *Note: In this reference implementation, manual fetching via CLI `blob-fetch` is sometimes required for advanced scenarios.*

### 7. Wallet

The **Wallet** tab manages Bobcoin earnings from hosting content (Supernode feature).
*   **Balance**: Confirmed and Pending earnings.
*   **Address**: Your payout address.
*   **Transactions**: History of payments and rewards.

### 8. Remote Management

You can use this Web UI to manage a remote Supernode (Java) or another Reference Client.
1.  Use the selector in the top header (next to the logo) to switch between **Local Node** and **Remote Supernode**.
2.  The UI will automatically proxy requests to the configured target.

### 9. Dashboard

The **Dashboard** provides an overview of:
*   **Storage Usage**: Total size of blobs stored locally.
*   **Network Status**: Connection to DHT and Tracker.
*   **Recent Activity**: Recently added blobs.

## Troubleshooting

*   **"Tracker Error"**: Ensure the tracker is running (`node server.js` in the project root) and reachable at `ws://localhost:8000`.
*   **"File not found"**: When ingesting, ensure you provide the *absolute path* to the file.
*   **"No peers found"**: The DHT takes a few moments to bootstrap. Wait a minute and try again.

## CLI Usage

The client also supports a Command Line Interface (CLI) for headless operation. See `node index.js --help` for details.
