# peerscout

![License](https://img.shields.io/github/license/matcra587/peerscout)
![Go](https://img.shields.io/github/go-mod/go-version/matcra587/peerscout?logo=go&logoColor=white)
![Release](https://img.shields.io/github/v/release/matcra587/peerscout)
![Go Report Card](https://goreportcard.com/badge/github.com/matcra587/peerscout)

Fetch live peers for Cosmos SDK chains from the Polkachu API.

## Background

A Go rewrite of [py_peerscout](https://github.com/matcra587/py_peerscout)
(now archived). The original was a Python script for retrieving and
filtering peers from Polkachu's live peer list. This version is a
proper CLI with parallel fetching, deduplication, multiple output
formats and configuration management.

## Installation

```bash
brew install matcra587/tap/peerscout
```

Also available via `go install` and pre-built binaries. All installs
support `peerscout update` for self-updating.
See [docs/installation.md](docs/installation.md) for details.

## Quick Start

```bash
peerscout find cosmos            # Fetch 5 peers (default)
peerscout find cosmos -n 15      # Fetch 15 peers
peerscout find cosmos -f csv     # Comma-separated for config files
peerscout find cosmos -f json    # JSON output
peerscout find cosmos --seed-node    # Polkachu seed node
peerscout find cosmos --state-sync   # State-sync RPC endpoint
peerscout find cosmos --addrbook     # Addrbook download URL
peerscout list                   # Show all supported networks
```

## Configuration

Settings are loaded in precedence order:

1.  Compiled defaults
2.  TOML config file (`~/.config/peerscout/config.toml`)
3.  Environment variables (`PEERSCOUT_*`)
4.  CLI flags

Manage settings directly:

```bash
peerscout config set count 10    # Set default peer count
peerscout config list            # Show current settings
peerscout config get count       # Show a single setting
peerscout config unset count     # Clear a setting
peerscout config path            # Show config file path
```

## Output Formats

| Format | Description |
|--------|-------------|
| `plain` (default) | One peer per line |
| `json` | Indented JSON |
| `csv` | Comma-separated single line |

Use `--agent` for AI agent consumption (forces JSON, quiet mode).
Use `-q`/`--quiet` to suppress all non-data output.

## Shell Completion

```bash
peerscout --install-completion   # Install for your current shell
```

Tab-completing `peerscout find <tab>` fetches the network list
from the API.

## API Coverage

| Endpoint | Description |
|----------|-------------|
| `GET /api/v2/chains` | List all supported networks |
| `GET /api/v2/chains/{network}/live_peers` | Fetch live peers |
| `GET /api/v2/chains/{network}` | Seed node, state-sync endpoint, addrbook URL |

Data sourced from the [Polkachu API v2](https://polkachu.com).

## Roadmap

Feature parity with [py_peerscout](https://github.com/matcra587/py_peerscout),
rebuilt incrementally:

*   [ ] Geolocation filtering - filter peers by country/region
*   [ ] Latency probing - ICMP with TCP fallback
*   [ ] Peer validation - verify peers are reachable
*   [ ] Daemon mode - systemd service with configurable interval
*   [ ] Metrics - emit alerts when peers become unviable

## Development

```bash
mise install          # Install Go, task, linters
task build            # Build to ./dist/
task test             # Run tests
task lint             # Run golangci-lint
task fmt              # Format with gofumpt
```
