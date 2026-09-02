package database

import (
	"database/sql"
	"fmt"

	"lms-cn-api/internal/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Connection struct {
	GORM *gorm.DB
	SQL  *sql.DB
}

func Open(cfg *config.Config) (*Connection, error) {
	logLevel := logger.Warn
	if cfg.AppEnv == "production" {
		logLevel = logger.Error
	}

	db, err := gorm.Open(mysql.Open(cfg.DatabaseURL), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logLevel),
		DisableForeignKeyConstraintWhenMigrating: true,
		TranslateError:                           true,
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get database handle: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.DatabaseMaxOpen)
	sqlDB.SetMaxIdleConns(cfg.DatabaseMaxIdle)
	sqlDB.SetConnMaxLifetime(cfg.DatabaseMaxLife)

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Connection{GORM: db, SQL: sqlDB}, nil
}

func (c *Connection) Close() error {
	if c == nil || c.SQL == nil {
		return nil
	}
	return c.SQL.Close()
}
