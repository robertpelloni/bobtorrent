# Deployment Instructions

## 🚀 Re-Deployment Protocol

To deploy the Bobtorrent suite, execute the following commands in order:

### 1. Pre-Deployment Sync
Ensure the Omni-Workspace is synchronized and built.
```bash
git fetch --all
git pull origin main
git submodule update --init --recursive
```

### 2. Tracker (Node.js) Deployment
Install dependencies and restart the tracker via PM2 (or equivalent process manager).
```bash
npm install
npm run build --if-present
pm2 restart bobtorrent-tracker
```

### 3. Supernode (Java) Deployment
Build the highly optimized shadow jar and deploy the Java daemon.
```bash
cd supernode-java
./gradlew clean shadowJar
systemctl restart supernode-java
```

### 4. Health Checks
Verify that the Tracker is responding to WebSockets and that the Supernode has connected to Kademlia DHT.
```bash
curl -i http://localhost:8000/stats.json
```
Check Supernode metrics dashboard (if configured) on `http://localhost:8080/metrics`.
