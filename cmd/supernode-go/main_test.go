package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHandleFHEOracleRequiresCiphertext(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/fhe-oracle", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	handleFHEOracle(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["success"] != false {
		t.Fatalf("expected success=false, got %#v", body["success"])
	}
	if body["error"] != "Encrypted payload missing" {
		t.Fatalf("expected missing payload error, got %#v", body["error"])
	}
}

func TestHandleFHEOracleReturnsComputedCiphertext(t *testing.T) {
	original := fheOracleRunner
	defer func() { fheOracleRunner = original }()

	fheOracleRunner = func(_ context.Context, cipherText string) (string, error) {
		if cipherText != "cipher_in" {
			t.Fatalf("expected cipher_in, got %q", cipherText)
		}
		return "cipher_out", nil
	}

	req := httptest.NewRequest(http.MethodPost, "/fhe-oracle", strings.NewReader(`{"cipherText":"cipher_in"}`))
	rec := httptest.NewRecorder()

	handleFHEOracle(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["success"] != true {
		t.Fatalf("expected success=true, got %#v", body["success"])
	}
	if body["resultCipher"] != "cipher_out" {
		t.Fatalf("expected cipher_out, got %#v", body["resultCipher"])
	}
}

func TestHandleFHEOracleSurfacesComputationFailure(t *testing.T) {
	original := fheOracleRunner
	defer func() { fheOracleRunner = original }()

	fheOracleRunner = func(_ context.Context, cipherText string) (string, error) {
		return "", context.DeadlineExceeded
	}

	req := httptest.NewRequest(http.MethodPost, "/fhe-oracle", strings.NewReader(`{"cipherText":"cipher_in"}`))
	rec := httptest.NewRecorder()

	handleFHEOracle(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["success"] != false {
		t.Fatalf("expected success=false, got %#v", body["success"])
	}
	if body["error"] != "Homomorphic computation failed." {
		t.Fatalf("expected homomorphic failure error, got %#v", body["error"])
	}
}

func TestHandleSignalingSocketMatchesPlayersAndRelaysSignals(t *testing.T) {
	original := signalingMatchmaker
	signalingMatchmaker = newMatchmaker()
	defer func() { signalingMatchmaker = original }()

	server := httptest.NewServer(http.HandlerFunc(handleSignalingSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	connA, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect first websocket: %v", err)
	}
	defer connA.Close()
	connB, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect second websocket: %v", err)
	}
	defer connB.Close()

	deadline := time.Now().Add(3 * time.Second)
	_ = connA.SetReadDeadline(deadline)
	_ = connB.SetReadDeadline(deadline)

	if err := connA.WriteJSON(map[string]interface{}{"type": "FIND_MATCH"}); err != nil {
		t.Fatalf("failed to queue first player: %v", err)
	}
	if err := connB.WriteJSON(map[string]interface{}{"type": "FIND_MATCH"}); err != nil {
		t.Fatalf("failed to queue second player: %v", err)
	}

	var msgA map[string]interface{}
	if err := connA.ReadJSON(&msgA); err != nil {
		t.Fatalf("failed to read first match message: %v", err)
	}
	var msgB map[string]interface{}
	if err := connB.ReadJSON(&msgB); err != nil {
		t.Fatalf("failed to read second match message: %v", err)
	}

	if msgA["type"] != "MATCH_FOUND" || msgA["initiator"] != true {
		t.Fatalf("unexpected first matchmaking payload: %#v", msgA)
	}
	if msgB["type"] != "MATCH_FOUND" || msgB["initiator"] != false {
		t.Fatalf("unexpected second matchmaking payload: %#v", msgB)
	}

	if err := connA.WriteJSON(map[string]interface{}{"type": "SIGNAL", "signal": map[string]interface{}{"sdp": "offer"}}); err != nil {
		t.Fatalf("failed to send signaling payload: %v", err)
	}

	var relayed map[string]interface{}
	if err := connB.ReadJSON(&relayed); err != nil {
		t.Fatalf("failed to read relayed signaling payload: %v", err)
	}
	if relayed["type"] != "SIGNAL" {
		t.Fatalf("expected SIGNAL relay, got %#v", relayed)
	}
	signal, ok := relayed["signal"].(map[string]interface{})
	if !ok || signal["sdp"] != "offer" {
		t.Fatalf("unexpected relayed signal payload: %#v", relayed["signal"])
	}
}

func TestHandleSignalingSocketNotifiesOpponentDisconnect(t *testing.T) {
	original := signalingMatchmaker
	signalingMatchmaker = newMatchmaker()
	defer func() { signalingMatchmaker = original }()

	server := httptest.NewServer(http.HandlerFunc(handleSignalingSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	connA, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect first websocket: %v", err)
	}
	connB, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect second websocket: %v", err)
	}
	defer connB.Close()

	deadline := time.Now().Add(3 * time.Second)
	_ = connA.SetReadDeadline(deadline)
	_ = connB.SetReadDeadline(deadline)

	_ = connA.WriteJSON(map[string]interface{}{"type": "FIND_MATCH"})
	_ = connB.WriteJSON(map[string]interface{}{"type": "FIND_MATCH"})

	var discard map[string]interface{}
	_ = connA.ReadJSON(&discard)
	_ = connB.ReadJSON(&discard)

	if err := connA.Close(); err != nil {
		t.Fatalf("failed to close first websocket: %v", err)
	}

	var msg map[string]interface{}
	if err := connB.ReadJSON(&msg); err != nil {
		t.Fatalf("failed to read disconnect notification: %v", err)
	}
	if msg["type"] != "OPPONENT_DISCONNECTED" {
		t.Fatalf("expected opponent disconnect message, got %#v", msg)
	}
}

func TestMatchmakerEvictsStaleWaitingPeerBeforeMatching(t *testing.T) {
	mm := newMatchmaker()
	stale := &matchPlayer{}
	stale.setWaitingSince(time.Now().Add(-2 * signalingWaitTimeout))
	mm.waiting = stale

	fresh := &matchPlayer{}
	opponent, matched := mm.queueOrMatch(fresh)
	if matched || opponent != nil {
		t.Fatalf("expected fresh player to be queued after stale eviction")
	}
	snapshot := mm.snapshot()
	if snapshot.WaitingPlayers != 1 {
		t.Fatalf("expected one waiting player, got %#v", snapshot)
	}
	if snapshot.StaleWaitingEvictions != 1 {
		t.Fatalf("expected one stale waiting eviction, got %#v", snapshot)
	}
}

func TestHandleServiceStatusIncludesSignalingSnapshot(t *testing.T) {
	original := signalingMatchmaker
	signalingMatchmaker = newMatchmaker()
	signalingMatchmaker.activeConnections = 2
	signalingMatchmaker.activePairs = 1
	signalingMatchmaker.totalMatches = 3
	defer func() { signalingMatchmaker = original }()

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()

	handleServiceStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode status response: %v", err)
	}
	signaling, ok := body["signaling"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected signaling snapshot in status response, got %#v", body["signaling"])
	}
	if signaling["activeConnections"] != float64(2) || signaling["activePairs"] != float64(1) || signaling["totalMatches"] != float64(3) {
		t.Fatalf("unexpected signaling snapshot: %#v", signaling)
	}
}
