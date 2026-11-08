# Default currency for CSV configs

## Context
`dkb_v1` has `CurrencyCol: -1`, so every imported DKB transaction has an empty `currency`.

## Scope
- Add `DefaultCurrency string` to `genericcsv.Config`. Use it when `CurrencyCol` is negative or the column is empty.
- Set `DefaultCurrency: "EUR"` for `dkb_v1` in `internal/importer/genericcsv/configs.go`.
- Update the dkb_v1 golden file.

## Out of scope
- Backfilling existing rows. See `repair_from_archives.md`.

## Acceptance criteria
Write failing tests first:
- dkb_v1 golden output has `Currency: "EUR"`.
- A config with a currency column uses the column value; an empty cell falls back to `DefaultCurrency`.

## Data warning
Currency is part of the ID hash.
Do not import real data after merging this task until `repair_from_archives.md` has run.

## Depends on
None.
