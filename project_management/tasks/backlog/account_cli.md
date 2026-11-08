# fin account command

## Context
See `architecture/overview.md` (Main Components). Accounts are managed in the CLI only.

## Scope
- `fin account list`: name, IBAN, currency, transaction count.
- `fin account add <name> [--iban <iban>] [--currency <code>]`
- `fin account rename <old> <new>`
- Same workspace checks as other commands (workspace found and open, schema up to date).
- Update `architecture/overview.md`: remove "(planned)" from `account`.

## Out of scope
- Deleting accounts.

## Acceptance criteria
Write failing tests first for the db functions the commands need (e.g. rename to an existing name fails).

## Depends on
`accounts_schema.md`. Use the helper from `cmd_workspace_db_helper.md` if it is merged.
