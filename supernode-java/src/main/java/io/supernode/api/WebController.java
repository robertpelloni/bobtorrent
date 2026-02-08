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
import io.supernode.storage.SupernodeStorage;
import io.supernode.storage.mux.Manifest;

import java.nio.charset.StandardCharsets;
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

        // Simple ingest: treat body as file content
        // In real impl, handle multipart/form-data
        // For testing, just post raw bytes

        String filename = "upload-" + System.currentTimeMillis() + ".bin";
        // Check if filename header exists
        if (req.headers().contains("X-Filename")) {
            filename = req.headers().get("X-Filename");
        }

        SupernodeStorage.IngestResult result = network.ingestFile(bytes, filename, DEFAULT_MASTER_KEY);

        ObjectNode json = mapper.createObjectNode();
        json.put("fileId", result.fileId());
        json.put("status", "ingested");

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
