# Small code cleanups

## Scope
- `internal/categorize/estimator.go`: `tokenize` compiles its regexp on every call. Make it a package-level variable.
- `internal/db/migrate.go`: `Migrate` and `NeedsUpgrade` both set goose global state (`SetBaseFS`, `SetDialect`). Use one `goose.NewProvider` constructor instead of the global API.

## Acceptance criteria
- Existing tests pass. No behavior change, so no new tests.

## Depends on
None.
