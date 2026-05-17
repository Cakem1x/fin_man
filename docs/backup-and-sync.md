# Backup and Sync

The finance tools do not implement syncing.

## Recommended Tools

- Syncthing
- rsync
- Nextcloud
- restic

## SQLite Notes

Prefer syncing while the database is closed.

Use WAL mode for better resilience.

## Backup Strategy

Recommended:

1. close tools
2. create snapshot
3. encrypt backup
4. sync externally
