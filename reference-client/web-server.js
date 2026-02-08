import fs from 'fs'
import path from 'path'
import http from 'http'
import { fileURLToPath } from 'url'
import { generateKeypair } from './lib/crypto.js'
import { createManifest } from './lib/manifest.js'
import { ingest, reassemble, createBlobClient } from './lib/storage.js'
import { BlobStore } from './lib/blob-store.js'
import { BlobNetwork } from './lib/blob-network.js'
import { BlobTracker } from './lib/blob-tracker.js'
import { SubscriptionStore } from './lib/subscription-store.js'
import { createChannel } from './lib/channels.js'
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
let blobStore = new BlobStore(STORAGE_DIR)
let blobNetwork = null
let blobTracker = null
let blobClient = null
let channel = null
let subscriptionStore = new SubscriptionStore(SUBSCRIPTIONS_FILE)
let filesIndex = [] // In-memory file index (persist to file in future)

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

  blobClient = createBlobClient(blobStore, blobNetwork, blobTracker)

  // Load subscriptions
  subscriptionStore.load()

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
  const parsedUrl = new URL(req.url, `http://${req.headers.host}`);
  const pathname = parsedUrl.pathname;
  const safeUrl = path.normalize(pathname).replace(/^(\.\.[\/\\])+/, '');
  let filePath = path.join(__dirname, 'web-ui', safeUrl === '/' || safeUrl === '.' ? 'index.html' : safeUrl);

  // Security: Ensure we stay within web-ui directory
  if (!filePath.startsWith(path.join(__dirname, 'web-ui'))) {
    res.writeHead(403);
    res.end('Forbidden');
    return;
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
        subscriptions: subscriptionStore.getAllSubscriptions().length
      }))
      return
    }

    if (route === 'subscriptions') {
      res.writeHead(200, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify(subscriptionStore.getAllSubscriptions()))
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
      const filePath = body.filePath
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
function startSubscriptionLoop() {
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
