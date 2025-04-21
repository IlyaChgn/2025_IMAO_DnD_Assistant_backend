package migrator

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"strconv"

	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

type Migrator struct {
	srcDriver source.Driver
}

func mustGetNewMigrator(sqlFiles embed.FS, dirName string) (*Migrator, error) {
	driver, err := iofs.New(sqlFiles, dirName)
	if err != nil {
		return nil, err
	}

	return &Migrator{
		srcDriver: driver,
	}, nil
}

func (m *Migrator) applyMigrations(db *sql.DB, versionStr string) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	migrator, err := migrate.NewWithInstance("migration_embedded_sql_files", m.srcDriver,
		"psql_db", driver)
	if err != nil {
		return err
	}
	defer migrator.Close()

	if versionStr == "latest" {
		err = migrator.Up()
	} else {
		version, convErr := strconv.ParseUint(versionStr, 10, 64)
		if convErr != nil {
			return fmt.Errorf("invalid migration version: %s", versionStr)
		}

		err = migrator.Migrate(uint(version))
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}
