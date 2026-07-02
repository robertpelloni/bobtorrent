package main
import ("net/http"; "os"; "strings")
type BlobInfo struct { ID string `json:"id"`; Size int64 `json:"size"`; Timestamp int64 `json:"timestamp"` }
func handleBlobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed); return }
	var blobs []BlobInfo; entries, err := os.ReadDir(torrentDataDir)
	if err != nil { writeJSON(w, http.StatusOK, map[string]interface{}{"blobs": []BlobInfo{}}); return }
	for _, entry := range entries {
		if entry.IsDir() { continue }
		name := entry.Name()
		if name == "identity.json" || name == "wallet.json" || strings.HasSuffix(name, ".db") || strings.HasPrefix(name, ".torrent") || strings.HasPrefix(name, "torrents.json") { continue }
		info, err := entry.Info(); if err != nil { continue }
		blobs = append(blobs, BlobInfo{ ID: name, Size: info.Size(), Timestamp: info.ModTime().UnixMilli() })
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"blobs": blobs})
}
