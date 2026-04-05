package bridges

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultFilecoinRecordsPath = "data/filecoin/deals.json"
)

// FilecoinDealRecord persists one archival publication attempt or verification
// result. This gives operators a durable, inspectable trail of what the bridge
// tried to do, whether it used a real Lotus RPC path or a fallback simulation,
// and what the latest observed deal state was.
type FilecoinDealRecord struct {
	DealID          string `json:"dealId"`
	CID             string `json:"cid"`
	Size            int64  `json:"size"`
	DurationDays    int    `json:"durationDays"`
	Mode            string `json:"mode"`
	State           string `json:"state"`
	PublishedAt     int64  `json:"publishedAt"`
	VerifiedAt      int64  `json:"verifiedAt,omitempty"`
	Provider        string `json:"provider,omitempty"`
	Wallet          string `json:"wallet,omitempty"`
	RPCURL          string `json:"rpcUrl,omitempty"`
	LastError       string `json:"lastError,omitempty"`
	VerificationRaw string `json:"verificationRaw,omitempty"`
}

// FilecoinBridgeStatus is a compact operator-facing snapshot of bridge state.
// It is surfaced via HTTP so the Go supernode can report whether Lotus is
// configured, how many deal records are persisted, and whether the bridge is
// currently operating in real RPC mode or safe fallback mode.
type FilecoinBridgeStatus struct {
	Enabled      bool   `json:"enabled"`
	Configured   bool   `json:"configured"`
	Mode         string `json:"mode"`
	RPCURL       string `json:"rpcUrl,omitempty"`
	Wallet       string `json:"wallet,omitempty"`
	Miner        string `json:"miner,omitempty"`
	DealCount    int    `json:"dealCount"`
	LatestDealID string `json:"latestDealId,omitempty"`
	LatestState  string `json:"latestState,omitempty"`
	RecordsPath  string `json:"recordsPath,omitempty"`
	LastError    string `json:"lastError,omitempty"`
}

// lotusRPCRequest/Response are intentionally generic because the bridge only
// needs a narrow JSON-RPC slice today and Lotus response details can vary by
// node version and method.
type lotusRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

type lotusRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type lotusRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *lotusRPCError  `json:"error,omitempty"`
}

// FilecoinBridge provides a real Lotus JSON-RPC integration path when the
// operator configures one, while preserving a safe fallback mode so the rest of
// the supernode can keep functioning in environments without Lotus.
type FilecoinBridge struct {
	nodeAddr    string
	rpcURL      string
	authToken   string
	wallet      string
	miner       string
	recordsPath string
	httpClient  *http.Client

	mu        sync.Mutex
	records   []FilecoinDealRecord
	lastError string
}

func NewFilecoinBridge(addr string) *FilecoinBridge {
	bridge := &FilecoinBridge{
		nodeAddr:    addr,
		rpcURL:      strings.TrimSpace(os.Getenv("BOBTORRENT_FILECOIN_RPC_URL")),
		authToken:   strings.TrimSpace(os.Getenv("BOBTORRENT_FILECOIN_AUTH_TOKEN")),
		wallet:      strings.TrimSpace(os.Getenv("BOBTORRENT_FILECOIN_WALLET")),
		miner:       strings.TrimSpace(os.Getenv("BOBTORRENT_FILECOIN_MINER")),
		recordsPath: strings.TrimSpace(os.Getenv("BOBTORRENT_FILECOIN_RECORDS")),
		httpClient:  &http.Client{Timeout: 20 * time.Second},
	}
	if bridge.recordsPath == "" {
		bridge.recordsPath = defaultFilecoinRecordsPath
	}
	if err := bridge.loadRecords(); err != nil {
		bridge.lastError = err.Error()
		log.Printf("[Filecoin] failed to load deal records: %v", err)
	}
	return bridge
}

func (b *FilecoinBridge) configured() bool {
	return strings.TrimSpace(b.rpcURL) != "" && strings.TrimSpace(b.wallet) != "" && strings.TrimSpace(b.miner) != ""
}

func (b *FilecoinBridge) mode() string {
	if b.configured() {
		return "lotus-rpc"
	}
	return "fallback-simulated"
}

func (b *FilecoinBridge) Status() FilecoinBridgeStatus {
	b.mu.Lock()
	defer b.mu.Unlock()
	status := FilecoinBridgeStatus{
		Enabled:     true,
		Configured:  b.configured(),
		Mode:        b.mode(),
		RPCURL:      b.rpcURL,
		Wallet:      b.wallet,
		Miner:       b.miner,
		DealCount:   len(b.records),
		RecordsPath: b.recordsPath,
		LastError:   b.lastError,
	}
	if len(b.records) > 0 {
		latest := b.records[0]
		status.LatestDealID = latest.DealID
		status.LatestState = latest.State
	}
	return status
}

func (b *FilecoinBridge) ListDeals() []FilecoinDealRecord {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]FilecoinDealRecord, len(b.records))
	copy(out, b.records)
	return out
}

// PublishDeal attempts a real Lotus deal publication when RPC configuration is
// present. If Lotus is not configured, it falls back to a local simulated deal
// record so the rest of the supernode can keep functioning without silently
// pretending a real network submission happened.
func (b *FilecoinBridge) PublishDeal(cid string, size int64, durationDays int) (string, error) {
	if strings.TrimSpace(cid) == "" {
		return "", fmt.Errorf("filecoin cid required")
	}

	if !b.configured() {
		dealID := fmt.Sprintf("sim-f0%d", time.Now().UnixNano())
		record := FilecoinDealRecord{
			DealID:       dealID,
			CID:          cid,
			Size:         size,
			DurationDays: durationDays,
			Mode:         "fallback-simulated",
			State:        "simulated",
			PublishedAt:  time.Now().UnixMilli(),
			Provider:     b.nodeAddr,
			Wallet:       b.wallet,
			RPCURL:       b.rpcURL,
			LastError:    "lotus rpc not configured; recorded simulated archival intent",
		}
		b.recordDeal(record)
		log.Printf("[Filecoin] Lotus RPC not configured; recorded simulated archival intent for %s as %s", cid, dealID)
		return dealID, nil
	}

	proposal := map[string]any{
		"Data": map[string]any{
			"TransferType": "manual",
			"Root":         map[string]any{"/": cid},
		},
		"Wallet":            b.wallet,
		"Miner":             b.miner,
		"EpochPrice":        "0",
		"MinBlocksDuration": durationDays * 2880,
	}

	var result any
	if err := b.callLotus("Filecoin.ClientStartDeal", []any{proposal}, &result); err != nil {
		b.setLastError(err)
		return "", err
	}

	dealID := normalizeLotusScalar(result)
	if dealID == "" {
		return "", fmt.Errorf("lotus returned empty deal id for cid %s", cid)
	}
	record := FilecoinDealRecord{
		DealID:       dealID,
		CID:          cid,
		Size:         size,
		DurationDays: durationDays,
		Mode:         "lotus-rpc",
		State:        "submitted",
		PublishedAt:  time.Now().UnixMilli(),
		Provider:     b.miner,
		Wallet:       b.wallet,
		RPCURL:       b.rpcURL,
	}
	b.recordDeal(record)
	log.Printf("[Filecoin] Lotus deal published for %s as %s via %s", cid, dealID, b.rpcURL)
	return dealID, nil
}

// VerifyStorage checks current deal state. In Lotus mode it queries the RPC;
// in fallback mode it preserves backward-compatible successful verification for
// simulated intents while updating the local record trail.
func (b *FilecoinBridge) VerifyStorage(dealID string) (bool, error) {
	if strings.TrimSpace(dealID) == "" {
		return false, fmt.Errorf("filecoin deal id required")
	}

	if !b.configured() {
		b.updateDealVerification(dealID, true, "simulated-active", "")
		return true, nil
	}

	var result map[string]any
	if err := b.callLotus("Filecoin.StateMarketStorageDeal", []any{dealID, nil}, &result); err != nil {
		b.setLastError(err)
		b.updateDealVerification(dealID, false, "verification_error", err.Error())
		return false, err
	}

	active, stateLabel := interpretLotusDealState(result)
	raw, _ := json.Marshal(result)
	b.updateDealVerification(dealID, active, stateLabel, string(raw))
	return active, nil
}

func interpretLotusDealState(result map[string]any) (bool, string) {
	state, _ := result["State"].(map[string]any)
	sectorStart := asInt64(state["SectorStartEpoch"])
	slashEpoch := asInt64(state["SlashEpoch"])
	if sectorStart > 0 && slashEpoch < 0 {
		return true, "active"
	}
	if slashEpoch >= 0 {
		return false, "slashed"
	}
	if sectorStart > 0 {
		return true, "sealed"
	}
	statusText := normalizeLotusScalar(state["State"])
	if statusText != "" {
		return false, strings.ToLower(statusText)
	}
	return false, "pending"
}

func normalizeLotusScalar(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case json.Number:
		return v.String()
	case map[string]any:
		if slash, ok := v["/"]; ok {
			return normalizeLotusScalar(slash)
		}
	}
	return fmt.Sprintf("%v", value)
}

func asInt64(value any) int64 {
	switch v := value.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case json.Number:
		i, _ := v.Int64()
		return i
	case string:
		i, _ := strconv.ParseInt(v, 10, 64)
		return i
	}
	return 0
}

func (b *FilecoinBridge) callLotus(method string, params []any, out any) error {
	requestBody, err := json.Marshal(lotusRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return fmt.Errorf("failed to encode lotus request %s: %w", method, err)
	}

	req, err := http.NewRequest(http.MethodPost, b.rpcURL, bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create lotus request %s: %w", method, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(b.authToken) != "" {
		req.Header.Set("Authorization", "Bearer "+b.authToken)
	}

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("lotus request %s failed: %w", method, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("lotus request %s returned http %d", method, resp.StatusCode)
	}

	var rpcResp lotusRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return fmt.Errorf("failed to decode lotus response %s: %w", method, err)
	}
	if rpcResp.Error != nil {
		return fmt.Errorf("lotus rpc %s error %d: %s", method, rpcResp.Error.Code, rpcResp.Error.Message)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(rpcResp.Result, out); err != nil {
		return fmt.Errorf("failed to decode lotus result %s: %w", method, err)
	}
	return nil
}

func (b *FilecoinBridge) loadRecords() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if strings.TrimSpace(b.recordsPath) == "" {
		return nil
	}
	raw, err := os.ReadFile(b.recordsPath)
	if err != nil {
		if os.IsNotExist(err) {
			b.records = nil
			return nil
		}
		return err
	}
	var records []FilecoinDealRecord
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &records); err != nil {
			return err
		}
	}
	b.records = records
	return nil
}

func (b *FilecoinBridge) persistRecordsLocked() error {
	if strings.TrimSpace(b.recordsPath) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(b.recordsPath), 0755); err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(b.records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(b.recordsPath, encoded, 0644)
}

func (b *FilecoinBridge) recordDeal(record FilecoinDealRecord) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.records = append([]FilecoinDealRecord{record}, b.records...)
	if err := b.persistRecordsLocked(); err != nil {
		b.lastError = err.Error()
		log.Printf("[Filecoin] failed to persist deal records: %v", err)
	}
}

func (b *FilecoinBridge) updateDealVerification(dealID string, active bool, state string, verificationRaw string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for index := range b.records {
		if b.records[index].DealID != dealID {
			continue
		}
		b.records[index].VerifiedAt = time.Now().UnixMilli()
		b.records[index].State = state
		b.records[index].VerificationRaw = verificationRaw
		if !active && verificationRaw != "" {
			b.records[index].LastError = verificationRaw
		}
		break
	}
	if err := b.persistRecordsLocked(); err != nil {
		b.lastError = err.Error()
		log.Printf("[Filecoin] failed to persist verified deal records: %v", err)
	}
}

func (b *FilecoinBridge) setLastError(err error) {
	if err == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lastError = err.Error()
}
