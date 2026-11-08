# Consolidate transaction queries

## Context
`internal/db/db.go` has three nearly identical functions: `GetAllTransactions`, `GetCategorizedTransactions`, `GetUnreviewedTransactions` (the last one is unused).
`hydrateTags` loads every tag link in the database for every call.
`RenameCategory` and `DeleteEmptyCategory` load all categories to filter by prefix in Go.
`DeleteCategories` is unused.

## Scope
- Add a `Filter` type (in `internal/db` or a new `internal/query` package):
  date from/to, account names, category path (including nested categories), reviewed state, payee substring, include/exclude uncategorized.
  Filter fields are optional. Account filtering applies once `accounts_schema.md` is merged; add it then if this task runs first.
- `ListTransactions(ctx, Filter)` with one shared `scanTransaction`. It keeps hiding split parents.
- Replace the three `Get*Transactions` functions with it. Update callers.
- `hydrateTags` only loads tags for the returned transactions.
- `RenameCategory` and `DeleteEmptyCategory` use `name = ? OR name LIKE ? || '/%'` in SQL.
- Delete `DeleteCategories`.
- Update `architecture/enrichment.md`, which names `GetUnreviewedTransactions`.

## Acceptance criteria
Write failing tests first:
- Each filter field narrows the result as expected.
- Split parents are never returned.
- Tags are hydrated only for returned transactions.

## Depends on
None.
