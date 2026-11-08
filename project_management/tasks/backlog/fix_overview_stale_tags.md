# Fix stale tags in the overview after saving

## Context
`OverviewModel.saveReview` in `internal/tui/overview.go` updates the cached transaction's category name, memo and review state after a save.
It does not update `Tags` or `CategoryID`. The overview table shows old tags until restart.

## Scope
- After `EnrichTransaction` succeeds, update the cached transaction completely: `CategoryID`, `CategoryName`, `Tags`, `Memo`, `IsReviewed`.
  Prefer reloading the transaction from the database over patching fields by hand.

## Out of scope
- The service layer refactoring (`app_service_layer.md`). Keep this fix small.

## Acceptance criteria
Write a failing test first:
- After saving a review with new tags, the overview's cached transaction has those tags.

## Depends on
None.
