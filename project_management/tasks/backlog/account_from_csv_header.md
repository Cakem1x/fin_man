# Detect the account from the CSV header

## Context
DKB exports start with the account IBAN in the header:

```text
"Girokonto";"DE30301010303010103030"
```

(see `internal/importer/genericcsv/testdata/dkb_v1/set1.csv`).
Reading it removes the need to pass `--account` and prevents importing a file into the wrong account.

## Scope
- `genericcsv.Config` gets `IBANLine` and `IBANCol` (1-based line within the skipped lines; negative or zero means "not available"). Set them for `dkb_v1` (line 1, column 2).
- The importer returns the detected IBAN together with the transactions.
- `fin import-csv`:
  - IBAN detected and matches an account: use it. `--account` is optional.
  - IBAN detected, no account has it: ask for confirmation and an account name, then create the account.
  - `--account` given and its IBAN differs from the detected one: fail.
- Expose the header parsing as a function so `repair_from_archives.md` can reuse it.

## Out of scope
- The balance line ("Kontostand"). Balance snapshots are a separate future task.

## Acceptance criteria
Write failing tests first:
- dkb_v1 testdata yields IBAN `DE30301010303010103030`.
- A config without IBAN settings yields no IBAN.

## Depends on
`accounts_schema.md`.
