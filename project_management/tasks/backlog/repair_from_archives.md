# Repair existing transactions from archived source files

## Context
Several fixes change data that is already stored:
- `fix_amount_parsing.md`: stored amounts may be one cent too low or too high.
- `csv_default_currency.md`: stored DKB currency is empty.
- `accounts_schema.md`: stored rows have no account.
- `transaction_note.md`: edited memos overwrote the imported memo.

The transaction ID is a hash of the imported columns (`internal/dedup`).
New imports compute IDs from the corrected data. Without a repair, overlapping imports insert near-duplicates.

This task also changes the hash to include the account ("hash v2"), so all rows are re-keyed exactly once.

## Scope
- `dedup.GenerateHash` includes the account ID. `import-csv` uses it.
- New command `fin workspace repair [--dry-run] [--default-account <name>]`.
- For each archived file under `store/archives/csv_<config>/`:
  - Look up the builtin config by name. Unknown config: report and skip.
  - Re-parse the file with the fixed parser. Detect the account from the header (`account_from_csv_header.md`), else use `--default-account`.
  - For each parsed row, compute the legacy ID: the old hash with the legacy truncated amount (`int64(f*100)`) and empty currency. Keep the legacy amount function inside the repair code only.
  - Find the stored row with that ID. Then, in one DB transaction:
    - restore `memo` from the file; if the stored memo differs and `note` is empty, move the stored memo to `note`
    - set the corrected `amount_cents` and `currency`
    - set `account_id`
    - change `id` to the hash v2 ID, and update `transaction_tags.transaction_id` and children's `parent_id`
- Rows not matched (no archive path, unknown config, or no match): assign `--default-account` and re-key with hash v2 from stored values. Report them.
- Split children (`parent_id IS NOT NULL`) keep their IDs and take the account of their parent.
- `--dry-run` prints the planned changes and changes nothing.
- Back up `finance.db` (copy next to it with a timestamp) before writing.

## Acceptance criteria
Write failing tests first, with a temporary database and archive:
- A row stored with a truncated amount, empty currency and an edited memo is repaired: correct amount, `EUR`, original memo, edited text in `note`, account set, new ID.
- Its tags and split children point to the new ID.
- Running the repair twice changes nothing the second time.
- Importing the archived file again after the repair inserts nothing.

## Depends on
`fix_amount_parsing.md`, `csv_default_currency.md`, `transaction_note.md`, `accounts_schema.md`, `account_from_csv_header.md`, `db_foreign_keys.md`.
