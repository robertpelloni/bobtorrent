import { test } from 'node:test'
import assert from 'node:assert'
import Server from '../server.js'

// Regression: listen() referenced an out-of-scope `onListening` when
// WebTransport was enabled, throwing ReferenceError synchronously.
// With the http3 module absent the starter must degrade gracefully
// (warning + listening) instead of crashing.
test('Server.listen with WebTransport enabled does not throw', async () => {
  const server = new Server({ webtransport: { port: 0 }, udp: false, ws: false })
  await new Promise((resolve, reject) => {
    server.on('warning', () => {})
    server.listen(0, () => resolve())
    setTimeout(() => reject(new Error('listening callback never fired')), 5000)
  })
  assert.strictEqual(server.listening, true)
  await new Promise(resolve => server.close(resolve))
})
