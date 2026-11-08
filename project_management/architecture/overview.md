# Architecture

## Goals

- local-first
- CLI-first
- composable tools
- low operational complexity
- privacy-focused

## Core Stack

- Go
- SQLite
- Nix Flake
- Bubble Tea (TUI)

## Principles

- raw imports are immutable
- enrichment is layered metadata
- suggestions are explainable; the user confirms every categorization
- encryption is delegated to gocryptfs; sync and backup to external tools

## Non-Goals

- web frontend or mobile app
- cloud sync service
- multi-user support
- automatic budgeting engine
- custom cryptography
- double-entry accounting

## Main Components

`fin` is the unified CLI:

- `workspace {init,open,close,upgrade}`: workspace and encrypted store lifecycle (see [encryption](encryption.md))
- `import-csv`: import a bank CSV export (see [importing](importing.md))
- `classify`: compute transaction embeddings with Ollama (see [categorization](categorization.md))
- `tui`: explore, categorize, tag, and enrich transactions (see [categorization](categorization.md), [enrichment](enrichment.md))
- `account` (planned): manage accounts (see [data model](data-model.md))
- `export` (planned): write filtered transactions as CSV or JSON (see [reporting](reporting.md))

## Execution Model (Separation of Concerns)

There is a strict separation between the **Codebase** and the **User Workspace**:

1. **Codebase (`fin_man/`)**: Stateless logic, schema definitions, and dev environment.
2. **User Workspace**: Stateful private data.

A workspace is a directory that contains `fin.toml`.
`fin` finds the workspace like git finds a repository: it walks up from the current directory until it finds `fin.toml`.
`fin -C <path> <subcommand>` starts the search at `<path>`.

Workspace layout while the workspace is open:

```text
my-finances/
├── fin.toml          # store_path: location of the encrypted gocryptfs store; optional [llm] section
└── store/            # mount point of the decrypted store
    ├── finance.db    # SQLite database
    └── archives/     # copies of imported source files, e.g. csv_dkb_v1/
```

Categories and tags exist only in the database.

## TUI UX Guidelines

When designing and maintaining the Terminal User Interface for `fin_man`, adhere to the following principles:

1. **Clear Focus Flow**: Auto-focus the most commonly interacted pane (e.g. lists). `Tab` / `Shift+Tab` explicitly cycle focus through logical panes.
2. **Explicit Text Editing**: Text inputs capture character keys (like `j/k`). To avoid navigation clashes, hide text inputs behind explicit actions (e.g., an "Edit Note" button) instead of defaulting them to active.
3. **Decouple Selection from Submission**: In lists, `Enter` should select or toggle items. It should not abruptly advance focus or submit entire forms, allowing users to scroll and review freely.
4. **Safe Escapes**: `Esc` should universally escape the current overlay or return to a safe state, preferably using explicit dialog boxes if data might be lost, instead of hidden `Ctrl+...` shortcuts.
