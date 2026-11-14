# Session Handoff (v11.60.47)

## What was broken at session start (all fixed + verified this session)
1. **`-headless` served no dashboard** — `main_wrapper.go` never called `startTrackerServices()`. The tray's launch mode had zero HTTP. Now headless = full service set, blocks on signals. Proven in logs: DHT :6882, libp2p messenger :6883, HTTP bind attempted, `Headless supernode v11.60.47 started`.
2. **`/status` lied about version** — hardcoded `11.40.0`; now `main.Version` from `-X main.Version` (VERSION file = single source). Test: `cmd/supernode-go/version_status_test.go`.
3. **No tray icon on Windows, ever** — gated on `$DISPLAY`, used `lsof` + extensionless paths. Rewritten cross-platform; GUI tray verified live (pystray+Pillow present). Quit stops services; stop only kills PIDs whose image matches our binaries.
4. **Airdrop button fabricated success** — now reports UNAVAILABLE, credits nothing (no such endpoint exists).
5. **`build.bat` could not build** — single-file `go build main.go` on a multi-file package, plus blocking `pause` + unrelated `npm install`. Rebuilt; `build.sh`/`build.bat` both default `CGO_ENABLED=0` (local mingw gcc emits objects Go cannot parse — this is also why plain `go test` fails here without it).

## Dashboard (single page, 5 tabs, unchanged IA — coverage completed)
Overview: telemetry, swarms, **Services & Network Health** (`/status`, `/status/i2p`, `/filecoin/status`, `/channels/browse`), **Add/Remove/Upload torrents**, blobs. Publishing: ingest/publish/subscribe/assets (unchanged). Mesh Chat + Mobile Preview (unchanged). Identity & Attestation (unchanged). Wallet: Solana wallet, **Node Economy** (`/bankroll`, `/transactions`, `/mint`, `/burn`), lattice ledger. Still no UI for: `/spora/*` proof, `/fhe-oracle`, `/matchmaking` signaling, `/debug/pprof` (operator-only by design — link from Services panel is a reasonable next step).

## NOT running right now (needs a human decision, not more code)
- **:8000 is owned by bobcoin's `go-supertorrent.exe` v8.117.3** ("Go Game Server orchestrator"), NOT bobtorrent. Bobtorrent's dashboard cannot bind :8000 until that process stops or the supernode gets a configurable port. I did not kill it (not this project's process). **Next step candidates: (a) stop bobcoin's server when you want bobtorrent in front, or (b) add `-port` flag to supernode-go (HTTP + latticeURL + tray + UI are all hardcoded to 8000 today).**
- **lattice-go (:4000) not running** — verified it binds and serves HTTP 200 below-normal; left stopped after the smoke test. Start via tray or `build\lattice-go.exe`.
- **I2P datagrams OFF** — no SAM bridge at localhost:7656 (expected; install/run i2pd to enable). Filecoin bridge mode depends on Lotus config.
- Edge CDP :9222 and HyperNexus kernel :7778 are up and healthy (not this repo's processes).

## State
- Version 11.60.47. Build: `cmd /c build.bat` OK (supernode-go.exe, lattice-go.exe, dht-proxy.exe, storage.wasm). Tests: all Go packages pass with `CGO_ENABLED=0`.
- Pre-existing uncommitted `bobcoin` submodule pointer bump (31c79ff→09c5012) and untracked `.hypernexus/` were left untouched — not mine to commit.
- The 15-minute autopilot loop (cron 91f8f159) re-runs "please continue to proceed as you advise"; delete it when the party should stop.

## v11.60.48 addendum
- `-port` flag added (default 8000): `build\supernode-go.exe -headless -port 8001` serves the full dashboard alongside bobcoin :8000 — verified live (dashboard + 9 endpoints HTTP 200, /status version=11.60.48). Tray honors `BOBTORRENT_PORT`.
- Lattice feed reconnect backoff 5s->60s (log spam fix).
- Still awaiting: GitHub re-auth to push commits b0253d5 + v11.60.48; lattice-go left stopped (start via tray); I2P needs i2pd SAM on :7656.

## v11.60.49 addendum
- Operator Tools wired + live-verified on :8001 (SPoRA 200 w/ chunkHash, pprof 200, FHE empty->400). Dashboard endpoint coverage is now complete except `/matchmaking` + `/announce` (protocol endpoints, not operator UI) and `/submit-proof` (covered via Identity attestation flow).
- Remaining blockers unchanged: GitHub re-auth for push (3 commits ahead after this one), bobcoin submodule bump decision, i2pd for I2P.

## v11.60.50 addendum
- Fabricated lattice ledger removed; /lattice proxies real chain (empty = []). To see real blocks, submit via lattice-go /process (bobcoin-compat proxied at supernode /process too).
- E2E pair (lattice:4000 + supernode:8001) verified working together; both left stopped (session processes do not persist).

## Genesis block (2026-11-14, live-verified)
- First real block submitted to lattice-go `POST /process`: SYSTEM_GENESIS open for supernode wallet `6RkjnNtnsxm99BsJJuJJcCkLhtJEkHr1xYuaXrS7wcZw`, hash `a07810ce3df0860acb71ee80a7225dfe09c0e68aed1890025e679a4851f6ebff`, balance 1000000, persisted in `data/lattice/lattice.db`. Supernode `/lattice` now returns this real block; `/balance` reads 999995 (demurrage applied — worth a UX note in the wallet card someday).
- Restart the pair (lattice-go + supernode -port 8001) and the dashboard ledger shows the genesis block; chain height grows only via real `/process` submissions.

## Send/receive verification (2026-11-14)
- Height-1 `send` (1,000 units, hash `d1aa7b0a...`) from genesis account to `BobReceiver111...` + receiver `open` (hash `9c595032...`) both accepted by lattice-go `/process` with demurrage-exact balance math (rate 0.0001/60000/ms). Chain: 3 blocks, persisted. Balances: A=997466 (decayed), B=1000. Supernode `/lattice` lists all 3 in order — dashboard ledger is fully real end-to-end.

## Receive transition (2026-11-14)
- A height-2 `send` (500, `6201d5a5...`) + B height-1 `receive` (`2262c498...`) accepted. Chain: 5 blocks. Balances A=995587, B=1499 (1000 decayed + 500). open/send/receive all live-verified through the real HTTP path; dashboard `/lattice` count=5. Remaining transition types (stake/swap/NFT/governance) are unit-tested in `internal/consensus` but not yet exercised live.

## Stake transition (2026-11-14)
- B height-2 `stake` of 200 (`54a775ee...`) accepted live. Chain: 6 blocks. B frontier: balance 1297, staked_balance 200, height 2. Live-verified types now: open, send, receive, stake. Remaining live-unverified: unstake (yield-bearing), swaps, NFT, governance, publish_manifest.
