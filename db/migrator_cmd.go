package migrator

import (
	"database/sql"
	"embed"
	"fmt"
	pool "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbinit"
	"log"
	"os"
)

const migrationsDir = "migrations"

//go:embed migrations/*.sql
var migrationsFS embed.FS

func ApplyMigrations(versionStr string) {
	migrator, err := mustGetNewMigrator(migrationsFS, migrationsDir)
	if err != nil {
		log.Fatal("Failed to initialize migrator: ", err)
	}

	postgresURL := pool.NewConnectionString(
		os.Getenv("POSTGRES_USERNAME"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_DB"),
	)
	connectionStr := fmt.Sprintf("%s?sslmode=disable", postgresURL)

	conn, err := sql.Open("postgres", connectionStr)
	if err != nil {
		log.Fatal("Unable to connect to database for migrations: ", err)
	}
	defer conn.Close()

	err = migrator.applyMigrations(conn, versionStr)
	if err != nil {
		log.Fatal("Failed to apply migrations: ", err)
	}

	if versionStr == "latest" {
		log.Println("All migrations have been applied")
	} else {
		log.Printf("Migration to version %s has been applied", versionStr)
	}
}
