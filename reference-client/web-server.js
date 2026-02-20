import fs from 'fs'
import path from 'path'
import http from 'http'
import { fileURLToPath } from 'url'
import { generateKeypair } from './lib/crypto.js'
import { createManifest } from './lib/manifest.js'
import { ingest, createBlobClient, createReadStream } from './lib/storage.js'
import { BlobStore } from './lib/blob-store.js'
import { BlobNetwork } from './lib/blob-network.js'
import { BlobTracker } from './lib/blob-tracker.js'
import { SubscriptionStore } from './lib/subscription-store.js'
import { createChannel } from './lib/channels.js'
import { WalletManager } from './lib/wallet.js'
import DHT from 'bittorrent-dht'
import Client from '../index.js'
import { createRequire } from 'module'

const require = createRequire(import.meta.url)
const packageJson = require('../package.json')
const __dirname = path.dirname(fileURLToPath(import.meta.url))

const PORT = process.env.PORT || 3000
const STORAGE_DIR = process.env.STORAGE_DIR || './storage'
const SUBSCRIPTIONS_FILE = process.env.SUBSCRIPTIONS_FILE || './subscriptions.json'
const TRACKER_URL = process.env.TRACKER_URL || 'ws://localhost:8000'

// Ensure storage exists
if (!fs.existsSync(STORAGE_DIR)) {
  fs.mkdirSync(STORAGE_DIR, { recursive: true })
}

// State
let dht = null
const blobStore = new BlobStore(STORAGE_DIR)
let blobNetwork = null
let blobTracker = null
let channel = null
const subscriptionStore = new SubscriptionStore(SUBSCRIPTIONS_FILE)
let filesIndex = [] // In-memory file index (persist to file in future)
const walletManager = new WalletManager(STORAGE_DIR)

// Initialize Network (Lazy)
async function initNetwork () {
  if (blobNetwork) return

  console.log('Initializing network...')
  dht = new DHT()
  await new Promise(resolve => dht.on('ready', resolve))

  blobTracker = new BlobTracker(dht, { port: 6881 }) // Standard DHT port
  blobNetwork = new BlobNetwork(blobStore, { port: 0 }) // Random port
  channel = await createChannel(dht)

  await blobNetwork.listen()
  console.log(`Blob network listening on port ${blobNetwork.port}`)

  createBlobClient(blobStore, blobNetwork, blobTracker)

  // Load subscriptions
  subscriptionStore.load()

  // Load Wallet
  if (!walletManager.load()) {
    walletManager.create()
    console.log('Created new wallet:', walletManager.getAddress())
  } else {
    console.log('Loaded wallet:', walletManager.getAddress())
  }

  // Load files index (mock persistence)
  if (fs.existsSync(path.join(STORAGE_DIR, 'files.json'))) {
    try {
      filesIndex = JSON.parse(fs.readFileSync(path.join(STORAGE_DIR, 'files.json')))
    } catch (e) { console.error('Failed to load files index', e) }
  }
}

// Start Server
const server = http.createServer(async (req, res) => {
  // Security: No CORS headers needed as we serve same-origin
  // This prevents malicious sites from accessing the local API via CSRF

  // API Routes
  if (req.url.startsWith('/api/')) {
    try {
      // Proxy to Remote Supernode if configured
      if (req.headers['x-target-node'] && req.headers['x-target-node'] !== 'local') {
        await proxyToSupernode(req, res, req.headers['x-target-node'])
        return
      }
      await handleApi(req, res)
    } catch (err) {
      console.error('API Error:', err)
      res.writeHead(500, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ error: err.message }))
    }
    return
  }

  // Static Files
  // Security: Sanitize path to prevent traversal
  const parsedUrl = new URL(req.url, `http://${req.headers.host}`)
  const pathname = parsedUrl.pathname
  const safeUrl = path.normalize(pathname).replace(/^(\.\.[\/\\])+/, '')
  const filePath = path.join(__dirname, 'web-ui', safeUrl === '/' || safeUrl === '.' ? 'index.html' : safeUrl)

  // Security: Ensure we stay within web-ui directory
  if (!filePath.startsWith(path.join(__dirname, 'web-ui'))) {
    res.writeHead(403)
    res.end('Forbidden')
    return
  }

  const extname = path.extname(filePath)
  let contentType = 'text/html'

  switch (extname) {
    case '.js': contentType = 'text/javascript'; break
    case '.css': contentType = 'text/css'; break
    case '.json': contentType = 'application/json'; break
    case '.png': contentType = 'image/png'; break
    case '.jpg': contentType = 'image/jpg'; break
  }

  fs.readFile(filePath, (err, content) => {
    if (err) {
      if (err.code === 'ENOENT') {
        res.writeHead(404)
        res.end('404 Not Found')
      } else {
        res.writeHead(500)
        res.end('Server Error: ' + err.code)
      }
    } else {
      res.writeHead(200, { 'Content-Type': contentType })
      res.end(content, 'utf-8')
    }
  })
})

async function proxyToSupernode (req, res, targetUrl) {
  // Simple proxy logic
  const url = new URL(req.url, targetUrl) // e.g. http://supernode:8080/api/...
  const options = {
    method: req.method,
    headers: { ...req.headers, host: new URL(targetUrl).host }
  }

  return new Promise((resolve, reject) => {
    const proxyReq = http.request(url, options, (proxyRes) => {
      res.writeHead(proxyRes.statusCode, proxyRes.headers)
      proxyRes.pipe(res)
      proxyRes.on('end', resolve)
    })

    proxyReq.on('error', (err) => {
      res.writeHead(502, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ error: 'Supernode Proxy Error: ' + err.message }))
      resolve()
    })

    req.pipe(proxyReq)
  })
}

async function handleApi (req, res) {
  const url = new URL(req.url, `http://${req.headers.host}`)
  const route = url.pathname.replace('/api/', '')

  if (req.method === 'GET') {
    if (route === 'status') {
      const stats = blobStore.stats()
      res.writeHead(200, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({
        version: packageJson.version,
        network: blobNetwork ? 'active' : 'inactive',
        dht: dht ? 'ready' : 'initializing',
        storage: {
          blobs: stats.blobCount,
          size: stats.currentSize,
          max: stats.maxSize,
          utilization: stats.utilization
        },
        subscriptions: subscriptionStore.getAllSubscriptions().length,
        networkDetails: {
          peerCount: dht ? dht.nodes.toArray().length : 0,
          transports: {
            'DHT (UDP)': {
              status: dht ? 'Running' : 'Initializing',
              address: `0.0.0.0:${blobTracker ? blobTracker.port : 6881}`,
              connectionsIn: 0, // Not tracked in JS ref yet
              connectionsOut: dht ? dht.nodes.toArray().length : 0,
              bytesReceived: 0,
              bytesSent: 0,
              errors: 0
            },
            'Blob (TCP)': {
              status: blobNetwork ? 'Running' : 'Stopped',
              address: blobNetwork ? `0.0.0.0:${blobNetwork.port}` : '-',
              connectionsIn: 0,
              connectionsOut: 0,
              bytesReceived: 0,
              bytesSent: 0,
              errors: 0
            }
          }
        },
        storageDetails: {
          isoSize: 'N/A (Simple)',
          totalFilesIngested: filesIndex.length,
          totalBytesIngested: blobStore.stats().currentSize,
          erasure: null
        }
      }))
      return
    }

    if (route === 'subscriptions') {
      res.writeHead(200, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify(subscriptionStore.getAllSubscriptions()))
      return
    }

    if (route === 'peers') {
      const peers = []
      if (dht) {
        dht.nodes.toArray().forEach(node => {
          peers.push({
            id: node.id.toString('hex'),
            address: `${node.host}:${node.port}`,
            transport: 'DHT (UDP)',
            latency: 0, // Not tracked in basic DHT
            score: 0,
            packets: '0/0',
            status: 'Seen'
          })
        })
      }
      res.writeHead(200, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify(peers))
      return
    }

    if (route === 'blobs') {
      const blobs = blobStore.list().slice(0, 50) // Limit to 50
      res.writeHead(200, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify(blobs))
      return
    }

    if (route === 'channels/browse') {
      const topic = url.searchParams.get('topic') || ''
      await initNetwork()
      try {
        const result = await channel.browse(topic)
        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify(result))
      } catch (err) {
        res.writeHead(500)
        res.end(JSON.stringify({ error: err.message }))
      }
      return
    }

    if (route === 'wallet') {
      const balance = await walletManager.refreshBalance()
      res.writeHead(200, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({
        address: walletManager.getAddress(),
        balance,
        pending: 0,
        transactions: []
      }))
      return
    }

    if (route === 'files') {
      // Calculate progress for each file
      const files = filesIndex.map(f => {
        const totalBlobs = f.chunks.length
        const havingBlobs = f.chunks.filter(c => blobStore.has(c.blobId)).length
        return {
          name: f.name,
          size: f.size,
          progress: Math.round((havingBlobs / totalBlobs) * 100),
          status: havingBlobs === totalBlobs ? 'Complete' : 'Downloading',
          id: f.chunks[0].blobId // Use first blob as ID for now
        }
      })
      res.writeHead(200, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify(files))
      return
    }

    if (route.startsWith('files/') && route.endsWith('/health')) {
      const fileId = route.substring(6, route.length - 7)
      const fileEntry = filesIndex.find(f => f.chunks && f.chunks[0].blobId === fileId)

      if (!fileEntry) {
        res.writeHead(404, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({ error: 'File not found' }))
        return
      }

      const chunks = fileEntry.chunks.map((c, i) => {
        const present = blobStore.has(c.blobId)
        return {
          index: i,
          status: present ? 'Healthy' : 'Missing',
          shards: [] // No erasure coding in Node.js ref client yet
        }
      })

      const healthyCount = chunks.filter(c => c.status === 'Healthy').length
      const status = healthyCount === chunks.length ? 'Healthy' : (healthyCount > 0 ? 'Degraded' : 'Critical')

      res.writeHead(200, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({
        fileId,
        status,
        totalChunks: chunks.length,
        healthyChunks: healthyCount,
        erasure: null,
        chunks
      }))
      return
    }

    if (route.startsWith('stream/')) {
      const fileId = route.replace('stream/', '')
      const fileEntry = filesIndex.find(f => f.chunks && f.chunks[0].blobId === fileId)

      if (!fileEntry) {
        res.writeHead(404)
        res.end('File not found')
        return
      }

      // Detect Mime Type roughly
      let mime = 'application/octet-stream'
      if (fileEntry.name.endsWith('.mp4')) mime = 'video/mp4'
      if (fileEntry.name.endsWith('.webm')) mime = 'video/webm'
      if (fileEntry.name.endsWith('.mp3')) mime = 'audio/mpeg'

      const getBlobFn = async (id) => blobStore.has(id) ? blobStore.get(id) : null

      const range = req.headers.range
      const fileSize = fileEntry.size

      if (range) {
        const parts = range.replace(/bytes=/, '').split('-')
        const start = parseInt(parts[0], 10)
        const end = parts[1] ? parseInt(parts[1], 10) : fileSize - 1
        const chunksize = (end - start) + 1

        res.writeHead(206, {
          'Content-Range': `bytes ${start}-${end}/${fileSize}`,
          'Accept-Ranges': 'bytes',
          'Content-Length': chunksize,
          'Content-Type': mime
        })

        const stream = createReadStream(fileEntry, getBlobFn, { start, end })
        stream.pipe(res)
        stream.on('error', (err) => {
          console.error('Stream error:', err)
          if (!res.headersSent) res.writeHead(500)
          res.end()
        })
      } else {
        res.writeHead(200, {
          'Content-Length': fileSize,
          'Content-Type': mime
        })
        createReadStream(fileEntry, getBlobFn).pipe(res)
      }
      return
    }
  }

  if (req.method === 'POST') {
    const body = await parseBody(req)

    if (route === 'key/generate') {
      const keypair = generateKeypair()
      res.writeHead(200, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({
        publicKey: keypair.publicKey.toString('hex'),
        secretKey: keypair.secretKey.toString('hex')
      }))
      return
    }

    if (route === 'ingest') {
      // Handle both local path ingest and remote supernode proxy ingest
      // If body has 'data', it's a proxy ingest request (handled by proxyToSupernode usually,
      // but if we hit this locally, it means we are emulating supernode or just normal local ingest).
      // BUT, the UI logic for 'Local Node' sends { filePath: ... }.
      // The UI logic for 'Remote Node' proxies the request directly.
      // So here we only care about Local Node logic.

      const filePath = body.filePath

      // If we receive options (erasure), we should store them in metadata even if we don't use them yet
      const options = body.options || {}

      if (!filePath || !fs.existsSync(filePath)) {
        throw new Error('File not found')
      }

      // Security: Check file size
      const stats = fs.statSync(filePath)
      if (stats.size > 1024 * 1024 * 1024 * 2) { // 2GB limit
        throw new Error('File too large for Web UI ingest (max 2GB). Use CLI for larger files.')
      }

      console.log(`Ingesting ${filePath}...`)
      const fileBuf = fs.readFileSync(filePath)
      const result = ingest(fileBuf, path.basename(filePath))

      // Attach options to result for reference
      if (options.enableErasure) {
          result.fileEntry.erasure = {
              dataShards: options.dataShards,
              parityShards: options.parityShards,
              // Mark as fake/metadata-only for Reference Client
              simulated: true
          }
      }

      // Save blobs
      result.blobs.forEach(blob => {
        blobStore.put(blob.id, blob.buffer)
      })

      // Add to files index
      filesIndex.push(result.fileEntry)
      fs.writeFileSync(path.join(STORAGE_DIR, 'files.json'), JSON.stringify(filesIndex))

      res.writeHead(200, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({
        fileEntry: result.fileEntry,
        blobCount: result.blobs.length
      }))
      return
    }

    if (route === 'publish') {
      const { fileEntry, identity } = body
      if (!fileEntry || !identity) throw new Error('Missing fileEntry or identity')

      await initNetwork()

      const keypair = {
        publicKey: Buffer.from(identity.publicKey, 'hex'),
        secretKey: Buffer.from(identity.secretKey, 'hex')
      }

      const collections = [{
        title: 'Default Collection',
        items: [fileEntry]
      }]

      const sequence = Date.now()
      const manifest = createManifest(keypair, sequence, collections)

      console.log('Publishing manifest:', manifest)

      // Connect to tracker and publish
      const client = new Client({
        infoHash: Buffer.alloc(20),
        peerId: Buffer.alloc(20),
        announce: [TRACKER_URL],
        port: 6666
      })

      // Send publish message
      // Note: This logic mirrors the CLI but needs to wait for connection
      await new Promise((resolve, reject) => {
        client.on('error', (err) => {
          console.error('Tracker Error:', err)
          // Don't reject immediately, retry or wait
        })

        // Give it a moment to connect
        setTimeout(() => {
          const trackers = client._trackers
          let sent = false
          for (const tracker of trackers) {
            if (tracker.socket && tracker.socket.readyState === 1) {
              tracker.socket.send(JSON.stringify({
                action: 'publish',
                manifest
              }))
              sent = true
            }
          }
          client.destroy()
          if (sent) resolve()
          else reject(new Error('No connected tracker found'))
        }, 2000)
      })

      res.writeHead(200, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ status: 'published', manifest }))
      return
    }

    if (route === 'wallet/airdrop') {
      try {
        await walletManager.requestAirdrop()
        const balance = await walletManager.refreshBalance()
        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({ status: 'success', balance }))
      } catch (err) {
        res.writeHead(500)
        res.end(JSON.stringify({ error: err.message }))
      }
      return
    }

    if (route === 'subscribe') {
      const { publicKey } = body
      if (!publicKey) throw new Error('Missing public key')

      await initNetwork()
      subscriptionStore.addSubscription(publicKey, {})

      // Trigger subscription logic (similar to CLI loop)
      // For now, we just add it to the store. The background loop (if running) would pick it up.
      // We should probably start a background loop here if not started.
      startSubscriptionLoop()

      res.writeHead(200, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ status: 'subscribed', publicKey }))
      return
    }
  }

  res.writeHead(404)
  res.end('Not Found')
}

function parseBody (req) {
  return new Promise((resolve, reject) => {
    let body = ''
    req.on('data', chunk => { body += chunk.toString() })
    req.on('end', () => {
      try {
        resolve(body ? JSON.parse(body) : {})
      } catch (err) {
        reject(err)
      }
    })
    req.on('error', reject)
  })
}

// Subscription Loop (Simple version)
let loopStarted = false
function startSubscriptionLoop () {
  if (loopStarted) return
  loopStarted = true

  console.log('Starting subscription loop...')
  setInterval(async () => {
    const subs = subscriptionStore.getAllSubscriptions()
    if (subs.length === 0) return

    // Create a temporary client to poll trackers
    // In a real app, we'd keep a persistent connection
    const client = new Client({
      infoHash: Buffer.alloc(20),
      peerId: Buffer.alloc(20),
      announce: [TRACKER_URL],
      port: 0
    })

    client.on('error', () => {}) // Ignore errors

    setTimeout(() => {
      const trackers = client._trackers
      for (const tracker of trackers) {
        if (tracker.socket && tracker.socket.readyState === 1) {
          subs.forEach(sub => {
            // Send subscribe
            tracker.socket.send(JSON.stringify({
              action: 'subscribe',
              key: sub.topicPath // In this simplified version, topicPath is the public key
            }))
          })

          // Listen for updates
          tracker.socket.onmessage = (event) => {
            try {
              const data = JSON.parse(event.data)
              if (data.action === 'publish') {
                console.log('Received update for', data.manifest.publicKey)
                // Validate and store/download
                // For now, just log it
                subscriptionStore.updateLastSeq(data.manifest.publicKey, data.manifest.sequence)
              }
            } catch (e) {}
          }
        }
      }
    }, 1000)

    // Destroy client after a short poll to avoid leaking resources in this simple loop
    // A better approach would be a persistent service
    setTimeout(() => {
      client.destroy()
    }, 5000)
  }, 30000) // Poll every 30s
}

// Initialize
initNetwork().catch(console.error)

server.listen(PORT, '127.0.0.1', () => {
  console.log(`Web UI running at http://127.0.0.1:${PORT}`)
  console.log(`Storage: ${STORAGE_DIR}`)
  console.log(`Tracker: ${TRACKER_URL}`)
})
