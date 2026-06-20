package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

type Config struct {
	Port           string
	AppSecret      string
	DBHost         string
	DBPort         string
	DBName         string
	DBUser         string
	DBPassword     string
	DevUserID      string
	DevAuthEnabled bool
}

func Load() Config {
	loadDotEnv(".env")

	return Config{
		Port:           env("PORT", "8080"),
		AppSecret:      env("APP_SECRET", "dev-secret-change-me"),
		DBHost:         env("DB_HOST", "127.0.0.1"),
		DBPort:         env("DB_PORT", "3306"),
		DBName:         env("DB_NAME", "temu_tools"),
		DBUser:         env("DB_USER", "root"),
		DBPassword:     env("DB_PASSWORD", ""),
		DevUserID:      env("DEV_USER_ID", "1"),
		DevAuthEnabled: envBool("DEV_AUTH_ENABLED", false),
	}
}

func (c Config) MySQLDSN() string {
	return c.mysqlDSN(c.DBName)
}

func (c Config) MySQLServerDSN() string {
	return c.mysqlDSN("")
}

func (c Config) mysqlDSN(dbName string) string {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		location = time.Local
	}

	cfg := mysql.NewConfig()
	cfg.User = c.DBUser
	cfg.Passwd = c.DBPassword
	cfg.Net = "tcp"
	cfg.Addr = c.DBHost + ":" + c.DBPort
	cfg.DBName = dbName
	cfg.ParseTime = true
	cfg.Loc = location
	cfg.Params = map[string]string{
		"charset":   "utf8mb4",
		"collation": "utf8mb4_unicode_ci",
	}

	return cfg.FormatDSN()
}

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" || os.Getenv(key) != "" {
			continue
		}

		_ = os.Setenv(key, value)
	}
}

func env(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
