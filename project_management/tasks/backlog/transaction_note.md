# Store user edits in a note instead of the memo

## Context
See `architecture/data-model.md`: imported columns are never updated; user edits go to enrichment columns.
Today the TUI's memo editor overwrites `transactions.memo` through `db.EnrichTransaction`.
The memo is also part of the ID hash, so the original memo is needed to explain the ID.

## Scope
- Migration: add `note TEXT NOT NULL DEFAULT ''` to `transactions`.
- `model.Transaction` gets `Note string`. All transaction queries in `internal/db/db.go` read it.
- `EnrichTransaction` writes `note` and never writes `memo`.
- TUI (`internal/tui/review.go`): "Edit Memo" becomes "Edit Note". The imported memo is shown read-only.
- Overview table: show the note when set, otherwise the memo.
- `TransactionDiff` (`internal/tui/transaction_diff.go`) diffs `Note` instead of `Memo`.

## Out of scope
- Recovering memos that were already overwritten. See `repair_from_archives.md`.
- The trigger that blocks memo updates. See `immutable_import_trigger.md`.

## Acceptance criteria
Write failing tests first:
- `EnrichTransaction` with a note leaves `memo` unchanged and stores `note`.
- The diff reports a note change as `Note: (empty) → <text>`.

## Depends on
None.
