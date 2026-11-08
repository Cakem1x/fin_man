# Reporting

## Approach

- The TUI overview shows category totals and stats for a selected time range.
- `fin export` (planned) writes filtered transactions as CSV or JSON.
  Filters: date range, account, category, review state, payee substring.
- Ad-hoc analysis uses external tools on the export (e.g. duckdb, visidata, jq, spreadsheets).

A built-in `fin report` command is deferred until a concrete need comes up.

## Future Export Formats

- ledger / hledger

## Query Philosophy

Reports and exports are read-only and reproducible.
