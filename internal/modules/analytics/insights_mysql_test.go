package analytics

import (
	"context"
	"database/sql"
	"net"
	"os"
	"testing"
	"time"

	"lms-cn-api/internal/authz"
	"lms-cn-api/internal/config"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Opt-in validation against the existing local schema, with no fixtures or writes.
func TestInsightsLocalMySQLReadOnly(t *testing.T) {
	if os.Getenv("LMS_ANALYTICS_READONLY_CHECK") != "1" {
		t.Skip("local read-only MySQL check is opt-in")
	}
	// go test starts in the package directory; load configuration from repository root.
	t.Chdir("../../..")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal("local development configuration is unavailable")
	}
	dsn, err := mysqlDriver.ParseDSN(cfg.DatabaseURL)
	if err != nil {
		t.Fatal("local database configuration is invalid")
	}
	host, _, err := net.SplitHostPort(dsn.Addr)
	if err != nil || dsn.Net != "tcp" || (host != "127.0.0.1" && host != "localhost" && host != "::1") || (cfg.AppEnv != "development" && cfg.AppEnv != "test") {
		t.Fatal("this check requires a loopback MySQL connection in development or test")
	}
	pool, err := sql.Open("mysql", cfg.DatabaseURL)
	if err != nil {
		t.Fatal("cannot initialize local MySQL connection")
	}
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	tx, err := pool.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		t.Fatal("local MySQL read-only transaction is unavailable")
	}
	defer tx.Rollback()
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: tx, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("cannot initialize read-only query adapter")
	}
	service := NewService(NewRepository(db), nil)
	for _, role := range []string{"admin", "teacher", "student"} {
		t.Run(role, func(t *testing.T) {
			result, err := service.DashboardInsights(ctx, authz.Principal{UserID: uuid.NewString(), Role: role})
			if err != nil {
				t.Fatal("dashboard aggregate queries failed against local schema")
			}
			if len(result.Activity) != 14 || len(result.Metrics) != 3 {
				t.Fatal("unexpected aggregate contract")
			}
			if role != "admin" {
				for _, metric := range result.Metrics {
					if metric.Value != 0 {
						t.Fatal("an identity without membership received aggregate data")
					}
				}
				for _, day := range result.Activity {
					if day.ExamSubmissions != 0 || day.AssignmentSubmissions != 0 {
						t.Fatal("an unrelated identity received activity data")
					}
				}
				for _, item := range result.Distribution {
					if item.Value != 0 {
						t.Fatal("an unrelated identity received distribution data")
					}
				}
				if len(result.RecentResults) != 0 {
					t.Fatal("an unrelated identity received result data")
				}
				if result.LearningProgress != nil && (result.LearningProgress.Completed != 0 || result.LearningProgress.Total != 0) {
					t.Fatal("an unrelated identity received material progress")
				}
			}
		})
	}
}
