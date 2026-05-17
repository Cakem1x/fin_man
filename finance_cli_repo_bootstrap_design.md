# Finance CLI Suite — Repository Bootstrap

This document restructures the architecture proposal into concrete repository files and documents.

---

# Repository Layout

```text
finance/
├── flake.nix
├── .envrc
├── .gitignore
├── README.md
├── Makefile
├── justfile
├── go.mod
├── go.sum
├── config/
│   ├── finance.example.toml
│   └── categories.example.toml
├── cmd/
│   ├── fin-import/
│   │   └── main.go
│   ├── fin-review/
│   │   └── main.go
│   ├── fin-query/
│   │   └── main.go
│   ├── fin-report/
│   │   └── main.go
│   ├── fin-rules/
│   │   └── main.go
│   └── fin-db/
│       └── main.go
├── internal/
│   ├── config/
│   ├── db/
│   ├── model/
│   ├── importer/
│   │   ├── genericcsv/
│   │   ├── dkb/
│   │   ├── ing/
│   │   └── revolut/
│   ├── normalize/
│   ├── dedup/
│   ├── categorize/
│   ├── similarity/
│   ├── rules/
│   ├── report/
│   ├── query/
│   ├── bankapi/
│   ├── tui/
│   └── export/
├── migrations/
│   ├── 0001_initial.sql
│   ├── 0002_categories.sql
│   └── 0003_rules.sql
├── scripts/
│   ├── mount-store.sh
│   ├── unmount-store.sh
│   ├── backup.sh
│   └── restore.sh
├── docs/
│   ├── architecture.md
│   ├── data-model.md
│   ├── categorization.md
│   ├── encryption.md
│   ├── importing.md
│   ├── reporting.md
│   ├── backup-and-sync.md
│   └── roadmap.md
├── testdata/
│   ├── csv/
│   └── anonymized/
└── data/
    └── .gitkeep
```

---

# File Responsibilities

# Root Files

## README.md

Purpose:
- onboarding
- installation
- development workflow
- philosophy
- quick examples

Should contain:
- project goals
- repository layout overview
- setup instructions
- example workflows
- security caveats

Should NOT contain:
- deep architecture details
- schema explanations
- implementation details

---

## flake.nix

Purpose:
- reproducible development environment
- shell provisioning
- formatter/linter tooling
- CI parity

Should provide:
- Go toolchain
- SQLite tools
- sqlc
- golangci-lint
- migrate/goose
- just
- git

Should NOT:
- perform runtime encryption
- contain secrets
- manage user data

---

## .envrc

Purpose:
- automatic shell activation via direnv

Minimal content:

```sh
use flake
```

---

## .gitignore

Must ignore:

```gitignore
/data/
*.db
*.db-shm
*.db-wal
*.sqlite
*.sqlite3
.env
```

Must NOT ignore:
- migrations
- configs
- test fixtures

---

## justfile

Purpose:
- developer convenience tasks

Example targets:

```text
just build
just test
just lint
just migrate
just fmt
```

Should NOT become:
- deployment orchestration
- complex scripting framework

---

# Config Files

## config/finance.example.toml

Purpose:
- application configuration template

Should include:
- db path
- encryption mount path
- default currency
- importer defaults
- report defaults

Should NOT include:
- passwords
- API secrets
- personal data

---

## config/categories.example.toml

Purpose:
- bootstrap category taxonomy

Example:

```toml
[[category]]
name = "groceries"

[[category]]
name = "rent"
```

---

# CLI Executables

# Philosophy

Each executable should:
- do one thing well
- compose with pipes/scripts
- remain independently understandable

Avoid giant shared command hierarchies.

---

## cmd/fin-import/

Purpose:
- ingest transaction data

Responsibilities:
- parse CSV
- normalize transactions
- store raw imports
- deduplicate
- insert canonical transactions

Should support:

```sh
fin-import dkb file.csv
fin-import generic --mapping mapping.toml file.csv
```

Should NOT:
- perform interactive categorization
- generate reports
- sync files

---

## cmd/fin-review/

Purpose:
- human enrichment workflow

Responsibilities:
- review uncategorized transactions
- accept suggestions
- edit metadata
- create rules
- tag transactions

Should use:
- terminal UI
- keyboard-driven interactions

Should NOT:
- import CSVs
- manage backups

---

## cmd/fin-query/

Purpose:
- ad-hoc querying

Example usage:

```sh
fin-query category groceries
fin-query merchant amazon
fin-query tag reimbursable
```

Should support:
- filtering
- sorting
- JSON output
- CSV output

Should NOT:
- mutate transaction data by default

---

## cmd/fin-report/

Purpose:
- aggregate analytics

Reports:
- monthly summaries
- merchant analysis
- category trends
- account balances
- cashflow

Output formats:
- terminal tables
- CSV
- HTML (later)

Should NOT:
- modify transaction metadata

---

## cmd/fin-rules/

Purpose:
- manage categorization rules

Responsibilities:
- list rules
- test rules
- reorder priorities
- enable/disable rules

Should support:

```sh
fin-rules list
fin-rules test transaction-id
```

---

## cmd/fin-db/

Purpose:
- low-level database operations

Responsibilities:
- migrations
- integrity checks
- vacuum
- backup helpers

Should NOT:
- expose raw SQL shell functionality initially

---

# Internal Packages

# internal/model/

Purpose:
- canonical domain types

Examples:
- Transaction
- Category
- Rule
- Account

Rules:
- no database logic
- no UI logic

---

# internal/db/

Purpose:
- database access layer

Responsibilities:
- SQLite initialization
- query execution
- transactions
- migrations integration

Recommended:
- raw SQL
- sqlc-generated code

Avoid:
- heavy ORM abstractions

---

# internal/importer/

Purpose:
- source-specific parsing

Each importer should:
- parse source format
- produce normalized intermediate model

Must NOT:
- write directly to DB
- contain categorization logic

---

# internal/normalize/

Purpose:
- canonical cleanup

Responsibilities:
- date normalization
- currency normalization
- whitespace cleanup
- merchant cleanup
- unicode normalization

This layer is critical for:
- deduplication
- categorization quality

---

# internal/dedup/

Purpose:
- duplicate detection

Responsibilities:
- stable hashing
- duplicate checks
- fuzzy duplicate analysis

Must remain:
- deterministic
- explainable

---

# internal/categorize/

Purpose:
- categorization engine

Architecture:

```text
rules
  ↓
similarity lookup
  ↓
manual review
```

Should expose:

```go
Suggest(transaction) -> suggestions
```

Should NOT:
- directly mutate DB

---

# internal/similarity/

Purpose:
- merchant/payee similarity analysis

Recommended techniques:
- trigram similarity
- token overlap
- normalized merchant extraction
- edit distance

Avoid initially:
- embeddings
- neural networks
- external AI APIs

---

# internal/rules/

Purpose:
- deterministic matching engine

Example rule types:
- payee contains
- regex match
- IBAN match
- amount range
- account-specific rule

Rules must be:
- transparent
- inspectable
- testable

---

# internal/report/

Purpose:
- aggregation and analytics

Responsibilities:
- grouping
- rolling summaries
- trend calculations
- export formatting

Should remain:
- stateless

---

# internal/query/

Purpose:
- reusable filtering/query primitives

Should support:
- composable filters
- reusable SQL fragments
- pagination

---

# internal/tui/

Purpose:
- terminal UI components

Recommended stack:
- Bubble Tea
- Lip Gloss

Responsibilities:
- transaction review screens
- keyboard bindings
- selection widgets

Must remain:
- thin UI layer

Business logic belongs elsewhere.

---

# internal/bankapi/

Purpose:
- future bank integration abstraction

Initial scope:
- interfaces only
- no real provider implementations

Example:

```go
type TransactionProvider interface {
    FetchTransactions(...) ([]Transaction, error)
}
```

---

# internal/export/

Purpose:
- export adapters

Future targets:
- CSV
- JSON
- ledger
- hledger
- parquet

---

# Migration Files

# migrations/

Purpose:
- schema evolution

Rules:
- append-only
- immutable after merge
- idempotent where possible

Must contain:
- schema creation
- indexes
- constraints

Should NOT:
- contain business logic

---

# Scripts

# scripts/mount-store.sh

Purpose:
- mount encrypted store

Recommended integration:
- gocryptfs

Example responsibilities:
- mount encrypted directory
- validate mount exists
- export DB path env var

Should NOT:
- implement encryption itself

---

# scripts/unmount-store.sh

Purpose:
- clean unmount workflow

Should:
- check for open DB handles
- flush writes
- unmount safely

---

# scripts/backup.sh

Purpose:
- snapshot backup helper

Recommended tools:
- restic
- rsync

Should:
- create consistent DB snapshot
- optionally compress
- optionally invoke external backup tool

---

# scripts/restore.sh

Purpose:
- restore snapshot workflow

Should:
- validate DB integrity after restore

---

# Documentation Files

# docs/architecture.md

Contains:
- overall architecture
- design philosophy
- subsystem relationships
- lifecycle diagrams

---

# docs/data-model.md

Contains:
- schema explanation
- transaction lifecycle
- metadata model
- deduplication strategy

Should include:
- ER diagrams later

---

# docs/categorization.md

Contains:
- rule engine
- similarity heuristics
- confidence scoring
- review workflow

Important:
- explain WHY deterministic rules dominate initially

---

# docs/encryption.md

Contains:
- gocryptfs workflow
- mount/unmount lifecycle
- backup considerations
- threat model assumptions

Should explicitly state:
- application does not implement encryption

---

# docs/importing.md

Contains:
- importer architecture
- CSV mapping strategy
- normalization examples
- deduplication pipeline

---

# docs/reporting.md

Contains:
- supported reports
- aggregation semantics
- category handling
- recurring transaction ideas

---

# docs/backup-and-sync.md

Contains:
- SQLite WAL caveats
- sync recommendations
- restic examples
- Syncthing recommendations
- backup validation procedures

---

# docs/roadmap.md

Should split work into phases:

Phase 1:
- schema
- importers
- queries

Phase 2:
- categorization
- TUI review

Phase 3:
- reports
- exports

Phase 4:
- bank APIs
- automation

Future:
- investments
- receipts
- recurring transactions
- forecasting

---

# Database Design Principles

# Core Principle

Raw imported data is immutable.

Never mutate:
- imported CSV payloads
- imported bank payloads
- raw transaction rows

All enrichment is layered on top.

---

# Categorization Principles

Priority order:

```text
manual override
  > deterministic rules
  > similarity suggestions
  > uncategorized
```

This guarantees predictability.

---

# Security Principles

The application itself:
- does not encrypt files
- does not sync files
- does not store secrets in repo

The application integrates cleanly with:
- encrypted filesystems
- external sync tools
- external backup tools

---

# Explicit Non-Goals

Do NOT build initially:

- web frontend
- mobile app
- cloud sync service
- collaborative multi-user support
- automatic budgeting engine
- AI chatbot finance assistant
- custom cryptography
- double-entry accounting system

These can all be future extensions.

---

# Recommended Initial Build Order

1. SQLite schema
2. migrations
3. importer pipeline
4. normalization
5. deduplication
6. query CLI
7. categorization rules
8. similarity engine
9. review TUI
10. reporting

