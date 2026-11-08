# Add a service layer between the TUI and the database

## Context
The TUI is meant to be a thin UI layer.
Today `OverviewModel` (`internal/tui/overview.go`) holds `*db.DB` and does persistence, in-memory cache patching, estimator training and category/tag list maintenance (`saveReview`).
This makes the TUI hard to test and blocks the TUI rework (`rework_transaction_edit.md`).

## Scope
- New package `internal/app` with a `Service` that owns the DB connection, the `categorize.Estimator`, and the cached transaction list.
- Methods (names indicative):
  - `Transactions(filter) []model.Transaction`
  - `Categories()`, `Tags()`
  - `Suggest(tx) *categorize.Estimation`: the only place the TUI gets suggestions from
  - `ApplyReview(txID, category, tags, note, reviewed) error`: persists, refreshes the cache entry, trains the estimator
  - `Split(txID, children) error`
  - `RenameCategory`, `DeleteCategory`
- `OverviewModel` takes a `*app.Service` and holds no `*db.DB`.
- `cmd/fin/cmd/tui.go` builds the service. It still loads cached embeddings (`categorize.LoadEmbeddings`) when `[llm]` is enabled and passes them to the service, which builds the estimator with `categorize.WithEmbeddings` (see `architecture/categorization.md`).

## Out of scope
- Moving the diff and draft model (`review_draft_model.md`).
- UI changes.

## Acceptance criteria
Write failing tests first (service with a temporary database):
- After `ApplyReview`, `Transactions` returns the updated category, tags and memo/note, and `Categories`/`Tags` include new names.
  This also fixes the old bug that the overview showed stale tags after a save (it updated only category, memo and review state in its cache).
- After `ApplyReview` with `reviewed = true`, `Suggest` for a similar transaction ranks the new category first.
- After `Split`, the parent is gone from `Transactions` and the children are present.

## Depends on
`db_list_transactions.md` (done).
