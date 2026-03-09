package io.supernode.storage;

import java.security.SecureRandom;
import java.time.Duration;
import java.time.Instant;
import java.util.*;
import java.util.concurrent.*;
import java.util.concurrent.atomic.AtomicLong;

/**
 * Automated storage benchmarking utility.
 * Measures throughput, latency, and correctness for ingest/retrieve operations
 * under configurable concurrency and data size parameters.
 */
public class StorageBenchmark {

    private final SupernodeStorage storage;
    private final SecureRandom random = new SecureRandom();

    public StorageBenchmark(SupernodeStorage storage) {
        this.storage = storage;
    }

    /**
     * Runs a full benchmark suite with the given options.
     */
    public BenchmarkReport run(BenchmarkOptions options) {
        List<BenchmarkResult> results = new ArrayList<>();

        // Sequential ingest benchmark
        results.add(benchmarkIngest("sequential-ingest", options.fileSizeBytes, 1, options.iterations));

        // Concurrent ingest benchmark
        if (options.concurrency > 1) {
            results.add(benchmarkIngest("concurrent-ingest", options.fileSizeBytes, options.concurrency, options.iterations));
        }

        // Sequential retrieve benchmark
        results.add(benchmarkRetrieve("sequential-retrieve", options.fileSizeBytes, 1, options.iterations));

        // Concurrent retrieve benchmark
        if (options.concurrency > 1) {
            results.add(benchmarkRetrieve("concurrent-retrieve", options.fileSizeBytes, options.concurrency, options.iterations));
        }

        // Data integrity verification
        results.add(benchmarkIntegrity("integrity-check", options.fileSizeBytes, options.iterations));

        return new BenchmarkReport(results, Instant.now(), options);
    }

    private BenchmarkResult benchmarkIngest(String name, int fileSizeBytes, int concurrency, int iterations) {
        byte[] masterKey = generateKey();
        ExecutorService pool = Executors.newFixedThreadPool(concurrency);
        List<Long> latencies = Collections.synchronizedList(new ArrayList<>());
        AtomicLong totalBytes = new AtomicLong();
        AtomicLong errors = new AtomicLong();

        Instant start = Instant.now();

        List<Future<?>> futures = new ArrayList<>();
        for (int i = 0; i < iterations; i++) {
            final int idx = i;
            futures.add(pool.submit(() -> {
                try {
                    byte[] data = new byte[fileSizeBytes];
                    random.nextBytes(data);
                    Instant opStart = Instant.now();
                    storage.ingest(data, "bench-" + idx + ".dat", masterKey);
                    long latencyMs = Duration.between(opStart, Instant.now()).toMillis();
                    latencies.add(latencyMs);
                    totalBytes.addAndGet(fileSizeBytes);
                } catch (Exception e) {
                    errors.incrementAndGet();
                }
            }));
        }

        awaitAll(futures);
        pool.shutdown();

        Duration elapsed = Duration.between(start, Instant.now());
        return buildResult(name, latencies, totalBytes.get(), errors.get(), elapsed, iterations);
    }

    private BenchmarkResult benchmarkRetrieve(String name, int fileSizeBytes, int concurrency, int iterations) {
        byte[] masterKey = generateKey();

        // Pre-ingest files for retrieval
        List<String> fileIds = new ArrayList<>();
        for (int i = 0; i < iterations; i++) {
            byte[] data = new byte[fileSizeBytes];
            random.nextBytes(data);
            SupernodeStorage.IngestResult result = storage.ingest(data, "retrieve-bench-" + i + ".dat", masterKey);
            fileIds.add(result.fileId());
        }

        ExecutorService pool = Executors.newFixedThreadPool(concurrency);
        List<Long> latencies = Collections.synchronizedList(new ArrayList<>());
        AtomicLong totalBytes = new AtomicLong();
        AtomicLong errors = new AtomicLong();

        Instant start = Instant.now();

        List<Future<?>> futures = new ArrayList<>();
        for (int i = 0; i < iterations; i++) {
            final String fileId = fileIds.get(i);
            futures.add(pool.submit(() -> {
                try {
                    Instant opStart = Instant.now();
                    SupernodeStorage.RetrieveResult result = storage.retrieve(fileId, masterKey);
                    long latencyMs = Duration.between(opStart, Instant.now()).toMillis();
                    latencies.add(latencyMs);
                    totalBytes.addAndGet(result.data().length);
                } catch (Exception e) {
                    errors.incrementAndGet();
                }
            }));
        }

        awaitAll(futures);
        pool.shutdown();

        Duration elapsed = Duration.between(start, Instant.now());
        return buildResult(name, latencies, totalBytes.get(), errors.get(), elapsed, iterations);
    }

    private BenchmarkResult benchmarkIntegrity(String name, int fileSizeBytes, int iterations) {
        byte[] masterKey = generateKey();
        List<Long> latencies = new ArrayList<>();
        AtomicLong totalBytes = new AtomicLong();
        long errors = 0;

        Instant start = Instant.now();

        for (int i = 0; i < iterations; i++) {
            byte[] data = new byte[fileSizeBytes];
            random.nextBytes(data);

            Instant opStart = Instant.now();
            SupernodeStorage.IngestResult ingestResult = storage.ingest(data, "integrity-" + i + ".dat", masterKey);
            SupernodeStorage.RetrieveResult retrieveResult = storage.retrieve(ingestResult.fileId(), masterKey);

            long latencyMs = Duration.between(opStart, Instant.now()).toMillis();
            latencies.add(latencyMs);
            totalBytes.addAndGet(fileSizeBytes);

            if (!Arrays.equals(data, retrieveResult.data())) {
                errors++;
            }
        }

        Duration elapsed = Duration.between(start, Instant.now());
        return buildResult(name, latencies, totalBytes.get(), errors, elapsed, iterations);
    }

    private BenchmarkResult buildResult(String name, List<Long> latencies, long totalBytes,
                                         long errors, Duration elapsed, int iterations) {
        if (latencies.isEmpty()) {
            return new BenchmarkResult(name, 0, 0, 0, 0, 0, 0, totalBytes, errors, elapsed, iterations);
        }

        Collections.sort(latencies);
        long min = latencies.getFirst();
        long max = latencies.getLast();
        double avg = latencies.stream().mapToLong(l -> l).average().orElse(0);
        long p50 = latencies.get(latencies.size() / 2);
        long p95 = latencies.get((int) (latencies.size() * 0.95));
        long p99 = latencies.get(Math.min((int) (latencies.size() * 0.99), latencies.size() - 1));

        double throughputMBps = elapsed.toMillis() > 0
                ? (totalBytes / (1024.0 * 1024.0)) / (elapsed.toMillis() / 1000.0)
                : 0;

        return new BenchmarkResult(name, min, max, avg, p50, p95, p99, totalBytes, errors, elapsed, iterations);
    }

    private void awaitAll(List<Future<?>> futures) {
        for (Future<?> f : futures) {
            try {
                f.get(60, TimeUnit.SECONDS);
            } catch (Exception e) {
                // Counted as error in the benchmark
            }
        }
    }

    private byte[] generateKey() {
        byte[] key = new byte[32];
        random.nextBytes(key);
        return key;
    }

    // --- Records ---

    public record BenchmarkOptions(int fileSizeBytes, int iterations, int concurrency) {
        public static BenchmarkOptions quick() {
            return new BenchmarkOptions(64 * 1024, 5, 2);      // 64KB x 5
        }

        public static BenchmarkOptions standard() {
            return new BenchmarkOptions(256 * 1024, 10, 4);     // 256KB x 10
        }

        public static BenchmarkOptions stress() {
            return new BenchmarkOptions(1024 * 1024, 20, 8);    // 1MB x 20
        }
    }

    public record BenchmarkResult(
        String name,
        long minLatencyMs,
        long maxLatencyMs,
        double avgLatencyMs,
        long p50LatencyMs,
        long p95LatencyMs,
        long p99LatencyMs,
        long totalBytes,
        long errors,
        Duration elapsed,
        int iterations
    ) {
        public double throughputMBps() {
            return elapsed.toMillis() > 0
                    ? (totalBytes / (1024.0 * 1024.0)) / (elapsed.toMillis() / 1000.0)
                    : 0;
        }

        public boolean passed() {
            return errors == 0;
        }
    }

    public record BenchmarkReport(
        List<BenchmarkResult> results,
        Instant completedAt,
        BenchmarkOptions options
    ) {
        public boolean allPassed() {
            return results.stream().allMatch(BenchmarkResult::passed);
        }

        public long totalErrors() {
            return results.stream().mapToLong(BenchmarkResult::errors).sum();
        }
    }
}
