package db

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"

	"gorm.io/gorm"

	"trade-buddy/api/migrations"
)

// Migrate runs all *_up.sql migration files in order (skips *_down.sql).
func Migrate(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get underlying sql.DB: %w", err)
	}
	if _, err := sqlDB.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		filename TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}

	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") || strings.HasSuffix(name, "_down.sql") {
			continue
		}
		applied, err := migrationApplied(sqlDB, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		content, err := migrations.FS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if _, err := sqlDB.Exec(string(content)); err != nil {
			if schemaAlreadyPresent(sqlDB) {
				if recordErr := recordMigration(sqlDB, name); recordErr != nil {
					return recordErr
				}
				slog.Info("migration marked applied for existing schema", "file", name)
				continue
			}
			return fmt.Errorf("exec migration %s: %w", name, err)
		}
		if err := recordMigration(sqlDB, name); err != nil {
			return err
		}
		slog.Info("migration applied", "file", name)
	}
	return nil
}

func migrationApplied(db *sql.DB, name string) (bool, error) {
	var applied bool
	err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE filename = $1)`, name).Scan(&applied)
	if err != nil {
		return false, fmt.Errorf("check migration %s: %w", name, err)
	}
	return applied, nil
}

func recordMigration(db *sql.DB, name string) error {
	if _, err := db.Exec(`INSERT INTO schema_migrations (filename) VALUES ($1) ON CONFLICT (filename) DO NOTHING`, name); err != nil {
		return fmt.Errorf("record migration %s: %w", name, err)
	}
	return nil
}

func schemaAlreadyPresent(db *sql.DB) bool {
	var exists bool
	err := db.QueryRow(`SELECT to_regclass('public.market_bars') IS NOT NULL`).Scan(&exists)
	return err == nil && exists
}
