---
name: peerscout
description: Use when a user needs Cosmos SDK blockchain peers, seed nodes, state-sync RPC endpoints or addrbook URLs. Triggers on mentions of persistent_peers, seeds, state-sync, addrbook, config.toml peer configuration, Polkachu, or Cosmos chain node setup.
---

# Using peerscout

peerscout is a CLI that fetches live peers from the Polkachu API for
Cosmos SDK chains. It has an agent mode that returns structured JSON.

## Agent mode

peerscout auto-detects AI agents via environment variables
(`CLAUDE_CODE`, `CURSOR_AGENT`, `CODEX`, etc.) and switches to agent
mode automatically. No extra flags needed.

Agent mode changes:

- Output becomes JSON wrapped in an envelope
- Spinners and logs are suppressed
- Colours are disabled

## Envelope format

Every command returns a JSON envelope:

```json
{"success":true,"command":"find","data":{...},"hints":["..."]}
```

| Field | Type | Description |
|-------|------|-------------|
| `success` | bool | Whether the command succeeded |
| `command` | string | Command name (`find`, `list`) |
| `data` | object/array | Result payload |
| `hints` | string[] | Optional guidance (omitted when empty) |
| `error` | object | Present on failure: `{code, message}`, optional `suggestion` |

Read `data` directly from the JSON output.

## Self-discovery

Before guessing at flags, use these commands:

```
peerscout agent schema            # JSON schema of all commands and flags
peerscout agent schema --compact  # Same, without descriptions (smaller)
peerscout agent guide <name>      # Embedded workflow guide (markdown)
```

Available guides: `peer-discovery`, `networks`, `configuration`.

## Key commands

### Fetch peers

```
peerscout find <network>
peerscout find cosmos
peerscout find celestia -n 10
```

Agent mode forces JSON envelope output. The `-f` flag is ignored.

Response shapes by variant:

| Variant | `data` fields |
|---------|---------------|
| `find <net>` | `network`, `peers` (array of `id@host:port`), `count` |
| `find <net> --seed-node` | `network`, `seed` |
| `find <net> --state-sync` | `network`, `state_sync` |
| `find <net> --addrbook` | `network`, `addrbook` |
| `list` | array of network name strings |

### Fetch seed node, state-sync or addrbook

These flags are mutually exclusive:

```
peerscout find cosmos --seed-node
peerscout find cosmos --state-sync
peerscout find cosmos --addrbook
```

If the service is inactive for a network, peerscout exits
successfully with no output (no JSON envelope). Check for empty
output before parsing.

### List supported networks

```
peerscout list
```

Returns `data` as a JSON array of network name strings.

## Common workflows

**Persistent peers for config.toml:**
Run `peerscout find <network> -n <count>`, read `data.peers` from the
envelope, then join with commas for the `persistent_peers` field.

**Check network support:**
Run `peerscout list`, check whether the network name appears in `data`.

**Full node bootstrap:**
1. `peerscout find <network> -n 10` for persistent peers
2. `peerscout find <network> --seed-node` for seeds
3. `peerscout find <network> --state-sync` for state-sync RPC

## Error handling

Failed commands return `success: false` with an error object:

```json
{"success":false,"command":"find","error":{"code":1,"message":"unknown network \"foo\" - run 'peerscout list' to see all supported networks"}}
```

Check `success` before reading `data`. On error, read
`error.message` for details and any suggested action.

## Common mistakes

- NEVER pass `--quiet`, `--no-color` or `--agent` - agent mode
  sets all three automatically via env detection.
- NEVER pass `-f` - agent mode forces the JSON envelope regardless.
- NEVER combine `--seed-node`, `--state-sync` and `--addrbook` -
  they are mutually exclusive and Cobra rejects the command.
- NEVER assume a specific peer count in one call - Polkachu returns
  up to 5 random peers per request. peerscout accumulates across
  calls, so `-n 10` works but takes 2-3 requests.

## Rules

- Network names are lowercase (e.g. `cosmos`, `celestia`, `dydx`).
- Not all networks support all services. Inactive services produce
  empty output rather than an error.
