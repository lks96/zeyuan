# Database Scripts

## Files

- `zeyuan_db_full_schema.sql`: current full database creation script. Use this for a brand-new environment.
- `init_zeyuan_db.sql`: production initialization script kept for compatibility with the previous deployment flow.
- `test-api.ps1`: local API smoke test helper.

## Runtime Entry

Startup/setup scripts call:

```bash
go run ./cmd/dbprepare
```

`dbprepare` checks the configured MySQL database:

- missing database or empty database: run `zeyuan_db_full_schema.sql`
- existing database with tables: run numbered migrations from `backend/migrations`

Fresh database initialization requires `ADMIN_INITIAL_PASSWORD` with at least 12 characters.

## Maintenance Rule

Schema changes should be maintained in two places:

1. Add a new numbered migration under `backend/migrations` for existing databases.
2. Update `zeyuan_db_full_schema.sql` so a fresh database can be created in one pass with the latest structure.

Do not rewrite old migration files after they may have been applied in a shared environment.
