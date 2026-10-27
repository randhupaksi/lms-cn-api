package database

import (
	"context"
	driver "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"lms-cn-api/internal/modules/governance"
	"lms-cn-api/internal/modules/users"
	"os"
	"strings"
	"testing"
	"time"
)

func TestDemoRestartPreservesGovernance(t *testing.T) {
	if os.Getenv("GOVERNANCE_SEED_TEST") != "1" {
		t.Skip("requires explicit disposable local database")
	}
	dsn := os.Getenv("GOVERNANCE_TEST_DATABASE_URL")
	parsed, err := driver.ParseDSN(dsn)
	if err != nil || !(strings.HasPrefix(parsed.Addr, "127.0.0.1:") || strings.HasPrefix(parsed.Addr, "localhost:")) || !(strings.HasSuffix(parsed.DBName, "_test") || parsed.DBName == "lms_governance_demo") {
		t.Fatal("disposable local database required")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), TranslateError: true})
	if err != nil {
		t.Fatal("fixture unavailable")
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	tx := db.Begin()
	defer tx.Rollback()
	var beforeCount int64
	tx.Model(&users.User{}).Count(&beforeCount)
	hash, _ := bcrypt.GenerateFromPassword([]byte("Synthetic-restart-fixture"), bcrypt.MinCost)
	if err := tx.Model(&users.User{}).Where("id=?", seedID(1)).Updates(map[string]any{"identifier": "fixture.renamed.operator", "full_name": "Fictional Renamed Operator", "status": "inactive", "role": "teacher", "password_hash": string(hash), "must_change_password": true}).Error; err != nil {
		t.Fatal("account fixture failed")
	}
	var request governance.ScheduleRequest
	if err := tx.First(&request, "id=?", seedID(900100)).Error; err != nil {
		t.Fatal("seeded request required")
	}
	start := time.Now().UTC().AddDate(0, 0, 20).Truncate(time.Second)
	end := start.Add(2 * time.Hour)
	if err := tx.Model(&request).Updates(map[string]any{"status": "approved", "proposed_starts_at": start, "proposed_ends_at": end, "decided_at": time.Now().UTC()}).Error; err != nil {
		t.Fatal("schedule fixture failed")
	}
	if err := tx.Model(&governance.Event{}).Where("id=?", seedID(900001)).Updates(map[string]any{"version": 2, "title": "Fictional edited agenda", "starts_at": start, "ends_at": end}).Error; err != nil {
		t.Fatal("agenda fixture failed")
	}
	if err := SeedDemoData(context.Background(), tx); err != nil {
		t.Fatal("restart seed failed")
	}
	var account users.User
	if err := tx.First(&account, "id=?", seedID(1)).Error; err != nil {
		t.Fatal("account disappeared")
	}
	if account.Identifier != "fixture.renamed.operator" || account.Status != users.StatusInactive || account.Role != users.RoleTeacher || account.PasswordHash != string(hash) || !account.MustChangePassword {
		t.Fatal("governed account was reset")
	}
	var schedule struct{ StartsAt time.Time }
	if err := tx.Table("exams").Where("id=?", request.ExamID).Take(&schedule).Error; err != nil || !schedule.StartsAt.Equal(start) {
		t.Fatal("approved schedule was reset")
	}
	var event governance.Event
	if err := tx.First(&event, "id=?", seedID(900001)).Error; err != nil || event.Version != 2 || event.Title != "Fictional edited agenda" || !event.StartsAt.Equal(start) {
		t.Fatal("edited agenda was reset")
	}
	var afterCount int64
	tx.Model(&users.User{}).Count(&afterCount)
	if beforeCount != afterCount {
		t.Fatal("restart duplicated accounts")
	}
}
