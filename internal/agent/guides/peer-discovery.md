# Peer Discovery

## Finding peers

```
peerscout find <network>
peerscout find cosmos
peerscout find dydx -n 10
```

Returns live peers from the Polkachu API for a Cosmos SDK network.
Default count is 5. Use `-n` to request more.

## Output formats

| Flag | Output |
|------|--------|
| (default) | One peer per line |
| `-f csv` | Comma-separated, single line (for config files) |
| `-f json` | JSON object with network, peers and count |

## Seed node, state-sync and addrbook

These flags are mutually exclusive with each other:

```
peerscout find cosmos --seed-node
peerscout find cosmos --state-sync
peerscout find cosmos --addrbook
```

Each returns a single value. Not all networks support all services.

## How accumulation works

Polkachu returns up to 5 random live peers per API call. peerscout
makes multiple calls until the desired count is reached, discarding
duplicates automatically. A request for 10 peers typically needs 2-3
API calls.

## Common patterns

Fetch peers and paste into a node's config.toml:

```
peerscout find cosmos -f csv
```

Check if a network is supported before fetching:

```
peerscout list
peerscout find dydx
```

## Flag reference

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-n, --count` | int | 5 | Number of peers to return |
| `-f, --format` | string | plain | Output format: plain, json, csv |
| `--seed-node` | bool | false | Return the seed node |
| `--state-sync` | bool | false | Return the state-sync RPC endpoint |
| `--addrbook` | bool | false | Return the addrbook download URL |

## Rules

- `--seed-node`, `--state-sync` and `--addrbook` are mutually exclusive
- Count must be a positive integer
- Unknown networks return an error listing valid alternatives
- The `-f` flag applies to all output modes including seed/state-sync/addrbook
