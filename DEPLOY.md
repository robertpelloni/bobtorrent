# Deployment Instructions (Omni-Workspace)

## Go Port Deployment (v11.5.0+)

### 1. Build the Binaries
Run the `build.bat` script or execute manually:
```bash
go build -o build/dht-proxy cmd/dht-proxy/main.go
go build -o build/supernode-go cmd/supernode-go/main.go
go build -o build/lattice-go cmd/lattice-go/main.go
```

### 2. Run the Consensus Node
The Lattice node must be running for the Supernode to sync.
```bash
./build/lattice-go
```
*Note: Default port is 4000.*

### 3. Run the Supernode
The Supernode provides tracker, DHT, and seeding services.
```bash
./build/supernode-go
```
*Note: Requires `lattice-go` for market polling and bid acceptance.*

### 4. Run the DHT Proxy
Optional utility for hiding client IPs.
```bash
./build/dht-proxy
```
*Note: Requires `GeoLite2-City.mmdb` in the same directory for proximity sorting.*

## Legacy Node.js Stack

### 1. Game Server
```bash
cd bobcoin/game-server
npm install
npm start
```

### 2. Frontend
```bash
cd bobcoin/frontend
npm install
npm run dev
```

## Docker Deployment
Refer to `docker-compose.yml` for containerized orchestration of the entire ecosystem.
```bash
docker-compose up --build
```
