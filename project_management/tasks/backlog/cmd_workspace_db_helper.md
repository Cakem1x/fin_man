# Share workspace and database setup across commands

## Context
`import-csv`, `tui`, `workspace open` and `workspace upgrade` (`cmd/fin/cmd/`) each repeat: find the workspace root, check it is open, build `store/finance.db`, open the database.
`tui` calls `log.Fatalf` and `os.Exit` inside `RunE`.
No command except `workspace open` checks whether the schema is outdated.

## Scope
- `workspace.Manager.DBPath()` returns `<root>/store/finance.db`.
- A helper in `cmd/fin/cmd`, e.g. `openWorkspaceDB(requireUpToDate bool) (*db.DB, func(), error)`:
  - finds the root, checks the workspace is open, opens the DB
  - with `requireUpToDate`, returns an error that tells the user to run `fin workspace upgrade` when migrations are pending
- `import-csv` and `tui` use it with `requireUpToDate = true`.
- Commands return errors; no `log.Fatalf` or `os.Exit` in `RunE`.
- Workspace commands use `db.Open` instead of `sql.Open("sqlite3", ...)`, so all connections share one DSN.
- `--workdir`: pass the start directory to the workspace lookup instead of calling `os.Chdir`.

## Acceptance criteria
Write failing tests first:
- `Manager.DBPath()` returns the expected path.
- The helper returns an error (not an exit) for a closed workspace and for an outdated schema. Use a workspace directory whose `store/` is a plain directory where needed, or test the pieces separately.

## Depends on
None.
