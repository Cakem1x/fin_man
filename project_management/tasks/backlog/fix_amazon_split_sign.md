# Fix Amazon split sign and extract split building

## Context
See `architecture/enrichment.md`.
`internal/tui/review.go` builds split children from Amazon items.
Bank debits are negative; Amazon item prices are positive.
The remainder `parent - sum(items)` is therefore wrong, e.g. parent -30.00 with items 10.00 + 20.00 gives a remainder of -60.00.

## Scope
- Move `internal/importer/amazon.go` to a new package `internal/enrich` (receipt matchers are enrichment, not importing).
- Add a pure function `enrich.BuildSplits(parent model.Transaction, items []Item) []model.Transaction`.
  - Item amounts take the sign of the parent amount.
  - A remainder child is added when the item sum differs from the parent amount.
  - Child IDs, dates, payee, currency and archive path stay as today.
- `review.go` calls `BuildSplits` instead of building splits inline.
- Update `architecture/enrichment.md` (package name).

## Acceptance criteria
Write failing tests first:
- parent -3000, items 1000 + 2000 → children -1000, -2000, no remainder
- parent -3500, items 1000 + 2000 → children -1000, -2000, remainder -500
- parent +3000 (refund), items 1000 + 2000 → children +1000, +2000

## Depends on
`fix_amount_parsing.md` (it changes the Amazon price parsing in the same file).
