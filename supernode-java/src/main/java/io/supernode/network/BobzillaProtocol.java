package io.supernode.network;

import java.io.*;
import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.time.Duration;
import java.time.Instant;
import java.util.*;
import java.util.concurrent.*;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicLong;
import java.util.function.Consumer;

/**
 * Bobzilla Client Protocol — Native wire protocol for cross-client P2P interoperability.
 *
 * Binary wire format:
 *   [MAGIC 4B][VERSION 1B][TYPE 1B][FLAGS 1B][RESERVED 1B][LENGTH 4B][PAYLOAD ...][CHECKSUM 4B]
 *
 * Total header: 12 bytes. Maximum payload: 16MB. CRC-32 checksum trailer.
 *
 * Message Types:
 *   HANDSHAKE(0x01)   — Client identity and capability negotiation
 *   BITFIELD(0x02)    — Announce which chunks are held
 *   REQUEST(0x03)     — Request a specific chunk
 *   PIECE(0x04)       — Deliver chunk data
 *   HAVE(0x05)        — Single chunk availability update
 *   CANCEL(0x06)      — Cancel a pending request
 *   CHOKE(0x07)       — Stop uploading to peer
 *   UNCHOKE(0x08)     — Resume uploading to peer
 *   INTERESTED(0x09)  — Indicate interest in peer's data
 *   NOT_INTERESTED(0x0A) — No longer interested
 *   KEEPALIVE(0x0B)   — Connection heartbeat
 *   MANIFEST(0x0C)    — Share a file manifest
 *   PROOF_CHALLENGE(0x0D) — Proof-of-seeding challenge
 *   PROOF_RESPONSE(0x0E)  — Proof-of-seeding response
 *   EXTENSION(0xFF)   — Protocol extension messages
 *
 * Features:
 *   - Version-negotiated handshake with capability flags
 *   - Efficient binary encoding (no JSON overhead)
 *   - CRC-32 integrity verification on every message
 *   - Extension system for third-party protocol additions
 */
public class BobzillaProtocol {

    // ==================== Constants ====================

    public static final byte[] MAGIC = {(byte) 0xB0, (byte) 0xB2, (byte) 0x11, (byte) 0xA0};
    public static final byte PROTOCOL_VERSION = 1;
    public static final int HEADER_SIZE = 12;
    public static final int CHECKSUM_SIZE = 4;
    public static final int MAX_PAYLOAD_SIZE = 16 * 1024 * 1024; // 16MB

    // ==================== Message Types ====================

    public enum MessageType {
        HANDSHAKE(0x01),
        BITFIELD(0x02),
        REQUEST(0x03),
        PIECE(0x04),
        HAVE(0x05),
        CANCEL(0x06),
        CHOKE(0x07),
        UNCHOKE(0x08),
        INTERESTED(0x09),
        NOT_INTERESTED(0x0A),
        KEEPALIVE(0x0B),
        MANIFEST(0x0C),
        PROOF_CHALLENGE(0x0D),
        PROOF_RESPONSE(0x0E),
        EXTENSION(0xFF);

        final int code;
        MessageType(int code) { this.code = code; }

        static MessageType fromCode(int code) {
            for (MessageType t : values()) {
                if (t.code == code) return t;
            }
            throw new IllegalArgumentException("Unknown message type: 0x" +
                Integer.toHexString(code));
        }
    }

    // ==================== Capability Flags ====================

    public static final int CAP_ENCRYPTION      = 1;       // AES-GCM support
    public static final int CAP_ERASURE_CODING   = 1 << 1; // Reed-Solomon
    public static final int CAP_DHT              = 1 << 2;  // DHT discovery
    public static final int CAP_PROOF_OF_SEEDING = 1 << 3;  // PoS challenges
    public static final int CAP_GAME_STREAMING   = 1 << 4;  // Game asset LOD
    public static final int CAP_MULTI_SWARM      = 1 << 5;  // Multi-swarm coordination
    public static final int CAP_TOR              = 1 << 6;  // Tor transport
    public static final int CAP_IPFS             = 1 << 7;  // IPFS bridging
    public static final int CAP_COMPRESSION      = 1 << 8;  // zstd compression
    public static final int CAP_EXTENDED_META    = 1 << 9;  // Extended metadata

    // ==================== Records ====================

    /** Raw wire message. */
    public record WireMessage(
        byte version,
        MessageType type,
        byte flags,
        byte[] payload,
        int checksum
    ) {}

    /** Handshake message payload. */
    public record Handshake(
        String peerId,
        String clientName,
        String clientVersion,
        int capabilities,
        int maxRequestSize,
        List<String> supportedTransports,
        byte[] infoHash
    ) {}

    /** Chunk request. */
    public record ChunkRequest(
        String fileId,
        int chunkIndex,
        int offset,
        int length
    ) {}

    /** Chunk piece (data delivery). */
    public record ChunkPiece(
        String fileId,
        int chunkIndex,
        int offset,
        byte[] data
    ) {}

    /** Protocol statistics. */
    public record ProtocolStats(
        long messagesSent,
        long messagesReceived,
        long bytesSent,
        long bytesReceived,
        long checksumErrors,
        long unknownMessages,
        Map<MessageType, Long> messageCounts
    ) {}

    // ==================== Codec — Encoder ====================

    /**
     * Encode a WireMessage to bytes for transmission.
     */
    public static byte[] encode(WireMessage message) {
        int payloadLen = message.payload() != null ? message.payload().length : 0;
        ByteBuffer buf = ByteBuffer.allocate(HEADER_SIZE + payloadLen + CHECKSUM_SIZE);
        buf.order(ByteOrder.BIG_ENDIAN);

        // Header
        buf.put(MAGIC);
        buf.put(message.version());
        buf.put((byte) message.type().code);
        buf.put(message.flags());
        buf.put((byte) 0); // reserved
        buf.putInt(payloadLen);

        // Payload
        if (payloadLen > 0) {
            buf.put(message.payload());
        }

        // Compute CRC-32 over header + payload
        int checksum = crc32(buf.array(), 0, HEADER_SIZE + payloadLen);
        buf.putInt(checksum);

        return buf.array();
    }

    /**
     * Encode a handshake message.
     */
    public static byte[] encodeHandshake(Handshake hs) {
        byte[] payload = serializeHandshake(hs);
        WireMessage msg = new WireMessage(PROTOCOL_VERSION, MessageType.HANDSHAKE,
            (byte) 0, payload, 0);
        return encode(msg);
    }

    /**
     * Encode a chunk request.
     */
    public static byte[] encodeRequest(ChunkRequest req) {
        byte[] payload = serializeChunkRequest(req);
        WireMessage msg = new WireMessage(PROTOCOL_VERSION, MessageType.REQUEST,
            (byte) 0, payload, 0);
        return encode(msg);
    }

    /**
     * Encode a chunk piece (data delivery).
     */
    public static byte[] encodePiece(ChunkPiece piece) {
        byte[] payload = serializeChunkPiece(piece);
        WireMessage msg = new WireMessage(PROTOCOL_VERSION, MessageType.PIECE,
            (byte) 0, payload, 0);
        return encode(msg);
    }

    /**
     * Encode a bitfield message.
     */
    public static byte[] encodeBitfield(String fileId, BitSet chunks, int totalChunks) {
        byte[] fileIdBytes = fileId.getBytes(StandardCharsets.UTF_8);
        byte[] bitfieldBytes = chunks.toByteArray();
        ByteBuffer payload = ByteBuffer.allocate(4 + fileIdBytes.length + 4 + 4 + bitfieldBytes.length);
        payload.putInt(fileIdBytes.length);
        payload.put(fileIdBytes);
        payload.putInt(totalChunks);
        payload.putInt(bitfieldBytes.length);
        payload.put(bitfieldBytes);

        WireMessage msg = new WireMessage(PROTOCOL_VERSION, MessageType.BITFIELD,
            (byte) 0, payload.array(), 0);
        return encode(msg);
    }

    /**
     * Encode a HAVE message (single chunk availability).
     */
    public static byte[] encodeHave(String fileId, int chunkIndex) {
        byte[] fileIdBytes = fileId.getBytes(StandardCharsets.UTF_8);
        ByteBuffer payload = ByteBuffer.allocate(4 + fileIdBytes.length + 4);
        payload.putInt(fileIdBytes.length);
        payload.put(fileIdBytes);
        payload.putInt(chunkIndex);

        WireMessage msg = new WireMessage(PROTOCOL_VERSION, MessageType.HAVE,
            (byte) 0, payload.array(), 0);
        return encode(msg);
    }

    /**
     * Encode a simple control message (CHOKE/UNCHOKE/INTERESTED/etc).
     */
    public static byte[] encodeControl(MessageType type) {
        WireMessage msg = new WireMessage(PROTOCOL_VERSION, type, (byte) 0, new byte[0], 0);
        return encode(msg);
    }

    /**
     * Encode a keepalive.
     */
    public static byte[] encodeKeepalive() {
        return encodeControl(MessageType.KEEPALIVE);
    }

    // ==================== Codec — Decoder ====================

    /**
     * Decode a WireMessage from a byte buffer.
     * Returns null if not enough data or invalid magic.
     */
    public static WireMessage decode(byte[] data) throws ProtocolException {
        if (data.length < HEADER_SIZE + CHECKSUM_SIZE) {
            throw new ProtocolException("Message too short: " + data.length + " bytes");
        }

        ByteBuffer buf = ByteBuffer.wrap(data);
        buf.order(ByteOrder.BIG_ENDIAN);

        // Verify magic
        byte[] magic = new byte[4];
        buf.get(magic);
        if (!Arrays.equals(magic, MAGIC)) {
            throw new ProtocolException("Invalid magic bytes");
        }

        byte version = buf.get();
        MessageType type = MessageType.fromCode(buf.get() & 0xFF);
        byte flags = buf.get();
        buf.get(); // reserved
        int payloadLen = buf.getInt();

        if (payloadLen < 0 || payloadLen > MAX_PAYLOAD_SIZE) {
            throw new ProtocolException("Invalid payload length: " + payloadLen);
        }

        if (data.length < HEADER_SIZE + payloadLen + CHECKSUM_SIZE) {
            throw new ProtocolException("Incomplete message: expected " +
                (HEADER_SIZE + payloadLen + CHECKSUM_SIZE) + " but got " + data.length);
        }

        byte[] payload = new byte[payloadLen];
        buf.get(payload);

        int receivedChecksum = buf.getInt();

        // Verify CRC-32
        int computedChecksum = crc32(data, 0, HEADER_SIZE + payloadLen);
        if (receivedChecksum != computedChecksum) {
            throw new ProtocolException("Checksum mismatch: expected 0x" +
                Integer.toHexString(computedChecksum) + " but got 0x" +
                Integer.toHexString(receivedChecksum));
        }

        return new WireMessage(version, type, flags, payload, receivedChecksum);
    }

    /**
     * Decode a handshake from a WireMessage payload.
     */
    public static Handshake decodeHandshake(byte[] payload) {
        ByteBuffer buf = ByteBuffer.wrap(payload);
        buf.order(ByteOrder.BIG_ENDIAN);

        int peerIdLen = buf.getInt();
        byte[] peerIdBytes = new byte[peerIdLen];
        buf.get(peerIdBytes);

        int nameLen = buf.getInt();
        byte[] nameBytes = new byte[nameLen];
        buf.get(nameBytes);

        int versionLen = buf.getInt();
        byte[] versionBytes = new byte[versionLen];
        buf.get(versionBytes);

        int capabilities = buf.getInt();
        int maxRequestSize = buf.getInt();

        int transportCount = buf.getInt();
        List<String> transports = new ArrayList<>();
        for (int i = 0; i < transportCount; i++) {
            int tLen = buf.getInt();
            byte[] tBytes = new byte[tLen];
            buf.get(tBytes);
            transports.add(new String(tBytes, StandardCharsets.UTF_8));
        }

        byte[] infoHash = new byte[32];
        if (buf.remaining() >= 32) {
            buf.get(infoHash);
        }

        return new Handshake(
            new String(peerIdBytes, StandardCharsets.UTF_8),
            new String(nameBytes, StandardCharsets.UTF_8),
            new String(versionBytes, StandardCharsets.UTF_8),
            capabilities, maxRequestSize, transports, infoHash
        );
    }

    /**
     * Decode a chunk request from payload.
     */
    public static ChunkRequest decodeRequest(byte[] payload) {
        ByteBuffer buf = ByteBuffer.wrap(payload);
        buf.order(ByteOrder.BIG_ENDIAN);

        int fileIdLen = buf.getInt();
        byte[] fileIdBytes = new byte[fileIdLen];
        buf.get(fileIdBytes);

        int chunkIndex = buf.getInt();
        int offset = buf.getInt();
        int length = buf.getInt();

        return new ChunkRequest(
            new String(fileIdBytes, StandardCharsets.UTF_8),
            chunkIndex, offset, length
        );
    }

    /**
     * Decode a chunk piece from payload.
     */
    public static ChunkPiece decodePiece(byte[] payload) {
        ByteBuffer buf = ByteBuffer.wrap(payload);
        buf.order(ByteOrder.BIG_ENDIAN);

        int fileIdLen = buf.getInt();
        byte[] fileIdBytes = new byte[fileIdLen];
        buf.get(fileIdBytes);

        int chunkIndex = buf.getInt();
        int offset = buf.getInt();
        int dataLen = buf.getInt();
        byte[] data = new byte[dataLen];
        buf.get(data);

        return new ChunkPiece(
            new String(fileIdBytes, StandardCharsets.UTF_8),
            chunkIndex, offset, data
        );
    }

    // ==================== Stream Codec ====================

    /**
     * Read a single WireMessage from an InputStream.
     * Blocks until a complete message is available.
     */
    public static WireMessage readMessage(InputStream in) throws IOException, ProtocolException {
        // Read header
        byte[] header = readExact(in, HEADER_SIZE);

        ByteBuffer hdr = ByteBuffer.wrap(header);
        hdr.order(ByteOrder.BIG_ENDIAN);

        byte[] magic = new byte[4];
        hdr.get(magic);
        if (!Arrays.equals(magic, MAGIC)) {
            throw new ProtocolException("Invalid magic bytes in stream");
        }

        hdr.position(8); // skip version, type, flags, reserved
        int payloadLen = hdr.getInt();

        if (payloadLen < 0 || payloadLen > MAX_PAYLOAD_SIZE) {
            throw new ProtocolException("Invalid payload length: " + payloadLen);
        }

        // Read payload + checksum
        byte[] rest = readExact(in, payloadLen + CHECKSUM_SIZE);

        // Combine for full decode
        byte[] full = new byte[HEADER_SIZE + payloadLen + CHECKSUM_SIZE];
        System.arraycopy(header, 0, full, 0, HEADER_SIZE);
        System.arraycopy(rest, 0, full, HEADER_SIZE, rest.length);

        return decode(full);
    }

    /**
     * Write a WireMessage to an OutputStream.
     */
    public static void writeMessage(OutputStream out, WireMessage message) throws IOException {
        out.write(encode(message));
        out.flush();
    }

    // ==================== Session Handler ====================

    /**
     * A Bobzilla protocol session — manages the state machine for a single
     * peer-to-peer connection.
     */
    public static class Session {
        private final String localPeerId;
        private final int localCapabilities;
        private String remotePeerId;
        private int remoteCapabilities;
        private boolean handshakeComplete = false;
        private boolean localChoked = true;
        private boolean remoteChoked = true;
        private boolean localInterested = false;
        private boolean remoteInterested = false;

        // Stats
        private final AtomicLong messagesSent = new AtomicLong();
        private final AtomicLong messagesReceived = new AtomicLong();
        private final AtomicLong bytesSent = new AtomicLong();
        private final AtomicLong bytesReceived = new AtomicLong();
        private final AtomicLong checksumErrors = new AtomicLong();
        private final ConcurrentHashMap<MessageType, AtomicLong> msgCounts = new ConcurrentHashMap<>();

        // Message handlers
        private Consumer<Handshake> onHandshake;
        private Consumer<ChunkRequest> onRequest;
        private Consumer<ChunkPiece> onPiece;
        private Consumer<MessageType> onControl;
        private Consumer<byte[]> onExtension;

        public Session(String localPeerId, int localCapabilities) {
            this.localPeerId = localPeerId;
            this.localCapabilities = localCapabilities;
        }

        /**
         * Process an incoming WireMessage and dispatch to handlers.
         */
        public void handleMessage(WireMessage msg) {
            messagesReceived.incrementAndGet();
            bytesReceived.addAndGet(msg.payload() != null ? msg.payload().length : 0);
            msgCounts.computeIfAbsent(msg.type(), k -> new AtomicLong()).incrementAndGet();

            switch (msg.type()) {
                case HANDSHAKE -> {
                    Handshake hs = decodeHandshake(msg.payload());
                    remotePeerId = hs.peerId();
                    remoteCapabilities = hs.capabilities();
                    handshakeComplete = true;
                    if (onHandshake != null) onHandshake.accept(hs);
                }
                case REQUEST -> {
                    if (onRequest != null) onRequest.accept(decodeRequest(msg.payload()));
                }
                case PIECE -> {
                    if (onPiece != null) onPiece.accept(decodePiece(msg.payload()));
                }
                case CHOKE -> {
                    remoteChoked = true;
                    if (onControl != null) onControl.accept(msg.type());
                }
                case UNCHOKE -> {
                    remoteChoked = false;
                    if (onControl != null) onControl.accept(msg.type());
                }
                case INTERESTED -> {
                    remoteInterested = true;
                    if (onControl != null) onControl.accept(msg.type());
                }
                case NOT_INTERESTED -> {
                    remoteInterested = false;
                    if (onControl != null) onControl.accept(msg.type());
                }
                case EXTENSION -> {
                    if (onExtension != null) onExtension.accept(msg.payload());
                }
                default -> {
                    if (onControl != null) onControl.accept(msg.type());
                }
            }
        }

        /**
         * Create our handshake message.
         */
        public byte[] createHandshake(byte[] infoHash) {
            Handshake hs = new Handshake(
                localPeerId, "Bobzilla", "0.6.0",
                localCapabilities, MAX_PAYLOAD_SIZE,
                List.of("tcp", "ws", "tor", "i2p", "ipfs", "hyphanet", "zeronet"),
                infoHash
            );
            return encodeHandshake(hs);
        }

        /**
         * Check if both sides have negotiated a specific capability.
         */
        public boolean hasSharedCapability(int capability) {
            return (localCapabilities & capability) != 0
                && (remoteCapabilities & capability) != 0;
        }

        public ProtocolStats getStats() {
            Map<MessageType, Long> counts = new HashMap<>();
            msgCounts.forEach((type, count) -> counts.put(type, count.get()));
            return new ProtocolStats(
                messagesSent.get(), messagesReceived.get(),
                bytesSent.get(), bytesReceived.get(),
                checksumErrors.get(), 0, counts
            );
        }

        // Getters
        public boolean isHandshakeComplete() { return handshakeComplete; }
        public boolean isLocalChoked() { return localChoked; }
        public boolean isRemoteChoked() { return remoteChoked; }
        public boolean isLocalInterested() { return localInterested; }
        public boolean isRemoteInterested() { return remoteInterested; }
        public String getRemotePeerId() { return remotePeerId; }
        public int getRemoteCapabilities() { return remoteCapabilities; }

        // Event handlers
        public void setOnHandshake(Consumer<Handshake> h) { this.onHandshake = h; }
        public void setOnRequest(Consumer<ChunkRequest> h) { this.onRequest = h; }
        public void setOnPiece(Consumer<ChunkPiece> h) { this.onPiece = h; }
        public void setOnControl(Consumer<MessageType> h) { this.onControl = h; }
        public void setOnExtension(Consumer<byte[]> h) { this.onExtension = h; }
    }

    // ==================== Utilities ====================

    private static byte[] serializeHandshake(Handshake hs) {
        byte[] peerId = hs.peerId().getBytes(StandardCharsets.UTF_8);
        byte[] name = hs.clientName().getBytes(StandardCharsets.UTF_8);
        byte[] version = hs.clientVersion().getBytes(StandardCharsets.UTF_8);

        int transportSize = 0;
        List<byte[]> transportBytes = new ArrayList<>();
        for (String t : hs.supportedTransports()) {
            byte[] tb = t.getBytes(StandardCharsets.UTF_8);
            transportBytes.add(tb);
            transportSize += 4 + tb.length;
        }

        int size = 4 + peerId.length + 4 + name.length + 4 + version.length
                   + 4 + 4 + 4 + transportSize + 32;

        ByteBuffer buf = ByteBuffer.allocate(size);
        buf.order(ByteOrder.BIG_ENDIAN);

        buf.putInt(peerId.length).put(peerId);
        buf.putInt(name.length).put(name);
        buf.putInt(version.length).put(version);
        buf.putInt(hs.capabilities());
        buf.putInt(hs.maxRequestSize());
        buf.putInt(hs.supportedTransports().size());
        for (byte[] tb : transportBytes) {
            buf.putInt(tb.length).put(tb);
        }
        buf.put(hs.infoHash() != null && hs.infoHash().length == 32
            ? hs.infoHash() : new byte[32]);

        return buf.array();
    }

    private static byte[] serializeChunkRequest(ChunkRequest req) {
        byte[] fileId = req.fileId().getBytes(StandardCharsets.UTF_8);
        ByteBuffer buf = ByteBuffer.allocate(4 + fileId.length + 12);
        buf.order(ByteOrder.BIG_ENDIAN);
        buf.putInt(fileId.length).put(fileId);
        buf.putInt(req.chunkIndex());
        buf.putInt(req.offset());
        buf.putInt(req.length());
        return buf.array();
    }

    private static byte[] serializeChunkPiece(ChunkPiece piece) {
        byte[] fileId = piece.fileId().getBytes(StandardCharsets.UTF_8);
        ByteBuffer buf = ByteBuffer.allocate(4 + fileId.length + 12 + piece.data().length);
        buf.order(ByteOrder.BIG_ENDIAN);
        buf.putInt(fileId.length).put(fileId);
        buf.putInt(piece.chunkIndex());
        buf.putInt(piece.offset());
        buf.putInt(piece.data().length).put(piece.data());
        return buf.array();
    }

    private static byte[] readExact(InputStream in, int count) throws IOException {
        byte[] data = new byte[count];
        int read = 0;
        while (read < count) {
            int r = in.read(data, read, count - read);
            if (r < 0) throw new IOException("Unexpected end of stream");
            read += r;
        }
        return data;
    }

    private static int crc32(byte[] data, int offset, int length) {
        java.util.zip.CRC32 crc = new java.util.zip.CRC32();
        crc.update(data, offset, length);
        return (int) crc.getValue();
    }

    // ==================== Exception ====================

    public static class ProtocolException extends Exception {
        public ProtocolException(String message) { super(message); }
        public ProtocolException(String message, Throwable cause) { super(message, cause); }
    }
}
