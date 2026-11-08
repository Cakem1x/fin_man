# Archive the source file before inserting

## Context
`cmd/fin/cmd/import_csv.go` inserts transactions with `archive_file_path` set and copies the file afterwards.
If the copy fails, the rows reference a file that does not exist.

## Scope
- Copy the source file to the archive path first.
- Insert the transactions.
- If the insert fails or inserts nothing, remove the archive copy.
- Move the copy logic out of `RunE` into a testable function.

## Acceptance criteria
Write failing tests first:
- When the copy fails, no transactions are inserted.
- When no transaction is new, no archive file remains.

## Depends on
`import_duplicates_nonfatal.md` (both change the same `RunE`; merge that first to avoid conflicts).
