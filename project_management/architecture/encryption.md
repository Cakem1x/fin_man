# Encryption

`fin` does not implement encryption.
`fin workspace` drives [gocryptfs](https://nuetzlich.net/gocryptfs/) to encrypt the workspace data.

## Lifecycle

1. `fin workspace init --store <path>`: creates the gocryptfs store at `<path>` (or reuses an existing one), writes `fin.toml`, mounts the store, and creates the database.
2. `fin workspace open`: asks for the password and mounts the store at `store/`.
3. Work with `fin import-csv`, `fin tui`, and other commands. They refuse to run while the workspace is closed.
4. `fin workspace close`: unmounts `store/` with `fusermount -u`.
5. Sync or back up the encrypted store (see [backup and sync](backup-and-sync.md)).

## Notes

The encrypted store can live outside the workspace directory, e.g. in a synced folder.
Only the encrypted store needs to be synced or backed up.
