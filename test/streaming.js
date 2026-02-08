import test from 'tape'
import crypto from 'crypto'
import { ingest, createReadStream } from '../reference-client/lib/storage.js'
import { BlobStore } from '../reference-client/lib/blob-store.js'
import fs from 'fs'
import path from 'path'

const TMP_DIR = './test-storage-stream'

if (!fs.existsSync(TMP_DIR)) fs.mkdirSync(TMP_DIR)

test('Streaming: Range Requests', async (t) => {
    // 1. Create a 3MB file (spanning 3 chunks of 1MB)
    const size = 3 * 1024 * 1024 + 500 // 3MB + 500 bytes
    const buffer = crypto.randomBytes(size)

    // 2. Ingest
    const { fileEntry, blobs } = ingest(buffer, 'test.bin')
    t.equal(fileEntry.chunks.length, 4, 'Should have 4 chunks')

    // 3. Mock Blob Store
    const store = new BlobStore(TMP_DIR)
    for (const blob of blobs) {
        store.put(blob.id, blob.buffer)
    }

    const getBlobFn = async (id) => store.get(id)

    // 4. Test Full Stream
    const stream = createReadStream(fileEntry, getBlobFn)
    const chunks = []
    for await (const chunk of stream) {
        chunks.push(chunk)
    }
    const resultFull = Buffer.concat(chunks)
    t.equal(resultFull.length, size, 'Full stream length match')
    t.ok(resultFull.equals(buffer), 'Full stream content match')

    // 5. Test Range (Middle of 1st chunk to Middle of 2nd chunk)
    // Chunk 1: 0 - 1MB
    // Chunk 2: 1MB - 2MB
    // Request: 0.5MB to 1.5MB
    const start = 512 * 1024
    const end = (1.5 * 1024 * 1024) - 1
    const expectedLength = end - start + 1

    const streamRange = createReadStream(fileEntry, getBlobFn, { start, end })
    const rangeChunks = []
    for await (const chunk of streamRange) {
        rangeChunks.push(chunk)
    }
    const resultRange = Buffer.concat(rangeChunks)

    t.equal(resultRange.length, expectedLength, 'Range stream length match')
    t.ok(resultRange.equals(buffer.slice(start, end + 1)), 'Range stream content match')

    // 6. Test Range (Small slice in last chunk)
    const lastStart = size - 100
    const lastEnd = size - 1
    const streamLast = createReadStream(fileEntry, getBlobFn, { start: lastStart, end: lastEnd })
    const lastChunks = []
    for await (const chunk of streamLast) {
        lastChunks.push(chunk)
    }
    const resultLast = Buffer.concat(lastChunks)
    t.ok(resultLast.equals(buffer.slice(lastStart, lastEnd + 1)), 'Last chunk slice match')

    // Cleanup
    fs.rmSync(TMP_DIR, { recursive: true, force: true })
    t.end()
})
