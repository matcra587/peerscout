# peerscout

Blockchain peer discovery CLI. Fetches live peers from the Polkachu
API for Cosmos SDK chains.

## Quick Start

```bash
mise install                     # Install Go, task, actionlint, rumdl, zizmor
task deps                        # Download dependencies
task build                       # Build binary to ./dist/peerscout-<os>-<arch>
task test                        # Run unit tests
task lint                        # Run golangci-lint
```

## Module

`github.com/matcra587/peerscout`

## Go Version

Managed via `go.mod` toolchain directive: `go 1.26` with
`toolchain go1.26.1`. mise bootstraps Go; the toolchain directive
pins the exact version.

## Dev Tools

Tools split across two managers:

**mise** (`.mise.toml`) - platform tools:

| Tool | Purpose |
|------|---------|
| go | Go runtime (version pinned by go.mod toolchain) |
| task | Task runner |
| actionlint | GitHub Actions linter |
| rumdl | Markdown linter |
| zizmor | Workflow security scanner |

**go.mod** `tool` directives - Go project tools:

| Tool | Run with |
|------|----------|
| gofumpt | `go tool gofumpt` |
| govulncheck | `go tool govulncheck` |
| golangci-lint | `go tool golangci-lint` |

## Architecture

```text
cmd/peerscout/       Cobra command definitions (find, list, config, version)
internal/config/     Configuration management (koanf, TOML, env)
internal/dirs/       XDG config/cache directory resolution
internal/polkachu/   Polkachu API v2 client
internal/version/    Build metadata
```

## Commands

| Command | Purpose |
|---------|---------|
| `peerscout find --network cosmos` | Fetch live peers for a network |
| `peerscout list` | Show all supported networks |
| `peerscout config init` | Interactive first-run setup |
| `peerscout config list` | Show current settings |
| `peerscout config get/set/unset` | Manage individual settings |
| `peerscout version` | Print build info |

## Key Dependencies

- `spf13/cobra` - CLI framework
- `gechr/clog` - structured CLI logging (chain: `.Info().Str(k,v).Msg(m)`)
- `gechr/clib` - CLI infrastructure (help rendering, completions, theme)
- `knadh/koanf` - configuration (TOML file, env vars, CLI flags)
- `charm.land/huh` - interactive prompts (config init wizard)
- `charm.land/lipgloss` - terminal styling and tables

## Gotchas

- Polkachu API returns up to 5 random live peers plus one Polkachu
  internal peer per request.
- `depguard` bans `log` and `log/slog` imports. Use `gechr/clog`.
- Config precedence: compiled defaults < TOML file < env vars
  (`PEERSCOUT_*`) < CLI flags.

## Writing Quality

When writing or editing prose (docs, README, error messages, commit
messages), use the `/writing-clearly-and-concisely` skill if installed.
