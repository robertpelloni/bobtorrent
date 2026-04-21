import crypto from 'crypto'
import { EventEmitter } from 'events'
import { Readable } from 'stream'

const CHUNK_SIZE = 1024 * 1024
const ALGORITHM = 'aes-256-gcm'
const KEY_SIZE = 32 // 256 bits
const NONCE_SIZE = 12 // 96 bits
const OVERHEAD_SIZE = NONCE_SIZE + 16 // Tag size is 16

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

export function createReadStream (fileEntry, getBlobFn, options = {}) {
  const start = options.start || 0
  const end = options.end || (fileEntry.size - 1)
  const readahead = options.readahead || 3 // Number of chunks to pre-fetch

  let cursor = start
  let currentChunkIndex = 0
  let chunkStartOffset = 0

  // Fast-forward to start chunk
  for (let i = 0; i < fileEntry.chunks.length; i++) {
    const len = fileEntry.chunks[i].length - OVERHEAD_SIZE
    if (cursor < chunkStartOffset + len) {
      currentChunkIndex = i
      break
    }
    chunkStartOffset += len
  }

  return new Readable({
    async read (size) {
      if (cursor > end) {
        this.push(null)
        return
      }

      if (currentChunkIndex >= fileEntry.chunks.length) {
        this.push(null)
        return
      }

      // Trigger predictive readahead
      for (let i = 1; i <= readahead; i++) {
        const nextIndex = currentChunkIndex + i
        if (nextIndex < fileEntry.chunks.length) {
          const nextChunk = fileEntry.chunks[nextIndex]
          // We fire-and-forget the fetch request. The underlying blob store/network logic
          // should handle deduplication of in-flight requests.
          // Since getBlobFn is async, we don't await it here to avoid blocking the current read.
          // This relies on getBlobFn caching the result or the network layer handling it.
          getBlobFn(nextChunk.blobId).catch(() => {})
        }
      }

      const chunkMeta = fileEntry.chunks[currentChunkIndex]
      const plaintextSize = chunkMeta.length - OVERHEAD_SIZE
      const chunkEndOffset = chunkStartOffset + plaintextSize - 1

      const sliceStart = cursor
      const sliceEnd = Math.min(end, chunkEndOffset)

      const relStart = sliceStart - chunkStartOffset
      const relEnd = sliceEnd - chunkStartOffset

      try {
        const blobBuffer = await getBlobFn(chunkMeta.blobId)
        if (!blobBuffer) {
          this.destroy(new Error(`Blob ${chunkMeta.blobId} missing`))
          return
        }

        const nonce = blobBuffer.slice(chunkMeta.offset, chunkMeta.offset + NONCE_SIZE)
        const encryptedWithTag = blobBuffer.slice(chunkMeta.offset + NONCE_SIZE, chunkMeta.offset + chunkMeta.length)
        const authTag = encryptedWithTag.slice(encryptedWithTag.length - 16)
        const ciphertext = encryptedWithTag.slice(0, encryptedWithTag.length - 16)
        const key = Buffer.from(chunkMeta.key, 'hex')

        const decipher = crypto.createDecipheriv(ALGORITHM, key, nonce)
        decipher.setAuthTag(authTag)
        const plaintext = Buffer.concat([decipher.update(ciphertext), decipher.final()])

        const data = plaintext.slice(relStart, relEnd + 1)
        this.push(data)

        cursor += data.length

        if (cursor > chunkEndOffset) {
          currentChunkIndex++
          chunkStartOffset += plaintextSize
        }
      } catch (err) {
        this.destroy(err)
      }
    }
  })
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
