# Block updates of imported columns

## Context
See `architecture/data-model.md`: imported columns are never updated.
A trigger enforces this so that no code path can change them by accident.

## Scope
- Migration with a trigger: `BEFORE UPDATE OF date, payee, amount_cents, currency, memo, archive_file_path ON transactions` → `RAISE(ABORT, 'imported columns are immutable')`.
- Document in `architecture/data-model.md`: a migration that must fix imported data drops the trigger, fixes the data, and recreates the trigger. Remove "(planned)" there.

## Out of scope
- `id` stays updatable (re-keying).

## Acceptance criteria
Write failing tests first:
- Updating `memo` fails.
- Updating `note`, `category_id` and `is_reviewed` succeeds.

## Depends on
`repair_from_archives.md` (the repair updates imported columns).
