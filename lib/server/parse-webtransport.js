/**
 * WebTransport Request Parser for Bobtorrent Tracker
 *
 * Parses incoming JSON messages from WebTransport bidirectional streams
 * and datagrams into the unified tracker request format consumed by _onRequest.
 *
 * WebTransport (RFC 9220) uses HTTP/3 (QUIC) to provide:
 *   - Ultra-low-latency, multiplexed streams
 *   - Built-in TLS 1.3 encryption
 *   - 0-RTT connection establishment
 *   - Unreliable datagram support for announce keepalives
 *
 * This parser follows the same contract as parse-websocket.js:
 *   Input:  { session, remoteAddress, remotePort } + raw JSON string
 *   Output: normalized params object for _onRequest pipeline
 */

import { bin2hex } from 'uint8-util'
import common from '../common.js'

/**
 * Parse a WebTransport message into tracker request params.
 *
 * @param {Object} session  - WebTransport session wrapper with { send, ip, port, addr, headers }
 * @param {Object} opts     - Server options (e.g. { trustProxy })
 * @param {string} rawData  - Raw JSON string from the WebTransport stream/datagram
 * @returns {Object}        - Normalized params for _onRequest
 */
export default function parseWebTransportRequest (session, opts, rawData) {
  if (!opts) opts = {}

  const params = JSON.parse(rawData) // may throw on malformed JSON

  // Tag the transport type for swarm accounting
  params.type = 'wt'
  params.socket = session

  if (params.action === 'announce') {
    params.action = common.ACTIONS.ANNOUNCE

    if (typeof params.info_hash !== 'string' || params.info_hash.length !== 20) {
      throw new Error('invalid info_hash')
    }
    params.info_hash = bin2hex(params.info_hash)

    if (typeof params.peer_id !== 'string' || params.peer_id.length !== 20) {
      throw new Error('invalid peer_id')
    }
    params.peer_id = bin2hex(params.peer_id)

    if (params.answer) {
      if (typeof params.to_peer_id !== 'string' || params.to_peer_id.length !== 20) {
        throw new Error('invalid `to_peer_id` (required with `answer`)')
      }
      params.to_peer_id = bin2hex(params.to_peer_id)
    }

    params.left = Number(params.left)
    if (Number.isNaN(params.left)) params.left = Infinity

    params.numwant = Math.min(
      Number(params.offers && params.offers.length) || 0,
      common.MAX_ANNOUNCE_PEERS
    )
    params.compact = -1 // return full peer objects (same as WebSocket)
  } else if (params.action === 'scrape') {
    params.action = common.ACTIONS.SCRAPE

    if (typeof params.info_hash === 'string') params.info_hash = [params.info_hash]
    if (Array.isArray(params.info_hash)) {
      params.info_hash = params.info_hash.map(binaryInfoHash => {
        if (typeof binaryInfoHash !== 'string' || binaryInfoHash.length !== 20) {
          throw new Error('invalid info_hash')
        }
        return bin2hex(binaryInfoHash)
      })
    }
  } else if (params.action === 'publish') {
    if (!params.manifest) throw new Error('missing manifest')
  } else if (params.action === 'subscribe') {
    if (!params.key) throw new Error('missing key')
  } else {
    throw new Error(`invalid action in WebTransport request: ${params.action}`)
  }

  // Attach connection metadata from the session
  params.ip = session.ip
  params.port = session.port
  params.addr = session.addr
  params.headers = session.headers || {}

  return params
}
