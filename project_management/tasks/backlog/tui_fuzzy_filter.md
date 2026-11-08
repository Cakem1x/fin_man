# TUI rework 4: fuzzy filtering

## Context
Part of `rework_transaction_edit.md` ("Fuzzy filter/search features are not in place or do not work properly").

## Scope
- Category and tag pickers and the transaction list support fuzzy filtering with the built-in filter of `bubbles/list` (or `bubbles/table` plus `sahilm/fuzzy`, already a transitive dependency of `bubbles/list`).
- While filtering, all shortcuts are disabled (`tui_keymap.md`).
- In pickers, `Enter` on a filter text that matches no item offers to create it as a new category or tag.

## Acceptance criteria
Write failing tests first:
- Filtering `grc` matches `Food/Groceries`.
- Typing shortcut letters while filtering only changes the filter.

## Depends on
`tui_keymap.md`.
