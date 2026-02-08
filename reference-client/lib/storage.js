import crypto from 'crypto'
import { EventEmitter } from 'events'

const CHUNK_SIZE = 1024 * 1024
const ALGORITHM = 'aes-256-gcm'
const KEY_SIZE = 32 // 256 bits
const NONCE_SIZE = 12 // 96 bits

function sha256 (buffer) {
  const hash = crypto.createHash('sha256')
  hash.update(buffer)
  return hash.digest('hex')
}

export function ingest (fileBuffer, fileName) {
  const totalSize = fileBuffer.length
  const chunks = []
  const blobs = []

  let offset = 0
  while (offset < totalSize) {
    const end = Math.min(offset + CHUNK_SIZE, totalSize)
    const chunkData = fileBuffer.slice(offset, end)

    // Generate random key
    const key = crypto.randomBytes(KEY_SIZE)

    // To match Java MuxEngine, we use a zero nonce for determinism per key
    // (Note: Java generates a new key per session/file, or per chunk?
    // MuxEngine encrypt(plaintext, key) uses zero nonce.
    // Here we generate a UNIQUE random key for EVERY chunk, so zero nonce is safe.)
    const nonce = Buffer.alloc(NONCE_SIZE) // Zeros

    const cipher = crypto.createCipheriv(ALGORITHM, key, nonce)
    const encrypted = Buffer.concat([cipher.update(chunkData), cipher.final()])
    const tag = cipher.getAuthTag()

    // Java MuxEngine format: Nonce (12) + Ciphertext (N) + Tag (16)
    // Wait, MuxEngine.java:
    // ByteBuffer result = ByteBuffer.allocate(NONCE_SIZE + ciphertext.length); // GCM ciphertext includes tag usually in Java?
    // Java doFinal() returns ciphertext + tag appended.
    // So result = Nonce + Ciphertext + Tag.

    const blobBuffer = Buffer.concat([nonce, encrypted, tag])
    const blobId = sha256(blobBuffer)

    blobs.push({
      id: blobId,
      buffer: blobBuffer
    })

    chunks.push({
      blobId,
      offset: 0,
      length: blobBuffer.length,
      key: key.toString('hex'),
      nonce: nonce.toString('hex')
    })

    offset = end
  }

  return {
    fileEntry: {
      name: fileName,
      size: totalSize,
      chunks
    },
    blobs
  }
}

export async function reassemble (fileEntry, getBlobFn) {
  const parts = []

  for (const chunkMeta of fileEntry.chunks) {
    const blobBuffer = await getBlobFn(chunkMeta.blobId)
    if (!blobBuffer) throw new Error(`Blob ${chunkMeta.blobId} not found`)

    // Extract parts based on Java format: Nonce (12) + Ciphertext + Tag (16)
    const nonce = blobBuffer.slice(chunkMeta.offset, chunkMeta.offset + NONCE_SIZE)
    const encryptedWithTag = blobBuffer.slice(chunkMeta.offset + NONCE_SIZE, chunkMeta.offset + chunkMeta.length)

    const authTag = encryptedWithTag.slice(encryptedWithTag.length - 16)
    const ciphertext = encryptedWithTag.slice(0, encryptedWithTag.length - 16)

    const key = Buffer.from(chunkMeta.key, 'hex')
    // We expect the stored nonce to match the one in metadata (if stored), or we just use the one in the blob
    // Java MuxEngine puts nonce at start of blob.

    const decipher = crypto.createDecipheriv(ALGORITHM, key, nonce)
    decipher.setAuthTag(authTag)

    try {
      const plaintext = Buffer.concat([decipher.update(ciphertext), decipher.final()])
      parts.push(plaintext)
    } catch (err) {
      throw new Error(`Decryption failed for blob ${chunkMeta.blobId}: ${err.message}`)
    }
  }

  return Buffer.concat(parts)
}

export class BlobClient extends EventEmitter {
  constructor (options = {}) {
    super()
    this.store = options.store
    this.network = options.network
    this.tracker = options.tracker
    this.maxParallel = options.maxParallel || 5
  }

  async seed (fileEntry) {
    const blobIds = fileEntry.chunks.map(c => c.blobId)
    
    for (const blobId of blobIds) {
      if (this.store.has(blobId)) {
        this.tracker.announce(blobId)
        this.network.announceBlob(blobId)
      }
    }
    
    this.emit('seeding', { blobIds })
    return blobIds
  }

  async fetch (fileEntry, options = {}) {
    const onProgress = options.onProgress || (() => {})
    const chunks = fileEntry.chunks
    const total = chunks.length
    let completed = 0
    
    const getBlobFn = async (blobId) => {
      if (this.store.has(blobId)) {
        return this.store.get(blobId)
      }
      
      const peers = await this.tracker.lookup(blobId)
      
      for (const peer of peers) {
        try {
          await this.network.connect(peer.address)
        } catch {
          continue
        }
      }
      
      this.network.queryBlob(blobId)
      
      await new Promise(resolve => setTimeout(resolve, 500))
      
      const blob = await this.network.requestBlob(blobId)
      
      completed++
      onProgress({ completed, total, blobId })
      
      return blob
    }
    
    return reassemble(fileEntry, getBlobFn)
  }

  async storeBlobs (blobs) {
    for (const blob of blobs) {
      this.store.put(blob.id, blob.buffer)
    }
  }
}

export function createBlobClient (store, network, tracker) {
  return new BlobClient({ store, network, tracker })
}
