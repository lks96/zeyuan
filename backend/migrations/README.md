# Database Migrations

This directory keeps the incremental database upgrade history.

For a brand-new production database, use the current full schema script instead:

- `backend/scripts/zeyuan_db_full_schema.sql`

The full schema script should contain the final table definitions in one pass. These migration files are still kept so existing databases can upgrade safely from their current version.

## Rules

- Use numbered files: `001_init.sql`, `002_permissions.sql`, `003_feature_name.sql`.
- After a migration has been applied to any shared environment, do not edit it.
- Add a new migration for every schema or seed-data change.
- When adding a migration, also update `backend/scripts/zeyuan_db_full_schema.sql` so new databases can be created directly with the latest structure.
- Keep migrations idempotent when possible with `IF NOT EXISTS`, `ON DUPLICATE KEY UPDATE`, or `INSERT IGNORE`.
- Do not store real secrets in SQL files.

## How It Runs

Run migrations from `backend`:

```powershell
go run ./cmd/migrate
```

The migrator maintains `schema_migrations` with:

- `version`
- `name`
- `checksum`
- `applied_at`

Already-applied migrations are skipped. If an applied file changes, checksum validation fails so the change can be captured in a new migration instead.

## Current Files

- `001_init.sql`: users, shops, user-shop assignments, initial accounts
- `002_permissions.sql`: button-level permission dictionary and role permissions
- `003_feature_interfaces.sql`: tool modules and system settings used by current pages
