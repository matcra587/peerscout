# Country Filtering for Peer Discovery

Filter discovered peers by ISO 3166-1 alpha-2 country code.
Target release: v0.4.1.

## Problem

Peerscout returns peers from any country.
Operators often want peers in specific regions for latency, compliance or network topology reasons.
Geolocation enrichment (v0.4.0) provides the country data — this feature uses it.

## CLI Surface

### `--country` flag (find command only)

```
peerscout find cosmos --country GB,US
peerscout find cosmos --country GB --country US
```

- Type: `StringSlice` (Cobra) — supports both comma-separated and repeated forms.
- Case-insensitive input, normalised to uppercase internally.
- Validated as 2-character alpha strings. Invalid codes produce an error.
- Mutually exclusive with `--seed-node`, `--state-sync` and `--addrbook`.
- When set with `geo_provider=none`: hard error — "cannot filter by country without a geo provider".

### `--max-retries` flag (find command only)

```
peerscout find cosmos --country GB --max-retries 10
```

- Type: `int`, default `5`.
- Caps how many fetch-enrich-filter rounds the discovery pipeline runs.
- Applies regardless of whether `--country` is set — gives users control over how hard the tool works to fill the requested count.

### Config file

```toml
country = ["GB", "US"]
max_retries = 5
```

### Environment variables

```
PEERSCOUT_COUNTRY=GB,US
PEERSCOUT_MAX_RETRIES=5
```

`PEERSCOUT_COUNTRY` is a comma-separated string, split on load.

### Precedence

Compiled defaults (empty / 5) < TOML file < env vars < CLI flags.

## Architecture

### Package changes

```
internal/discovery/    New — pipeline: fetch → dedup → enrich → filter → retry
internal/filter/       New — ByCountry (country filtering logic)
internal/polkachu/     Remove AccumulatePeers; keep as thin API client
internal/config/       Add Country []string, MaxRetries int
main.go / config.go    Wire --country, --max-retries, call discovery.Run
```

### `internal/discovery/`

Owns the peer discovery pipeline.
Replaces the accumulation logic currently in `polkachu.AccumulatePeers` and the enrichment block in `runFind`.

```go
// Run executes the discovery pipeline: fetch → dedup → enrich → filter → retry.
func Run(ctx context.Context, opts Opts) (*Result, error)

type Opts struct {
    Client     *polkachu.Client
    Network    string
    Count      int
    MaxRetries int
    Locator    locator
    Countries  []string
    OnProgress ProgressFunc
}

type Result struct {
    Peers      []EnrichedPeer
    Duplicates int
    Retries    int
}
```

Each round:

1. Call `client.FetchLivePeers` (two parallel fetches, as today).
2. Deduplicate against a running `seen` set.
3. Enrich new peers with geo data via `Locator`.
4. Filter by country (if countries are set).
5. Append matches to the result set.
6. If `len(result) >= count` or round yielded zero new peers, stop.
7. If round count hits `maxRetries`, stop.

When the loop ends with fewer matches than requested, log a warning:
`found 3 of 10 requested peers after 5 retries`.

### `internal/filter/`

```go
package filter

// ByCountry returns peers whose CountryCode is in the allowed set.
// Peers with an empty CountryCode are excluded.
func ByCountry(peers []EnrichedPeer, codes []string) []EnrichedPeer
```

The `codes` parameter is pre-normalised to uppercase by the caller.
The function builds a set from `codes` for O(1) lookup.

This package will later hold `ByLatency` and other filter functions.

### `internal/polkachu/` changes

Remove `AccumulatePeers`, `AccumulateResult` and `ProgressFunc`.
The client keeps: `ListChains`, `FetchLivePeers`, `ChainDetail`, `CheckLivePeersActive`.

### `internal/config/` changes

```go
type Config struct {
    // ...existing fields...
    Country    []string `koanf:"country"`
    MaxRetries int      `koanf:"max_retries"`
}
```

Defaults: `Country` empty (no filter), `MaxRetries` 5.

Env var `PEERSCOUT_COUNTRY` needs a comma-split transform in the env provider callback since koanf does not split comma-separated strings into slices automatically.

### `config.go` changes

- Add `"country"` and `"max_retries"` to `configKeys`.
- Add descriptions to `configDescriptions`.
- Add validation in `parseConfigValue`:
  - `country`: list of 2-character alpha strings, normalised to uppercase.
  - `max_retries`: positive integer.
- Update `configToMap` to include both keys.
- Add `"country"` to the masked/display logic (no masking needed).

### `main.go` changes

- Add `--country` and `--max-retries` flags to `findCmd`.
- Mark `--country` mutually exclusive with `--seed-node`, `--state-sync`, `--addrbook`.
- Validate `--country` + `geo_provider=none` → hard error.
- Replace the `AccumulatePeers` call and geo enrichment block with `discovery.Run`.
- Move the `locator` interface to `internal/discovery/` (it is consumed there now).
- Move `enrichedPeer` to `internal/discovery/` as `EnrichedPeer` (the discovery package builds and returns them).
- Keep `peerResult` in `main.go` — it is a presentation type for output rendering.

### Shimmer output

Single summary log line on completion:

```
INF discovered peers network=dydx duration=3.2s found=5/5 retries=2/5 duplicates=4
```

- `found=x/X` shows matched vs requested.
- `retries=n/N` shows rounds used vs max. Omitted when no country filter is active.
- Live shimmer updates with both fields during the loop.

When no country filter is active:

```
INF discovered peers network=dydx duration=1s found=11 duplicates=1
```

## Error Handling

| Condition | Behaviour |
|-----------|-----------|
| `--country` + `geo_provider=none` | Hard error before any API call |
| `--country` + `--seed-node`/`--state-sync`/`--addrbook` | Cobra mutual exclusivity error |
| Geo lookup fails for all peers in a round | Peers excluded from that round (no country code = excluded) |
| Zero matches after all retries | Return empty result with warning |
| Fewer matches than count after all retries | Return partial result with warning |
| Invalid country code format | Hard error at flag/config validation |

## Testing

### `internal/filter/`

- `TestByCountry`: table-driven — matches, no matches, empty country code excluded, case already normalised.

### `internal/discovery/`

- `TestRun_NoFilter`: basic accumulation without country filter.
- `TestRun_CountryFilter`: mock locator returns mixed countries, verify only matching peers returned.
- `TestRun_MaxRetriesExhausted`: verify loop stops at max retries and returns partial result.
- `TestRun_DeduplicatesAcrossRounds`: same peer from multiple rounds counted once.
- `TestRun_ZeroNewPeersStops`: loop exits when a round yields nothing new.

### `main.go`

- `--country` + `geo_provider=none` returns error.
- Mutual exclusivity enforced (Cobra handles this, but a smoke test confirms).
- Agent envelope test updated for new fields.

### `internal/polkachu/`

- Remove `AccumulatePeers` tests.
- `FetchLivePeers` tests remain unchanged.

## Migration

`AccumulatePeers` is internal and has no external consumers.
The move to `internal/discovery/` is a straight refactor with no compatibility concerns.
Existing tests for `AccumulatePeers` behaviour migrate to `internal/discovery/` tests.
