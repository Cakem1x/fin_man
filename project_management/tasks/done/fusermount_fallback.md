# Unmount with fusermount3, fall back to fusermount

## Context
See `architecture/encryption.md`.
`Manager.Close` in `internal/workspace/workspace.go` runs `fusermount -u <store>`.
FUSE 3 systems only ship `fusermount3`. On NixOS it is a setuid wrapper at `/run/wrappers/bin/fusermount3`; there is no `fusermount`.
On such systems `fin workspace close` fails with `exec: "fusermount": executable file not found in $PATH`, and `TestWorkspaceLifecycle` fails at close and leaves gocryptfs mounts behind.

FUSE tools from nixpkgs cannot fix this: they are not setuid, so unprivileged unmount fails with "Operation not permitted". Do not add them to the dev shell.

## Scope
- `Manager.Close` uses `fusermount3 -u` if `fusermount3` is in `PATH`, otherwise `fusermount -u`. If neither exists, return an error that names both.
- Keep the binary lookup in a small function that takes the lookup function as a parameter, so it is testable without FUSE.
- `TestWorkspaceLifecycle` must clean up its mount even when a subtest fails (`t.Cleanup` that unmounts if still mounted), so failed runs leave no mounts behind.

## Acceptance criteria
Write failing tests first:
- Lookup: only `fusermount3` available → `fusermount3`; only `fusermount` → `fusermount`; both → `fusermount3`; none → error.
- `nix develop -c just test` passes on this machine (NixOS, only `fusermount3`), including `TestWorkspaceLifecycle`, and `grep gocryptfs /proc/mounts` shows no new test mounts afterwards.

## Depends on
None.
