package ipinfo_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matcra587/peerscout/internal/geo/ipinfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocate(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	// The SDK POSTs a JSON array of IPs to /batch and expects a JSON
	// object mapping each IP to its Core data.
	mux.HandleFunc("POST /batch", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"1.2.3.4": {"ip":"1.2.3.4","country":"US"},
			"5.6.7.8": {"ip":"5.6.7.8","country":"DE"}
		}`))
	})

	client := ipinfo.NewWithHTTP(server.Client(), server.URL+"/", "test-token")
	result := client.Locate(context.Background(), []string{"1.2.3.4", "5.6.7.8"})

	require.Len(t, result, 2)
	assert.Equal(t, "US", result["1.2.3.4"].CountryCode)
	assert.Equal(t, "United States", result["1.2.3.4"].Country)
	assert.Equal(t, "DE", result["5.6.7.8"].CountryCode)
	assert.Equal(t, "Germany", result["5.6.7.8"].Country)
}

func TestLocate_EmptyInput(t *testing.T) {
	t.Parallel()

	client := ipinfo.New("test-token")
	result := client.Locate(context.Background(), nil)

	assert.Nil(t, result)
}

func TestLocate_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := ipinfo.New("test-token")
	result := client.Locate(ctx, []string{"1.2.3.4"})

	assert.Empty(t, result)
}

func TestLocate_NilCore(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("POST /batch", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return a null value for one IP to exercise the nil-core branch.
		_, _ = w.Write([]byte(`{
			"1.2.3.4": {"ip":"1.2.3.4","country":"US"},
			"9.9.9.9": null
		}`))
	})

	client := ipinfo.NewWithHTTP(server.Client(), server.URL+"/", "test-token")
	result := client.Locate(context.Background(), []string{"1.2.3.4", "9.9.9.9"})

	require.Len(t, result, 1)
	assert.Equal(t, "US", result["1.2.3.4"].CountryCode)
}

func TestLocate_UnknownCountryCode(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("POST /batch", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"1.2.3.4": {"ip":"1.2.3.4","country":"ZZ"}}`))
	})

	client := ipinfo.NewWithHTTP(server.Client(), server.URL+"/", "test-token")
	result := client.Locate(context.Background(), []string{"1.2.3.4"})

	require.Len(t, result, 1)
	assert.Equal(t, "ZZ", result["1.2.3.4"].CountryCode)
	assert.Empty(t, result["1.2.3.4"].Country)
}

func TestLocate_BatchError(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("POST /batch", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	client := ipinfo.NewWithHTTP(server.Client(), server.URL+"/", "test-token")
	result := client.Locate(context.Background(), []string{"1.2.3.4"})

	assert.Empty(t, result)
}
