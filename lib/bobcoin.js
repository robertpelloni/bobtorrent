// Megatorrent Integration for Bobcoin
// Accesses the 'bobcoin' submodule functionality

import fs from 'fs'
import path from 'path'
// We import directly from the submodule source code
// In production, this would be a dependency or compiled lib
import { BobcoinChain } from '../bobcoin/index.js'

export class BobcoinService {
    constructor(opts = {}) {
        this.enabled = opts.enabled || false
        this.nodePath = opts.nodePath || path.resolve('./bobcoin')
        this.chain = null
    }

    async start() {
        if (!this.enabled) return
        console.log('[Bobcoin] Starting Embedded Node...')

        try {
            this.chain = new BobcoinChain()
            console.log('[Bobcoin] Blockchain initialized. Height:', this.chain.getHeight())
        } catch (e) {
            console.error('[Bobcoin] Failed to initialize chain:', e.message)
        }
    }

    async getBalance(publicKey) {
        return 0 // Stub
    }

    // API for Arcades/DDR pads
    async submitDance(danceData) {
        if (!this.chain) throw new Error('Bobcoin not running')

        console.log('[Bobcoin] Mining attempt with dance data...')
        try {
            const block = this.chain.mineBlock(danceData, 'local_miner_wallet')
            console.log(`[Bobcoin] 💃 Block Mined! Hash: ${block.hash} Index: ${block.index}`)
            return block
        } catch (e) {
            console.error('[Bobcoin] Mining failed:', e.message)
            return null
        }
    }
}
