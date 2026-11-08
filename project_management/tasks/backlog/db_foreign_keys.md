# Enable SQLite foreign key enforcement

## Context
The schema uses `REFERENCES ... ON DELETE CASCADE` and `ON DELETE SET NULL` (`internal/db/migrations/`).
SQLite ignores these unless `PRAGMA foreign_keys = ON` is set per connection. `db.Open` does not set it.

## Scope
- Add `_foreign_keys=1` to the DSN in `db.Open` (`internal/db/db.go`).
- Workspace commands that open the database with `sql.Open` directly (`cmd/fin/cmd/workspace.go`) must use the same DSN.
- `fin workspace upgrade` runs `PRAGMA foreign_key_check` and reports orphaned rows before migrating.

## Acceptance criteria
Write failing tests first:
- Deleting a tag removes its `transaction_tags` rows.
- Deleting a category sets `category_id` of its transactions to NULL.
- Inserting a `transaction_tags` row with an unknown transaction ID fails.

## Depends on
None.
