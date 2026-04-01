---
description: >
  Security invariants for the peerscout CLI.
paths:
  - "**/*.go"
---

# Security

## HTTP Client

- HTTP timeout set to 15s. Never remove or increase without reason.
- Response body capped at 1MB via `io.LimitReader`.
- URL path segments escaped with `url.PathEscape` to prevent injection.

## Configuration

- Config file paths come from user input (`--config` flag or XDG default).
  Annotate file operations with `//nolint:gosec` and a trust boundary comment.

## Context Lifecycle

- Pass `context.Context` through all API-calling functions.
- Respect cancellation in long-running operations.
