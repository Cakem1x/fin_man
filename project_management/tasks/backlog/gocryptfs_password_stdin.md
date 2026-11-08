# Pass the workspace password to gocryptfs on stdin

## Context
See `architecture/encryption.md`.
`Manager.runGocryptfs` in `internal/workspace/workspace.go` writes the password to a temp file and passes `-passfile`.
The plain-text password lands on disk (usually `/tmp`).

## Scope
- Pass the password on gocryptfs' stdin instead. gocryptfs reads the password from stdin when stdin is not a terminal.
- Remove the temp file logic.
- Covers `-init` (init) and mounting (open).

## Acceptance criteria
- `internal/workspace/workspace_test.go` lifecycle test still passes (init, open, close).
- No file is created in `os.TempDir()` during init or open (assert in the test).

## Depends on
None.
