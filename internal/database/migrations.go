package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

const migrationTable = "schema_migrations"
const migrationLockName = "CONCAT('lms:', LEFT(SHA2(DATABASE(), 256), 60))"

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
	return db.WithContext(ctx).Connection(func(conn *gorm.DB) (result error) {
		// MySQL DDL can commit implicitly, so a transaction or row lock
		// cannot serialize the complete migration across application replicas.
		acquireCtx, cancel := context.WithTimeout(ctx, 35*time.Second)
		defer cancel()
		var acquired sql.NullInt64
		err := conn.WithContext(acquireCtx).Raw("SELECT GET_LOCK(" + migrationLockName + ", 30)").Row().Scan(&acquired)
		if err != nil {
			discardMigrationConnection(conn)
			return fmt.Errorf("acquire migration lock: %w", err)
		}
		if !acquired.Valid || acquired.Int64 != 1 {
			return errors.New("could not acquire migration lock")
		}
		defer func() {
			releaseCtx, cancelRelease := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancelRelease()
			var released sql.NullInt64
			err := conn.WithContext(releaseCtx).Raw("SELECT RELEASE_LOCK(" + migrationLockName + ")").Row().Scan(&released)
			if err != nil || !released.Valid || released.Int64 != 1 {
				discardMigrationConnection(conn)
				result = errors.Join(result, errors.New("could not release migration lock"))
			}
		}()
		return applyMigrations(conn.Session(&gorm.Session{NewDB: true}), directory, files)
	})
}

func discardMigrationConnection(db *gorm.DB) {
	if conn, ok := db.Statement.ConnPool.(*sql.Conn); ok {
		_ = conn.Raw(func(any) error { return driver.ErrBadConn })
	}
}

func applyMigrations(db *gorm.DB, directory string, files []string) error {
	var serverVersion string
	if err := db.Raw("SELECT VERSION()").Scan(&serverVersion).Error; err != nil {
		return fmt.Errorf("read database engine: %w", err)
	}

	if err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) NOT NULL PRIMARY KEY,
		applied_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`).Error; err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}

	for _, filename := range files {
		var applied int64
		if err := db.Table(migrationTable).Where("version = ?", filename).Count(&applied).Error; err != nil {
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

		if err := db.Transaction(func(tx *gorm.DB) error {
			for index, statement := range statements {
				statement = migrationForEngine(statement, serverVersion)
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

func migrationForEngine(statement, version string) string {
	if strings.Contains(strings.ToLower(version), "mariadb") {
		return strings.ReplaceAll(statement, "DROP CHECK chk_users_role", "DROP CONSTRAINT chk_users_role")
	}
	return statement
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
