# Data Model

## Core Tables

### transactions

Canonical normalized transactions.


### transaction_metadata

Categories, notes, review state.

### tags

Reusable labels.

### categorization_rules

Deterministic matching rules.

## Design Principles

- never mutate archived raw import files
- use stable deduplication hashes
- enrichment remains separate from facts
- schema migrations are append-only
