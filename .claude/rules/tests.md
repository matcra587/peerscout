---
description: >
  Testing conventions: TDD workflow, testify usage, httptest patterns,
  naming, and what to cover. Loaded for test files.
paths:
  - "**/*_test.go"
  - "internal/**/*.go"
---

# Testing

## Principles

Tests are executable specifications. They constrain behaviour and
make failures self-explanatory. Never write tests to hit coverage
targets.

### Test behaviour, not wiring

Test observable inputs and outputs. Do not test that Cobra calls
RunE, that `cobracli.Extend` sets metadata, or that `fmt.Println`
prints. Those are framework guarantees.

### One test, one concern

Each test function or subtest verifies one behaviour.

### Do not duplicate coverage

Error paths handled centrally (e.g. 404 handling in the Polkachu
client) are tested once. Per-method tests only need the happy path
plus method-specific error codes.

### Skip pointless tests

Do not test:
- Cobra command registration or flag binding
- Simple getters/setters with no logic
- Framework behaviour (arg validation, help rendering)

## Testify Usage

- Use `stretchr/testify` - `require` for guards, `assert` for
  verifications
- `require` when the next line would panic on nil (e.g. after
  `NoError`, `NotNil`, `Len`)
- `assert` for the actual verification (e.g. `Equal`, `Contains`)
- Argument order is always `(expected, actual)`
- Use `ErrorIs`/`require.ErrorIs` for errors, not `Equal`

## Test Structure

- `t.Parallel()` on all tests and subtests that do not share
  mutable state
- Table-driven tests for multiple cases with named subtests
- Each test sets up its own `httptest.NewServer` and calls
  `t.Cleanup(server.Close)`
- Test naming: `TestFunctionName` or `TestFunctionName_Variant`
- Use `server` and `client` as variable names for httptest servers
  and API clients

## API Client Test Pattern

```go
func TestFetchLivePeers(t *testing.T) {
    t.Parallel()
    mux := http.NewServeMux()
    server := httptest.NewServer(mux)
    t.Cleanup(server.Close)

    mux.HandleFunc("GET /chains/cosmos/live_peers", func(w http.ResponseWriter, _ *http.Request) {
        _, _ = w.Write([]byte(`{"network":"cosmos","polkachu_peer":"abc@1.2.3.4:26656","live_peers":["def@5.6.7.8:26656"]}`))
    })

    client := polkachu.NewClientWithHTTP(server.Client(), server.URL+"/api/v2")
    peers, err := client.FetchLivePeers(context.Background(), "cosmos")

    require.NoError(t, err)
    assert.Equal(t, "cosmos", peers.Network)
    require.Len(t, peers.LivePeers, 1)
}
```
