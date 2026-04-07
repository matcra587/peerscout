package countryis_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matcra587/peerscout/internal/geo/countryis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocate(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /1.2.3.4", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"country":"US"}`))
	})
	mux.HandleFunc("GET /5.6.7.8", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"country":"DE"}`))
	})

	client := countryis.NewWithHTTP(server.Client(), server.URL)
	result := client.Locate(context.Background(), []string{"1.2.3.4", "5.6.7.8"})

	require.Len(t, result, 2)
	assert.Equal(t, "US", result["1.2.3.4"].CountryCode)
	assert.Equal(t, "United States", result["1.2.3.4"].Country)
	assert.Equal(t, "DE", result["5.6.7.8"].CountryCode)
	assert.Equal(t, "Germany", result["5.6.7.8"].Country)
}

func TestLocate_PartialFailure(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /1.2.3.4", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"country":"US"}`))
	})
	mux.HandleFunc("GET /9.9.9.9", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	client := countryis.NewWithHTTP(server.Client(), server.URL)
	result := client.Locate(context.Background(), []string{"1.2.3.4", "9.9.9.9"})

	require.Len(t, result, 1)
	assert.Equal(t, "US", result["1.2.3.4"].CountryCode)
}

func TestLocate_AllFail(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	client := countryis.NewWithHTTP(server.Client(), server.URL)
	result := client.Locate(context.Background(), []string{"1.2.3.4"})

	assert.Empty(t, result)
}

func TestLocate_EmptyInput(t *testing.T) {
	t.Parallel()

	client := countryis.New()
	result := client.Locate(context.Background(), nil)

	assert.Empty(t, result)
}

func TestLocate_ContextCancellation(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /1.2.3.4", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"country":"US"}`))
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := countryis.NewWithHTTP(server.Client(), server.URL)
	result := client.Locate(ctx, []string{"1.2.3.4"})

	assert.Empty(t, result)
}

func TestLocate_UnknownCountryCode(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /1.2.3.4", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"country":"ZZ"}`))
	})

	client := countryis.NewWithHTTP(server.Client(), server.URL)
	result := client.Locate(context.Background(), []string{"1.2.3.4"})

	require.Len(t, result, 1)
	assert.Equal(t, "ZZ", result["1.2.3.4"].CountryCode)
	assert.Empty(t, result["1.2.3.4"].Country)
}
