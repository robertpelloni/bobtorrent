# Bobtorrent Omni-Workspace Handoff (v11.26.0)

## Session Objective
Eliminate the biggest architectural weakness in the Go lattice by introducing durable persistence and cold-boot recovery, then sync docs/versioning around the new operational reality.

## What Was Implemented

### 1. Durable SQLite-backed lattice persistence
Files:
- `internal/consensus/store.go`
- `internal/consensus/lattice.go`
- `internal/consensus/server.go`
- `cmd/lattice-go/main.go`

Implemented a durable confirmed-block log using `modernc.org/sqlite`.

Current behavior:
- each confirmed block is appended to SQLite
- replay order is preserved via an autoincrement sequence
- `cmd/lattice-go` now boots in persistent mode by default
- database path defaults to `data/lattice/lattice.db`
- path can be overridden via `BOBTORRENT_LATTICE_DB`

### 2. Cold-boot replay recovery
Added `NewPersistentLattice` / `NewPersistentServer`.

Startup now:
- loads persisted confirmed blocks from SQLite
- replays them in commit order
- reconstructs chains
- reconstructs pending transfers
- reconstructs proposals / votes
- reconstructs bids / swaps / NFTs
- reconstructs manifest anchors and typed proof metadata
- restores rolling state hash deterministically

### 3. Atomic persistence guard
`ProcessBlock` now snapshots the in-memory lattice state before mutation when persistence is enabled.
If the SQLite append fails:
- the in-memory mutation is rolled back
- the API sees the block as rejected
- consensus state does not drift ahead of persistence

This is important because it prevents a partial-commit failure mode where memory would accept a block that durable storage did not.

### 4. Operational visibility
`/status` now reports:
- whether persistence is enabled
- persistence DB path
- persisted block count

### 5. Validation
Executed successfully:
- `go test ./internal/consensus -buildvcs=false`
- `go build -buildvcs=false ./...`
- `cd bobcoin/frontend && npm run build`

Added test:
- `TestPersistentLatticeReplaysConfirmedBlocksOnRestart`

This test proves that a persisted manifest-anchor flow survives restart and restores typed publisher proof metadata after replay.

## Strategic State After This Session
The project is materially more credible now because the lattice is no longer process-ephemeral.

Before this session:
- consensus correctness existed only in memory
- restart implied state loss

After this session:
- confirmed lattice history survives restart
- derived consensus state can be rebuilt on boot
- the biggest production-readiness blocker has been partially retired

## Remaining Gaps
1. Snapshot acceleration
   - replay works, but large histories will eventually need materialized snapshots for faster startup
2. Persistence repair/integrity tooling
   - no corruption detection / repair path yet
3. Real Filecoin bridge
   - still simulated
4. Deeper peer sync / catch-up
   - still not a full distributed state-sync protocol
5. Broader automated testing
   - persistence exists, but test breadth still lags overall feature scope

## Recommended Next Step
1. Expand long-horizon source reliability analysis
2. Add snapshot acceleration for lattice persistence
3. Deepen publisher attestation semantics further

## Notes for the Next Agent
- The lattice persistence design currently uses a durable append-only confirmed block log plus deterministic replay.
- This was chosen as the safest first persistence milestone because it preserves existing consensus logic rather than duplicating derived-state rules into SQL tables immediately.
- The next durable-state improvement should likely be periodic materialized snapshots rather than a full relational rewrite of every consensus index.
