# TUI: accounts

## Context
See `architecture/data-model.md` (accounts) and `accounts_schema.md`.

## Scope
- Overview: an account filter next to the time filter (All, then each account). Header stats follow it.
- Transaction list: an account column when more than one account is visible.
- Edit view header shows the account (read-only).
- Totals are grouped by currency. No currency conversion.

## Out of scope
- Internal transfers between own accounts (deferred; see `architecture/roadmap.md`).

## Acceptance criteria
Write failing tests first:
- The account filter hides transactions of other accounts and updates the stats.
- With transactions in EUR and USD, totals show two lines, one per currency.

## Depends on
`accounts_schema.md`, `tui_panel_navigation.md`.
