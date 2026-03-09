package io.supernode.network;

import com.fasterxml.jackson.databind.ObjectMapper;
import io.supernode.network.transport.Transport;
import io.supernode.network.transport.TransportType;
import io.supernode.storage.StorageBenchmark;
import java.util.LinkedHashMap;
import java.util.Map;
import io.netty.bootstrap.ServerBootstrap;
import io.netty.buffer.Unpooled;
import io.netty.channel.*;
import io.netty.channel.nio.NioEventLoopGroup;
import io.netty.channel.socket.SocketChannel;
import io.netty.channel.socket.nio.NioServerSocketChannel;
import io.netty.handler.codec.http.*;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import com.fasterxml.jackson.databind.SerializationFeature;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.nio.charset.StandardCharsets;
import java.util.concurrent.CompletableFuture;

/**
 * A lightweight HTTP server providing a modern monitoring dashboard for Supernode metrics.
 * Serves an embedded HTML visualization at "/" and raw JSON at "/api/metrics".
 */
public class MetricsServer {

    private static final Logger log = LoggerFactory.getLogger(MetricsServer.class);
    
    // We bind against UnifiedNetwork or SupernodeNetwork as the source of truth for all metrics.
    private final UnifiedNetwork network;
    private final int port;
    
    private EventLoopGroup bossGroup;
    private EventLoopGroup workerGroup;
    private Channel serverChannel;
    private final ObjectMapper mapper;

    public MetricsServer(UnifiedNetwork network, int port) {
        this.network = network;
        this.port = port;
        this.mapper = new ObjectMapper();
        this.mapper.registerModule(new JavaTimeModule());
        this.mapper.disable(SerializationFeature.WRITE_DATES_AS_TIMESTAMPS);
    }

    public CompletableFuture<Void> start() {
        CompletableFuture<Void> future = new CompletableFuture<>();
        bossGroup = new NioEventLoopGroup(1);
        workerGroup = new NioEventLoopGroup();

        try {
            ServerBootstrap b = new ServerBootstrap();
            b.group(bossGroup, workerGroup)
             .channel(NioServerSocketChannel.class)
             .childHandler(new ChannelInitializer<SocketChannel>() {
                 @Override
                 public void initChannel(SocketChannel ch) {
                     ch.pipeline().addLast(new HttpServerCodec());
                     ch.pipeline().addLast(new HttpObjectAggregator(65536));
                     ch.pipeline().addLast(new MetricsHandler());
                 }
             });

            b.bind(port).addListener((ChannelFutureListener) f -> {
                if (f.isSuccess()) {
                    serverChannel = f.channel();
                    log.info("Monitoring Dashboard running on http://localhost:{}", port);
                    future.complete(null);
                } else {
                    future.completeExceptionally(f.cause());
                }
            });
        } catch (Exception e) {
            future.completeExceptionally(e);
        }
        return future;
    }

    public CompletableFuture<Void> stop() {
        CompletableFuture<Void> future = new CompletableFuture<>();
        if (serverChannel != null) {
            serverChannel.close().addListener(f -> {
                bossGroup.shutdownGracefully();
                workerGroup.shutdownGracefully();
                future.complete(null);
            });
        } else {
            future.complete(null);
        }
        return future;
    }

    private class MetricsHandler extends SimpleChannelInboundHandler<FullHttpRequest> {

        @Override
        protected void channelRead0(ChannelHandlerContext ctx, FullHttpRequest req) throws Exception {
            if (!req.decoderResult().isSuccess()) {
                sendError(ctx, HttpResponseStatus.BAD_REQUEST);
                return;
            }

            if ("/api/metrics".equals(req.uri())) {
                UnifiedNetwork.NetworkStats stats = network.stats();
                byte[] json = mapper.writeValueAsBytes(stats);
                sendJson(ctx, req, json);
            } else if ("/api/health".equals(req.uri())) {
                byte[] json = mapper.writeValueAsBytes(buildHealthPayload());
                sendJson(ctx, req, json);
            } else if ("/api/benchmark".equals(req.uri())) {
                // Run async to avoid blocking Netty thread
                ctx.channel().eventLoop().execute(() -> {
                    try {
                        StorageBenchmark bench = new StorageBenchmark(network.getStorage());
                        StorageBenchmark.BenchmarkReport report = bench.run(StorageBenchmark.BenchmarkOptions.quick());
                        byte[] json2 = mapper.writeValueAsBytes(report);
                        sendJson(ctx, req, json2);
                    } catch (Exception e) {
                        sendError(ctx, HttpResponseStatus.INTERNAL_SERVER_ERROR);
                    }
                });
                return; // Don't send response yet
            } else if ("/".equals(req.uri()) || "/index.html".equals(req.uri())) {
                byte[] html = getDashboardHtml().getBytes(StandardCharsets.UTF_8);
                
                FullHttpResponse res = new DefaultFullHttpResponse(
                    HttpVersion.HTTP_1_1, HttpResponseStatus.OK, Unpooled.wrappedBuffer(html));
                res.headers().set(HttpHeaderNames.CONTENT_TYPE, "text/html; charset=UTF-8");
                res.headers().set(HttpHeaderNames.CONTENT_LENGTH, res.content().readableBytes());
                
                sendResponse(ctx, req, res);
            } else {
                sendError(ctx, HttpResponseStatus.NOT_FOUND);
            }
        }

        private void sendJson(ChannelHandlerContext ctx, FullHttpRequest req, byte[] json) {
            FullHttpResponse res = new DefaultFullHttpResponse(
                HttpVersion.HTTP_1_1, HttpResponseStatus.OK, Unpooled.wrappedBuffer(json));
            res.headers().set(HttpHeaderNames.CONTENT_TYPE, "application/json");
            res.headers().set(HttpHeaderNames.CONTENT_LENGTH, res.content().readableBytes());
            res.headers().set(HttpHeaderNames.ACCESS_CONTROL_ALLOW_ORIGIN, "*");
            sendResponse(ctx, req, res);
        }

        private void sendResponse(ChannelHandlerContext ctx, FullHttpRequest req, FullHttpResponse res) {
            boolean keepAlive = HttpUtil.isKeepAlive(req);
            if (!keepAlive) {
                ctx.writeAndFlush(res).addListener(ChannelFutureListener.CLOSE);
            } else {
                res.headers().set(HttpHeaderNames.CONNECTION, HttpHeaderValues.KEEP_ALIVE);
                ctx.writeAndFlush(res);
            }
        }

        private void sendError(ChannelHandlerContext ctx, HttpResponseStatus status) {
            FullHttpResponse response = new DefaultFullHttpResponse(
                HttpVersion.HTTP_1_1, status, Unpooled.copiedBuffer("Failure: " + status + "\r\n", StandardCharsets.UTF_8));
            response.headers().set(HttpHeaderNames.CONTENT_TYPE, "text/plain; charset=UTF-8");
            ctx.writeAndFlush(response).addListener(ChannelFutureListener.CLOSE);
        }

        private Map<String, Object> buildHealthPayload() {
            Map<String, Object> payload = new LinkedHashMap<>();
            Map<TransportType, Transport.HealthStatus> healthMap = network.getTransportManager().getAllHealthStatuses();
            
            for (var entry : healthMap.entrySet()) {
                Map<String, Object> t = new LinkedHashMap<>();
                Transport.HealthStatus hs = entry.getValue();
                t.put("state", hs.state().name());
                t.put("message", hs.message());
                t.put("consecutiveFailures", hs.consecutiveFailures());
                t.put("latencyMs", hs.latencyMs());
                
                var cbStatus = network.getTransportManager().getCircuitBreakerStatus(entry.getKey());
                if (cbStatus != null) {
                    Map<String, Object> cb = new LinkedHashMap<>();
                    cb.put("open", cbStatus.open());
                    cb.put("failures", cbStatus.failures());
                    cb.put("threshold", cbStatus.threshold());
                    t.put("circuitBreaker", cb);
                }
                
                Transport transport = network.getTransportManager().getTransport(entry.getKey());
                if (transport != null) {
                    t.put("running", transport.isRunning());
                    t.put("available", transport.isAvailable());
                }
                
                payload.put(entry.getKey().name(), t);
            }
            return payload;
        }
    }

    private String getDashboardHtml() {
        return """
            <!DOCTYPE html>
            <html lang="en">
            <head>
                <meta charset="UTF-8">
                <meta name="viewport" content="width=device-width, initial-scale=1.0">
                <title>Supernode Intelligence Dashboard</title>
                <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
                <style>
                    :root {
                        --bg-dark: #0f172a;
                        --bg-card: rgba(30, 41, 59, 0.7);
                        --text-main: #f8fafc;
                        --text-muted: #94a3b8;
                        --accent-blue: #38bdf8;
                        --accent-green: #34d399;
                        --accent-purple: #a78bfa;
                    }
                    body {
                        font-family: 'Inter', -apple-system, sans-serif;
                        background-color: var(--bg-dark);
                        color: var(--text-main);
                        margin: 0;
                        padding: 2rem;
                        background-image: radial-gradient(circle at 15% 50%, rgba(56, 189, 248, 0.05), transparent 25%),
                                          radial-gradient(circle at 85% 30%, rgba(167, 139, 250, 0.05), transparent 25%);
                    }
                    .dashboard-header {
                        display: flex;
                        justify-content: space-between;
                        align-items: center;
                        margin-bottom: 2rem;
                        border-bottom: 1px solid rgba(255,255,255,0.1);
                        padding-bottom: 1rem;
                    }
                    .live-indicator {
                        display: flex;
                        align-items: center;
                        gap: 8px;
                        font-size: 0.9rem;
                        color: var(--accent-green);
                        font-weight: 500;
                    }
                    .dot {
                        width: 8px;
                        height: 8px;
                        background-color: var(--accent-green);
                        border-radius: 50%;
                        box-shadow: 0 0 10px var(--accent-green);
                        animation: pulse 2s infinite;
                    }
                    @keyframes pulse {
                        0% { opacity: 1; transform: scale(1); }
                        50% { opacity: 0.5; transform: scale(1.2); }
                        100% { opacity: 1; transform: scale(1); }
                    }
                    .grid {
                        display: grid;
                        grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
                        gap: 1.5rem;
                        margin-bottom: 1.5rem;
                    }
                    .card {
                        background: var(--bg-card);
                        backdrop-filter: blur(12px);
                        border: 1px solid rgba(255,255,255,0.05);
                        border-radius: 16px;
                        padding: 1.5rem;
                        box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
                        transition: transform 0.2s ease;
                    }
                    .card:hover {
                        transform: translateY(-2px);
                        border-color: rgba(255,255,255,0.1);
                    }
                    .stat-value {
                        font-size: 2.5rem;
                        font-weight: 700;
                        margin: 0.5rem 0;
                        background: linear-gradient(135deg, #fff, #a5b4fc);
                        -webkit-background-clip: text;
                        -webkit-text-fill-color: transparent;
                    }
                    .stat-label {
                        color: var(--text-muted);
                        font-size: 0.875rem;
                        text-transform: uppercase;
                        letter-spacing: 0.05em;
                        font-weight: 600;
                    }
                    .chart-container {
                        position: relative;
                        height: 250px;
                        width: 100%;
                    }
                    .wide-card {
                        grid-column: 1 / -1;
                    }
                    .metrics-row {
                        display: flex;
                        justify-content: space-between;
                        padding: 0.75rem 0;
                        border-bottom: 1px solid rgba(255,255,255,0.05);
                    }
                    .metrics-row:last-child {
                        border-bottom: none;
                    }
                </style>
            </head>
            <body>
                <div class="dashboard-header">
                    <div>
                        <h1 style="margin:0; font-weight:800; font-size:2rem;">Supernode<span style="color:var(--accent-blue)">Intelligence</span></h1>
                        <p style="margin:0; margin-top:0.5rem; color:var(--text-muted);">Real-time Storage & Network Telemetry</p>
                    </div>
                    <div class="live-indicator">
                        <div class="dot"></div>
                        SYSTEM LIVE
                    </div>
                </div>

                <div class="grid">
                    <div class="card">
                        <div class="stat-label">Total Connected Peers</div>
                        <div class="stat-value" id="peer-count">0</div>
                        <div style="font-size:0.85rem; color:var(--text-muted)">Across all transport layers</div>
                    </div>
                    
                    <div class="card">
                        <div class="stat-label">Total Blobs Managed</div>
                        <div class="stat-value" id="blob-count">0</div>
                        <div style="font-size:0.85rem; color:var(--text-muted)">Content addressed chunks</div>
                    </div>

                    <div class="card">
                        <div class="stat-label">Storage Footprint</div>
                        <div class="stat-value" id="storage-bytes">0 B</div>
                        <div style="font-size:0.85rem; color:var(--text-muted)" id="ingestion-ratio">Ingested: 0 B | Retrieved: 0 B</div>
                    </div>
                </div>

                <div class="grid">
                    <div class="card wide-card">
                        <h3 style="margin-top:0; color:var(--text-muted)">Network Throughput (Messages/s)</h3>
                        <div class="chart-container">
                            <canvas id="trafficChart"></canvas>
                        </div>
                    </div>
                </div>

                <div class="grid">
                    <div class="card wide-card">
                        <h3 style="margin-top:0; color:var(--text-muted)">Transport Health</h3>
                        <div id="transport-health" style="display:grid; grid-template-columns:repeat(auto-fit,minmax(220px,1fr)); gap:1rem;"></div>
                    </div>
                </div>

                <div class="grid">
                    <div class="card wide-card">
                        <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:1rem">
                            <h3 style="margin:0; color:var(--text-muted)">Performance Benchmark</h3>
                            <button id="run-bench" onclick="runBenchmark()" style="background:linear-gradient(135deg,#38bdf8,#a78bfa);color:#fff;border:none;padding:8px 20px;border-radius:8px;font-weight:600;cursor:pointer;font-size:0.85rem;transition:transform 0.15s"
                                onmouseover="this.style.transform='scale(1.05)'" onmouseout="this.style.transform='scale(1)'">\u25B6 Run Benchmark</button>
                        </div>
                        <div id="bench-results" style="font-size:0.85rem;color:var(--text-muted)">Click 'Run Benchmark' to measure storage throughput and latency.</div>
                    </div>
                </div>
                
                <div class="grid">
                    <div class="card">
                        <h3 style="margin-top:0; color:var(--text-muted)">Storage Statistics</h3>
                        <div id="storage-details">
                            <div class="metrics-row"><span>Manifests Stored</span><span id="manifest-count" style="font-weight:600; color:var(--accent-blue)">0</span></div>
                            <div class="metrics-row"><span>Erasure Config</span><span id="erasure-config" style="font-weight:600; color:var(--accent-purple)">N/A</span></div>
                            <div class="metrics-row"><span>Files Ingested</span><span id="files-ingested" style="font-weight:600; color:var(--text-main)">0</span></div>
                            <div class="metrics-row"><span>Active Operations</span><span id="active-ops" style="font-weight:600; color:var(--accent-green)">0</span></div>
                        </div>
                    </div>
                    
                    <div class="card">
                        <h3 style="margin-top:0; color:var(--text-muted)">Transport Breakdown</h3>
                        <div class="chart-container" style="height: 200px;">
                            <canvas id="transportPie"></canvas>
                        </div>
                    </div>
                </div>

                <script>
                    function formatBytes(bytes, decimals = 2) {
                        if (!+bytes) return '0 B';
                        const k = 1024, dm = decimals < 0 ? 0 : decimals, sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
                        const i = Math.floor(Math.log(bytes) / Math.log(k));
                        return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`;
                    }
                    
                    const ctx = document.getElementById('trafficChart').getContext('2d');
                    const trafficChart = new Chart(ctx, {
                        type: 'line',
                        data: {
                            labels: Array(20).fill(''),
                            datasets: [{
                                label: 'Messages Sent/s',
                                borderColor: '#38bdf8',
                                backgroundColor: 'rgba(56, 189, 248, 0.1)',
                                borderWidth: 2,
                                fill: true,
                                tension: 0.4,
                                data: Array(20).fill(0)
                            }, {
                                label: 'Messages Received/s',
                                borderColor: '#a78bfa',
                                backgroundColor: 'rgba(167, 139, 250, 0.1)',
                                borderWidth: 2,
                                fill: true,
                                tension: 0.4,
                                data: Array(20).fill(0)
                            }]
                        },
                        options: {
                            responsive: true,
                            maintainAspectRatio: false,
                            animation: { duration: 0 },
                            scales: {
                                y: { beginAtZero: true, grid: { color: 'rgba(255, 255, 255, 0.05)' } },
                                x: { display: false, grid: { display: false } }
                            },
                            plugins: {
                                legend: { labels: { color: '#94a3b8' } }
                            }
                        }
                    });
                    
                    let lastIn = 0, lastOut = 0;

                    async function updateMetrics() {
                        try {
                            const res = await fetch('/api/metrics');
                            const data = await res.json();
                            
                            // High level numbers
                            document.getElementById('peer-count').innerText = data.peerCount || 0;
                            
                            if (data.storage) {
                                document.getElementById('blob-count').innerText = data.storage.blobCount || 0;
                                document.getElementById('storage-bytes').innerText = formatBytes(data.storage.totalBytes || 0);
                                document.getElementById('ingestion-ratio').innerText = `Ingested: ${formatBytes(data.storage.totalBytesIngested)} | Retrieved: ${formatBytes(data.storage.totalBytesRetrieved)}`;
                                
                                document.getElementById('manifest-count').innerText = data.storage.manifestCount || 0;
                                if (data.storage.erasure) {
                                    document.getElementById('erasure-config').innerText = `${data.storage.erasure.dataShards}+${data.storage.erasure.parityShards}`;
                                }
                                document.getElementById('files-ingested').innerText = data.storage.totalFilesIngested || 0;
                                document.getElementById('active-ops').innerText = data.storage.activeOperations || 0;
                            }
                            
                            // Traffic Graph
                            if (data.transport) {
                                let currIn = data.transport.totalMessagesReceived || 0;
                                let currOut = data.transport.totalMessagesSent || 0;
                                
                                let diffIn = lastIn === 0 ? 0 : currIn - lastIn;
                                let diffOut = lastOut === 0 ? 0 : currOut - lastOut;
                                
                                lastIn = currIn;
                                lastOut = currOut;
                                
                                const datasets = trafficChart.data.datasets;
                                datasets[0].data.shift();
                                datasets[0].data.push(diffOut);
                                datasets[1].data.shift();
                                datasets[1].data.push(diffIn);
                                trafficChart.update();
                            }
                            
                        } catch (e) {
                            console.error('Failed to fetch metrics:', e);
                        }
                    }

                    const stateColors = {
                        HEALTHY: '#34d399', DEGRADED: '#fbbf24', UNHEALTHY: '#f87171',
                        STOPPED: '#64748b', UNKNOWN: '#94a3b8'
                    };

                    async function updateHealth() {
                        try {
                            const res = await fetch('/api/health');
                            const data = await res.json();
                            const container = document.getElementById('transport-health');
                            container.innerHTML = '';
                            for (const [name, info] of Object.entries(data)) {
                                const color = stateColors[info.state] || '#94a3b8';
                                const cbOpen = info.circuitBreaker?.open ? '<span style="color:#f87171;font-size:0.75rem">⚡ BREAKER OPEN</span>' : '';
                                container.innerHTML += `
                                    <div style="background:rgba(15,23,42,0.6);border:1px solid ${color}33;border-radius:12px;padding:1rem;">
                                        <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.5rem">
                                            <span style="font-weight:700;font-size:0.9rem">${name}</span>
                                            <span style="background:${color}22;color:${color};padding:2px 8px;border-radius:8px;font-size:0.75rem;font-weight:600">${info.state}</span>
                                        </div>
                                        <div style="color:var(--text-muted);font-size:0.8rem">${info.message || ''}</div>
                                        <div style="display:flex;justify-content:space-between;margin-top:0.5rem;font-size:0.8rem;color:var(--text-muted)">
                                            <span>Latency: ${info.latencyMs}ms</span>
                                            <span>Failures: ${info.consecutiveFailures}</span>
                                        </div>
                                        <div style="margin-top:0.25rem">${cbOpen}</div>
                                    </div>`;
                            }
                        } catch (e) {
                            console.error('Failed to fetch health:', e);
                        }
                    }

                    setInterval(updateMetrics, 1000);
                    setInterval(updateHealth, 2000);
                    updateMetrics();
                    updateHealth();

                    async function runBenchmark() {
                        const btn = document.getElementById('run-bench');
                        const results = document.getElementById('bench-results');
                        btn.disabled = true;
                        btn.innerText = '\u23F3 Running...';
                        results.innerHTML = '<span style="color:var(--accent-blue)">Benchmark in progress...</span>';
                        try {
                            const res = await fetch('/api/benchmark');
                            const data = await res.json();
                            let html = '<div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:1rem;margin-top:0.5rem">';
                            for (const r of data.results) {
                                const color = r.errors === 0 ? 'var(--accent-green)' : '#f87171';
                                const tput = (r.totalBytes / (1024*1024)) / (r.elapsed.seconds || 1);
                                html += `
                                    <div style="background:rgba(15,23,42,0.6);border:1px solid rgba(255,255,255,0.05);border-radius:10px;padding:1rem">
                                        <div style="font-weight:700;font-size:0.85rem;margin-bottom:0.5rem">${r.name}</div>
                                        <div style="font-size:0.8rem;color:var(--text-muted)">Avg: ${r.avgLatencyMs.toFixed(1)}ms &middot; P95: ${r.p95LatencyMs}ms</div>
                                        <div style="font-size:0.8rem;color:var(--text-muted)">Throughput: ${tput.toFixed(2)} MB/s</div>
                                        <div style="font-size:0.75rem;margin-top:0.25rem;color:${color}">${r.errors === 0 ? '\u2713 PASS' : '\u2717 ' + r.errors + ' errors'}</div>
                                    </div>`;
                            }
                            html += '</div>';
                            results.innerHTML = html;
                        } catch (e) {
                            results.innerHTML = '<span style="color:#f87171">Benchmark failed: ' + e.message + '</span>';
                        }
                        btn.disabled = false;
                        btn.innerText = '\u25B6 Run Benchmark';
                    }
                </script>
            </body>
            </html>
            """;
    }
}
