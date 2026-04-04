# Installation

## Homebrew (recommended)

```bash
brew install matcra587/tap/peerscout
```

Homebrew sets up shell completions automatically.

## GitHub Releases

Download a pre-built binary from the
[releases page](https://github.com/matcra587/peerscout/releases)
and place it on your `PATH`.

## Go

Requires Go `1.26+`.

```bash
go install github.com/matcra587/peerscout@latest
```

## Updating

peerscout can update itself regardless of install method:

```bash
peerscout update
```

It detects how it was installed and delegates accordingly:

| Method | Detection | Action |
|--------|-----------|--------|
| Homebrew | Binary path under Homebrew prefix | `brew upgrade matcra587/tap/peerscout` |
| `go install` | Module path in embedded build info | `go install .../peerscout@latest` |
| Binary | Any other path | Downloads the latest release asset and replaces the binary in place |

## Shell Completion

Homebrew sets up completions automatically. If you installed via
GitHub Releases or `go install`, run:

```bash
peerscout --install-completion
```

Tab-completing `peerscout find <tab>` fetches the network list
from the API.
