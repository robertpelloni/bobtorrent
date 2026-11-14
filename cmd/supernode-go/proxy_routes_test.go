package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// The dashboard needs read access to lattice state (NFTs, swaps, anchors,
// market bids, frontiers) through the supernode origin. These routes must
// be reverse-proxied to lattice-go instead of falling through to 404.
func TestBobcoinProxiesLatticeReadRoutes(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"proxied":"` + r.URL.Path + `"}`))
	}))
	defer upstream.Close()

	old := latticeURL
	latticeURL = upstream.URL
	defer func() { latticeURL = old }()

	mux := http.NewServeMux()
	registerBobcoinProxies(mux)

	for _, path := range []string{"/nfts", "/swaps", "/anchors", "/market/bids", "/frontier/abc", "/balance/abc"} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: expected 200 via proxy, got %d", path, rec.Code)
		}
		if want := `{"proxied":"` + path + `"}`; rec.Body.String() != want {
			t.Fatalf("GET %s: expected %s, got %s", path, want, rec.Body.String())
		}
	}
}
