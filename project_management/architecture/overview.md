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
- deterministic rules before ML
- external encryption/sync tools

## Main Components

- `fin` (Unified CLI)
  - `import-csv`
  - `review`
  - `query`
  - `report`
  - `rules`
  - `db`

## Execution Model (Separation of Concerns)

There is a strict separation between the **Codebase** and the **User Workspace**:

1. **Codebase (`fin_man/`)**: Stateless logic, schema definitions, and dev environment.
2. **User Workspace (`~/finance/`)**: Stateful private data, configurations, and database.

The `fin` CLI is invoked by pointing it at a workspace configuration:
```sh
fin --config path/to/workspace/finance.toml <subcommand>
```

## TUI UX Guidelines

When designing and maintaining the Terminal User Interface for `fin_man`, adhere to the following principles:

1. **Clear Focus Flow**: Auto-focus the most commonly interacted pane (e.g. lists). `Tab` / `Shift+Tab` explicitly cycle focus through logical panes.
2. **Explicit Text Editing**: Text inputs capture character keys (like `j/k`). To avoid navigation clashes, hide text inputs behind explicit actions (e.g., an "Edit Memo" button) instead of defaulting them to active.
3. **Decouple Selection from Submission**: In lists, `Enter` should select or toggle items. It should not abruptly advance focus or submit entire forms, allowing users to scroll and review freely.
4. **Safe Escapes**: `Esc` should universally escape the current overlay or return to a safe state, preferably using explicit dialog boxes if data might be lost, instead of hidden `Ctrl+...` shortcuts.
