package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"temu-tools/backend/internal/config"
	"temu-tools/backend/internal/database"
)

func main() {
	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	serverDB, err := sql.Open("mysql", cfg.MySQLServerDSN())
	if err != nil {
		log.Fatalf("open mysql: %v", err)
	}
	defer serverDB.Close()

	state, err := inspectDatabase(ctx, serverDB, cfg.DBName)
	if err != nil {
		log.Fatalf("inspect database: %v", err)
	}

	if state.IsFresh {
		if err := runFullSchema(ctx, serverDB, cfg); err != nil {
			log.Fatalf("initialize fresh database: %v", err)
		}
		log.Printf("fresh database initialized with full schema: %s", cfg.DBName)
		return
	}

	if err := database.Migrate(ctx, cfg, "migrations"); err != nil {
		log.Fatalf("migration failed: %v", err)
	}
	log.Printf("existing database migrated: %s", cfg.DBName)
}

type databaseState struct {
	Exists  bool
	Tables  int
	IsFresh bool
}

func inspectDatabase(ctx context.Context, db *sql.DB, dbName string) (databaseState, error) {
	var exists int
	if err := db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?`,
		dbName,
	).Scan(&exists); err != nil {
		return databaseState{}, err
	}
	if exists == 0 {
		return databaseState{Exists: false, IsFresh: true}, nil
	}

	var tables int
	if err := db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = ?`,
		dbName,
	).Scan(&tables); err != nil {
		return databaseState{}, err
	}

	return databaseState{
		Exists:  true,
		Tables:  tables,
		IsFresh: tables == 0,
	}, nil
}

func runFullSchema(ctx context.Context, db *sql.DB, cfg config.Config) error {
	adminPassword := strings.TrimSpace(os.Getenv("ADMIN_INITIAL_PASSWORD"))
	if adminPassword == "" {
		adminPassword = strings.TrimSpace(os.Getenv("ADMIN_PASSWORD"))
	}
	if len(adminPassword) < 12 || adminPassword == "CHANGE_ME_BEFORE_RUN" || adminPassword == "change_me" {
		return fmt.Errorf("fresh database requires ADMIN_INITIAL_PASSWORD with a real value of at least 12 characters")
	}

	path := filepath.Join("scripts", "zeyuan_db_full_schema.sql")
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	sqlText := string(content)
	adminUsername := envWithDefault("ADMIN_USERNAME", "admin")
	adminDisplayName := envWithDefault("ADMIN_DISPLAY_NAME", "系统管理员")
	sqlText = strings.ReplaceAll(sqlText, "CREATE DATABASE IF NOT EXISTS `zeyuan_db`", "CREATE DATABASE IF NOT EXISTS `"+escapeIdentifier(cfg.DBName)+"`")
	sqlText = strings.ReplaceAll(sqlText, "USE `zeyuan_db`;", "USE `"+escapeIdentifier(cfg.DBName)+"`;")
	sqlText = replaceSetVariable(sqlText, "admin_username", adminUsername)
	sqlText = replaceSetVariable(sqlText, "admin_initial_password", adminPassword)
	sqlText = replaceSetVariable(sqlText, "admin_display_name", adminDisplayName)

	for _, statement := range splitSQLWithDelimiters(sqlText) {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("execute full schema statement %q: %w", statementPreview(statement), err)
		}
	}
	return nil
}

func replaceSetVariable(sqlText string, name string, value string) string {
	line := fmt.Sprintf("SET @%s = '%s';", name, escapeSQLString(value))
	scanner := bufio.NewScanner(strings.NewReader(sqlText))
	var builder strings.Builder
	for scanner.Scan() {
		current := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(current), "SET @"+name+" = ") {
			builder.WriteString(line)
		} else {
			builder.WriteString(current)
		}
		builder.WriteByte('\n')
	}
	return builder.String()
}

func splitSQLWithDelimiters(content string) []string {
	delimiter := ";"
	statements := make([]string, 0)
	var builder strings.Builder

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		upper := strings.ToUpper(trimmed)
		if strings.HasPrefix(upper, "DELIMITER ") {
			delimiter = strings.TrimSpace(trimmed[len("DELIMITER "):])
			continue
		}

		builder.WriteString(line)
		builder.WriteByte('\n')
		statementText := strings.TrimSpace(builder.String())
		if statementText == "" {
			builder.Reset()
			continue
		}
		if strings.HasSuffix(strings.TrimSpace(statementText), delimiter) {
			statementText = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(statementText), delimiter))
			if !isEmptySQL(statementText) {
				statements = append(statements, statementText)
			}
			builder.Reset()
		}
	}

	rest := strings.TrimSpace(builder.String())
	if !isEmptySQL(rest) {
		statements = append(statements, rest)
	}
	return statements
}

func isEmptySQL(statement string) bool {
	statement = strings.TrimSpace(statement)
	if statement == "" {
		return true
	}
	for _, line := range strings.Split(statement, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "--") {
			return false
		}
	}
	return true
}

func statementPreview(statement string) string {
	statement = strings.Join(strings.Fields(statement), " ")
	if len(statement) > 120 {
		return statement[:120] + "..."
	}
	return statement
}

func envWithDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func escapeIdentifier(value string) string {
	return strings.ReplaceAll(value, "`", "``")
}

func escapeSQLString(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}
