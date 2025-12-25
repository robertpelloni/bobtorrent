import { spawn, execSync } from 'child_process'
import path from 'path'
import fs from 'fs'

const ROOT = process.cwd()
const NODE = process.argv[0]
const INDEX = path.join(ROOT, 'index.js')

const DATA_A = path.join(ROOT, 'data/node-a')
const DATA_B = path.join(ROOT, 'data/node-b')
const DATA_C = path.join(ROOT, 'data/node-c')

const dirs = [DATA_A, DATA_B, DATA_C]
dirs.forEach(dir => {
  if (fs.existsSync(dir)) fs.rmSync(dir, { recursive: true, force: true })
  fs.mkdirSync(dir, { recursive: true })
})

function spawnNode (name, dir, port, p2pPort, bootstrap = null) {
  const args = ['serve', '--dir', dir, '--port', port, '--p2p-port', p2pPort]
  if (bootstrap) args.push('--bootstrap', bootstrap)

  console.log(`[${name}] Starting on RPC ${port}, P2P ${p2pPort}...`)
  const proc = spawn(NODE, [INDEX, ...args], {
    env: { ...process.env, DEBUG: '' },
    stdio: ['ignore', 'pipe', 'pipe']
  })

  proc.stdout.on('data', d => console.log(`[${name}] ${d.toString().trim()}`))
  proc.stderr.on('data', d => console.error(`[${name}] ERR: ${d.toString().trim()}`))

  return proc
}

async function sleep (ms) { return new Promise(resolve => setTimeout(resolve, ms)) }

async function run () {
  console.log('>>> STARTING NETWORK SIMULATION <<<')

  // 1. Start Nodes A, B, C
  const nodeA = spawnNode('Node A', DATA_A, '3001', '4001')
  await sleep(2000)
  const nodeB = spawnNode('Node B', DATA_B, '3002', '4002', '127.0.0.1:4001')
  const nodeC = spawnNode('Node C', DATA_C, '3003', '4003', '127.0.0.1:4001')
  await sleep(3000) // Allow DHT bootstrap

  // 2. Generate Content on B
  console.log('\n>>> NODE B: Ingesting <<<')
  const keyFile = path.join(DATA_B, 'identity.json')
  execSync(`${NODE} ${INDEX} gen-key -k ${keyFile}`)
  const keyData = JSON.parse(fs.readFileSync(keyFile))
  console.log(`Publisher Key: ${keyData.publicKey}`)

  const dummyFile = path.join(DATA_B, 'video.mp4')
  fs.writeFileSync(dummyFile, Buffer.alloc(1024 * 1024 * 1, 'x'))

  // Use --json to get clean output
  const ingestJson = execSync(`${NODE} ${INDEX} ingest -i ${dummyFile} -d ${DATA_B} --json`).toString()
  const fileEntryPath = path.join(DATA_B, 'video.mp4.json')
  fs.writeFileSync(fileEntryPath, ingestJson)
  console.log('Ingested via CLI.')

  // Note: Since 'ingest --json' exits, Node B (Daemon) doesn't know about blobs in RAM.
  // But Node B (Daemon) checks disk?
  // No, `MegatorrentClient` doesn't scan disk automatically.
  // We need to tell Node B to seed these files.
  // BUT, I didn't implement 'reload' RPC.
  // Workaround: We restart Node B?
  // Or: We run `ingest` without --json in background to seed?

  // Actually, 'ingest' logic writes to disk.
  // If we restart Node B, it will load DHT state. But it won't load `heldBlobs` unless we scan?
  // `MegatorrentClient` in `lib/client.js` initializes `heldBlobs = new Set()`.
  // It does NOT scan disk.
  // This is a flaw.
  // Fix: I will add `scanStorage()` to `MegatorrentClient.start()`.

  console.log('Restarting Node B to pick up blobs...')
  nodeB.kill()
  await sleep(1000)
  const nodeB_restarted = spawnNode('Node B', DATA_B, '3002', '4002', '127.0.0.1:4001')
  await sleep(2000)

  // 3. Publish
  console.log('Publishing Manifest...')
  execSync(`${NODE} ${INDEX} publish -k ${keyFile} -i ${fileEntryPath} -d ${DATA_B} --bootstrap 127.0.0.1:4001`)

  // 4. Subscribe Node C
  console.log('\n>>> NODE C: Subscribing <<<')
  try {
    const res = await fetch('http://localhost:3003/api/rpc', {
      method: 'POST',
      body: JSON.stringify({
        method: 'addSubscription',
        params: { uri: `megatorrent://${keyData.publicKey}` }
      })
    })
    console.log('RPC Result:', await res.json())
  } catch (e) {
    console.error('RPC Failed:', e.message)
  }

  console.log('Waiting for transfer (30s)...')
  await sleep(30000)

  const downloadedFile = path.join(DATA_C, 'video.mp4')
  if (fs.existsSync(downloadedFile)) {
    console.log(`SUCCESS: File downloaded on Node C!`)
  } else {
    console.error('FAILURE: File not found on Node C.')
  }

  nodeA.kill()
  nodeB_restarted.kill()
  nodeC.kill()
  process.exit(0)
}

run()
