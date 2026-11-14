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
