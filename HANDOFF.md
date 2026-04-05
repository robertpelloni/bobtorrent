# Bobtorrent Omni-Workspace Handoff (v11.43.0)

## Session Objective
Continue persistence hardening by making snapshot cadence/retention operator-tunable instead of relying only on hardcoded defaults.

## What Was Implemented

### 1. Explicit snapshot config surface
Files:
- `internal/consensus/store.go`
- `internal/consensus/lattice.go`
- `internal/consensus/server.go`

Added a real snapshot configuration surface:
- `SnapshotConfig`
- `DefaultSnapshotConfig()`
- `NewLatticeStoreWithConfig()`
- `NewPersistentLatticeWithConfig()`
- env-driven `SnapshotConfigFromEnv()`

### 2. Operator environment controls
`NewPersistentLattice()` now honors:
- `BOBTORRENT_LATTICE_SNAPSHOT_INTERVAL`
- `BOBTORRENT_LATTICE_SNAPSHOT_RETENTION`

Behavior notes:
- interval can be set to `0` to disable automatic snapshot creation
- retention must remain at least `1`
- defaults still remain `25` / `3`

### 3. Runtime visibility
Lattice status now reports:
- `snapshotInterval`
- `snapshotRetention`

This gives operators a direct way to confirm the active persistence tuning rather than inferring it from defaults or source code.

### 4. Regression coverage
File:
- `internal/consensus/lattice_test.go`

Added `TestPersistentLatticeWithCustomSnapshotConfigHonorsIntervalAndRetention`, proving that custom config changes:
- snapshot cadence behavior
- retained snapshot count
- exported persistence metadata

## Validation
Executed successfully:
- `go test ./internal/consensus ./cmd/supernode-go ./internal/... -buildvcs=false`
- `go build -buildvcs=false ./...`

## Strategic State After This Session
Persistence hardening now includes:
- replay-backed block durability
- snapshots
- verification/repair
- export/backup/import/restore
- secure backup bundles
- mixed transition replay coverage
- operator-tunable snapshot cadence/retention

## Recommended Next Steps
1. Decide whether snapshot controls should remain startup-config-only or gain runtime/API mutability
2. Continue expanding persistence-aware replay coverage toward even larger mixed multi-account webs
3. Consider signed/shareable diagnostics packaging beyond the current plain JSON export
4. Continue evaluating which specialized Node surfaces are still worth porting further

## Notes for the Next Agent
- No running processes were terminated in this session.
- Snapshot interval `0` now disables automatic snapshot creation, while retention remains validated as at least `1`.
