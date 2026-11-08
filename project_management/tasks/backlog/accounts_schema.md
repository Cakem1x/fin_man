# Add accounts

## Context
See `architecture/data-model.md`. Transactions have no account today.
Multiple bank accounts need to be told apart for filtering and, later, balances.

## Scope
- Migration:
  - `accounts(id TEXT PRIMARY KEY, name TEXT UNIQUE NOT NULL, iban TEXT UNIQUE, currency TEXT NOT NULL DEFAULT 'EUR')`
  - `transactions.account_id TEXT REFERENCES accounts(id)` (nullable; existing rows stay NULL until `repair_from_archives.md`)
- `model.Account`; `model.Transaction` gets `AccountID *string` and `AccountName *string` (hydrated).
- `internal/db`: `CreateAccount`, `GetAccountByName`, `ListAccounts`. Account IDs are random hex like category IDs (`category_tag_ids.md`).
- `fin import-csv --account <name>` is required. Unknown name: ask for confirmation (`internal/prompt.Confirm`), then create the account.
- Inserted transactions get `account_id`.
- Add an account filter (account names) to `db.Filter` and `ListTransactions` (`internal/db/list.go`).

## Out of scope
- Including the account in the ID hash. That happens in `repair_from_archives.md` together with re-keying.
- Reading the account from the CSV header (`account_from_csv_header.md`).
- TUI changes (`tui_accounts.md`).

## Acceptance criteria
Write failing tests first:
- Inserting transactions with an account stores `account_id`.
- `GetAllTransactions` (or its successor) hydrates `AccountName`.

## Data warning
After this task, import real data only after `repair_from_archives.md` has run.
The repair changes the ID hash to include the account.

## Depends on
`category_tag_ids.md` (shared random-ID helper).
