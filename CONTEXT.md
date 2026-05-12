# TaskDeck Domain Glossary

## Task

A unit of work managed by TaskDeck. Each task has a numeric auto-increment `id` and an optional unique string `key`.

- **State**: `active` (work queue, ordered) | `waiting` (condition-gated pool) | `done` (completed)
- **Identity**: tasks are identified by either `id` (int64) or `key` (string) interchangeably in all operations

## Key

A user-assigned unique string identifier for a task. Optional on creation, unique across all non-deleted tasks.

- When creating a task with a `key` that already exists on an active/waiting task, the existing task is updated (upsert semantics) — title and context are overwritten, status is preserved
- When the keyed task is in `done` status, upsert is rejected
- When a task is soft-deleted, its `key` is released (set to NULL) so it can be reused
- Primary use case: agent automation marking tasks with session IDs for idempotent task management
