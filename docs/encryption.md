# Encryption

The finance tools do not implement encryption directly.

## Recommended Setup

### Active Store

Use:

- gocryptfs
- CryFS

The SQLite database lives inside the mounted encrypted directory.

### Backups

Use:

- restic
- age

## Workflow

1. mount encrypted store
2. work with tools
3. close tools
4. unmount store
5. sync/backup

## Notes

Avoid syncing a live SQLite database when possible.
