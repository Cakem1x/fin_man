# Backup and Sync

Backups and syncing are explicitly out of scope for the `fin_man` project.

It is strongly recommended to handle backups and syncing externally using your preferred tools (e.g., Syncthing, rsync, Nextcloud, restic).

## SQLite Notes

Prefer syncing while the database is closed.

Use WAL mode for better resilience.
