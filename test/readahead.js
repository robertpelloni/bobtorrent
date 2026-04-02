import test from 'tape'
import crypto from 'crypto'
import { ingest, createReadStream } from '../reference-client/lib/storage.js'
import { BlobStore } from '../reference-client/lib/blob-store.js'
import fs from 'fs'
import path from 'path'

const TMP_DIR = './test-readahead'

if (!fs.existsSync(TMP_DIR)) fs.mkdirSync(TMP_DIR)

test('Predictive Readahead', async (t) => {
    // 1. Create a file spanning 5 chunks (1MB each approx)
    const size = 5 * 1024 * 1024
    const buffer = crypto.randomBytes(size)
    const { fileEntry, blobs } = ingest(buffer, 'test.bin')

    // 2. Mock Blob Fetcher
    const requestedBlobs = []
    const getBlobFn = async (id) => {
        requestedBlobs.push(id)
        const found = blobs.find(b => b.id === id)
        return found ? found.buffer : null
    }

    // 3. Create stream with readahead = 2
    // Reading chunk 0 should trigger fetch for 0, 1, 2
    const stream = createReadStream(fileEntry, getBlobFn, { readahead: 2 })

    // Read first chunk only
    const reader = stream[Symbol.asyncIterator]()
    await reader.next()

    // Verify
    // We expect requests for chunk 0 (to satisfy read) AND chunk 1, 2 (readahead)
    // Note: Readahead is async/fire-and-forget, so we might need a tiny delay
    await new Promise(r => setTimeout(r, 50))

    t.ok(requestedBlobs.includes(fileEntry.chunks[0].blobId), 'Chunk 0 requested')
    t.ok(requestedBlobs.includes(fileEntry.chunks[1].blobId), 'Chunk 1 requested (readahead)')
    t.ok(requestedBlobs.includes(fileEntry.chunks[2].blobId), 'Chunk 2 requested (readahead)')
    t.notOk(requestedBlobs.includes(fileEntry.chunks[3].blobId), 'Chunk 3 NOT requested')

    t.end()
})
