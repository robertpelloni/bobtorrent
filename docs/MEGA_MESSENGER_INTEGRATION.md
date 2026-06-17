# Mega-Messenger: Element-Web Integration Path

## 1. Architectural Alignment
The goal is to repurpose the `element-web` UI/UX while replacing the Matrix backend with the Go-native libp2p mesh.

## 2. Integration Strategy
- **Control Plane**: `supernode-go` remains the heavy backend.
- **Frontend**: Extract React components from `element-web` packages.
- **Bridge**: UI communicates with backend via `/mega-bridge` WebSocket using the `Envelope` protocol.

## 3. Implementation Steps
1. **Payload Translation**: Map Matrix events (m.room.message) to Bobtorrent Envelopes (TypeChat).
2. **WebSocket Driver**: Implement a custom `MatrixClient` backend in JS that talks to `/mega-bridge` instead of a Matrix HomeServer.
3. **Decoupled Deployment**: Serve the extracted frontend from the Go node or wrap it in Tauri/Electron.

## 4. Security
- Use Ed25519 for end-to-end identity.
- Implement DH/AES-GCM within the Envelope `encrypted_body` for private rooms.
