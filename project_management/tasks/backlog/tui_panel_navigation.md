# TUI rework 2: left-to-right panel navigation

## Context
Part of `rework_transaction_edit.md` (read the "Full UI Changes" section; it is the UX spec).

## Scope
- `Enter` moves one level deeper (to the right panel); `Esc` moves one level up.
- The focused panel is large, the next panel small; width ratio is the golden ratio (about 62/38).
- Overview: categories with stats on the left; transactions of the highlighted category on the right.
  `Enter` on a category focuses the transaction list. `Enter` on a transaction opens the edit view (full screen; the current review view until `tui_edit_view.md` replaces it).
- Leave room in the header for a second filter (account filter, `tui_accounts.md`).
- Use the keymaps from `tui_keymap.md`.

## Acceptance criteria
Write failing tests first:
- `Enter` on a category focuses the transaction panel; `Esc` returns focus to the category panel.
- A pure layout function returns golden-ratio widths for a given terminal width.

## Depends on
`tui_keymap.md`.
