# TUI rework 1: consistent keymaps

## Context
Part of `rework_transaction_edit.md` (read it first; it is the UX spec). Also see the TUI UX guidelines in `architecture/overview.md`.
Today keybindings conflict, e.g. typing `n` in a filter opens the new-category dialog.

## Scope
- One `bubbles/key` keymap per view in `internal/tui`, plus a help line rendered with `bubbles/help`.
- While a text input or a list filter is active, no shortcut fires; all keys go to the input. `Esc` leaves the input.
- No key has different meanings in different panels of the same view.
- Keep the current layout. Layout changes are `tui_panel_navigation.md`.

## Acceptance criteria
Write failing tests first (drive the models with `tea.KeyMsg`):
- Typing `n` into the category filter adds `n` to the filter and opens no dialog.
- The help line lists the bindings of the focused view.

## Depends on
None. Easier after `app_service_layer.md`.
