package dhtproxy

type Peer struct {
	IP        string  `json:"ip"`
	Port      int     `json:"port"`
	Country   string  `json:"country"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Torrent struct {
	InfoHash    string `json:"info_hash"`
	DisplayName string `json:"display_name"`
}
