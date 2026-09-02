package database

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/gorm"
)

const migrationTable = "schema_migrations"

// RunMigrations applies pending versioned SQL migrations in filename order.
// Each version is recorded only after all statements in that file succeed.
func RunMigrations(ctx context.Context, db *gorm.DB, directory string) error {
	if db == nil {
		return errors.New("cannot run migrations with a nil database")
	}
	if strings.TrimSpace(directory) == "" {
		return errors.New("migration directory is required")
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("read migration directory: %w", err)
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)

	if err := db.WithContext(ctx).Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) NOT NULL PRIMARY KEY,
		applied_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`).Error; err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}

	for _, filename := range files {
		var applied int64
		if err := db.WithContext(ctx).Table(migrationTable).Where("version = ?", filename).Count(&applied).Error; err != nil {
			return fmt.Errorf("check migration %s: %w", filename, err)
		}
		if applied > 0 {
			continue
		}

		path := filepath.Join(directory, filename)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", filename, err)
		}
		statements := splitMigrationStatements(string(content))
		if len(statements) == 0 {
			return fmt.Errorf("migration %s contains no SQL statements", filename)
		}

		if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			for index, statement := range statements {
				if err := tx.Exec(statement).Error; err != nil {
					return fmt.Errorf("execute statement %d: %w", index+1, err)
				}
			}
			return tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", filename).Error
		}); err != nil {
			return fmt.Errorf("apply migration %s: %w", filename, err)
		}
	}
	return nil
}

func splitMigrationStatements(content string) []string {
	parts := strings.Split(content, ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		statement := strings.TrimSpace(part)
		if statement != "" {
			statements = append(statements, statement)
		}
	}
	return statements
}
