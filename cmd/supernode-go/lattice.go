package main
import ("crypto/sha256"; "encoding/hex"; "encoding/json"; "fmt"; "net/http"; "time")
type BlockInfo struct { Hash string `json:"hash"`; Height int64 `json:"height"`; Producer string `json:"producer"`; Timestamp time.Time `json:"timestamp"` }
func handleLattice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed); return }
	var blocks []BlockInfo; now := time.Now(); producer := "Unknown"
	if nodeWallet != nil { producer = nodeWallet.PublicKey }
	for i := 0; i < 5; i++ {
		h := sha256.New(); h.Write([]byte(fmt.Sprintf("block_data_%d_%s", i, now.String()))); hashStr := hex.EncodeToString(h.Sum(nil))
		blocks = append(blocks, BlockInfo{ Hash: hashStr, Height: int64(1024000 - i), Producer: producer, Timestamp: now.Add(-time.Duration(i*30) * time.Second) })
	}
	w.Header().Set("Content-Type", "application/json"); json.NewEncoder(w).Encode(blocks)
}
