# Country Filtering Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Filter discovered peers by ISO 3166-1 alpha-2 country code, with retry logic to accumulate enough matching peers.

**Architecture:** Move peer accumulation from `polkachu.AccumulatePeers` into a new `internal/discovery/` package that owns the full fetch → dedup → enrich → filter → retry pipeline. The polkachu client becomes a thin HTTP layer. Country filtering is an unexported helper in `discovery`. Config and CLI gain `--country` and `--max-retries` flags.

**Tech Stack:** Go 1.26, Cobra (CLI), koanf (config), testify (testing), gechr/clog (logging), gechr/clib (CLI extensions)

**Spec:** `docs/superpowers/specs/2026-04-07-country-filtering-design.md`

---

### Task 1: Add `Country` and `MaxRetries` to config

**Files:**
- Modify: `internal/config/config.go`
- Modify: `config.go:287-328` (configKeys, configDescriptions, parseConfigValue, configToMap)

- [ ] **Step 1: Add fields to Config struct and Defaults**

In `internal/config/config.go`, add the new fields to `Config` and set defaults:

```go
type Config struct {
	Count int `koanf:"count"`

	// Global
	Debug     bool   `koanf:"debug"`
	NoColor   bool   `koanf:"no_color"`
	LogFormat string `koanf:"log_format"`

	// Geolocation
	GeoProvider string `koanf:"geo_provider"`
	GeoToken    string `koanf:"geo_token"`

	// Filtering
	Country    []string `koanf:"country"`
	MaxRetries int      `koanf:"max_retries"`
}

func Defaults() Config {
	return Config{
		Count:       5,
		LogFormat:   "auto",
		GeoProvider: "countryis",
		MaxRetries:  5,
	}
}
```

- [ ] **Step 2: Handle `PEERSCOUT_COUNTRY` comma-split in env provider**

In `internal/config/config.go`, replace the existing env provider block with a version that splits the `country` key on commas. The koanf env provider delivers all values as strings, so `PEERSCOUT_COUNTRY=GB,US` would become the literal string `"GB,US"` — which fails to unmarshal into `[]string`. Use koanf's `env.ProviderWithValue` to intercept and split:

```go
err := k.Load(env.ProviderWithValue("PEERSCOUT_", ".", func(key, value string) (string, any) {
	key = strings.ToLower(strings.TrimPrefix(key, "PEERSCOUT_"))
	if key == "country" && value != "" {
		return key, strings.Split(value, ",")
	}
	return key, value
}), nil)
if err != nil {
	return Config{}, fmt.Errorf("loading env vars: %w", err)
}
```

Note: this replaces the `env.Provider` call. Check that `env.ProviderWithValue` is available in the project's koanf version. If not, handle the split post-unmarshal as a fallback.

- [ ] **Step 3: Add config keys, descriptions, validation and map in `config.go`**

Update `configKeys`:

```go
var configKeys = []string{
	"count",
	"country",
	"geo_provider",
	"geo_token",
	"max_retries",
}
```

Update `configDescriptions`:

```go
var configDescriptions = map[string]string{
	"count":        "Number of peers to return",
	"country":      "Country codes to filter by (ISO 3166-1 alpha-2)",
	"geo_provider": "Geolocation provider (countryis, ipinfo, none)",
	"geo_token":    "API token for geolocation provider",
	"max_retries":  "Maximum discovery retry rounds",
}
```

Update `parseConfigValue` to add cases for the new keys:

```go
case "max_retries":
	n, err := strconv.Atoi(val)
	if err != nil || n < 1 {
		return nil, fmt.Errorf("max_retries must be a positive integer, got %s", val)
	}
	return n, nil
case "country":
	codes := strings.Split(val, ",")
	for i, c := range codes {
		c = strings.TrimSpace(strings.ToUpper(c))
		if len(c) != 2 || !isAlpha(c) {
			return nil, fmt.Errorf("invalid country code %q — must be 2-letter ISO 3166-1 alpha-2", c)
		}
		codes[i] = c
	}
	return codes, nil
```

Add the `isAlpha` helper at the bottom of `config.go`:

```go
func isAlpha(s string) bool {
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}
```

Update `configToMap`:

```go
func configToMap(cfg config.Config) map[string]string {
	return map[string]string{
		"count":        strconv.Itoa(cfg.Count),
		"country":      strings.Join(cfg.Country, ","),
		"geo_provider": cfg.GeoProvider,
		"geo_token":    cfg.GeoToken,
		"max_retries":  strconv.Itoa(cfg.MaxRetries),
	}
}
```

- [ ] **Step 4: Run tests to verify nothing is broken**

Run: `task test`
Expected: All existing tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go config.go
git commit -m "feat(config): add country and max_retries settings"
```

---

### Task 2: Test config loading for new fields

**Files:**
- Modify: `config_test.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write config env var round-trip test**

Create `internal/config/config_test.go`:

```go
package config_test

import (
	"testing"

	"github.com/matcra587/peerscout/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_CountryFromEnv(t *testing.T) {
	t.Parallel()
	t.Setenv("PEERSCOUT_COUNTRY", "GB,US")

	cfg, err := config.Load("", nil)
	require.NoError(t, err)

	assert.Equal(t, []string{"GB", "US"}, cfg.Country)
}

func TestLoad_MaxRetriesDefault(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load("", nil)
	require.NoError(t, err)

	assert.Equal(t, 5, cfg.MaxRetries)
}

func TestLoad_MaxRetriesFromEnv(t *testing.T) {
	t.Parallel()
	t.Setenv("PEERSCOUT_MAX_RETRIES", "10")

	cfg, err := config.Load("", nil)
	require.NoError(t, err)

	assert.Equal(t, 10, cfg.MaxRetries)
}

func TestLoad_CountryFromFile(t *testing.T) {
	t.Parallel()
	cfgFile := t.TempDir() + "/config.toml"
	if err := writeTestConfig(cfgFile, `country = ["DE", "FR"]`); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgFile, nil)
	require.NoError(t, err)

	assert.Equal(t, []string{"DE", "FR"}, cfg.Country)
}

func writeTestConfig(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}
```

Add the `"os"` import.

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test ./internal/config/ -v -run TestLoad_Country -run TestLoad_MaxRetries`
Expected: All new tests pass.

- [ ] **Step 3: Write parseConfigValue tests for country validation**

Add to `config_test.go` (the one in the root, alongside existing config tests):

```go
func TestParseConfigValue_Country(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		want    any
		wantErr bool
	}{
		{name: "valid single", input: "GB", want: []string{"GB"}},
		{name: "valid multiple", input: "gb,us", want: []string{"GB", "US"}},
		{name: "trims spaces", input: " DE , FR ", want: []string{"DE", "FR"}},
		{name: "invalid length", input: "GBR", wantErr: true},
		{name: "invalid chars", input: "G1", wantErr: true},
		{name: "empty string", input: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseConfigValue("country", tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseConfigValue_MaxRetries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		want    any
		wantErr bool
	}{
		{name: "valid", input: "10", want: 10},
		{name: "zero", input: "0", wantErr: true},
		{name: "negative", input: "-1", wantErr: true},
		{name: "not a number", input: "abc", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseConfigValue("max_retries", tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
```

- [ ] **Step 4: Run all tests**

Run: `task test`
Expected: All tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/config/config_test.go config_test.go
git commit -m "test(config): add country and max_retries config tests"
```

---

### Task 3: Create `internal/discovery/` package — types and interfaces

**Files:**
- Create: `internal/discovery/discovery.go`

- [ ] **Step 1: Create the package with types and interfaces**

```go
// Package discovery owns the peer discovery pipeline:
// fetch → deduplicate → enrich → filter → retry.
package discovery

import (
	"context"

	"github.com/matcra587/peerscout/internal/geo"
	"github.com/matcra587/peerscout/internal/polkachu"
)

// EnrichedPeer is a peer address with optional geolocation data.
type EnrichedPeer struct {
	Address     string `json:"address"`
	CountryCode string `json:"country_code,omitempty"`
	Country     string `json:"country,omitempty"`
}

// Fetcher abstracts peer fetching for testability.
// *polkachu.Client satisfies this interface.
type Fetcher interface {
	FetchLivePeers(ctx context.Context, network string) (polkachu.ChainLivePeers, error)
}

// Locator looks up geographic locations for IP addresses.
type Locator interface {
	Locate(ctx context.Context, ips []string) map[string]geo.Location
}

// ProgressFunc reports the current count of matched peers.
type ProgressFunc func(found, retries int)

// Opts configures a discovery run.
type Opts struct {
	Fetcher    Fetcher
	Network    string
	Count      int
	MaxRetries int
	Locator    Locator
	Countries  []string
	OnProgress ProgressFunc
}

// Result holds the outcome of a discovery run.
type Result struct {
	Peers      []EnrichedPeer
	Duplicates int
	Retries    int
}
```

- [ ] **Step 2: Verify the package compiles**

Run: `go build ./internal/discovery/`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add internal/discovery/discovery.go
git commit -m "feat(discovery): add types, interfaces and opts for discovery pipeline"
```

---

### Task 4: Implement `filterByCountry` with tests (TDD)

**Files:**
- Modify: `internal/discovery/discovery.go`
- Create: `internal/discovery/discovery_test.go`

- [ ] **Step 1: Write the failing test for filterByCountry**

Create `internal/discovery/discovery_test.go`:

```go
package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/discovery/ -v -run TestFilterByCountry`
Expected: FAIL — `filterByCountry` not defined.

- [ ] **Step 3: Implement filterByCountry**

Add to `internal/discovery/discovery.go`:

```go
// filterByCountry returns peers whose CountryCode is in the allowed set.
// Peers with an empty CountryCode are excluded.
// The codes parameter must be pre-normalised to uppercase.
func filterByCountry(peers []EnrichedPeer, codes []string) []EnrichedPeer {
	if len(codes) == 0 {
		return nil
	}

	allowed := make(map[string]struct{}, len(codes))
	for _, c := range codes {
		if c != "" {
			allowed[c] = struct{}{}
		}
	}

	var matched []EnrichedPeer
	for _, p := range peers {
		if p.CountryCode == "" {
			continue
		}
		if _, ok := allowed[p.CountryCode]; ok {
			matched = append(matched, p)
		}
	}
	return matched
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/discovery/ -v -run TestFilterByCountry`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/discovery/discovery.go internal/discovery/discovery_test.go
git commit -m "feat(discovery): add filterByCountry helper with tests"
```

---

### Task 5: Implement `discovery.Run` with tests (TDD)

**Files:**
- Modify: `internal/discovery/discovery.go`
- Modify: `internal/discovery/discovery_test.go`

- [ ] **Step 1: Write the failing test for Run — no filter (basic accumulation)**

Add to `internal/discovery/discovery_test.go`:

```go
import (
	"context"
	"net"
	"strings"
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
		Fetcher:    fetcher,
		Network:    "cosmos",
		Count:      5,
		MaxRetries: 5,
		Locator:    locator,
	})

	require.NoError(t, err)
	require.Len(t, result.Peers, 5)
	assert.Equal(t, "GB", result.Peers[1].CountryCode)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/discovery/ -v -run TestRun_NoFilter`
Expected: FAIL — `Run` not defined.

- [ ] **Step 3: Implement `Run`**

Add to `internal/discovery/discovery.go`:

```go
import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"

	"github.com/matcra587/peerscout/internal/geo"
	"github.com/matcra587/peerscout/internal/polkachu"
)

// Run executes the discovery pipeline: fetch → dedup → enrich → filter → retry.
func Run(ctx context.Context, opts Opts) (*Result, error) {
	seen := make(map[string]struct{})
	var matched []EnrichedPeer
	var duplicates int
	retries := 0

	for retries < opts.MaxRetries {
		retries++

		// Two parallel fetches per round.
		var (
			mu      sync.Mutex
			results []polkachu.ChainLivePeers
			errs    []error
			wg      sync.WaitGroup
		)

		for range 2 {
			wg.Go(func() {
				resp, err := opts.Fetcher.FetchLivePeers(ctx, opts.Network)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					errs = append(errs, err)
					return
				}
				results = append(results, resp)
			})
		}
		wg.Wait()

		// If all fetches failed, return what we have or the error.
		if len(results) == 0 {
			if len(matched) > 0 {
				break
			}
			return nil, errors.Join(errs...)
		}

		// Deduplicate raw peers.
		var newPeers []string
		for _, r := range results {
			for _, p := range append(r.LivePeers, r.PolkachuPeer) {
				if p == "" {
					continue
				}
				if _, ok := seen[p]; ok {
					duplicates++
					continue
				}
				seen[p] = struct{}{}
				newPeers = append(newPeers, p)
			}
		}

		// API exhausted — no new unique peers this round.
		if len(newPeers) == 0 {
			break
		}

		// Enrich with geo data.
		var enriched []EnrichedPeer
		if opts.Locator != nil {
			ips := geo.ExtractIPs(newPeers)
			locations := opts.Locator.Locate(ctx, ips)
			enriched = buildEnrichedPeers(newPeers, locations)
		} else {
			enriched = buildEnrichedPeers(newPeers, nil)
		}

		// Filter by country if requested.
		if len(opts.Countries) > 0 {
			enriched = filterByCountry(enriched, opts.Countries)
		}

		matched = append(matched, enriched...)

		if opts.OnProgress != nil {
			opts.OnProgress(len(matched), retries)
		}

		if len(matched) >= opts.Count {
			break
		}
	}

	// Truncate to requested count.
	if opts.Count > 0 && len(matched) > opts.Count {
		matched = matched[:opts.Count]
	}

	return &Result{
		Peers:      matched,
		Duplicates: duplicates,
		Retries:    retries,
	}, nil
}

// buildEnrichedPeers creates EnrichedPeer values from raw peer strings
// and an optional locations map.
func buildEnrichedPeers(peers []string, locations map[string]geo.Location) []EnrichedPeer {
	enriched := make([]EnrichedPeer, 0, len(peers))
	for _, p := range peers {
		ep := EnrichedPeer{Address: p}
		if _, hostPort, ok := strings.Cut(p, "@"); ok {
			if host, _, err := net.SplitHostPort(hostPort); err == nil {
				if loc, ok := locations[host]; ok {
					ep.CountryCode = loc.CountryCode
					ep.Country = loc.Country
				}
			}
		}
		enriched = append(enriched, ep)
	}
	return enriched
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/discovery/ -v -run TestRun_NoFilter`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/discovery/discovery.go internal/discovery/discovery_test.go
git commit -m "feat(discovery): implement Run pipeline with fetch, dedup, enrich and filter"
```

---

### Task 6: Test discovery edge cases

**Files:**
- Modify: `internal/discovery/discovery_test.go`

- [ ] **Step 1: Write country filter test**

```go
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
		Fetcher:    fetcher,
		Network:    "cosmos",
		Count:      2,
		MaxRetries: 5,
		Locator:    locator,
		Countries:  []string{"GB"},
	})

	require.NoError(t, err)
	require.Len(t, result.Peers, 2)
	assert.Equal(t, "GB", result.Peers[0].CountryCode)
	assert.Equal(t, "GB", result.Peers[1].CountryCode)
}
```

- [ ] **Step 2: Write max retries exhausted test**

```go
func TestRun_MaxRetriesExhausted(t *testing.T) {
	t.Parallel()

	// Each call returns one peer with a non-matching country.
	// Use enough responses to cover max retries (2 fetches per round).
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
		Fetcher:    fetcher,
		Network:    "cosmos",
		Count:      5,
		MaxRetries: 3,
		Locator:    locator,
		Countries:  []string{"GB"},
	})

	require.NoError(t, err)
	assert.Empty(t, result.Peers)
	assert.Equal(t, 3, result.Retries)
}
```

Add `"fmt"` to the imports.

- [ ] **Step 3: Write deduplication across rounds test**

```go
func TestRun_DeduplicatesAcrossRounds(t *testing.T) {
	t.Parallel()

	// Both rounds return the same peer.
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
		Fetcher:    fetcher,
		Network:    "cosmos",
		Count:      5,
		MaxRetries: 5,
		Locator:    locator,
	})

	require.NoError(t, err)
	require.Len(t, result.Peers, 1)
	assert.Positive(t, result.Duplicates)
}
```

- [ ] **Step 4: Write API exhausted stops test**

```go
func TestRun_APIExhaustedStops(t *testing.T) {
	t.Parallel()

	// First round returns peers, second round returns the same (no new).
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
		Fetcher:    fetcher,
		Network:    "cosmos",
		Count:      10,
		MaxRetries: 5,
		Locator:    locator,
	})

	require.NoError(t, err)
	assert.Len(t, result.Peers, 1)
	// Should have stopped after round 2 because no new unique peers.
	assert.Equal(t, 2, result.Retries)
}
```

- [ ] **Step 5: Write filter rejects but continues test**

```go
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
		Fetcher:    fetcher,
		Network:    "cosmos",
		Count:      1,
		MaxRetries: 5,
		Locator:    locator,
		Countries:  []string{"GB"},
	})

	require.NoError(t, err)
	require.Len(t, result.Peers, 1)
	assert.Equal(t, "GB", result.Peers[0].CountryCode)
	// Should have continued past round 1 (filter rejected all) into round 2.
	assert.GreaterOrEqual(t, result.Retries, 2)
}
```

- [ ] **Step 6: Run all discovery tests**

Run: `go test ./internal/discovery/ -v`
Expected: All tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/discovery/discovery_test.go
git commit -m "test(discovery): add edge case tests for filtering, retries and deduplication"
```

---

### Task 7: Remove `AccumulatePeers` from polkachu

**Files:**
- Modify: `internal/polkachu/client.go`
- Modify: `internal/polkachu/client_test.go`

- [ ] **Step 1: Remove AccumulatePeers, AccumulateResult and ProgressFunc from client.go**

Delete lines 165-244 from `internal/polkachu/client.go` — the `AccumulateResult` struct, `ProgressFunc` type, and `AccumulatePeers` method. Keep everything else (the client, `get`, `doGet`, `FetchLivePeers`, etc.).

- [ ] **Step 2: Remove AccumulatePeers tests from client_test.go**

Remove `TestAccumulatePeers_Deduplication` and `TestAccumulatePeers_PartialOnError` (and any other `AccumulatePeers` tests). Keep all `FetchLivePeers`, `ListChains` and `ChainDetail` tests.

- [ ] **Step 3: Run tests to verify nothing is broken**

Run: `task test`
Expected: All tests pass. No references to `AccumulatePeers` remain.

- [ ] **Step 4: Verify no remaining references**

Run: `grep -r "AccumulatePeers\|AccumulateResult\|ProgressFunc" --include="*.go" .`
Expected: Only hits in `internal/discovery/` (if any reference the old type name in comments). No hits in `internal/polkachu/` or `main.go`.

- [ ] **Step 5: Commit**

```bash
git add internal/polkachu/client.go internal/polkachu/client_test.go
git commit -m "refactor(polkachu): remove AccumulatePeers, now in internal/discovery"
```

---

### Task 8: Wire `--country` and `--max-retries` flags and integrate `discovery.Run`

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Add flags to `findCmd`**

In `findCmd()`, add the new flags after the existing ones:

```go
cmd.Flags().StringSliceP("country", "c", nil, "Filter peers by country code (ISO 3166-1 alpha-2)")
cmd.Flags().Int("max-retries", 5, "Maximum discovery retry rounds")
cobracli.Extend(cmd.Flags().Lookup("country"), cobracli.FlagExtra{
	Group:       "Filters",
	Placeholder: "CODE",
	Terse:       "country filter",
})
cobracli.Extend(cmd.Flags().Lookup("max-retries"), cobracli.FlagExtra{
	Group:       "Filters",
	Placeholder: "N",
	Terse:       "max retry rounds",
})
cmd.MarkFlagsMutuallyExclusive("country", "seed-node")
cmd.MarkFlagsMutuallyExclusive("country", "state-sync")
cmd.MarkFlagsMutuallyExclusive("country", "addrbook")
```

- [ ] **Step 2: Remove `locator` interface and `enrichedPeer` from main.go**

Delete the `locator` interface (lines 46-48) and `enrichedPeer` struct (lines 50-54) from `main.go`. Update `peerResult` to use `discovery.EnrichedPeer`:

```go
type peerResult struct {
	Network string                  `json:"network"`
	Peers   []discovery.EnrichedPeer `json:"peers"`
	Count   int                     `json:"count"`
}
```

Add the import:

```go
"github.com/matcra587/peerscout/internal/discovery"
```

- [ ] **Step 3: Replace accumulation and enrichment in `runFind` with `discovery.Run`**

Replace the entire section from the `count` extraction through to the `enriched` slice building (roughly lines 408-503) with the new discovery integration. The new code:

```go
count := cfg.Count
if cmd.Flags().Changed("count") {
	count, _ = cmd.Flags().GetInt("count")
}
if count < 1 {
	return fmt.Errorf("count must be a positive integer, got %d", count)
}

maxRetries := cfg.MaxRetries
if cmd.Flags().Changed("max-retries") {
	maxRetries, _ = cmd.Flags().GetInt("max-retries")
}

// Resolve country filter.
countries := cfg.Country
if cmd.Flags().Changed("country") {
	countries, _ = cmd.Flags().GetStringSlice("country")
}
// Normalise to uppercase.
for i, c := range countries {
	countries[i] = strings.ToUpper(c)
}

// Validate: country filter requires a geo provider.
if len(countries) > 0 && cfg.GeoProvider == "none" {
	return fmt.Errorf("cannot filter by country without a geo provider — set geo_provider or remove --country")
}

// Build geo locator.
var loc discovery.Locator
switch cfg.GeoProvider {
case "ipinfo":
	if cfg.GeoToken == "" {
		clog.Warn().Msg("ipinfo requires a token — set PEERSCOUT_GEO_TOKEN; skipping geo enrichment")
	} else {
		loc = geoipinfo.New(cfg.GeoToken)
	}
case "none":
	// Enrichment disabled.
default:
	loc = countryis.New()
}

// Run discovery pipeline.
var result *discovery.Result
if quiet {
	var err error
	result, err = discovery.Run(ctx, discovery.Opts{
		Fetcher:    client,
		Network:    network,
		Count:      count,
		MaxRetries: maxRetries,
		Locator:    loc,
		Countries:  countries,
	})
	if err != nil {
		return fmt.Errorf("discovering peers: %w", err)
	}
} else {
	filtering := len(countries) > 0
	var discoverErr error
	shimmer := clog.Shimmer("discovering peers").
		Str("network", network).
		Elapsed("duration")

	if filtering {
		_ = shimmer.Progress(ctx, func(ctx context.Context, u *clogfx.Update) error {
			result, discoverErr = discovery.Run(ctx, discovery.Opts{
				Fetcher:    client,
				Network:    network,
				Count:      count,
				MaxRetries: maxRetries,
				Locator:    loc,
				Countries:  countries,
				OnProgress: func(found, retries int) {
					u.Str("found", fmt.Sprintf("%d/%d", found, count)).
						Str("retries", fmt.Sprintf("%d/%d", retries, maxRetries)).
						Send()
				},
			})
			return discoverErr
		}).
			Int("duplicates", result.Duplicates).
			Str("found", fmt.Sprintf("%d/%d", len(result.Peers), count)).
			Str("retries", fmt.Sprintf("%d/%d", result.Retries, maxRetries)).
			Msg("discovered peers")
	} else {
		_ = shimmer.Progress(ctx, func(ctx context.Context, u *clogfx.Update) error {
			result, discoverErr = discovery.Run(ctx, discovery.Opts{
				Fetcher:    client,
				Network:    network,
				Count:      count,
				MaxRetries: maxRetries,
				Locator:    loc,
				OnProgress: func(found, _ int) {
					u.Int("found", found).Send()
				},
			})
			return discoverErr
		}).
			Int("duplicates", result.Duplicates).
			Msg("discovered peers")
	}

	if discoverErr != nil {
		return fmt.Errorf("discovering peers: %w", discoverErr)
	}
}

if len(countries) > 0 && len(result.Peers) < count {
	clog.Warn().
		Str("found", fmt.Sprintf("%d/%d", len(result.Peers), count)).
		Int("retries", result.Retries).
		Msg("fewer peers matched the country filter than requested")
}

enriched := result.Peers
```

- [ ] **Step 4: Update the output section to use `discovery.EnrichedPeer`**

The output section references `enrichedPeer` fields. Update to use `discovery.EnrichedPeer` — the field names (`Address`, `CountryCode`, `Country`) are identical, so only the type reference in `peerResult` needs changing (done in Step 2). The rest of the output code (`ep.Address`, `ep.CountryCode`, etc.) works unchanged since the field names match.

Also remove the `"net"` import from `main.go` if it is no longer used (the `net.SplitHostPort` call was in the enrichment block which is now in discovery).

- [ ] **Step 5: Verify it compiles**

Run: `go build ./...`
Expected: No errors.

- [ ] **Step 6: Commit**

```bash
git add main.go
git commit -m "feat(find): wire --country and --max-retries flags with discovery pipeline"
```

---

### Task 9: Add `"Filters"` command group and update help

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Ensure Filters group exists for flag categorisation**

The `cobracli.Extend` calls in Task 8 use `Group: "Filters"`. Verify that this group renders correctly in help output. The `cobracli` library groups flags automatically by the `Group` field — no explicit group registration is needed for flags (unlike command groups). Run:

Run: `go run . find --help`
Expected: `--country` and `--max-retries` appear under a "Filters" section in help output.

- [ ] **Step 2: Update find command examples**

Add country filter examples to `findCmd()`:

```go
Example: `  # Fetch peers for cosmos
  $ peerscout find cosmos

  # Return 10 peers
  $ peerscout find cosmos -n 10

  # Filter by country
  $ peerscout find cosmos --country GB,US

  # Get the seed node instead
  $ peerscout find cosmos --seed-node

  # Comma-separated for config files
  $ peerscout find cosmos -f csv

  # Output as JSON
  $ peerscout find cosmos -f json`,
```

- [ ] **Step 3: Commit**

```bash
git add main.go
git commit -m "docs(find): add country filter examples to help text"
```

---

### Task 10: Update agent envelope test

**Files:**
- Modify: `agent_envelope_test.go`

- [ ] **Step 1: Verify existing agent envelope tests pass**

Run: `go test -v -run TestAgentMode`
Expected: Existing tests pass. If the `peerResult` type change in Task 8 affected the agent output format, the tests may need updating.

- [ ] **Step 2: Update tests if needed**

The `peerResult` struct now uses `discovery.EnrichedPeer` instead of the old `enrichedPeer`. The JSON field names are identical (`address`, `country_code`, `country`), so the agent envelope format is unchanged. Verify the test still passes. If any test references the old type by name in assertions, update it.

- [ ] **Step 3: Run full test suite**

Run: `task test`
Expected: All tests pass.

- [ ] **Step 4: Commit (if changes were needed)**

```bash
git add agent_envelope_test.go
git commit -m "test: update agent envelope test for discovery.EnrichedPeer"
```

---

### Task 11: Lint, format and final verification

**Files:**
- All modified files

- [ ] **Step 1: Format**

Run: `go tool gofumpt -w .`

- [ ] **Step 2: Lint**

Run: `task lint`
Expected: No lint errors. Fix any that appear.

- [ ] **Step 3: Run full test suite**

Run: `task test`
Expected: All tests pass.

- [ ] **Step 4: Run with race detector**

Run: `go test -race ./...`
Expected: No new data races (the pre-existing `configPath` race may still appear — that is out of scope).

- [ ] **Step 5: Commit any formatting or lint fixes**

```bash
git add -A
git commit -m "style: format and lint fixes for country filtering"
```

---

### Task 12: Update README

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Read the current README**

Read `README.md` to understand the current structure and find where to add country filtering documentation.

- [ ] **Step 2: Add country filtering to the README**

Add a section or update the existing `find` command documentation to mention `--country` and `--max-retries`. Keep it brief — just the flag names, a one-line description and an example. Follow the existing README style.

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: add country filtering to README"
```
