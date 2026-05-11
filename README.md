# TaskDeck

A personal context-task manager — your war room cockpit for wrangling async AI agents and personal tasks across work queues and waiting pools.

## Install

```bash
git clone git@github.com:rgb-24bit/taskdeck.git
cd taskdeck
make install    # → ~/.local/bin/td
```

Requires Go 1.20+.

## Quickstart

```bash
td serve                        # start daemon (localhost:10086)

td add "review PR #42"          # add to work queue
td add -w -t 30m "wait for CI"  # add to waiting pool (auto-activate in 30m)
td list                          # view work queue
td list -w                       # view waiting pool
td done 1                        # mark task done
td context 1 --append "notes"   # append to task context
open http://localhost:10086      # web dashboard
```

## Concepts

| Area | Description |
|------|-------------|
| **Work Queue** | Ordered list of tasks needing your direct involvement. Process top-to-bottom, reorder via drag/CLI. |
| **Waiting Pool** | Tasks waiting on conditions — AI agent working, CI running, external event. No fixed order. Activates into work queue when ready. |

A task moves between these two states: `active` → `waiting` → `active` → `done`.

## CLI Reference

```
td add [-w] [-t 30m] <title>       add task (queue or pool)
td list                              list active tasks
td list -w                           list waiting pool tasks
td list -d                           list done tasks
td show <id>                         task details
td edit <id> title <text>            edit title
td edit <id> timeout <dur>           set timeout condition
td done <id>                         mark done
td delete <id>                       hard delete
td move <id> --top|--bottom          reorder in queue
td move <id> --after <target_id>     place after another task
td move <id> --wait [-t 30m]        move to waiting pool
td activate <id>                     move from pool to queue
td context <id>                      view context
td context <id> --append <text>      append to context
td context <id> --set <text>         replace context
td cleanup [duration]                hard-delete done tasks (default: 30d)
```

## Web Dashboard

| Page | Description |
|------|-------------|
| `/` | Dual panel: work queue (ordered, drag-to-reorder) + waiting pool (grouped by timeout/manual, drag-to-change-group). Click task → slide-out drawer with detail/edit/actions. |
| `/history` | Completed tasks by date range. |

## API

```
GET    /api/tasks?status=active|waiting      list tasks
GET    /api/tasks?status=done&from=...&to=...  list completed (with date range)
POST   /api/tasks                             create task (JSON or form)
GET    /api/tasks/:id                         get task (JSON, or HTML if HX-Request)
PATCH  /api/tasks/:id                         update task fields
DELETE /api/tasks/:id                         hard delete
POST   /api/tasks/:id/done                     mark done
POST   /api/tasks/:id/activate                 waiting → active
POST   /api/tasks/:id/wait                     active → waiting (with condition)
POST   /api/tasks/:id/reorder                  {after_id, position: "top"|"bottom"}
POST   /api/tasks/cleanup                      {older_than: "7d"}
```

## Config

`~/.taskdeck/config.yaml` (auto-created with defaults):

```yaml
port: 10086
db_path: ~/.taskdeck/taskdeck.db
log_path: ~/.taskdeck/taskdeck.log
pid_path: ~/.taskdeck/taskdeck.pid
cleanup:
  retain_done_days: 30
default_timeout: 30m
```

## Files

```
~/.taskdeck/
├── config.yaml
├── taskdeck.db      # SQLite (WAL mode)
├── taskdeck.pid
└── taskdeck.log     # rotated at 10MB
```

## Tech stack

Go · HTMX · Pico CSS · SortableJS · SQLite (pure Go, no CGO)
