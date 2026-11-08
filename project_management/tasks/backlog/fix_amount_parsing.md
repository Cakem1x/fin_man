# Fix amount parsing truncation

## Context
`parseAmountToCents` in `internal/importer/genericcsv/parser.go` converts amounts with `int64(f*100)`.
Float errors truncate some values by one cent: `"1,15"` becomes 114, `"4,35"` becomes 434.
`internal/importer/amazon.go` has the same bug.

## Scope
- Add `normalize.ParseAmountCents(s, decimalSep, thousandsSep string) (int64, error)` in `internal/normalize`.
  It parses the decimal string with integer arithmetic only. No floats.
- Use it in `genericcsv` and in the Amazon matcher. Remove `parseAmountToCents`.
- Update the dkb_v1 golden file if values change.

## Out of scope
- Repairing amounts already stored in the database. See `repair_from_archives.md`.

## Acceptance criteria
Write failing tests first:
- `"1,15"` → 115, `"-4,35"` → -435, `"1.234,5"` → 123450, `"-13"` → -1300, `"0,29"` → 29
- `" 12,10 "` (surrounding whitespace) → 1210
- empty string → 0 (current behavior)
- invalid input (`"abc"`, `"1,2,3"`, more than two decimal digits) → error

## Data warning
This changes the ID hash of affected transactions.
Do not import real data after merging this task until `repair_from_archives.md` has run.
Otherwise, overlapping imports insert near-duplicates that are one cent off.

## Depends on
None.
