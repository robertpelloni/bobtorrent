package test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/bobtorrent/bobtorrent/internal/api"
	"github.com/bobtorrent/bobtorrent/internal/wallet"
	"github.com/bobtorrent/bobtorrent/pkg/dht"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestServer(t *testing.T, dataDir string) (*api.Server, *httptest.Server) {
	err := os.MkdirAll(dataDir, 0755)
	require.NoError(t, err)

	w, err := wallet.NewWallet(dataDir)
	require.NoError(t, err)

	engine, err := dht.NewEngine(dataDir)
	require.NoError(t, err)

	server := &api.Server{
		Wallet:  w,
		Engine:  engine,
		DataDir: dataDir,
	}

	mux := server.SetupRoutes()
	ts := httptest.NewServer(mux)

	return server, ts
}

func TestE2E_IngestAndStream(t *testing.T) {
	dirA, err := os.MkdirTemp("", "bobtorrent-nodeA-*")
	require.NoError(t, err)
	defer os.RemoveAll(dirA)

	dirB, err := os.MkdirTemp("", "bobtorrent-nodeB-*")
	require.NoError(t, err)
	defer os.RemoveAll(dirB)

	nodeA, tsA := setupTestServer(t, dirA)
	defer tsA.Close()
	defer nodeA.Engine.Close()

	nodeB, tsB := setupTestServer(t, dirB)
	defer tsB.Close()
	defer nodeB.Engine.Close()

	dummyContent := []byte("This is a highly secret, incredibly important BobTorrent video file used for predictive streaming readahead verification.")

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "dummy.txt")
	require.NoError(t, err)
	part.Write(dummyContent)
	err = writer.Close()
	require.NoError(t, err)

	req, err := http.NewRequest("POST", tsA.URL+"/api/ingest", body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var ingestResult map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&ingestResult)
	require.NoError(t, err)
	assert.Equal(t, true, ingestResult["success"])

	fileID := ingestResult["fileId"].(string)
	assert.NotEmpty(t, fileID)

	manifestBytes, err := os.ReadFile(dirA + "/manifests/" + fileID + ".json")
	require.NoError(t, err)

	err = os.MkdirAll(dirB + "/manifests", 0755)
	require.NoError(t, err)
	err = os.WriteFile(dirB + "/manifests/" + fileID + ".json", manifestBytes, 0644)
	require.NoError(t, err)

	var manifest struct {
		Chunks []struct{ BlobID string `json:"blobId"` } `json:"chunks"`
	}
	json.Unmarshal(manifestBytes, &manifest)
	blobID := manifest.Chunks[0].BlobID

	blobBytes, err := os.ReadFile(dirA + "/" + blobID)
	require.NoError(t, err)
	err = os.WriteFile(dirB + "/" + blobID, blobBytes, 0644)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	streamReq, err := http.NewRequest("GET", tsB.URL+"/api/stream/"+fileID, nil)
	require.NoError(t, err)

	streamReq.Header.Set("Range", "bytes=0-14")

	streamResp, err := client.Do(streamReq)
	require.NoError(t, err)
	defer streamResp.Body.Close()

	assert.Equal(t, http.StatusPartialContent, streamResp.StatusCode)

	streamedBytes, err := io.ReadAll(streamResp.Body)
	require.NoError(t, err)

	expectedRange := dummyContent[0:15]
	assert.Equal(t, expectedRange, streamedBytes, "Streamed decrypted range should match original plaintext")
}
