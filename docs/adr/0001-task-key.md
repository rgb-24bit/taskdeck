# ADR 0001: Task Key — Secondary Unique Identifier

## Status

Accepted (2026-05-12)

## Context

TaskDeck tasks are identified solely by auto-increment integer `id`. Agents need idempotent task management: an agent with a session ID should be able to create or update a task without tracking the auto-incremented numeric ID. This requires a second, string-based unique identifier.

## Decision

Add an optional `key` column to the `tasks` table with the following semantics:

1. **Uniqueness**: `key` is enforced unique via SQLite UNIQUE index. SQLite allows multiple NULLs, so tasks without a key are unaffected.
2. **Upsert**: creating a task with an existing `key` updates the existing task's `title` and `context` (if provided), but does not change its `status`.
3. **Upsert guard**: if the existing task is in `done` status, the upsert is rejected — the user must run `td cleanup` first.
4. **Soft-delete release**: when a task is soft-deleted (`deleted_at` set), its `key` is set to NULL so the key can be reused.
5. **Identifier resolution**: all endpoints that accept a task identifier (`/api/tasks/{id-or-key}`) resolve the identifier by first attempting `ParseInt` → ID lookup, then falling back to key lookup via UNIQUE index.
6. **CLI flag**: `-k, --key` on `td add`. All other commands (`td show`, `td edit`, `td done`, etc.) accept either a numeric ID or a string key as the positional argument.
7. **Display**: key shown in CLI `list` output and in Web UI drawer; not shown in Web panel list.

## Alternatives considered

### Separate lookup endpoint (`/api/tasks/key/...`)

Rejected: adds routing complexity for no benefit. Auto-detection is backward-compatible and handles the edge case of all-digit keys gracefully (they're treated as ID, which is the overwhelmingly common case).

### Always required key

Rejected: forces friction on human users who just want to quickly add a task.

### Allow duplicate keys (no uniqueness)

Rejected: would make key-based lookup ambiguous, defeating the purpose of a "unique identifier."

## Consequences

- All store methods that accept `id int64` now have `ByKey` variants (`GetByKey`, `UpdateByKey`, etc.), or the client/server layer handles the resolution before calling the same store method.
- Soft-delete now always NULLs the key column.
- CLI help text must document that all `<id>` positional arguments can also be a `<key>` string.
