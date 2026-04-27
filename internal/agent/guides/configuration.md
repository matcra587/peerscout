# Configuration

## Config file location

peerscout stores configuration in a TOML file at the XDG config path:
`$XDG_CONFIG_HOME/peerscout/config.toml` (typically
`~/.config/peerscout/config.toml`).

Override with `--config PATH`.

## Commands

```bash
peerscout config init          # Interactive first-run wizard
peerscout config list          # Show current settings
peerscout config get <key>     # Get a single value
peerscout config set <key> <value>  # Set a value
peerscout config unset <key>   # Remove a value
```

## Available keys

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `count` | int | 5 | Default number of peers for `find` |

## Precedence

Settings resolve in this order (highest wins):

1.  CLI flags (`-n 10`)
2.  Environment variables (`PEERSCOUT_COUNT=10`)
3.  TOML config file
4.  Compiled defaults

## Rules

*   `config init` is interactive and requires a TTY
*   Environment variables use the `PEERSCOUT_` prefix with uppercase key names
*   Only changed CLI flags override; unchanged flags fall through to lower layers
