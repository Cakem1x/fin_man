# Backup and Sync

Backups and syncing are explicitly out of scope for the `fin_man` project.

Handle backups and syncing externally (e.g., Syncthing, rsync, Nextcloud, restic).
Sync or back up the encrypted gocryptfs store, not the mounted `store/` directory (see [encryption](encryption.md)).

## SQLite Notes

Sync while the workspace is closed (`fin workspace close`).
An open workspace may have uncommitted WAL data.

`fin` opens the database in WAL mode.
