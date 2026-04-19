package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func main() {
	direction := flag.String("direction", "up", "Migration direction: up or down")
	flag.Parse()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required\n" +
			"Example: DATABASE_URL=postgres://postgres:password@localhost:5432/member_onboarding?sslmode=disable")
	}

	// Resolve the migrations directory relative to the current working directory.
	// os.DirFS takes an ordinary OS path, so Windows paths work without any
	// file:// URL construction.
	migrationsDir, err := filepath.Abs("db/migrations")
	if err != nil {
		log.Fatalf("Failed to resolve migrations path: %v", err)
	}

	fsys := os.DirFS(migrationsDir)
	d, err := iofs.New(fsys, ".")
	if err != nil {
		log.Fatalf("Failed to load migration files from %s: %v", migrationsDir, err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, databaseURL)
	if err != nil {
		log.Fatalf("Failed to initialise migrator: %v", err)
	}
	defer m.Close()

	switch *direction {
	case "up":
		if err := m.Up(); err != nil {
			if err == migrate.ErrNoChange {
				fmt.Println("No new migrations to apply")
				return
			}
			log.Fatalf("migrate up failed: %v", err)
		}
		fmt.Println("Migrations applied successfully")
	case "down":
		if err := m.Steps(-1); err != nil {
			log.Fatalf("migrate down failed: %v", err)
		}
		fmt.Println("Migration rolled back successfully")
	default:
		log.Fatalf("Unknown direction %q — use 'up' or 'down'", *direction)
	}
}
