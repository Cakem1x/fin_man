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

## Non-Goals

- web app
- cloud sync platform
- mobile app
- custom encryption
- double-entry accounting initially
