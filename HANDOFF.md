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

## Unstake transition (2026-11-14)
- B height-3 `unstake` of full 200 (`9ab08d91...`) accepted live. Chain: 7 blocks. B: balance 1495, staked 0, height 3. Live-verified: open/send/receive/stake/unstake. **Finding:** staking yield truncates to 0 for small/short stakes (200 units x ~35min at 5%/yr = <0.01 -> int64 0); yield only materializes for large or long-held stakes — consider surfacing accrued-yield preview in UI before unstaking.

## NFT mint/transfer (2026-11-14)
- B height-4 `mint_nft` ('Genesis Shard #1', cost 50, NFT id `8d8d79c7...`) + height-5 `transfer_nft` to genesis account A (fee 1) accepted live. Chain: 9 blocks. `GET /nfts` shows owner=A. Live-verified: open/send/receive/stake/unstake/mint_nft/transfer_nft. Remaining live-unverified: swaps (initiate/claim/refund), governance proposals/votes, publish_manifest.

## Swap HTLC lifecycle (2026-11-14)
- A height-3 `initiate_swap` (300 locked to B, `fdbaa3b6...`) + B height-6 `claim_swap` with secret preimage accepted live; `GET /swaps` status CLAIMED. Chain: 11 blocks, B balance 1739. Live-verified: open/send/receive/stake/unstake/mint_nft/transfer_nft/initiate_swap/claim_swap. Remaining: refund_swap (needs expiry), governance, publish_manifest. Note: one transient lattice startup failure observed (empty logs, likely DB-lock race after force-kill) — startup succeeded on retry; graceful shutdown would reduce this.

## CONSENSUS MATRIX CLOSED (2026-11-14) — all 17 types live-verified on real chain
Chain: **20 blocks** persisted in `data/lattice/lattice.db` (accounts: genesis wallet `6Rkjn...` + `BobReceiver111...`). Every dispatch type in `internal/consensus/lattice.go` accepted via `POST /process` against lattice-go and visible through supernode `/lattice` (count=20):
- open (genesis SYSTEM_GENESIS + receiver open), send x3, receive, stake, unstake,
- mint_nft + transfer_nft (NFT 'Genesis Shard #1' now owned by genesis account),
- initiate_swap + claim_swap (HTLC CLAIMED), initiate_swap + refund_swap (expired, REFUNDED),
- proposal (cost 10) + vote (quadratic power 41.69 -> status passed),
- publish_manifest (anchor 'manifest-genesis-shard-1'), market_bid (50) + accept_bid (+50),
- data_anchor (cost 10; /anchors count=2), achievement_unlock (informational, no balance change).
Notes: `GET /market/bids` returns null when no OPEN bids remain (accepted bids filtered) — expected. First governance attempt failed on my arithmetic (used cost 100 vs ProposalCost 10); engine rejection message was exact and self-correcting. Demurrage (0.0001/60000 per ms) must be replicated exactly for every balance-bearing block.

## v11.60.51: dashboard lattice-state panels
- All lattice state (proposals/NFTs/swaps/anchors/bids/frontiers) reachable same-origin via supernode proxy and rendered in Wallet & Lattice tab. One transient full-suite failure observed then PASS on verbose rerun (TestHandleMessengerSocket timing suspected flake — watch it).

## Signed blocks + flake hunt (2026-11-14)
- **Signature path live-verified both ways**: bad-signature control rejected (400 'invalid signature for block 3ef569ab...'), valid Ed25519-signed `send` (wallet `6Rkjn...`, signed via python `cryptography`) accepted. Chain: **21 blocks**. Unsigned blocks remain accepted by design (signature verified only when present) — requiring signatures is a policy decision for the user.
- **Messenger test flake**: not reproduced in 8 consecutive runs (5x focused messenger/signaling/matchmaker + 3x full package, plus vet clean). The single earlier failure coincided with a parallel `build.bat` in the same command; suspected CPU contention with the 2.26s messenger test. `go test -race` unavailable (needs CGO; local mingw broken). Watch, don't chase.

## v11.60.52 two-node sync
- lattice-go `-port`/`-peers` added. Fresh node2 (:4002, temp DB) + node1 (:4000, real 21-block DB): bootstrap applied 21 blocks, hashes match. Port map warning: bobcoin runs go-lattice on :4001 and go-supertorrent on :8000 — bobtorrent defaults collide with both; use -port 4002+/8001+ alongside bobcoin.

## Autonomous background sync (2026-11-14)
- Node2 (peer-registered via `-peers`) caught up 21->22 within one 30s sync interval after a new `achievement_unlock` block (`a278f148...`) was submitted to node1 only — no bootstrap call. Log: 'Autonomous sync: Peer http://127.0.0.1:4000 is ahead. Catching up...'. Chain now **22 blocks** on both node DBs. Multi-node story complete: bootstrap catch-up + autonomous reconciliation both live-proven.

## Divergence safety (2026-11-14)
- Foreign-genesis node (:4003, own SYSTEM_GENESIS supply 777) vs main 22-block chain: `POST /reconcile` -> relationship `divergent`, suggestedAction `investigate_divergence`; `POST /reconcile/apply` -> **409 refused** ('Safe reconciliation refuses to execute when histories are ambiguous or divergent'), executed=false. Neither chain mutated (node3 stayed 1 block, node1 stayed 22). Temp divergent DB deleted. The reconcile layer fails closed exactly as designed.

## v11.60.53 peer panel
- UI coverage complete: every supernode + lattice endpoint family now has a dashboard surface (only /announce + /matchmaking remain protocol-only by design). One transient Go compiler trace seen during a loaded build; clean rebuild + tests pass, binary reports 11.60.53 — treat as environment flake unless it recurs.

## Persistence check (2026-11-14)
- `GET /persistence/verify` on real DB: healthy, quick_check ok, 22 blocks, sequence 22, export 200 (10,388 bytes). Full `go test ./...` green. Durable findings consolidated into MEMORY.md (headless parity, version chain, CGO, port map, consensus math, multi-node, dashboard rule).

## Secret untracking (2026-11-14)
- `storage/wallet.json` (contains `secretKey`) removed from the index (`git rm --cached`); file remains on disk and is covered by the existing `/storage/` ignore rule. It stays in git HISTORY (commit 0605fb6) — not rewritten. **Owner action: rotate that keypair and decide on a history purge.** `MobileMessenger/ios/.xcode.env` appears in historical paths only; not present in the worktree.

## Second tracked secret (2026-11-14)
- `data/node-b/identity.json` (test node-b identity: publicKey + 128-char secretKey) also untracked via `git rm --cached`; on disk, `data/` ignored. Full tracked-tree scan for secret-shaped keys: remaining hits are source-code field names/tests, no other data files. **Rotation list: (1) storage/wallet.json key, (2) node-b identity key.** Both remain in history. `data/node-b/video.mp4` is tracked test media (~large) — owner decision to untrack/LFS.

## Full-component smoke (2026-11-14, v11.60.55 binaries)
All three Go binaries launched together below-normal: lattice :4000 (200), supernode :8001 — dashboard /, mobile_preview.html, storage.wasm (22MB), wasm_exec.js, /status, /stats, /debug/pprof all 200; dht-proxy :9080 up (root / is 404 **by design** — only /api/announce + /api/torrent/add exist; GeoIP DB absent -> graceful non-enriched mode, logged as warning not error). Only degraded item remains I2P (no SAM on :7656).

## Snapshot cold boot (2026-11-14)
- Submitted 4 `achievement_unlock` blocks to cross the snapshot interval (25): chain 22->26, `snapshotCount=1, latestSnapshotSequence=25`. Cold restart replayed snapshot+tail: total=26, latest hash identical (`812bdacc...`), verify healthy. Snapshot-backed boot is live-proven, not just unit-tested.
- Repo state: another actor merged `main` into `master` and synced the `bobcoin` submodule pointer (Protocols #300-305); branch was fully up to date with origin at check time.

## v11.60.56 dashboard write path
- Wallet tab Submit Lattice Block (Advanced) posts raw/wrapped JSON to `/process` via supernode proxy; verified live through :8001 (chain 26->27, block `860b7155...`). Chain now **27 blocks**. Dashboard coverage: read + write complete.
- Branch audit: all local branches (jules-*, fix-wasm, monorepo, main) fully merged into master. Tray controller verified: menu Open/Start/Restart/Stop/Refresh/Quit(stops services); Windows `.exe` + `taskkill`/CIM guarded stop.

## v11.60.58 require-signatures
- Opt-in enforcement shipped (flag, default off). Owner policy decision is now: run lattice-go with `-require-signatures` or not — note signed-only mode refuses peer sync of unsigned history by design.

## Node test scope fix (2026-11-16)
- Root `npm test` discovered tests across submodules (element-web Jest/TS): 355 false failures. Repo's own suite scoped (`test/*.test.js`): **113/113 pass**. `package.json` test script scoped accordingly (one line). `npm test` still gates on `pretest` lint: 669 pre-existing standard-style findings in legacy JS (index.js/server.js/lib/test) — left untouched (mass style rewrite not advised); fix with `npx standard --fix` as a dedicated change if desired.

## CI node bump + gate audit (2026-11-16)
- CI workflow Node 18 -> 22: scoped test glob needs Node >=21 test-runner glob support; local is v22.23.4, engines permit it. go vet clean. gofmt: 46 pre-existing unformatted Go files (compact vendored style) deliberately not mass-formatted; new session files are gofmt-clean. JS lint debt later fixed to 0 (v11.60.59), superseding the earlier note.

## CI not firing (2026-11-17, owner action)
- GitHub Actions is enabled and ci/Release/stale workflows are active, but gh run list shows ZERO ci/Release runs for all Nov 14-16 master pushes (only scheduled stale runs). Local gates pass (Go suite, npm test 114/114, CI bumped to Node 22), so this is a repo/token-side trigger issue to inspect in the Actions tab (push token workflow scope is the usual suspect). Also: release.yml runs semantic-release with no config in-repo while governance is VERSION-file based, and package.json version had drifted to 11.2.4 - synced to 11.60.59 this turn. Decide: keep semantic-release or retire release.yml in favor of VERSION-file releases.

## CI resolved (2026-11-17)
- The not-firing mystery was a gh default-repo artifact: gh resolved to upstream webtorrent/bittorrent-tracker. On the fork (explicit -R robertpelloni/bobtorrent), ci.yml gained a Go build/vet/test job + workflow_dispatch; manual dispatch run 44030837396 is GREEN (Go 28s, Node 50s) after fixing npm install with --legacy-peer-deps (ERESOLVE on @lightprotocol/compressed-token). Push-triggered runs still unobserved - dispatch is the proven path; check push token if automatic runs are wanted.

## Push-trigger diagnosis (2026-11-17)
- Fork API confirms: only workflow_dispatch + Copilot dynamic events have EVER run; pushes create zero runs. Verified NOT the cause: Actions enabled (allowed_actions=all), ci+Release workflow state=active, push credential is owner OAuth (gho_, repo+workflow scopes), commits attributed to owner, workflow files valid (dispatch run 44030837396 green: Go 28s + Node 50s). Remaining suspects live only in GitHub Settings UI (fork workflow approval / Actions policy). Workaround proven: gh workflow run ci.yml -R robertpelloni/bobtorrent. Also note gh CLI defaults to upstream webtorrent/bittorrent-tracker in this workspace - always pass -R robertpelloni/bobtorrent.

## Disaster-recovery drill (2026-11-17)
- Full operator path live-verified on the real 27-block chain: POST /persistence/backup-bundle (passphrase, unsigned) -> wrong passphrase rejected 400 -> POST /persistence/restore-bundle -> fresh lattice-go cold boot on restored DB: 27/27 blocks, latest hash match. File-level proof at each stage (plain backup 27 rows; restored file 27 rows pre-boot and post-boot). One earlier drill run reported 0 blocks - not reproduced in two subsequent end-to-end runs; suspected concurrent-start WAL race in the drill harness (a hung first node attempt), not restore code. Durable unit proof added: internal/consensus/secure_bundle_snapshot_test.go (snapshot-crossed chain through bundle round-trip). Drill artifacts in TEMP were passphrase test files only - delete freely.

## v11.60.61 + port map update (2026-11-17)
- /subscribe key validation fixed + tested + live-verified on :8002. PORT MAP CHANGED: bobcoin go-casino.exe now owns :8001 too (in addition to go-supertorrent :8000, go-lattice :4001). Use supernode -port 8002 (or any free port) alongside bobcoin lanes; fixed sleeps are too short under multi-lane load - always readiness-loop before probing.

## Submodule repair (2026-11-17)
- Dead gitlink removed: MobileMessenger pointed at commit 215ce1e with no .gitmodules mapping and no such GitHub repo (Jules scaffold artifact); every git submodule command fataled on it. Removed from index only - pointer preserved in history (8fb7538), empty dir kept. git submodule status now lists bobcoin + element-web cleanly. Restore MobileMessenger as a real submodule if its repo URL is provided.

## Submodule catch-up audit (2026-11-17)
- bobcoin submodule is 24 commits behind its origin/main (through v8.117.31, ZK/SP1/casino releases from the bobcoin lane). NOT advanced here: pointer bumps are the sync owner (Protocols) lane and pull frontend changes against the Go compat surface. element-web checked separately. Parent repo gates (Go suite, npm test, fork CI) all green at current pointers.
