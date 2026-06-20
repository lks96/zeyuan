package main

import (
	"context"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"temu-tools/backend/internal/config"
	"temu-tools/backend/internal/database"
)

func main() {
	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := database.Migrate(ctx, cfg, "migrations"); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	log.Println("migration completed")
}
