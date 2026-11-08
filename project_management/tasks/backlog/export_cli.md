# fin export command

## Context
See `architecture/reporting.md`. `fin export` replaces separate query and report commands.
Ad-hoc analysis happens in external tools on the export.

## Scope
- `fin export [--from YYYY-MM-DD] [--to YYYY-MM-DD] [--account <name>]... [--category <path>] [--reviewed|--unreviewed] [--payee <substring>] [--format csv|json] [-o <file>]`
- Default output is stdout, default format is CSV.
- Columns: id, date, account, payee, amount (decimal string, e.g. `-12.10`), currency, memo, note, category, tags (`;`-separated), reviewed, parent_id.
- Uses `ListTransactions(ctx, Filter)` from `db_list_transactions.md`.
- Update `architecture/overview.md` and `architecture/reporting.md`: remove "(planned)".

## Out of scope
- ledger/hledger format.

## Acceptance criteria
Write failing tests first:
- CSV and JSON writers produce the expected output for a fixed set of transactions (golden files).
- Amounts are formatted from cents without floats (`-1210` → `-12.10`).

## Depends on
`db_list_transactions.md`. Account and note columns need `accounts_schema.md` and `transaction_note.md`; leave them out if those are not merged yet.
