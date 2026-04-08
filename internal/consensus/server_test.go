package consensus

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"bobtorrent/pkg/torrent"

	"github.com/gorilla/websocket"
)

func mustGenerateKeypair(t *testing.T) *torrent.Keypair {
	t.Helper()
	wallet, err := torrent.GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair failed: %v", err)
	}
	return wallet
}

func mustSignBlock(t *testing.T, block *torrent.Block, privateKey string) {
	t.Helper()
	if err := block.Sign(privateKey); err != nil {
		t.Fatalf("Sign failed for block %s: %v", block.Type, err)
	}
}

func postJSON(t *testing.T, handler http.Handler, path string, payload interface{}) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func decodeJSONBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	return body
}

func TestHandleProcessReturnsDuplicateMetadataForKnownBlock(t *testing.T) {
	server := NewServer()
	handler := server.HTTPHandler()
	wallet := mustGenerateKeypair(t)

	genesis := torrent.NewBlock("open", wallet.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	mustSignBlock(t, genesis, wallet.PrivateKey)

	first := postJSON(t, handler, "/process", map[string]interface{}{"block": genesis})
	if first.Code != http.StatusOK {
		t.Fatalf("expected first process status %d, got %d with %s", http.StatusOK, first.Code, first.Body.String())
	}
	firstBody := decodeJSONBody(t, first)
	if firstBody["accepted"] != true || firstBody["duplicate"] != false {
		t.Fatalf("expected first process to accept new block, got %#v", firstBody)
	}

	second := postJSON(t, handler, "/process", map[string]interface{}{"block": genesis})
	if second.Code != http.StatusOK {
		t.Fatalf("expected second process status %d, got %d with %s", http.StatusOK, second.Code, second.Body.String())
	}
	secondBody := decodeJSONBody(t, second)
	if secondBody["accepted"] != false || secondBody["duplicate"] != true {
		t.Fatalf("expected second process to report duplicate block, got %#v", secondBody)
	}
}

func TestHandleProcessAcceptsBothRawAndWrappedFormats(t *testing.T) {
	server := NewServer()
	handler := server.HTTPHandler()

	wallet1 := mustGenerateKeypair(t)
	rawGenesis := torrent.NewBlock("open", wallet1.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	mustSignBlock(t, rawGenesis, wallet1.PrivateKey)

	rawRec := postJSON(t, handler, "/process", rawGenesis)
	if rawRec.Code != http.StatusOK {
		t.Fatalf("expected raw block submission to succeed, got %d with %s", rawRec.Code, rawRec.Body.String())
	}
	if len(server.lattice.blocks) != 1 {
		t.Fatalf("expected 1 block in lattice after raw submission, got %d", len(server.lattice.blocks))
	}

	wrappedBlock := torrent.NewBlock("achievement_unlock", wallet1.PublicKey, &rawGenesis.Hash, rawGenesis.Balance, rawGenesis.StakedBalance, rawGenesis.Height+1, "a1", nil, map[string]interface{}{"achievement": "A1"})
	mustSignBlock(t, wrappedBlock, wallet1.PrivateKey)

	wrappedRec := postJSON(t, handler, "/process", map[string]interface{}{"block": wrappedBlock})
	if wrappedRec.Code != http.StatusOK {
		t.Fatalf("expected wrapped block submission to succeed, got %d with %s", wrappedRec.Code, wrappedRec.Body.String())
	}
	if len(server.lattice.blocks) != 2 {
		t.Fatalf("expected 2 blocks in lattice after wrapped submission, got %d", len(server.lattice.blocks))
	}
}

func TestHandleWebSocketEmitsLiveFeedOnNewBlock(t *testing.T) {
	server := NewServer()
	httpServer := httptest.NewServer(server.HTTPHandler())
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v", err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	var welcome map[string]interface{}
	if err := conn.ReadJSON(&welcome); err != nil {
		t.Fatalf("failed to read welcome message: %v", err)
	}
	if welcome["event"] != "CONNECTED" || welcome["type"] != "STATS" {
		t.Fatalf("expected STATS CONNECTED welcome message, got %#v", welcome)
	}

	wallet := mustGenerateKeypair(t)
	genesis := torrent.NewBlock("open", wallet.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	mustSignBlock(t, genesis, wallet.PrivateKey)

	processRec := postJSON(t, server.HTTPHandler(), "/process", genesis)
	if processRec.Code != http.StatusOK {
		t.Fatalf("expected block submission to succeed, got %d with %s", processRec.Code, processRec.Body.String())
	}

	var liveEvent map[string]interface{}
	if err := conn.ReadJSON(&liveEvent); err != nil {
		t.Fatalf("failed to read live block event: %v", err)
	}

	if liveEvent["type"] != "NEW_BLOCK" || liveEvent["event"] != "NEW_BLOCK" {
		t.Fatalf("expected NEW_BLOCK live event, got %#v", liveEvent)
	}
	blockPayload, ok := liveEvent["block"].(map[string]interface{})
	if !ok || blockPayload["hash"] != genesis.Hash {
		t.Fatalf("expected live event to carry genesis block hash %s, got block payload %#v", genesis.Hash, liveEvent["block"])
	}
	if liveEvent["stateHash"] == nil || liveEvent["stateHash"] == strings.Repeat("0", 64) {
		t.Fatalf("expected updated state hash in live event, got %v", liveEvent["stateHash"])
	}
}

func TestHandleBlocksPagesOrderedHistory(t *testing.T) {
	server := NewServer()
	handler := server.HTTPHandler()
	wallet := mustGenerateKeypair(t)

	genesis := torrent.NewBlock("open", wallet.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	mustSignBlock(t, genesis, wallet.PrivateKey)
	if err := server.lattice.ProcessBlock(genesis); err != nil {
		t.Fatalf("ProcessBlock genesis failed: %v", err)
	}

	blockOne := torrent.NewBlock("achievement_unlock", wallet.PublicKey, &genesis.Hash, genesis.Balance, genesis.StakedBalance, genesis.Height+1, "a1", nil, map[string]interface{}{"achievement": "A1"})
	mustSignBlock(t, blockOne, wallet.PrivateKey)
	if err := server.lattice.ProcessBlock(blockOne); err != nil {
		t.Fatalf("ProcessBlock blockOne failed: %v", err)
	}

	blockTwo := torrent.NewBlock("achievement_unlock", wallet.PublicKey, &blockOne.Hash, blockOne.Balance, blockOne.StakedBalance, blockOne.Height+1, "a2", nil, map[string]interface{}{"achievement": "A2"})
	mustSignBlock(t, blockTwo, wallet.PrivateKey)
	if err := server.lattice.ProcessBlock(blockTwo); err != nil {
		t.Fatalf("ProcessBlock blockTwo failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/blocks?limit=2", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected blocks status %d, got %d with %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	body := decodeJSONBody(t, rec)
	blocks, ok := body["blocks"].([]interface{})
	if !ok || len(blocks) != 2 {
		t.Fatalf("expected first page to contain 2 blocks, got %#v", body["blocks"])
	}
	if body["hasMore"] != true || body["cursorFound"] != true {
		t.Fatalf("expected first page to have more results, got %#v", body)
	}

	secondPageReq := httptest.NewRequest(http.MethodGet, "/blocks?after="+url.QueryEscape(blockOne.Hash)+"&limit=2", nil)
	secondPageRec := httptest.NewRecorder()
	handler.ServeHTTP(secondPageRec, secondPageReq)
	if secondPageRec.Code != http.StatusOK {
		t.Fatalf("expected second page status %d, got %d with %s", http.StatusOK, secondPageRec.Code, secondPageRec.Body.String())
	}
	secondBody := decodeJSONBody(t, secondPageRec)
	secondBlocks, ok := secondBody["blocks"].([]interface{})
	if !ok || len(secondBlocks) != 1 {
		t.Fatalf("expected second page to contain 1 block, got %#v", secondBody["blocks"])
	}
	lastBlock, ok := secondBlocks[0].(map[string]interface{})
	if !ok || lastBlock["hash"] != blockTwo.Hash {
		t.Fatalf("expected second page to end with blockTwo, got %#v", secondBlocks[0])
	}

	missingCursorReq := httptest.NewRequest(http.MethodGet, "/blocks?after=missing-hash&limit=2", nil)
	missingCursorRec := httptest.NewRecorder()
	handler.ServeHTTP(missingCursorRec, missingCursorReq)
	if missingCursorRec.Code != http.StatusOK {
		t.Fatalf("expected missing cursor status %d, got %d with %s", http.StatusOK, missingCursorRec.Code, missingCursorRec.Body.String())
	}
	missingBody := decodeJSONBody(t, missingCursorRec)
	if missingBody["cursorFound"] != false {
		t.Fatalf("expected cursorFound=false for missing cursor, got %#v", missingBody)
	}
}

func TestHandlePeersSyncsLateJoinerAndLearnsPeerList(t *testing.T) {
	origin := NewServer()
	originHandler := origin.HTTPHandler()
	originHTTP := httptest.NewServer(originHandler)
	defer originHTTP.Close()

	wallet := mustGenerateKeypair(t)
	genesis := torrent.NewBlock("open", wallet.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	mustSignBlock(t, genesis, wallet.PrivateKey)
	if err := origin.lattice.ProcessBlock(genesis); err != nil {
		t.Fatalf("origin genesis failed: %v", err)
	}
	blockOne := torrent.NewBlock("achievement_unlock", wallet.PublicKey, &genesis.Hash, genesis.Balance, genesis.StakedBalance, genesis.Height+1, "origin-a1", nil, map[string]interface{}{"achievement": "ORIGIN_A1"})
	mustSignBlock(t, blockOne, wallet.PrivateKey)
	if err := origin.lattice.ProcessBlock(blockOne); err != nil {
		t.Fatalf("origin blockOne failed: %v", err)
	}
	origin.lattice.AddPeer("peer-c:4000")

	joiner := NewServer()
	joinerHandler := joiner.HTTPHandler()

	registerRec := postJSON(t, joinerHandler, "/peers", map[string]interface{}{"addr": originHTTP.URL, "sync": true})
	if registerRec.Code != http.StatusOK {
		t.Fatalf("expected peer registration sync status %d, got %d with %s", http.StatusOK, registerRec.Code, registerRec.Body.String())
	}
	registerBody := decodeJSONBody(t, registerRec)
	syncPayload, ok := registerBody["sync"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected sync payload in peer registration response, got %#v", registerBody)
	}
	if syncPayload["appliedBlocks"] != float64(2) {
		t.Fatalf("expected 2 applied blocks during sync, got %#v", syncPayload)
	}

	if len(joiner.lattice.blocks) != 2 {
		t.Fatalf("expected joiner to catch up to 2 blocks, got %d", len(joiner.lattice.blocks))
	}
	peers := joiner.lattice.GetPeers()
	parsedOrigin, err := url.Parse(originHTTP.URL)
	if err != nil {
		t.Fatalf("failed to parse origin test server url: %v", err)
	}
	foundOrigin := false
	foundDiscovered := false
	for _, peer := range peers {
		if peer == parsedOrigin.Host {
			foundOrigin = true
		}
		if peer == "peer-c:4000" {
			foundDiscovered = true
		}
	}
	if !foundOrigin {
		t.Fatalf("expected joiner to retain origin peer registration, got %#v", peers)
	}
	if !foundDiscovered {
		t.Fatalf("expected joiner to learn discovered peer, got %#v", peers)
	}

	peerRec := httptest.NewRecorder()
	peerReq := httptest.NewRequest(http.MethodGet, "/peers", nil)
	joinerHandler.ServeHTTP(peerRec, peerReq)
	if peerRec.Code != http.StatusOK {
		t.Fatalf("expected peers status %d, got %d with %s", http.StatusOK, peerRec.Code, peerRec.Body.String())
	}
	peerBody := decodeJSONBody(t, peerRec)
	healthSummary, ok := peerBody["healthSummary"].(map[string]interface{})
	if !ok || healthSummary["healthy"] == nil {
		t.Fatalf("expected health summary in peers response, got %#v", peerBody)
	}
	diagnostics, ok := peerBody["diagnostics"].([]interface{})
	if !ok || len(diagnostics) == 0 {
		t.Fatalf("expected peer diagnostics in peers response, got %#v", peerBody["diagnostics"])
	}
}

func TestHandlePeersSyncRetriesTransientBlockFetchFailure(t *testing.T) {
	origin := NewServer()
	originHandler := origin.HTTPHandler()
	wallet := mustGenerateKeypair(t)
	genesis := torrent.NewBlock("open", wallet.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	mustSignBlock(t, genesis, wallet.PrivateKey)
	if err := origin.lattice.ProcessBlock(genesis); err != nil {
		t.Fatalf("origin genesis failed: %v", err)
	}

	blockFailures := 0
	flakyHTTP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/blocks") && blockFailures == 0 {
			blockFailures++
			http.Error(w, "temporary upstream fault", http.StatusBadGateway)
			return
		}
		originHandler.ServeHTTP(w, r)
	}))
	defer flakyHTTP.Close()

	joiner := NewServer()
	joinerHandler := joiner.HTTPHandler()
	registerRec := postJSON(t, joinerHandler, "/peers", map[string]interface{}{"addr": flakyHTTP.URL, "sync": true})
	if registerRec.Code != http.StatusOK {
		t.Fatalf("expected peer registration sync status %d, got %d with %s", http.StatusOK, registerRec.Code, registerRec.Body.String())
	}
	registerBody := decodeJSONBody(t, registerRec)
	syncPayload, ok := registerBody["sync"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected sync payload after retrying transient error, got %#v", registerBody)
	}
	if syncPayload["retryCount"] == float64(0) {
		t.Fatalf("expected retryCount > 0 after transient failure, got %#v", syncPayload)
	}
	if len(joiner.lattice.blocks) != 1 {
		t.Fatalf("expected joiner to catch up to 1 block after retry, got %d", len(joiner.lattice.blocks))
	}

	statusRec := httptest.NewRecorder()
	statusReq := httptest.NewRequest(http.MethodGet, "/status", nil)
	joinerHandler.ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("expected status response %d, got %d with %s", http.StatusOK, statusRec.Code, statusRec.Body.String())
	}
	statusBody := decodeJSONBody(t, statusRec)
	peerSync, ok := statusBody["peerSync"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected peerSync diagnostics in status response, got %#v", statusBody)
	}
	peersPayload, ok := peerSync["peers"].([]interface{})
	if !ok || len(peersPayload) == 0 {
		t.Fatalf("expected peer telemetry entries in status response, got %#v", peerSync)
	}
	parsedFlaky, err := url.Parse(flakyHTTP.URL)
	if err != nil {
		t.Fatalf("failed to parse flaky server url: %v", err)
	}
	matched := false
	for _, entry := range peersPayload {
		peerEntry, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if peerEntry["peer"] == parsedFlaky.Host {
			matched = true
			if peerEntry["lastRetryCount"] == float64(0) {
				t.Fatalf("expected lastRetryCount to record retry usage, got %#v", peerEntry)
			}
		}
	}
	if !matched {
		t.Fatalf("expected telemetry entry for flaky peer %s, got %#v", parsedFlaky.Host, peersPayload)
	}
}

func TestHandlePeersSkipsSyncDuringCooldownAfterFailure(t *testing.T) {
	server := NewServer()
	handler := server.HTTPHandler()

	first := postJSON(t, handler, "/peers", map[string]interface{}{"addr": "127.0.0.1:1", "sync": true})
	if first.Code != http.StatusBadGateway {
		t.Fatalf("expected initial failing sync status %d, got %d with %s", http.StatusBadGateway, first.Code, first.Body.String())
	}

	second := postJSON(t, handler, "/peers", map[string]interface{}{"addr": "127.0.0.1:1", "sync": true})
	if second.Code != http.StatusOK {
		t.Fatalf("expected cooldown-skipped sync status %d, got %d with %s", http.StatusOK, second.Code, second.Body.String())
	}
	body := decodeJSONBody(t, second)
	syncPayload, ok := body["sync"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected sync payload on cooldown skip, got %#v", body)
	}
	if syncPayload["skippedDueToCooldown"] != true {
		t.Fatalf("expected skippedDueToCooldown=true, got %#v", syncPayload)
	}
	if syncPayload["cooldownRemainingMs"] == nil || syncPayload["cooldownRemainingMs"] == float64(0) {
		t.Fatalf("expected cooldownRemainingMs to be populated, got %#v", syncPayload)
	}
}

func TestHandlePeersMarksDivergenceWhenRemoteLacksLocalCursor(t *testing.T) {
	remote := NewServer()
	remoteWallet := mustGenerateKeypair(t)
	remoteGenesis := torrent.NewBlock("open", remoteWallet.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	mustSignBlock(t, remoteGenesis, remoteWallet.PrivateKey)
	if err := remote.lattice.ProcessBlock(remoteGenesis); err != nil {
		t.Fatalf("remote genesis failed: %v", err)
	}
	remoteHTTP := httptest.NewServer(remote.HTTPHandler())
	defer remoteHTTP.Close()

	local := NewServer()
	localWallet := mustGenerateKeypair(t)
	localGenesis := torrent.NewBlock("open", localWallet.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	mustSignBlock(t, localGenesis, localWallet.PrivateKey)
	if err := local.lattice.ProcessBlock(localGenesis); err != nil {
		t.Fatalf("local genesis failed: %v", err)
	}

	registerRec := postJSON(t, local.HTTPHandler(), "/peers", map[string]interface{}{"addr": remoteHTTP.URL, "sync": true, "force": true})
	if registerRec.Code != http.StatusBadGateway {
		t.Fatalf("expected divergence sync status %d, got %d with %s", http.StatusBadGateway, registerRec.Code, registerRec.Body.String())
	}

	statusRec := httptest.NewRecorder()
	statusReq := httptest.NewRequest(http.MethodGet, "/status", nil)
	local.HTTPHandler().ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("expected status response %d, got %d with %s", http.StatusOK, statusRec.Code, statusRec.Body.String())
	}
	statusBody := decodeJSONBody(t, statusRec)
	peerSync, ok := statusBody["peerSync"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected peerSync diagnostics in status response, got %#v", statusBody)
	}
	peersPayload, ok := peerSync["peers"].([]interface{})
	if !ok || len(peersPayload) == 0 {
		t.Fatalf("expected peer telemetry entries in status response, got %#v", peerSync)
	}
	parsedRemote, err := url.Parse(remoteHTTP.URL)
	if err != nil {
		t.Fatalf("failed to parse remote server url: %v", err)
	}
	matched := false
	for _, entry := range peersPayload {
		peerEntry, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if peerEntry["peer"] == parsedRemote.Host {
			matched = true
			if peerEntry["health"] != "diverged" {
				t.Fatalf("expected diverged health, got %#v", peerEntry)
			}
			if peerEntry["lastDivergenceReason"] == nil || peerEntry["lastDivergenceReason"] == "" {
				t.Fatalf("expected divergence reason to be recorded, got %#v", peerEntry)
			}
		}
	}
	if !matched {
		t.Fatalf("expected telemetry entry for divergent peer %s, got %#v", parsedRemote.Host, peersPayload)
	}
}

func TestHandleReconcileReportsRemoteAhead(t *testing.T) {
	wallet := mustGenerateKeypair(t)
	genesis := torrent.NewBlock("open", wallet.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	mustSignBlock(t, genesis, wallet.PrivateKey)

	remote := NewServer()
	if err := remote.lattice.ProcessBlock(genesis); err != nil {
		t.Fatalf("remote genesis failed: %v", err)
	}
	blockOne := torrent.NewBlock("achievement_unlock", wallet.PublicKey, &genesis.Hash, genesis.Balance, genesis.StakedBalance, genesis.Height+1, "remote-a1", nil, map[string]interface{}{"achievement": "REMOTE_A1"})
	mustSignBlock(t, blockOne, wallet.PrivateKey)
	if err := remote.lattice.ProcessBlock(blockOne); err != nil {
		t.Fatalf("remote blockOne failed: %v", err)
	}
	remoteHTTP := httptest.NewServer(remote.HTTPHandler())
	defer remoteHTTP.Close()

	local := NewServer()
	if err := local.lattice.ProcessBlock(genesis); err != nil {
		t.Fatalf("local genesis failed: %v", err)
	}

	reconcileRec := postJSON(t, local.HTTPHandler(), "/reconcile", map[string]interface{}{"peer": remoteHTTP.URL})
	if reconcileRec.Code != http.StatusOK {
		t.Fatalf("expected reconcile status %d, got %d with %s", http.StatusOK, reconcileRec.Code, reconcileRec.Body.String())
	}
	body := decodeJSONBody(t, reconcileRec)
	report, ok := body["reconciliation"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected reconciliation payload, got %#v", body)
	}
	if report["relationship"] != "remote_ahead" {
		t.Fatalf("expected remote_ahead relationship, got %#v", report)
	}
	if report["suggestedAction"] != "bootstrap_from_peer" {
		t.Fatalf("expected bootstrap_from_peer suggestion, got %#v", report)
	}
}

func TestHandleReconcileReportsDivergence(t *testing.T) {
	remote := NewServer()
	remoteWallet := mustGenerateKeypair(t)
	remoteGenesis := torrent.NewBlock("open", remoteWallet.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	mustSignBlock(t, remoteGenesis, remoteWallet.PrivateKey)
	if err := remote.lattice.ProcessBlock(remoteGenesis); err != nil {
		t.Fatalf("remote genesis failed: %v", err)
	}
	remoteHTTP := httptest.NewServer(remote.HTTPHandler())
	defer remoteHTTP.Close()

	local := NewServer()
	localWallet := mustGenerateKeypair(t)
	localGenesis := torrent.NewBlock("open", localWallet.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	mustSignBlock(t, localGenesis, localWallet.PrivateKey)
	if err := local.lattice.ProcessBlock(localGenesis); err != nil {
		t.Fatalf("local genesis failed: %v", err)
	}

	reconcileRec := postJSON(t, local.HTTPHandler(), "/reconcile", map[string]interface{}{"peer": remoteHTTP.URL})
	if reconcileRec.Code != http.StatusOK {
		t.Fatalf("expected reconcile status %d, got %d with %s", http.StatusOK, reconcileRec.Code, reconcileRec.Body.String())
	}
	body := decodeJSONBody(t, reconcileRec)
	report, ok := body["reconciliation"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected reconciliation payload, got %#v", body)
	}
	if report["relationship"] != "divergent" {
		t.Fatalf("expected divergent relationship, got %#v", report)
	}
	if report["suggestedAction"] != "investigate_divergence" {
		t.Fatalf("expected investigate_divergence suggestion, got %#v", report)
	}
}

func TestHandleReconcileApplyExecutesSafeRemoteAheadSync(t *testing.T) {
	wallet := mustGenerateKeypair(t)
	genesis := torrent.NewBlock("open", wallet.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	mustSignBlock(t, genesis, wallet.PrivateKey)

	remote := NewServer()
	if err := remote.lattice.ProcessBlock(genesis); err != nil {
		t.Fatalf("remote genesis failed: %v", err)
	}
	blockOne := torrent.NewBlock("achievement_unlock", wallet.PublicKey, &genesis.Hash, genesis.Balance, genesis.StakedBalance, genesis.Height+1, "remote-a1", nil, map[string]interface{}{"achievement": "REMOTE_A1"})
	mustSignBlock(t, blockOne, wallet.PrivateKey)
	if err := remote.lattice.ProcessBlock(blockOne); err != nil {
		t.Fatalf("remote blockOne failed: %v", err)
	}
	remoteHTTP := httptest.NewServer(remote.HTTPHandler())
	defer remoteHTTP.Close()

	local := NewServer()
	if err := local.lattice.ProcessBlock(genesis); err != nil {
		t.Fatalf("local genesis failed: %v", err)
	}

	applyRec := postJSON(t, local.HTTPHandler(), "/reconcile/apply", map[string]interface{}{"peer": remoteHTTP.URL})
	if applyRec.Code != http.StatusOK {
		t.Fatalf("expected reconcile apply status %d, got %d with %s", http.StatusOK, applyRec.Code, applyRec.Body.String())
	}
	body := decodeJSONBody(t, applyRec)
	apply, ok := body["apply"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected apply payload, got %#v", body)
	}
	if apply["executed"] != true || apply["executionMode"] != "remote_to_local_sync" {
		t.Fatalf("expected executed remote_to_local_sync result, got %#v", apply)
	}
	if len(local.lattice.blocks) != 2 {
		t.Fatalf("expected local node to catch up to 2 blocks, got %d", len(local.lattice.blocks))
	}
}

func TestHandleReconcileApplyRefusesDivergentPeer(t *testing.T) {
	remote := NewServer()
	remoteWallet := mustGenerateKeypair(t)
	remoteGenesis := torrent.NewBlock("open", remoteWallet.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	mustSignBlock(t, remoteGenesis, remoteWallet.PrivateKey)
	if err := remote.lattice.ProcessBlock(remoteGenesis); err != nil {
		t.Fatalf("remote genesis failed: %v", err)
	}
	remoteHTTP := httptest.NewServer(remote.HTTPHandler())
	defer remoteHTTP.Close()

	local := NewServer()
	localWallet := mustGenerateKeypair(t)
	localGenesis := torrent.NewBlock("open", localWallet.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	mustSignBlock(t, localGenesis, localWallet.PrivateKey)
	if err := local.lattice.ProcessBlock(localGenesis); err != nil {
		t.Fatalf("local genesis failed: %v", err)
	}

	applyRec := postJSON(t, local.HTTPHandler(), "/reconcile/apply", map[string]interface{}{"peer": remoteHTTP.URL, "force": true})
	if applyRec.Code != http.StatusConflict {
		t.Fatalf("expected reconcile apply conflict status %d, got %d with %s", http.StatusConflict, applyRec.Code, applyRec.Body.String())
	}
	body := decodeJSONBody(t, applyRec)
	if body["success"] != false {
		t.Fatalf("expected success=false for refused divergent reconciliation, got %#v", body)
	}
	apply, ok := body["apply"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected apply payload, got %#v", body)
	}
	if apply["executionMode"] != "refused" {
		t.Fatalf("expected refused execution mode, got %#v", apply)
	}
}

func TestAutonomousSyncLoop(t *testing.T) {
	origin := NewServer()
	wallet := mustGenerateKeypair(t)
	genesis := torrent.NewBlock("open", wallet.PublicKey, nil, 1000, 0, 0, "SYSTEM_GENESIS", nil, nil)
	mustSignBlock(t, genesis, wallet.PrivateKey)
	if err := origin.lattice.ProcessBlock(genesis); err != nil {
		t.Fatalf("origin genesis failed: %v", err)
	}
	originHTTP := httptest.NewServer(origin.HTTPHandler())
	defer originHTTP.Close()

	joiner := NewServer()
	joiner.lattice.AddPeer(originHTTP.URL)

	// Start loop with very short interval for testing
	joiner.StartBackgroundSync(100 * time.Millisecond)
	defer joiner.StopBackgroundSync()

	// Wait for sync
	time.Sleep(500 * time.Millisecond)

	if len(joiner.lattice.blocks) != 1 {
		t.Fatalf("expected autonomous sync to catch up 1 block, got %d", len(joiner.lattice.blocks))
	}
}
