package main
import ("encoding/json"; "net/http")
func handleBrowseChannels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var peersCount int64 = 0; var dhtNodes int64 = 0
	if torrentClient != nil {
		torrents := torrentClient.Torrents()
		for _, t := range torrents { peersCount += int64(len(t.PeerConns())) }
		if dhtNode != nil { dhtNodes = int64(dhtNode.Stats().GoodNodes) }
	}
	json.NewEncoder(w).Encode([]map[string]interface{}{{ "id": "bobtorrent-public-dht", "name": "BobTorrent Public DHT Swarm", "description": "The active public DHT swarm powered by anacrolix/torrent mapping AES blobs.", "peers": peersCount, "dhtNodes": dhtNodes }})
}
