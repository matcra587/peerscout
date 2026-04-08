package polkachu_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matcra587/peerscout/internal/polkachu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListChains(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /chains", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`["cosmos","osmosis","celestia"]`))
	})

	client := polkachu.NewClientWithHTTP(server.Client(), server.URL)
	chains, err := client.ListChains(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"cosmos", "osmosis", "celestia"}, chains)
}

func TestListChains_ServerError(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /chains", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"success":false,"message":"internal error"}`))
	})

	client := polkachu.NewClientWithHTTP(server.Client(), server.URL)
	_, err := client.ListChains(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "internal error")
}

func TestFetchLivePeers(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /chains/cosmos/live_peers", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"network":"cosmos","polkachu_peer":"abc@1.2.3.4:26656","live_peers":["def@5.6.7.8:26656"]}`))
	})

	client := polkachu.NewClientWithHTTP(server.Client(), server.URL)
	peers, err := client.FetchLivePeers(context.Background(), "cosmos")

	require.NoError(t, err)
	assert.Equal(t, "cosmos", peers.Network)
	assert.Equal(t, "abc@1.2.3.4:26656", peers.PolkachuPeer)
	require.Len(t, peers.LivePeers, 1)
	assert.Equal(t, "def@5.6.7.8:26656", peers.LivePeers[0])
}

func TestFetchLivePeers_NotFound(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /chains/unknown/live_peers", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"success":false,"message":"chain not found"}`))
	})

	client := polkachu.NewClientWithHTTP(server.Client(), server.URL)
	_, err := client.FetchLivePeers(context.Background(), "unknown")

	require.Error(t, err)
	var notFound *polkachu.NotFoundError
	require.ErrorAs(t, err, &notFound)
	assert.Contains(t, notFound.Message, "chain not found")
}

func TestChainDetail(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /chains/cosmos", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"network":"cosmos",
			"name":"Cosmos Hub",
			"chain_id":"cosmoshub-4",
			"polkachu_services":{
				"seed":{"active":true,"seed":"abc@1.2.3.4:26656"},
				"addrbook":{"active":true,"download_url":"https://example.com/addrbook.json"},
				"state_sync":{"active":true,"node":"https://rpc.example.com:443"},
				"live_peers":{"active":true}
			}
		}`))
	})

	client := polkachu.NewClientWithHTTP(server.Client(), server.URL)
	detail, err := client.ChainDetail(context.Background(), "cosmos")

	require.NoError(t, err)
	assert.Equal(t, "cosmos", detail.Network)
	assert.Equal(t, "cosmoshub-4", detail.ChainID)
	assert.True(t, detail.Services.Seed.Active)
	assert.Equal(t, "abc@1.2.3.4:26656", detail.Services.Seed.Seed)
	assert.True(t, detail.Services.StateSync.Active)
	assert.Equal(t, "https://rpc.example.com:443", detail.Services.StateSync.Node)
	assert.True(t, detail.Services.Addrbook.Active)
}

func TestGet_RateLimitRetry(t *testing.T) {
	t.Parallel()

	attempts := 0
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /chains", func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts <= 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`["cosmos"]`))
	})

	client := polkachu.NewClientWithHTTP(server.Client(), server.URL)
	chains, err := client.ListChains(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"cosmos"}, chains)
	assert.Equal(t, 3, attempts)
}

func TestGet_RateLimitExhausted(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /chains", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	})

	client := polkachu.NewClientWithHTTP(server.Client(), server.URL)
	_, err := client.ListChains(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate limited")
}

func TestChainDetail_PathEscaping(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /chains/cosmos%2Fhub", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"network":"cosmos/hub","name":"Test","chain_id":"test-1","polkachu_services":{"seed":{},"addrbook":{},"state_sync":{},"live_peers":{}}}`))
	})

	client := polkachu.NewClientWithHTTP(server.Client(), server.URL)
	detail, err := client.ChainDetail(context.Background(), "cosmos/hub")

	require.NoError(t, err)
	assert.Equal(t, "cosmos/hub", detail.Network)
}
