# Move the review draft and diff model out of the TUI

## Context
`TransactionDiff` (`internal/tui/transaction_diff.go`) and the draft state in `ReviewModel` (`internal/tui/review.go`: selected category, selected tags, note, `draftTransaction`, `HasPendingChanges`) are domain logic.
The TUI rework (`tui_edit_view.md`) builds a new edit view on them, so they must not depend on the current view code.

## Scope
- Move `TransactionDiff`, `BuildTransactionDiff` and a `Draft` type (category, tags, note, reviewed; `Apply(tx)`, `HasChanges`) to `internal/app`.
- `ReviewModel` holds an `app.Draft` and only renders it.
- `ApplyReview` in the service takes a `Draft`.
- Move the existing diff tests along.

## Acceptance criteria
- Moved tests pass in `internal/app`.
- Write failing tests first for `Draft`: a draft with no edits has no changes; reverting an edit removes the change.

## Depends on
`app_service_layer.md`.
