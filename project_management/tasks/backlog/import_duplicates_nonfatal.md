# Make duplicate imports non-fatal

## Context
Bank exports often overlap. Duplicates are expected (see `architecture/importing.md`).
Today `db.InsertTransactions` returns an error when it finds duplicates, and `fin import-csv` exits 1.

Two different cases exist:
- **Duplicate of an existing row**: the transaction is already in the database. Expected for overlaps.
- **Identical rows within one file**: two rows in the same file have the same ID. These may be two real transactions (e.g. two equal coffees on one day). Only one is stored. The hash stays unchanged; the user must be warned.

## Scope
- `InsertTransactions` returns an error only for real failures. It still returns the duplicate pairs.
- Detect identical rows within one file before inserting (same ID twice in the parsed slice).
- `import-csv` output:
  - existing duplicates: an info summary (count; details with `-v`)
  - identical rows within the file: a warning listing each row
- Exit 0 in both cases. Add `--strict`, which exits 1 when any duplicate occurs.

## Acceptance criteria
Write failing tests first:
- `InsertTransactions` with existing duplicates returns `err == nil` and the duplicate pairs.
- A function that finds identical rows in a parsed slice returns them (test it in isolation).

## Depends on
None.
