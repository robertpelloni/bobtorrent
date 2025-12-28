import DHT from 'bittorrent-dht'
import crypto from 'crypto'

const dht = new DHT()
const nodeId = crypto.randomBytes(32).toString('hex') // 64 chars

console.log(`Testing with 32-byte ID: ${nodeId}`)

try {
    dht.announce(nodeId, 1234, (err) => {
        if (err) console.error('Announce Error:', err.message)
        else console.log('Announce Success (Callback called)')
    })

    // Also try lookup
    dht.lookup(nodeId, (err) => {
        if (err) console.error('Lookup Error:', err.message)
        else console.log('Lookup Success (Callback called)')
    })

} catch (e) {
    console.error('Crash:', e.message)
}

setTimeout(() => {
    dht.destroy()
}, 1000)
