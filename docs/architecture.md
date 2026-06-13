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
