# TUI rework 3: edit view built on the diff

## Context
Part of `rework_transaction_edit.md` (read the "Edit Transactions" section; it is the UX spec).

## Scope
- The top panel shows the transaction with the diff always visible. Added values green, removed red; new tags and categories highlighted.
- The diff is navigable. Moving the cursor to "Category" or "Tags" shows the picker in the right panel (small, golden ratio). `Enter` enlarges it for editing; `Enter` or `Esc` returns.
- Suggestions keep their score ordering and percentages (`architecture/categorization.md`).
- "Edit Note" and "Receipt Matcher" are actions on the same level.
- The exit dialog with the diff stays.
- The view works on `app.Draft` (`review_draft_model.md`).

## Acceptance criteria
Write failing tests first:
- Selecting a category in the picker updates the draft and the rendered diff.
- `Esc` with unsaved changes opens the exit dialog; without changes it closes.

## Depends on
`review_draft_model.md`, `tui_panel_navigation.md`.
