# Data Model

## Core Tables

### transactions

One row per transaction.

Imported columns: `date`, `payee`, `amount_cents`, `currency`, `memo`, `archive_file_path`.
The `id` is a hash of the imported columns (see [importing](importing.md)).

Enrichment columns: `category_id`, `is_reviewed`, `note` (planned).

`parent_id` links a split child to the transaction it splits (see [enrichment](enrichment.md)).

### categories

Mutually exclusive classification. Nested categories use `/` in the name (e.g. `Food/Groceries`).

### tags

Reusable labels. `transaction_tags` links tags to transactions.

### accounts (planned)

Bank accounts: `name`, `iban`, `currency`. `transactions.account_id` references an account.

## Design Principles

- never mutate archived raw import files
- imported columns are never updated; a trigger enforces this (planned)
- user edits go to enrichment columns, e.g. an edited memo goes to `note`
- use stable deduplication hashes
- schema migrations are append-only
