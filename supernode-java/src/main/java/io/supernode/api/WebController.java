package io.supernode.api;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.node.ArrayNode;
import com.fasterxml.jackson.databind.node.ObjectNode;
import io.netty.bootstrap.ServerBootstrap;
import io.netty.buffer.ByteBuf;
import io.netty.buffer.Unpooled;
import io.netty.channel.*;
import io.netty.channel.nio.NioEventLoopGroup;
import io.netty.channel.socket.SocketChannel;
import io.netty.channel.socket.nio.NioServerSocketChannel;
import io.netty.handler.codec.http.*;
import io.netty.handler.stream.ChunkedWriteHandler;
import io.supernode.network.UnifiedNetwork;
import io.supernode.network.DHTDiscovery;
import io.supernode.network.ManifestDistributor;
import io.supernode.storage.SupernodeStorage;
import io.supernode.storage.mux.Manifest;

import java.nio.charset.StandardCharsets;
import java.security.KeyPair;
import java.security.KeyPairGenerator;
import java.time.Duration;
import java.util.Base64;
import java.util.HexFormat;
import java.util.List;
import java.util.Optional;

public class WebController {

    private final int port;
    private final UnifiedNetwork network;
    private final ObjectMapper mapper = new ObjectMapper();
    // In a real scenario, manage keys properly. For this reference impl, we use a fixed master key or generate one.
    // The Node.js client generates random keys per file ingest.
    // Here, we can use a fixed one for testing or accept it in headers.
    private static final byte[] DEFAULT_MASTER_KEY = new byte[32];

    public WebController(int port, UnifiedNetwork network) {
        this.port = port;
        this.network = network;
    }

    public void start() throws Exception {
        EventLoopGroup bossGroup = new NioEventLoopGroup(1);
        EventLoopGroup workerGroup = new NioEventLoopGroup();
        try {
            ServerBootstrap b = new ServerBootstrap();
            b.group(bossGroup, workerGroup)
             .channel(NioServerSocketChannel.class)
             .childHandler(new ChannelInitializer<SocketChannel>() {
                 @Override
                 public void initChannel(SocketChannel ch) {
                     ChannelPipeline p = ch.pipeline();
                     p.addLast(new HttpServerCodec());
                     p.addLast(new HttpObjectAggregator(100 * 1024 * 1024)); // 100MB max body
                     p.addLast(new ChunkedWriteHandler());
                     p.addLast(new SimpleChannelInboundHandler<FullHttpRequest>() {
                         @Override
                         protected void channelRead0(ChannelHandlerContext ctx, FullHttpRequest req) throws Exception {
                             handleRequest(ctx, req);
                         }
                     });
                 }
             });

            Channel ch = b.bind(port).sync().channel();
            System.out.println("Web UI API started on http://127.0.0.1:" + port);
            ch.closeFuture().sync();
        } finally {
            bossGroup.shutdownGracefully();
            workerGroup.shutdownGracefully();
        }
    }

    private void handleRequest(ChannelHandlerContext ctx, FullHttpRequest req) {
        String uri = req.uri();
        HttpMethod method = req.method();

        try {
            if (uri.equals("/api/status") && method.equals(HttpMethod.GET)) {
                handleStatus(ctx);
            } else if (uri.equals("/api/files") && method.equals(HttpMethod.GET)) {
                handleFiles(ctx);
            } else if (uri.equals("/api/ingest") && method.equals(HttpMethod.POST)) {
                handleIngest(ctx, req);
            } else if (uri.startsWith("/api/stream/") && method.equals(HttpMethod.GET)) {
                handleStream(ctx, req);
            } else if (uri.startsWith("/api/files/") && uri.endsWith("/health") && method.equals(HttpMethod.GET)) {
                handleFileHealth(ctx, req);
            } else if (uri.equals("/api/key/generate") && method.equals(HttpMethod.POST)) {
                handleKeyGenerate(ctx);
            } else if (uri.equals("/api/publish") && method.equals(HttpMethod.POST)) {
                handlePublish(ctx, req);
            } else if (uri.equals("/api/subscriptions") && method.equals(HttpMethod.GET)) {
                handleSubscriptions(ctx);
            } else if (uri.equals("/api/subscribe") && method.equals(HttpMethod.POST)) {
                handleSubscribe(ctx, req);
            } else if (uri.startsWith("/api/channels/browse") && method.equals(HttpMethod.GET)) {
                handleBrowse(ctx, req);
            } else if (uri.equals("/api/wallet") && method.equals(HttpMethod.GET)) {
                handleWallet(ctx);
            } else {
                sendError(ctx, HttpResponseStatus.NOT_FOUND);
            }
        } catch (Exception e) {
            e.printStackTrace();
            sendError(ctx, HttpResponseStatus.INTERNAL_SERVER_ERROR);
        }
    }

    private void handleStatus(ChannelHandlerContext ctx) {
        ObjectNode json = mapper.createObjectNode();
        json.put("version", "1.6.0-java");
        json.put("network", "active");

        UnifiedNetwork.NetworkStats stats = network.stats();
        ObjectNode storage = json.putObject("storage");
        storage.put("blobs", stats.storage().blobCount());
        storage.put("size", stats.storage().totalBytes());
        storage.put("max", -1); // Unlimited

        json.put("dht", "ready");
        json.put("subscriptions", 0);

        // Add detailed network stats
        ObjectNode networkDetails = json.putObject("networkDetails");
        networkDetails.put("peerCount", stats.peerCount());

        ObjectNode transports = networkDetails.putObject("transports");
        stats.transport().byTransport().forEach((type, tStats) -> {
            ObjectNode t = transports.putObject(type.toString());
            t.put("connectionsIn", tStats.connectionsIn());
            t.put("connectionsOut", tStats.connectionsOut());
            t.put("bytesReceived", tStats.bytesReceived());
            t.put("bytesSent", tStats.bytesSent());
            t.put("errors", tStats.errors());

            // Add address if available
            io.supernode.network.transport.TransportAddress addr = stats.addresses().get(type);
            if (addr != null) {
                t.put("address", addr.toString());
                t.put("status", "Running");
            } else {
                t.put("status", "Stopped");
            }
        });

        // Add detailed storage stats
        ObjectNode storageDetails = json.putObject("storageDetails");
        io.supernode.storage.SupernodeStorage.StorageStats sStats = stats.storage();
        storageDetails.put("isoSize", sStats.isoSize());
        storageDetails.put("totalFilesIngested", sStats.totalFilesIngested());
        storageDetails.put("totalBytesIngested", sStats.totalBytesIngested());

        if (sStats.erasure() != null) {
            ObjectNode erasure = storageDetails.putObject("erasure");
            erasure.put("dataShards", sStats.erasure().dataShards());
            erasure.put("parityShards", sStats.erasure().parityShards());
            erasure.put("totalShards", sStats.erasure().totalShards());
        }

        sendJson(ctx, json);
    }

    private void handleFiles(ChannelHandlerContext ctx) {
        ArrayNode files = mapper.createArrayNode();
        List<String> fileIds = network.getStorage().listFiles();

        for (String id : fileIds) {
            Optional<byte[]> manifestBytes = network.getStorage().getManifest(id);
            if (manifestBytes.isPresent()) {
                try {
                    // Try to decrypt with default master key
                    byte[] manifestKey = Manifest.deriveManifestKey(DEFAULT_MASTER_KEY, id);
                    Manifest m = Manifest.decrypt(manifestBytes.get(), manifestKey);

                    ObjectNode f = files.addObject();
                    f.put("id", id);
                    f.put("name", m.getFileName());
                    f.put("size", m.getFileSize());
                    f.put("status", "Complete"); // Since we have manifest
                    f.put("progress", 100);
                } catch (Exception e) {
                    // Decryption failed or invalid
                }
            }
        }

        sendJson(ctx, files);
    }

    private void handleIngest(ChannelHandlerContext ctx, FullHttpRequest req) {
        ByteBuf content = req.content();
        byte[] bytes = new byte[content.readableBytes()];
        content.readBytes(bytes);

        // Default values
        String filename = "upload-" + System.currentTimeMillis() + ".bin";
        byte[] fileData = bytes;
        SupernodeStorage.IngestOptions options = null;

        // Try to parse JSON wrapper if Content-Type is application/json
        String contentType = req.headers().get(HttpHeaderNames.CONTENT_TYPE);
        if (contentType != null && contentType.contains("application/json")) {
            try {
                // If it's a JSON request with base64 data and options
                // { "filename": "...", "data": "base64...", "options": { "enableErasure": true, ... } }
                // This is needed because standard multipart is hard to parse without a library in Netty raw
                // And we want to pass options + file in one go.

                ObjectNode node = (ObjectNode) mapper.readTree(new String(bytes, StandardCharsets.UTF_8));
                if (node.has("data")) {
                    fileData = Base64.getDecoder().decode(node.get("data").asText());
                    if (node.has("filename")) {
                        filename = node.get("filename").asText();
                    }
                    if (node.has("options")) {
                        ObjectNode opts = (ObjectNode) node.get("options");
                        boolean enableErasure = opts.has("enableErasure") && opts.get("enableErasure").asBoolean();
                        int dataShards = opts.has("dataShards") ? opts.get("dataShards").asInt() : 4;
                        int parityShards = opts.has("parityShards") ? opts.get("parityShards").asInt() : 2;
                        options = new SupernodeStorage.IngestOptions(enableErasure, dataShards, parityShards);
                    }
                }
            } catch (Exception e) {
                // Not JSON wrapped, treat as raw body
            }
        } else {
            // Raw upload, check header for filename
            if (req.headers().contains("X-Filename")) {
                filename = req.headers().get("X-Filename");
            }
        }

        SupernodeStorage.IngestResult result = network.ingestFile(fileData, filename, DEFAULT_MASTER_KEY, options);

        ObjectNode json = mapper.createObjectNode();
        json.put("fileId", result.fileId());
        json.put("status", "ingested");

        sendJson(ctx, json);
    }

    private void handleFileHealth(ChannelHandlerContext ctx, FullHttpRequest req) {
        // Extract fileId from /api/files/{id}/health
        String path = req.uri();
        String fileId = path.substring("/api/files/".length(), path.length() - "/health".length());

        io.supernode.storage.SupernodeStorage.FileHealth health = network.getStorage().getFileHealth(fileId, DEFAULT_MASTER_KEY);

        ObjectNode json = mapper.createObjectNode();
        json.put("fileId", health.fileId());
        json.put("status", health.status());
        json.put("totalChunks", health.totalChunks());
        json.put("healthyChunks", health.healthyChunks());

        if (health.erasure() != null) {
            ObjectNode ec = json.putObject("erasure");
            ec.put("dataShards", health.erasure().dataShards());
            ec.put("parityShards", health.erasure().parityShards());
        }

        ArrayNode chunks = json.putArray("chunks");
        for (io.supernode.storage.SupernodeStorage.ChunkHealth ch : health.chunks()) {
            ObjectNode chunk = chunks.addObject();
            chunk.put("index", ch.index());
            chunk.put("status", ch.status());

            if (!ch.shards().isEmpty()) {
                ArrayNode shards = chunk.putArray("shards");
                for (io.supernode.storage.SupernodeStorage.ShardHealth sh : ch.shards()) {
                    ObjectNode shard = shards.addObject();
                    shard.put("index", sh.index());
                    shard.put("present", sh.present());
                }
            }
        }

        sendJson(ctx, json);
    }

    private void handleStream(ChannelHandlerContext ctx, FullHttpRequest req) {
        String fileId = req.uri().substring("/api/stream/".length());

        try {
            SupernodeStorage.RetrieveResult result = network.retrieveFile(fileId, DEFAULT_MASTER_KEY);
            byte[] data = result.data();

            // Handle Range
            String range = req.headers().get(HttpHeaderNames.RANGE);
            if (range != null) {
                // Simplified Range support (bytes=start-end)
                String[] parts = range.replace("bytes=", "").split("-");
                int start = Integer.parseInt(parts[0]);
                int end = parts.length > 1 && !parts[1].isEmpty() ? Integer.parseInt(parts[1]) : data.length - 1;

                if (start >= data.length) {
                     sendError(ctx, HttpResponseStatus.REQUESTED_RANGE_NOT_SATISFIABLE);
                     return;
                }

                int len = end - start + 1;
                ByteBuf buf = Unpooled.wrappedBuffer(data, start, len);

                FullHttpResponse response = new DefaultFullHttpResponse(
                    HttpVersion.HTTP_1_1, HttpResponseStatus.PARTIAL_CONTENT, buf);

                response.headers().set(HttpHeaderNames.CONTENT_TYPE, "application/octet-stream");
                response.headers().set(HttpHeaderNames.CONTENT_LENGTH, len);
                response.headers().set(HttpHeaderNames.CONTENT_RANGE, "bytes " + start + "-" + end + "/" + data.length);
                response.headers().set(HttpHeaderNames.ACCESS_CONTROL_ALLOW_ORIGIN, "*"); // Allow CORS

                ctx.writeAndFlush(response);
            } else {
                ByteBuf buf = Unpooled.wrappedBuffer(data);
                FullHttpResponse response = new DefaultFullHttpResponse(
                    HttpVersion.HTTP_1_1, HttpResponseStatus.OK, buf);
                response.headers().set(HttpHeaderNames.CONTENT_TYPE, "application/octet-stream");
                response.headers().set(HttpHeaderNames.CONTENT_LENGTH, data.length);
                response.headers().set(HttpHeaderNames.ACCESS_CONTROL_ALLOW_ORIGIN, "*");
                ctx.writeAndFlush(response);
            }
        } catch (Exception e) {
            e.printStackTrace();
            sendError(ctx, HttpResponseStatus.NOT_FOUND);
        }
    }

    private void handleKeyGenerate(ChannelHandlerContext ctx) {
        try {
            KeyPairGenerator kpg = KeyPairGenerator.getInstance("EC");
            kpg.initialize(256);
            KeyPair kp = kpg.generateKeyPair();

            ObjectNode json = mapper.createObjectNode();
            json.put("publicKey", HexFormat.of().formatHex(kp.getPublic().getEncoded()));
            json.put("secretKey", HexFormat.of().formatHex(kp.getPrivate().getEncoded()));

            sendJson(ctx, json);
        } catch (Exception e) {
            e.printStackTrace();
            sendError(ctx, HttpResponseStatus.INTERNAL_SERVER_ERROR);
        }
    }

    private void handlePublish(ChannelHandlerContext ctx, FullHttpRequest req) {
        try {
            ByteBuf content = req.content();
            ObjectNode body = (ObjectNode) mapper.readTree(content.toString(StandardCharsets.UTF_8));

            // Extract fileEntry
            ObjectNode fileEntry = (ObjectNode) body.get("fileEntry");
            String fileId = fileEntry.get("chunks").get(0).get("blobId").asText(); // Assuming first chunk for now

            // In a real impl, we'd sign with identity. Here we just announce via DHT/ManifestDistributor
            network.getManifestDistributor().announceManifest(fileId);

            ObjectNode json = mapper.createObjectNode();
            json.put("status", "published");
            ObjectNode manifest = json.putObject("manifest");
            manifest.put("fileId", fileId);

            sendJson(ctx, json);
        } catch (Exception e) {
            e.printStackTrace();
            sendError(ctx, HttpResponseStatus.BAD_REQUEST);
        }
    }

    private void handleSubscriptions(ChannelHandlerContext ctx) {
        ArrayNode subs = mapper.createArrayNode();
        for (ManifestDistributor.AnnouncedManifest am : network.getManifestDistributor().getAnnouncedManifests()) {
            ObjectNode sub = subs.addObject();
            sub.put("topicPath", am.fileId());
            sub.put("lastSequence", am.announcedAt());
        }
        sendJson(ctx, subs);
    }

    private void handleSubscribe(ChannelHandlerContext ctx, FullHttpRequest req) {
        try {
            ByteBuf content = req.content();
            ObjectNode body = (ObjectNode) mapper.readTree(content.toString(StandardCharsets.UTF_8));
            String key = body.get("publicKey").asText();

            // For now, we treat subscription as "finding manifest peers"
            network.getManifestDistributor().findManifestPeers(key, 5000);

            ObjectNode json = mapper.createObjectNode();
            json.put("status", "subscribed");
            json.put("publicKey", key);
            sendJson(ctx, json);
        } catch (Exception e) {
            sendError(ctx, HttpResponseStatus.BAD_REQUEST);
        }
    }

    private void handleBrowse(ChannelHandlerContext ctx, FullHttpRequest req) {
        // DHT Browse logic
        QueryStringDecoder query = new QueryStringDecoder(req.uri());
        String topic = query.parameters().containsKey("topic") ? query.parameters().get("topic").get(0) : "";

        ObjectNode result = mapper.createObjectNode();
        ArrayNode subtopics = result.putArray("subtopics");
        ArrayNode publishers = result.putArray("publishers");

        // Query DHT for topic peers (using SHA1 of topic path)
        if (!topic.isEmpty()) {
            // This is a simplification. Real browsing involves traversing topic manifests.
            // For now, we list active peers found via DHT for this topic/blob.
            List<DHTDiscovery.PeerInfo> peers = network.getDht().findPeers(topic, Duration.ofMillis(2000)).join();
            for (DHTDiscovery.PeerInfo peer : peers) {
                ObjectNode p = publishers.addObject();
                p.put("name", peer.host());
                p.put("pk", "unknown"); // DHT peers don't advertise PK directly in this lookup
            }
        } else {
            // Root topics
            subtopics.add("video");
            subtopics.add("audio");
            subtopics.add("documents");
        }

        sendJson(ctx, result);
    }

    private void handleWallet(ChannelHandlerContext ctx) {
        ObjectNode wallet = mapper.createObjectNode();

        // Simple file-based wallet persistence for reference implementation
        String address = "0xSupernodeWalletJava";
        java.io.File walletFile = new java.io.File("wallet.json");

        if (walletFile.exists()) {
            try {
                ObjectNode saved = (ObjectNode) mapper.readTree(walletFile);
                if (saved.has("publicKey")) {
                    address = saved.get("publicKey").asText();
                }
            } catch (Exception e) {
                // Ignore load error
            }
        }

        wallet.put("address", address);
        wallet.put("balance", 1000); // Mock balance for Java Supernode (Bridge integration is next phase)
        wallet.put("pending", 50);
        wallet.putArray("transactions");

        sendJson(ctx, wallet);
    }

    private void sendJson(ChannelHandlerContext ctx, ObjectNode json) {
        ByteBuf content = Unpooled.copiedBuffer(json.toString(), StandardCharsets.UTF_8);
        FullHttpResponse response = new DefaultFullHttpResponse(
                HttpVersion.HTTP_1_1, HttpResponseStatus.OK, content);
        response.headers().set(HttpHeaderNames.CONTENT_TYPE, "application/json");
        response.headers().set(HttpHeaderNames.CONTENT_LENGTH, content.readableBytes());
        response.headers().set(HttpHeaderNames.ACCESS_CONTROL_ALLOW_ORIGIN, "*"); // Allow CORS
        ctx.writeAndFlush(response);
    }

    private void sendJson(ChannelHandlerContext ctx, ArrayNode json) {
        ByteBuf content = Unpooled.copiedBuffer(json.toString(), StandardCharsets.UTF_8);
        FullHttpResponse response = new DefaultFullHttpResponse(
                HttpVersion.HTTP_1_1, HttpResponseStatus.OK, content);
        response.headers().set(HttpHeaderNames.CONTENT_TYPE, "application/json");
        response.headers().set(HttpHeaderNames.CONTENT_LENGTH, content.readableBytes());
        response.headers().set(HttpHeaderNames.ACCESS_CONTROL_ALLOW_ORIGIN, "*");
        ctx.writeAndFlush(response);
    }

    private void sendError(ChannelHandlerContext ctx, HttpResponseStatus status) {
        FullHttpResponse response = new DefaultFullHttpResponse(
                HttpVersion.HTTP_1_1, status);
        ctx.writeAndFlush(response);
    }
}
