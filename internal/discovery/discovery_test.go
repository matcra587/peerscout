package discovery

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/matcra587/peerscout/internal/geo"
	"github.com/matcra587/peerscout/internal/polkachu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFetcher returns canned responses for each call.
type mockFetcher struct {
	responses []polkachu.ChainLivePeers
	errors    []error
	callCount atomic.Int32
}

func (m *mockFetcher) FetchLivePeers(_ context.Context, _ string) (polkachu.ChainLivePeers, error) {
	i := int(m.callCount.Add(1)) - 1
	if i >= len(m.responses) {
		return polkachu.ChainLivePeers{}, nil
	}
	var err error
	if i < len(m.errors) {
		err = m.errors[i]
	}
	return m.responses[i], err
}

// mockLocator returns a fixed map of IP → Location.
type mockLocator struct {
	locations map[string]geo.Location
}

func (m *mockLocator) Locate(_ context.Context, ips []string) map[string]geo.Location {
	if m.locations == nil {
		return nil
	}
	result := make(map[string]geo.Location)
	for _, ip := range ips {
		if loc, ok := m.locations[ip]; ok {
			result[ip] = loc
		}
	}
	return result
}

func TestFilterByCountry(t *testing.T) {
	t.Parallel()

	peers := []EnrichedPeer{
		{Address: "a@1.2.3.4:26656", CountryCode: "GB", Country: "United Kingdom"},
		{Address: "b@5.6.7.8:26656", CountryCode: "US", Country: "United States"},
		{Address: "c@9.10.11.12:26656", CountryCode: "DE", Country: "Germany"},
		{Address: "d@13.14.15.16:26656", CountryCode: "", Country: ""},
	}

	tests := []struct {
		name  string
		codes []string
		want  []EnrichedPeer
	}{
		{
			name:  "single match",
			codes: []string{"GB"},
			want:  []EnrichedPeer{peers[0]},
		},
		{
			name:  "multiple matches",
			codes: []string{"GB", "US"},
			want:  []EnrichedPeer{peers[0], peers[1]},
		},
		{
			name:  "no matches",
			codes: []string{"FR"},
			want:  nil,
		},
		{
			name:  "empty country code excluded",
			codes: []string{"GB", ""},
			want:  []EnrichedPeer{peers[0]},
		},
		{
			name:  "empty codes returns nil",
			codes: nil,
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := filterByCountry(peers, tt.codes)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRun_NoFilter(t *testing.T) {
	t.Parallel()

	fetcher := &mockFetcher{
		responses: []polkachu.ChainLivePeers{
			{
				Network:      "cosmos",
				PolkachuPeer: "polka@1.1.1.1:26656",
				LivePeers:    []string{"a@2.2.2.2:26656", "b@3.3.3.3:26656"},
			},
			{
				Network:   "cosmos",
				LivePeers: []string{"c@4.4.4.4:26656", "d@5.5.5.5:26656"},
			},
		},
	}

	locator := &mockLocator{
		locations: map[string]geo.Location{
			"1.1.1.1": {CountryCode: "US", Country: "United States"},
			"2.2.2.2": {CountryCode: "GB", Country: "United Kingdom"},
			"3.3.3.3": {CountryCode: "DE", Country: "Germany"},
			"4.4.4.4": {CountryCode: "FR", Country: "France"},
			"5.5.5.5": {CountryCode: "JP", Country: "Japan"},
		},
	}

	result, err := Run(context.Background(), Opts{
		Fetcher:   fetcher,
		Network:   "cosmos",
		Count:     5,
		MaxRounds: 5,
		Locator:   locator,
	})

	require.NoError(t, err)
	require.Len(t, result.Peers, 5)

	// Geo enrichment must have run: every peer should have a country code.
	countryCodes := make([]string, len(result.Peers))
	for i, p := range result.Peers {
		countryCodes[i] = p.CountryCode
	}
	assert.Contains(t, countryCodes, "GB")
}

func TestRun_CountryFilter(t *testing.T) {
	t.Parallel()

	fetcher := &mockFetcher{
		responses: []polkachu.ChainLivePeers{
			{
				Network:   "cosmos",
				LivePeers: []string{"a@1.1.1.1:26656", "b@2.2.2.2:26656", "c@3.3.3.3:26656"},
			},
			{
				Network:   "cosmos",
				LivePeers: []string{"d@4.4.4.4:26656", "e@5.5.5.5:26656"},
			},
		},
	}

	locator := &mockLocator{
		locations: map[string]geo.Location{
			"1.1.1.1": {CountryCode: "GB", Country: "United Kingdom"},
			"2.2.2.2": {CountryCode: "US", Country: "United States"},
			"3.3.3.3": {CountryCode: "DE", Country: "Germany"},
			"4.4.4.4": {CountryCode: "GB", Country: "United Kingdom"},
			"5.5.5.5": {CountryCode: "FR", Country: "France"},
		},
	}

	result, err := Run(context.Background(), Opts{
		Fetcher:   fetcher,
		Network:   "cosmos",
		Count:     2,
		MaxRounds: 5,
		Locator:   locator,
		Countries: []string{"GB"},
	})

	require.NoError(t, err)
	require.Len(t, result.Peers, 2)
	assert.Equal(t, "GB", result.Peers[0].CountryCode)
	assert.Equal(t, "GB", result.Peers[1].CountryCode)
}

func TestRun_MaxRetriesExhausted(t *testing.T) {
	t.Parallel()

	// Each call returns one peer with a non-matching country.
	// 2 fetches per round × 3 rounds = need 6 responses minimum.
	responses := make([]polkachu.ChainLivePeers, 10)
	locations := make(map[string]geo.Location)
	for i := range 10 {
		ip := fmt.Sprintf("10.0.0.%d", i+1)
		addr := fmt.Sprintf("node%d@%s:26656", i, ip)
		responses[i] = polkachu.ChainLivePeers{
			Network:   "cosmos",
			LivePeers: []string{addr},
		}
		locations[ip] = geo.Location{CountryCode: "DE", Country: "Germany"}
	}

	fetcher := &mockFetcher{responses: responses}
	locator := &mockLocator{locations: locations}

	result, err := Run(context.Background(), Opts{
		Fetcher:   fetcher,
		Network:   "cosmos",
		Count:     5,
		MaxRounds: 3,
		Locator:   locator,
		Countries: []string{"GB"},
	})

	require.NoError(t, err)
	assert.Empty(t, result.Peers)
	assert.Equal(t, 3, result.Rounds)
}

func TestRun_DeduplicatesAcrossRounds(t *testing.T) {
	t.Parallel()

	// All rounds return the same peer.
	fetcher := &mockFetcher{
		responses: []polkachu.ChainLivePeers{
			{Network: "cosmos", LivePeers: []string{"a@1.1.1.1:26656"}},
			{Network: "cosmos", LivePeers: []string{"a@1.1.1.1:26656"}},
			{Network: "cosmos", LivePeers: []string{"a@1.1.1.1:26656"}},
			{Network: "cosmos", LivePeers: []string{"a@1.1.1.1:26656"}},
		},
	}
	locator := &mockLocator{
		locations: map[string]geo.Location{
			"1.1.1.1": {CountryCode: "GB", Country: "United Kingdom"},
		},
	}

	result, err := Run(context.Background(), Opts{
		Fetcher:   fetcher,
		Network:   "cosmos",
		Count:     5,
		MaxRounds: 5,
		Locator:   locator,
	})

	require.NoError(t, err)
	require.Len(t, result.Peers, 1)
	assert.Positive(t, result.Duplicates)
}

func TestRun_APIExhaustedStops(t *testing.T) {
	t.Parallel()

	// First round returns peers, second round returns same (no new unique).
	fetcher := &mockFetcher{
		responses: []polkachu.ChainLivePeers{
			{Network: "cosmos", LivePeers: []string{"a@1.1.1.1:26656"}},
			{Network: "cosmos", LivePeers: []string{"a@1.1.1.1:26656"}},
		},
	}
	locator := &mockLocator{
		locations: map[string]geo.Location{
			"1.1.1.1": {CountryCode: "GB", Country: "United Kingdom"},
		},
	}

	result, err := Run(context.Background(), Opts{
		Fetcher:   fetcher,
		Network:   "cosmos",
		Count:     10,
		MaxRounds: 5,
		Locator:   locator,
	})

	require.NoError(t, err)
	assert.Len(t, result.Peers, 1)
	// Should stop early because API is exhausted (no new unique peers).
	assert.Equal(t, 2, result.Rounds)
}

func TestRun_FilterRejectsContinues(t *testing.T) {
	t.Parallel()

	// Round 1: two peers, neither matches GB. Round 2: one peer matches GB.
	fetcher := &mockFetcher{
		responses: []polkachu.ChainLivePeers{
			{Network: "cosmos", LivePeers: []string{"a@1.1.1.1:26656"}},
			{Network: "cosmos", LivePeers: []string{"b@2.2.2.2:26656"}},
			{Network: "cosmos", LivePeers: []string{"c@3.3.3.3:26656"}},
			{Network: "cosmos", LivePeers: []string{"d@4.4.4.4:26656"}},
		},
	}
	locator := &mockLocator{
		locations: map[string]geo.Location{
			"1.1.1.1": {CountryCode: "DE", Country: "Germany"},
			"2.2.2.2": {CountryCode: "FR", Country: "France"},
			"3.3.3.3": {CountryCode: "GB", Country: "United Kingdom"},
			"4.4.4.4": {CountryCode: "US", Country: "United States"},
		},
	}

	result, err := Run(context.Background(), Opts{
		Fetcher:   fetcher,
		Network:   "cosmos",
		Count:     1,
		MaxRounds: 5,
		Locator:   locator,
		Countries: []string{"GB"},
	})

	require.NoError(t, err)
	require.Len(t, result.Peers, 1)
	assert.Equal(t, "GB", result.Peers[0].CountryCode)
	// Should have continued past round 1 into round 2.
	assert.GreaterOrEqual(t, result.Rounds, 2)
}

func TestRun_NilLocator(t *testing.T) {
	t.Parallel()

	fetcher := &mockFetcher{
		responses: []polkachu.ChainLivePeers{
			{Network: "cosmos", LivePeers: []string{"a@1.1.1.1:26656", "b@2.2.2.2:26656"}},
			{Network: "cosmos", LivePeers: []string{}},
		},
	}

	result, err := Run(context.Background(), Opts{
		Fetcher:   fetcher,
		Network:   "cosmos",
		Count:     2,
		MaxRounds: 3,
	})

	require.NoError(t, err)
	require.Len(t, result.Peers, 2)
	assert.Empty(t, result.Peers[0].CountryCode)
	assert.Empty(t, result.Peers[1].CountryCode)
}

func TestRun_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Pre-cancelled.

	fetcher := &mockFetcher{
		responses: []polkachu.ChainLivePeers{
			{Network: "cosmos", LivePeers: []string{"a@1.1.1.1:26656"}},
			{Network: "cosmos", LivePeers: []string{"b@2.2.2.2:26656"}},
		},
	}

	result, err := Run(ctx, Opts{
		Fetcher:   fetcher,
		Network:   "cosmos",
		Count:     5,
		MaxRounds: 5,
	})

	require.NoError(t, err)
	assert.Empty(t, result.Peers)
}
