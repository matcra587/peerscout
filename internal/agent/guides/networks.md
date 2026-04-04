# Networks

## Listing supported networks

```
peerscout list
peerscout list -f json
peerscout list -f csv
```

Returns all Cosmos SDK networks supported by the Polkachu API.

## Output formats

| Flag | Output |
|------|--------|
| (default) | Multi-column layout (TTY) or one per line (piped) |
| `-f json` | JSON array of network names |
| `-f csv` | Comma-separated, single line |

## Validating a network name

Use `list` to check whether a network is supported before calling
`find`. The network list comes from Polkachu and may change over time
as chains are added or removed.

## Rules

- No arguments or flags beyond `--format`
- The default column layout adapts to terminal width
- Piped output (non-TTY) prints one network per line, no colour
