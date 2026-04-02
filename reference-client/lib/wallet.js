import { Keypair, Connection, LAMPORTS_PER_SOL, PublicKey } from '@solana/web3.js'
import fs from 'fs'
import path from 'path'

const DEVNET_URL = 'https://api.devnet.solana.com'

export class WalletManager {
  constructor (storageDir) {
    this.storageDir = storageDir
    this.walletPath = path.join(storageDir, 'wallet.json')
    this.connection = new Connection(DEVNET_URL, 'confirmed')
    this.keypair = null
    this.balance = 0
  }

  load () {
    if (fs.existsSync(this.walletPath)) {
      try {
        const data = JSON.parse(fs.readFileSync(this.walletPath))
        const secretKey = Uint8Array.from(Buffer.from(data.secretKey, 'hex'))
        this.keypair = Keypair.fromSecretKey(secretKey)
        return true
      } catch (err) {
        console.error('Failed to load wallet:', err)
      }
    }
    return false
  }

  create () {
    this.keypair = Keypair.generate()
    this.save()
    return this.keypair
  }

  save () {
    if (!this.keypair) return
    const data = {
      publicKey: this.keypair.publicKey.toBase58(),
      secretKey: Buffer.from(this.keypair.secretKey).toString('hex')
    }
    fs.writeFileSync(this.walletPath, JSON.stringify(data, null, 2))
  }

  getAddress () {
    return this.keypair ? this.keypair.publicKey.toBase58() : null
  }

  async refreshBalance () {
    if (!this.keypair) return 0
    try {
      this.balance = await this.connection.getBalance(this.keypair.publicKey)
      return this.balance / LAMPORTS_PER_SOL
    } catch (err) {
      console.error('Balance check failed:', err.message)
      return this.balance / LAMPORTS_PER_SOL // Return cached
    }
  }

  async requestAirdrop () {
    if (!this.keypair) throw new Error('No wallet')
    try {
      const signature = await this.connection.requestAirdrop(
        this.keypair.publicKey,
        1 * LAMPORTS_PER_SOL
      )
      await this.connection.confirmTransaction(signature)
      await this.refreshBalance()
      return true
    } catch (err) {
      console.error('Airdrop failed:', err)
      throw err
    }
  }
}
