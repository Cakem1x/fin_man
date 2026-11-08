# Enable SQLite foreign key enforcement

## Context
The schema uses `REFERENCES ... ON DELETE CASCADE` and `ON DELETE SET NULL` (`internal/db/migrations/`).
SQLite ignores these unless `PRAGMA foreign_keys = ON` is set per connection. `db.Open` does not set it.

## Scope
- Add `_foreign_keys=1` to the DSN in `db.Open` (`internal/db/db.go`). All commands open the database through `db.Open`.
- Check that existing code paths still work with enforcement on, e.g. `RenameCategory`, `DeleteEmptyCategory`, `SplitTransaction`, and the embedding cache (`transaction_embeddings`, migration 0006).
- `fin workspace upgrade` runs `PRAGMA foreign_key_check` and reports orphaned rows before migrating.

## Acceptance criteria
Write failing tests first:
- Deleting a tag removes its `transaction_tags` rows.
- Deleting a category sets `category_id` of its transactions to NULL.
- Inserting a `transaction_tags` row with an unknown transaction ID fails.
- Deleting a transaction removes its `transaction_embeddings` rows.

## Depends on
None.
