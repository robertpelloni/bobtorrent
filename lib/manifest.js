import stringify from 'fast-json-stable-stringify'
import sodium from 'sodium-native'

export function verify (message, signature, publicKey) {
  const msgBuffer = Buffer.isBuffer(message) ? message : Buffer.from(message)
  return sodium.crypto_sign_verify_detached(signature, msgBuffer, publicKey)
}

// Create Signed Manifest (Optional Encryption)
export function createManifest (keypair, sequence, collections, readKeyHex = null) {
  let content = {
    collections,
    timestamp: Date.now()
  }

  // Encrypt content if readKey provided
  if (readKeyHex) {
    const readKey = Buffer.from(readKeyHex, 'hex')
    const nonce = Buffer.alloc(sodium.crypto_secretbox_NONCEBYTES)
    sodium.randombytes_buf(nonce)

    const plaintext = Buffer.from(stringify(content))
    const cipher = Buffer.alloc(plaintext.length + sodium.crypto_secretbox_MACBYTES)
    sodium.crypto_secretbox_easy(cipher, plaintext, nonce, readKey)

    content = {
      encrypted: true,
      nonce: nonce.toString('hex'),
      ciphertext: cipher.toString('hex')
    }
  }

  const manifest = {
    publicKey: keypair.publicKey.toString('hex'),
    sequence,
    ...content
  }

  // Sign canonical JSON (keys sorted)
  const canonical = Buffer.from(stringify(manifest))

  const signature = Buffer.alloc(sodium.crypto_sign_BYTES)
  sodium.crypto_sign_detached(signature, canonical, keypair.secretKey)

  manifest.signature = signature.toString('hex')
  return manifest
}

export function validateManifest (manifest) {
  if (!manifest || typeof manifest !== 'object') throw new Error('Invalid manifest')
  if (!manifest.publicKey || !manifest.signature) throw new Error('Missing keys')

  // Validation
  if (typeof manifest.publicKey !== 'string' || !/^[0-9a-fA-F]{64}$/.test(manifest.publicKey)) {
    throw new Error('Invalid public key format')
  }
  if (typeof manifest.signature !== 'string' || !/^[0-9a-fA-F]{128}$/.test(manifest.signature)) {
    throw new Error('Invalid signature format')
  }
  if (typeof manifest.sequence !== 'number') throw new Error('Invalid sequence')

  if (!manifest.encrypted) {
    if (typeof manifest.timestamp !== 'number') throw new Error('Invalid timestamp')
    if (!Array.isArray(manifest.collections)) throw new Error('Invalid collections')
  }

  // Reconstruct the payload to verify (exclude signature)
  const clean = { ...manifest }
  delete clean.signature
  const jsonString = stringify(clean)
  const publicKey = Buffer.from(manifest.publicKey, 'hex')
  const signature = Buffer.from(manifest.signature, 'hex')

  return verify(jsonString, signature, publicKey)
}

export function decryptManifest (manifest, readKeyHex) {
  if (!manifest.encrypted) return manifest
  if (!readKeyHex) throw new Error('Manifest is encrypted but no key provided')

  const readKey = Buffer.from(readKeyHex, 'hex')
  const nonce = Buffer.from(manifest.nonce, 'hex')
  const cipher = Buffer.from(manifest.ciphertext, 'hex')
  const plain = Buffer.alloc(cipher.length - sodium.crypto_secretbox_MACBYTES)

  if (!sodium.crypto_secretbox_open_easy(plain, cipher, nonce, readKey)) {
    throw new Error('Failed to decrypt manifest')
  }

  const content = JSON.parse(plain.toString())
  return {
    publicKey: manifest.publicKey,
    sequence: manifest.sequence,
    ...content
  }
}
