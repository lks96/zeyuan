package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"temu-tools/backend/internal/config"
)

func Migrate(ctx context.Context, cfg config.Config, migrationsDir string) error {
	serverDB, err := sql.Open("mysql", cfg.MySQLServerDSN())
	if err != nil {
		return err
	}
	defer serverDB.Close()

	if _, err := serverDB.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS `"+escapeIdentifier(cfg.DBName)+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		fmt.Printf("skip database creation: %v\n", err)
	}

	db, err := Open(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := ensureMigrationTable(ctx, db); err != nil {
		return err
	}

	appliedMigrations, err := loadAppliedMigrations(ctx, db)
	if err != nil {
		return err
	}

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return err
	}
	sort.Strings(files)

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		name := filepath.Base(file)
		version := migrationVersion(name)
		if version == "" {
			return fmt.Errorf("invalid migration name: %s", name)
		}

		checksum := checksum(content)
		if appliedChecksum, ok := appliedMigrations[version]; ok {
			if appliedChecksum != checksum {
				return fmt.Errorf("migration checksum mismatch for %s; create a new migration instead of editing applied SQL", name)
			}
			fmt.Printf("skip applied migration: %s\n", name)
			continue
		}

		for _, statement := range splitSQLStatements(string(content)) {
			if _, err := db.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("apply %s: %w", name, err)
			}
		}

		if err := recordMigration(ctx, db, version, name, checksum); err != nil {
			return err
		}
		fmt.Printf("applied migration: %s\n", name)
	}

	return nil
}

func ensureMigrationTable(ctx context.Context, db *sql.DB) error {
	const query = `
CREATE TABLE IF NOT EXISTS schema_migrations (
  version VARCHAR(32) NOT NULL,
  name VARCHAR(255) NOT NULL,
  checksum CHAR(64) NOT NULL,
  applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (version),
  UNIQUE KEY uk_schema_migrations_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`

	_, err := db.ExecContext(ctx, query)
	return err
}

func loadAppliedMigrations(ctx context.Context, db *sql.DB) (map[string]string, error) {
	const query = `SELECT version, checksum FROM schema_migrations`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	appliedMigrations := make(map[string]string)
	for rows.Next() {
		var version string
		var checksum string
		if err := rows.Scan(&version, &checksum); err != nil {
			return nil, err
		}
		appliedMigrations[version] = checksum
	}

	return appliedMigrations, rows.Err()
}

func recordMigration(ctx context.Context, db *sql.DB, version string, name string, checksum string) error {
	const query = `
INSERT INTO schema_migrations (version, name, checksum)
VALUES (?, ?, ?)`

	_, err := db.ExecContext(ctx, query, version, name, checksum)
	return err
}

func migrationVersion(name string) string {
	version, _, ok := strings.Cut(name, "_")
	if !ok {
		version = strings.TrimSuffix(name, ".sql")
	}
	version = strings.TrimSpace(version)
	if version == "" || strings.Contains(version, ".") {
		return ""
	}
	return version
}

func checksum(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func splitSQLStatements(content string) []string {
	parts := strings.Split(content, ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		statement := strings.TrimSpace(part)
		if statement == "" {
			continue
		}
		statements = append(statements, statement)
	}
	return statements
}

func escapeIdentifier(value string) string {
	return strings.ReplaceAll(value, "`", "``")
}
