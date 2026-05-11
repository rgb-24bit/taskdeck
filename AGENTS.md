# AGENTS.md

## Project Purpose

TaskDeck is a personal context-task manager that helps a single user manage frequent context switches between AI agent async work and manual tasks. It provides a **work queue** (ordered, one-at-a-time) and a **waiting pool** (unordered, condition-gated) with CLI, Web dashboard, and REST API interfaces.

## Tech Stack

- **Go 1.20** — single binary (`td`), no external runtime dependencies
- **SQLite** via `modernc.org/sqlite` (pure Go, no CGO) — `internal/store/store.go`
- **HTMX + Pico CSS + SortableJS** — all loaded from CDN, no bundler/build step
- Templates embedded with `//go:embed` — `internal/server/web.go`
- Config in `~/.taskdeck/config.yaml` (YAML via `gopkg.in/yaml.v3`)

## Directory Map

```
cmd/td/main.go                  Entry point — CLI subcommand routing
internal/
  config/config.go              YAML config loading, defaults, typed Config struct
  model/task.go                 Task struct, status/source/condition constants, request DTOs
  store/store.go                SQLite schema migration, full CRUD, all queries
  server/server.go              HTTP router + all REST handlers (JSON + form)
  server/web.go                 Template functions, renderIndex/renderHistory, embed setup
  server/templates/*.html       index (dual panel), history, partials/drawer
  client/client.go              HTTP client used by CLI to talk to daemon
  daemon/daemon.go              Daemonize, PID file, log rotation (10MB), timeout checker
```

## Key Commands

```bash
make build         # compile bin/td
make install       # build + copy to ~/.local/bin/td
make run           # build + start daemon
```

Run `td` without args for the full CLI reference.

## How the Pieces Fit Together

1. `td serve` daemonizes the HTTP server (port 10086 by default) and starts a background goroutine that checks for expired timeout tasks every 30s.
2. CLI commands (`td add`, `td list`, etc.) detect if the daemon is running via PID file. If not, they auto-start it, then talk to it over HTTP via `internal/client`.
3. The Web dashboard at `/` renders active + waiting tasks server-side; HTMX handles interactions without a SPA. SortableJS provides drag-and-drop reorder + cross-panel movement.
4. Tasks move between states: `active` ←→ `waiting` → `done`. Soft delete is `status=done`; hard delete is `deleted_at IS NOT NULL` (via `td delete` or `td cleanup`).

## Working Guidelines

- Model changes go in `internal/model/task.go`. Store changes go in `internal/store/store.go`. Keep the scan helper (`taskRow`) in sync with schema when adding columns.
- API handlers in `internal/server/server.go` support both JSON (`Content-Type: application/json`) and form-encoded bodies. HTMX requests are detected via `HX-Request` header and return HTML partials instead of JSON.
- Templates live under `internal/server/templates/` and are embedded — run `go build` after editing them.
- The daemon binary is self-contained; `~/.taskdeck/` holds all runtime state. Test with `rm -f ~/.taskdeck/taskdeck.db` to reset.
- Time fields in SQLite are stored as RFC3339 TEXT. The `taskRow` struct scans them as `*string` and converts to `*time.Time` in `toTask()`.
- CLI UX uses ANSI escape codes for colored output. Keep them in `cmd/td/main.go` helper functions like `statusIcon()` and `statusTag()`.
