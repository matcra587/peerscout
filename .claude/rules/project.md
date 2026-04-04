---
description: >
  Project architecture: Polkachu API client, CLI framework (clib/clog),
  configuration and Cobra patterns. Loaded for all Go files.
paths:
  - "**/*.go"
---

# Project Architecture

Structure of the peer discovery CLI: Polkachu API client, CLI
framework (clib/clog) and configuration.

## Principles

- Clear, concise, easy to follow.
- No files, packages or features before they're needed (YAGNI).
- No abstractions for one-time operations.

---

## Polkachu API Client (internal/polkachu/)

- HTTP client wrapping Polkachu API v2
- Base URL: `https://polkachu.com/api/v2`
- Methods: `ListChains()`, `FetchLivePeers()`, `GetChainDetail()`, `CheckLivePeersActive()`
- Pass `context.Context` as first argument to every method
- HTTP timeout: 15s
- 404 returns `*NotFoundError`; 500 returns generic error
- Response body capped at 1MB via `io.LimitReader`
- Types in `types.go`: only map fields the project uses, but use named structs

---

## CLI Framework (gechr/clib)

Cobra extensions, themed help, shell completion and terminal detection.
Import sub-packages directly; alias the Cobra integration as `cobracli`.

### Imports

```go
cobracli "github.com/gechr/clib/cli/cobra"   // Cobra flag extensions, help, completion
         "github.com/gechr/clib/help"         // Help rendering
         "github.com/gechr/clib/terminal"     // TTY detection
         "github.com/gechr/clib/theme"        // Help theme
```

### Flag Extension Pattern

Define Cobra flags first, then extend with `cobracli.Extend`:

```go
cmd.Flags().String("network", "", "Blockchain network to query")

cobracli.Extend(cmd.Flags().Lookup("network"), cobracli.FlagExtra{
    Placeholder: "NAME",
    Terse:       "network name",
})
```

### FlagExtra Fields

| Field | Type | Purpose |
|-------|------|---------|
| Group | string | Category in help (Global, Filters, Output) |
| Placeholder | string | Value placeholder (NAME, PATH) |
| Terse | string | One-line description |
| Enum | []string | Allowed values |
| EnumTerse | []string | Short descriptions for enum values (parallel to Enum) |
| EnumDefault | string | Default for display |
| Hint | string | Input hint (e.g. `"file"`) |

### Help Rendering

```go
th := theme.New(
    theme.WithEnumStyle(theme.EnumStyleHighlightBoth),
    theme.WithHelpRepeatEllipsisEnabled(true),
)
renderer := help.NewRenderer(th)
root.SetHelpFunc(cobracli.HelpFunc(renderer, cobracli.SectionsWithOptions(cobracli.WithSubcommandOptional())))
```

### Command Groups

```go
root.AddGroup(
    &cobra.Group{ID: "peers", Title: "Peer Discovery"},
    &cobra.Group{ID: "config", Title: "Configuration"},
)
```

---

## CLI Logging (gechr/clog)

`gechr/clog` is the only logger. `depguard` bans `log` and `log/slog`.

### Configuration

```go
clog.SetEnvPrefix("PEERSCOUT")
clog.SetVerbose(cfg.Debug)
clog.SetColorMode(clog.ColorNever)
```

### Error Handling Pattern

Route Cobra errors through clog:

```go
SilenceErrors: true,
SilenceUsage:  true,

// In main():
if err := root.Execute(); err != nil {
    clog.Error().Err(err).Msg("fatal")
    os.Exit(exitCode(err))
}
```

---

## Agent Mode

Every command must produce a JSON envelope when running in agent
mode. No exceptions.

### Detection and context

Agent mode is detected once in `PersistentPreRunE` and stored in
the command context. Retrieve it with `AgentFromContext(cmd)`.
Never re-detect via env vars in individual commands.

### Format selection

Use `output.DetectFormat` to determine the output format. Priority:
agent mode > explicit `--format` flag > plain.

```go
det := AgentFromContext(cmd)
format, _ := cmd.Flags().GetString("format")
isTTY := terminal.Is(os.Stdout)

switch output.DetectFormat(output.FormatOpts{AgentMode: det.Active, Format: format}) {
case output.FormatAgentJSON:
    return output.RenderAgentJSON(w, "command", data, nil)
case output.FormatJSON:
    return output.RenderJSON(w, data, isTTY)
default:
    // plain/csv output
}
```

### Envelope format

```json
{"success":true,"command":"<name>","data":{...}}
```

### Adding new commands

1. Get `det := AgentFromContext(cmd)` in the `RunE` function.
2. Use `output.DetectFormat` to select the output path.
3. Build a structured `data` value (named struct, map or slice).
4. Add the command to `TestAgentMode_JSONEnvelope` in
   `agent_envelope_test.go`.

### Testing

`TestAgentMode_JSONEnvelope` exercises every non-API command with
`--agent` and verifies the output is a valid JSON envelope. New
commands must be added to its test table.

---

## Configuration (internal/config/)

- koanf-based loading from TOML file, env vars and CLI flags
- Precedence: compiled defaults < TOML file < env vars (`PEERSCOUT_*`) < CLI flags
- Auto-discovers config at XDG config path via `internal/dirs/`
- `NO_COLOR` env var respected (standard convention)
- Only changed CLI flags override (checked via `f.Changed`)
