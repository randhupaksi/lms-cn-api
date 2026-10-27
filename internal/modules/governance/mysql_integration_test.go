package governance

import (
	"context"
	"crypto/sha256"
	"fmt"
	driver "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"lms-cn-api/internal/authz"
	"lms-cn-api/internal/modules/academics"
	"lms-cn-api/internal/modules/audit"
	"lms-cn-api/internal/modules/auth"
	"lms-cn-api/internal/modules/exams"
	"lms-cn-api/internal/modules/users"
	"lms-cn-api/pkg/apperror"
	"os"
	"strings"
	"testing"
	"time"
)

// Opt in against a migrated, seeded, disposable local database. All writes roll back.
func TestGovernanceMySQLTransactions(t *testing.T) {
	if os.Getenv("GOVERNANCE_MYSQL_TEST") != "1" {
		t.Skip("requires explicit disposable local MySQL fixture")
	}
	dsn := os.Getenv("GOVERNANCE_TEST_DATABASE_URL")
	parsed, err := driver.ParseDSN(dsn)
	if err != nil || !(strings.HasPrefix(parsed.Addr, "127.0.0.1:") || strings.HasPrefix(parsed.Addr, "localhost:")) || !(strings.HasSuffix(parsed.DBName, "_test") || parsed.DBName == "lms_governance_demo") {
		t.Fatal("disposable local test database required")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), TranslateError: true})
	if err != nil {
		t.Fatal("cannot open fixture")
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal("cannot start fixture transaction")
	}
	defer tx.Rollback()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	password := "Synthetic-test-only-2026"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	makeUser := func(role users.Role) users.User {
		u := users.User{ID: uuid.NewString(), Identifier: "fixture." + uuid.NewString(), FullName: "Fictional Governance Fixture", Role: role, Status: users.StatusActive, PasswordHash: string(hash)}
		if err := tx.Create(&u).Error; err != nil {
			t.Fatal("user fixture failed")
		}
		return u
	}
	ownerUser := makeUser(users.RoleSuperadmin)
	teacher := makeUser(users.RoleTeacher)
	target := makeUser(users.RoleAdmin)
	actor := authz.Principal{UserID: ownerUser.ID, Role: "superadmin"}
	teacherActor := authz.Principal{UserID: teacher.ID, Role: "teacher"}
	var course academics.Course
	if err := tx.First(&course).Error; err != nil {
		t.Fatal("seeded course required")
	}
	if err := tx.Create(&academics.CourseTeacher{CourseID: course.ID, TeacherID: teacher.ID, AssignedAt: now}).Error; err != nil {
		t.Fatal("membership fixture failed")
	}
	exam := exams.Exam{ID: uuid.NewString(), CourseID: course.ID, AuthorID: teacher.ID, Title: "Fictional governance transaction check", Status: "draft", StartsAt: now.Add(24 * time.Hour), EndsAt: now.Add(26 * time.Hour), DurationMinutes: 60, MaxAttempts: 1, ResultPolicy: "after_publish"}
	if err := tx.Create(&exam).Error; err != nil {
		t.Fatal("exam fixture failed")
	}
	repo := NewRepository(tx)
	school := NewService(repo, academics.NewService(academics.NewRepository(tx), audit.NewService(tx)))
	if _, err := school.Overview(ctx, actor); err != nil {
		t.Fatalf("overview query: %v", err)
	}
	if _, _, err := school.Reports(ctx, actor, "", "", 1, 100); err != nil {
		t.Fatalf("report query: %v", err)
	}
	input := ScheduleInput{ExamID: exam.ID, Reason: "A synthetic schedule change used only for isolated verification.", ProposedStartsAt: now.Add(48 * time.Hour), ProposedEndsAt: now.Add(50 * time.Hour)}
	if err := school.CreateRequest(ctx, teacherActor, input); err != nil {
		t.Fatal(err)
	}
	assertCode := func(err error, code string) {
		t.Helper()
		app, ok := apperror.As(err)
		if !ok || app.Code != code {
			t.Fatalf("wanted %s", code)
		}
	}
	assertCode(school.CreateRequest(ctx, teacherActor, input), "PENDING_REQUEST_EXISTS")
	var request ScheduleRequest
	if err := tx.Where("exam_id=?", exam.ID).First(&request).Error; err != nil {
		t.Fatal(err)
	}
	decision := DecisionInput{Decision: "approved", Note: "Synthetic verification approved after checking the proposed window."}
	if err := school.Decide(ctx, actor, request.ID, decision); err != nil {
		t.Fatal(err)
	}
	assertCode(school.Decide(ctx, actor, request.ID, decision), "REQUEST_ALREADY_DECIDED")
	var changed exams.Exam
	if err := tx.First(&changed, "id=?", exam.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !changed.StartsAt.Equal(input.ProposedStartsAt) {
		t.Fatal("approval did not update exam atomically")
	}
	settings, err := school.Settings(ctx, actor)
	if err != nil {
		t.Fatal(err)
	}
	settingsInput := SettingsInput{SchoolName: settings.SchoolName, Motto: settings.Motto, ContactEmail: settings.ContactEmail, SupportMessage: settings.SupportMessage, PassingReference: 70, Version: settings.Version, Reason: "Synthetic settings transaction verification."}
	if err := school.SaveSettings(ctx, actor, settingsInput); err != nil {
		t.Fatal(err)
	}
	assertCode(school.SaveSettings(ctx, actor, settingsInput), "STALE_SETTINGS")
	history, err := school.History(ctx, actor)
	if err != nil || len(history) == 0 {
		t.Fatal("settings history absent")
	}
	eventInput := EventInput{Title: "Fictional school agenda", Category: "meeting", StartsAt: now.Add(24 * time.Hour), EndsAt: now.Add(25 * time.Hour)}
	if err := school.SaveEvent(ctx, actor, "", eventInput); err != nil {
		t.Fatal(err)
	}
	var event Event
	if err := tx.Where("created_by=?", actor.UserID).First(&event).Error; err != nil {
		t.Fatal(err)
	}
	eventInput.Version = event.Version
	if err := school.SaveEvent(ctx, actor, event.ID, eventInput); err != nil {
		t.Fatal(err)
	}
	assertCode(school.SaveEvent(ctx, actor, event.ID, eventInput), "STALE_EVENT")
	if err := school.ArchiveEvent(ctx, actor, event.ID, 2); err != nil {
		t.Fatal(err)
	}
	userService := users.NewService(users.NewRepository(tx))
	_, err = userService.Update(ctx, authz.Principal{Role: "admin", UserID: target.ID}, ownerUser.ID, users.UpdateRequest{Status: users.StatusInactive})
	assertCode(err, "PRIVILEGED_ACCOUNT_PROTECTED")
	roleInput := users.ChangeRoleRequest{Role: users.RoleSuperadmin, Reason: "Synthetic administrative role verification.", CurrentPassword: password}
	digest := sha256.Sum256([]byte(uuid.NewString()))
	session := auth.Session{ID: uuid.NewString(), UserID: target.ID, RefreshTokenHash: fmt.Sprintf("%x", digest), ExpiresAt: now.Add(time.Hour), LastUsedAt: now, CreatedAt: now}
	if err := tx.Create(&session).Error; err != nil {
		t.Fatal("session fixture failed")
	}
	if err := userService.ChangeRole(ctx, actor, target.ID, roleInput); err != nil {
		t.Fatal(err)
	}
	sessions, err := userService.Sessions(ctx, actor, target.ID)
	if err != nil || len(sessions) != 0 {
		t.Fatal("role change did not revoke sessions")
	}
	secondDigest := sha256.Sum256([]byte(uuid.NewString()))
	newSession := auth.Session{ID: uuid.NewString(), UserID: target.ID, RefreshTokenHash: fmt.Sprintf("%x", secondDigest), ExpiresAt: now.Add(time.Hour), LastUsedAt: now, CreatedAt: now}
	if err := tx.Create(&newSession).Error; err != nil {
		t.Fatal("session comparison fixture failed")
	}
	authService := auth.NewService(auth.NewRepository(tx), users.NewRepository(tx), nil, audit.NewService(tx), time.Hour, 24*time.Hour, nil)
	if err := authService.ValidateSession(ctx, authz.Principal{UserID: target.ID, SessionID: newSession.ID, Role: "admin"}); err == nil {
		t.Fatal("stale token role admitted")
	}
	if err := authService.ValidateSession(ctx, authz.Principal{UserID: target.ID, SessionID: newSession.ID, Role: "superadmin"}); err != nil {
		t.Fatal("current token role denied")
	}
	assertCode(userService.ChangeRole(ctx, actor, teacher.ID, roleInput), "ACADEMIC_ROLE_LOCKED")
	var auditCount int64
	tx.Table("audit_logs").Where("actor_id=?", actor.UserID).Count(&auditCount)
	if auditCount < 5 {
		t.Fatal("sensitive changes missing audit")
	}
	_, err = userService.Update(ctx, actor, actor.UserID, users.UpdateRequest{Status: users.StatusInactive})
	assertCode(err, "SELF_DEACTIVATION_DENIED")
	if err := tx.Model(&users.User{}).Where("role=? AND id<>?", users.RoleSuperadmin, target.ID).Update("status", users.StatusInactive).Error; err != nil {
		t.Fatal("last-account fixture failed")
	}
	_, err = userService.Update(ctx, actor, target.ID, users.UpdateRequest{Status: users.StatusInactive})
	assertCode(err, "LAST_SUPERADMIN_PROTECTED")
}
