import net from 'net'
import fs from 'fs'
import path from 'path'
import sodium from 'sodium-native'
import { SocksClient } from 'socks'

// Global Proxy Config
let globalProxy = null
export function setGlobalProxy (proxyUrl) {
  if (!proxyUrl) return
  const url = new URL(proxyUrl)
  globalProxy = {
    host: url.hostname,
    port: parseInt(url.port),
    type: 5 // SOCKS5
  }
}

function encryptStream (socket, isServer) {
  const ephemeral = {
    publicKey: Buffer.alloc(sodium.crypto_box_PUBLICKEYBYTES),
    secretKey: Buffer.alloc(sodium.crypto_box_SECRETKEYBYTES)
  }
  sodium.crypto_box_keypair(ephemeral.publicKey, ephemeral.secretKey)

  let sharedRx = null
  let sharedTx = null
  const nonceRx = Buffer.alloc(sodium.crypto_secretbox_NONCEBYTES)
  const nonceTx = Buffer.alloc(sodium.crypto_secretbox_NONCEBYTES)

  const pendingWrites = []
  const label = isServer ? 'SRV' : 'CLI'

  const flushWrites = () => {
    if (!sharedTx) return
    while (pendingWrites.length > 0) {
      const { chunk, encoding, cb } = pendingWrites.shift()
      writeEncrypted(chunk, encoding, cb)
    }
  }

  const writeEncrypted = (chunk, encoding, cb) => {
    const buf = Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk, encoding)
    const cipher = Buffer.alloc(buf.length + sodium.crypto_secretbox_MACBYTES)
    sodium.crypto_secretbox_easy(cipher, buf, nonceTx, sharedTx)
    sodium.sodium_increment(nonceTx)
    const len = Buffer.alloc(2)
    len.writeUInt16BE(cipher.length)
    socket.write(Buffer.concat([len, cipher]), cb)
  }

  return {
    write: (chunk, encoding, cb) => {
      if (!sharedTx) {
        pendingWrites.push({ chunk, encoding, cb })
      } else {
        writeEncrypted(chunk, encoding, cb)
      }
    },

    handshake: () => new Promise((resolve, reject) => {
      socket.write(ephemeral.publicKey)

      let buffer = Buffer.alloc(0)
      const onData = (data) => {
        buffer = Buffer.concat([buffer, data])
        if (buffer.length >= 32) {
          const remotePub = buffer.slice(0, 32)
          buffer = buffer.slice(32)

          const sharedPoint = Buffer.alloc(sodium.crypto_scalarmult_BYTES)
          sodium.crypto_scalarmult(sharedPoint, ephemeral.secretKey, remotePub)

          const kdf = (salt) => {
            const out = Buffer.alloc(sodium.crypto_secretbox_KEYBYTES)
            const saltBuf = Buffer.from(salt)
            sodium.crypto_generichash(out, Buffer.concat([sharedPoint, saltBuf]))
            return out
          }

          if (isServer) {
            sharedTx = kdf('S')
            sharedRx = kdf('C')
          } else {
            sharedTx = kdf('C')
            sharedRx = kdf('S')
          }

          socket.removeListener('data', onData)

          socket.on('data', (d) => {
            socket.emit('encrypted-data', d)
          })

          if (buffer.length > 0) {
            socket.emit('encrypted-data', buffer)
          }

          flushWrites()
          resolve()
        }
      }
      socket.on('data', onData)
    }),

    onDecrypted: (cb) => {
      let internalBuf = Buffer.alloc(0)
      socket.on('encrypted-data', (data) => {
        internalBuf = Buffer.concat([internalBuf, data])

        while (true) {
          if (internalBuf.length < 2) break
          const len = internalBuf.readUInt16BE(0)
          if (internalBuf.length < 2 + len) break

          const frame = internalBuf.slice(2, 2 + len)
          internalBuf = internalBuf.slice(2 + len)

          const plain = Buffer.alloc(frame.length - sodium.crypto_secretbox_MACBYTES)
          const success = sodium.crypto_secretbox_open_easy(plain, frame, nonceRx, sharedRx)
          sodium.sodium_increment(nonceRx)

          if (!success) {
            console.error(`[${label}] Decryption failed`)
            socket.destroy()
            return
          }
          cb(plain)
        }
      })
    }
  }
}

export function startSecureServer (storageDir, port = 0) {
  const server = net.createServer(async socket => {
    const secure = encryptStream(socket, true)
    try {
      await secure.handshake()

      secure.onDecrypted(plain => {
        const request = plain.toString().trim()
        if (request.startsWith('GET ')) {
          const blobId = request.split(' ')[1]
          const filePath = path.join(storageDir, blobId)

          if (fs.existsSync(filePath)) {
            const data = fs.readFileSync(filePath)
            secure.write(data)
            setTimeout(() => socket.end(), 500)
          } else {
            secure.write('ERROR: Not Found')
            setTimeout(() => socket.end(), 500)
          }
        }
      })
    } catch (e) {
      socket.destroy()
    }
  })
  server.listen(port)
  if (server.address()) server.port = server.address().port
  else server.port = port
  return server
}

export function downloadSecureBlob (peer, blobId) {
  return new Promise((resolve, reject) => {
    const [host, portStr] = peer.split(':')
    const port = parseInt(portStr)

    const connect = async () => {
      let socket
      try {
        if (globalProxy) {
          const info = await SocksClient.createConnection({
            proxy: globalProxy,
            command: 'connect',
            destination: {
              host,
              port
            }
          })
          socket = info.socket
        } else {
          socket = new net.Socket()
          await new Promise((resolveConnect, rejectConnect) => {
            socket.connect(port, host, resolveConnect)
            socket.on('error', rejectConnect)
          })
          socket.removeAllListeners('error')
        }
      } catch (e) {
        return reject(new Error('Connection failed: ' + e.message))
      }

      const secure = encryptStream(socket, false)
      const chunks = []

      const cleanup = () => {
        socket.destroy()
      }

      socket.on('error', (err) => {
        reject(err)
      })

      socket.on('close', () => {
        const buffer = Buffer.concat(chunks)
        if (buffer.length === 0) reject(new Error('Empty response'))
        else if (buffer.toString().startsWith('ERROR')) reject(new Error('Peer Error'))
        else resolve(buffer)
      })

      try {
        await secure.handshake()
        secure.write(`GET ${blobId}`)
      } catch (e) {
        cleanup()
        return reject(e)
      }

      secure.onDecrypted(data => {
        chunks.push(data)
      })

      setTimeout(() => cleanup(), 10000)
    }

    connect().catch(reject)
  })
}
