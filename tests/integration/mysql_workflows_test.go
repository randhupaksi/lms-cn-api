package integration_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	driver "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"lms-cn-api/internal/authz"
	"lms-cn-api/internal/config"
	"lms-cn-api/internal/database"
	"lms-cn-api/internal/modules/academics"
	"lms-cn-api/internal/modules/assignments"
	"lms-cn-api/internal/modules/attempts"
	"lms-cn-api/internal/modules/audit"
	"lms-cn-api/internal/modules/exams"
	"lms-cn-api/internal/modules/grading"
	"lms-cn-api/internal/modules/materials"
	"lms-cn-api/internal/modules/questions"
	"lms-cn-api/internal/modules/results"
	"lms-cn-api/internal/modules/users"
	"lms-cn-api/internal/router"
	"lms-cn-api/pkg/apperror"

	"golang.org/x/crypto/bcrypt"
)

func TestMySQLSchoolWorkflows(t *testing.T) {
	dsn, ok := os.LookupEnv("LMS_INTEGRATION_TEST_DSN")
	if !ok {
		t.Skip("requires an isolated local database")
	}
	parsed, err := driver.ParseDSN(dsn)
	if err != nil || parsed.Net != "tcp" || !strings.HasPrefix(parsed.Addr, "127.0.0.1:") || !strings.HasSuffix(parsed.DBName, "_test") {
		t.Fatal("requires a disposable loopback database ending in _test")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("cannot open isolated database")
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal("cannot access isolated database pool")
	}
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	migrations := filepath.Join("..", "..", "migrations")
	var baselineYearID string
	if !db.Migrator().HasTable("schema_migrations") {
		priorMigrations, err := copyPriorMigrations(t, migrations)
		if err != nil {
			t.Fatalf("prepare prior migration set: %v", err)
		}
		if err := database.RunMigrations(ctx, db, priorMigrations); err != nil {
			t.Fatalf("apply schema through migration 000008: %v", err)
		}
		baselineYearID = uuid.NewString()
		baselineYear := academics.AcademicYear{
			ID: baselineYearID, Name: "Prior schema fixture", Status: "active",
			StartsOn: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
			EndsOn:   time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC),
		}
		if err := db.WithContext(ctx).Create(&baselineYear).Error; err != nil {
			t.Fatalf("create record on prior schema: %v", err)
		}
	}
	migrated := make(chan error, 2)
	begin := make(chan struct{})
	for range 2 {
		go func() {
			<-begin
			migrated <- database.RunMigrations(ctx, db, migrations)
		}()
	}
	close(begin)
	for range 2 {
		if err := <-migrated; err != nil {
			t.Errorf("concurrent migration: %v", err)
		}
	}
	if t.Failed() {
		return
	}
	if baselineYearID != "" {
		if !db.Migrator().HasTable(&materials.Attachment{}) {
			t.Fatal("migration 000009 did not create the material attachment table")
		}
		var preserved academics.AcademicYear
		if err := db.WithContext(ctx).First(&preserved, "id = ?", baselineYearID).Error; err != nil {
			t.Fatalf("migration 000009 did not preserve existing academic data: %v", err)
		}
	}
	if err := database.RunMigrations(ctx, db, migrations); err != nil {
		t.Fatalf("repeat migrations: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	unique := uuid.NewString()[:12]
	const studentPassword = "Synthetic-workflow-only-2026"
	studentHash, err := bcrypt.GenerateFromPassword([]byte(studentPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal("cannot hash disposable student password")
	}
	teacher := users.User{ID: uuid.NewString(), Identifier: "teacher." + unique, FullName: "Synthetic Teacher", Role: users.RoleTeacher, Status: users.StatusActive, PasswordHash: string(studentHash)}
	curriculum := users.User{ID: uuid.NewString(), Identifier: "curriculum." + unique, FullName: "Synthetic Curriculum Coordinator", Role: users.RoleCurriculum, Status: users.StatusActive, PasswordHash: string(studentHash)}
	admin := users.User{ID: uuid.NewString(), Identifier: "admin." + unique, FullName: "Synthetic Admin", Role: users.RoleAdmin, Status: users.StatusActive, PasswordHash: string(studentHash)}
	principal := users.User{ID: uuid.NewString(), Identifier: "principal." + unique, FullName: "Synthetic Principal", Role: users.RolePrincipal, Status: users.StatusActive, PasswordHash: string(studentHash)}
	superadmin := users.User{ID: uuid.NewString(), Identifier: "superadmin." + unique, FullName: "Synthetic Superadmin", Role: users.RoleSuperadmin, Status: users.StatusActive, PasswordHash: string(studentHash)}
	student := users.User{ID: uuid.NewString(), Identifier: "student." + unique, FullName: "Synthetic Student", Role: users.RoleStudent, Status: users.StatusActive, PasswordHash: string(studentHash)}
	outsider := users.User{ID: uuid.NewString(), Identifier: "outsider." + unique, FullName: "Synthetic Unenrolled Student", Role: users.RoleStudent, Status: users.StatusActive, PasswordHash: "fixture-cannot-log-in"}
	year := academics.AcademicYear{ID: uuid.NewString(), Name: "Year " + unique, Status: "active", StartsOn: now.AddDate(0, -1, 0), EndsOn: now.AddDate(0, 1, 0)}
	class := academics.ClassGroup{ID: uuid.NewString(), AcademicYearID: year.ID, Name: "Synthetic Class", GradeLevel: 12}
	subject := academics.Subject{ID: uuid.NewString(), Code: unique, Name: "Subject " + unique}
	course := academics.Course{ID: uuid.NewString(), AcademicYearID: year.ID, ClassGroupID: class.ID, SubjectID: subject.ID, Name: "Synthetic Course", Status: "active"}
	create := func(model any) {
		t.Helper()
		if err := db.WithContext(ctx).Create(model).Error; err != nil {
			t.Fatalf("create synthetic %T: %v", model, err)
		}
	}
	for _, model := range []any{&teacher, &curriculum, &admin, &principal, &superadmin, &student, &outsider, &year, &class, &subject, &course} {
		create(model)
	}
	create(&academics.CourseTeacher{CourseID: course.ID, TeacherID: teacher.ID, AssignedAt: now})
	create(&academics.CourseTeacher{CourseID: course.ID, TeacherID: curriculum.ID, AssignedAt: now})
	create(&academics.CourseStudent{CourseID: course.ID, StudentID: student.ID, EnrolledAt: now})
	studentActor := authz.Principal{UserID: student.ID, Role: string(student.Role)}
	teacherActor := authz.Principal{UserID: teacher.ID, Role: string(teacher.Role)}
	curriculumActor := authz.Principal{UserID: curriculum.ID, Role: string(curriculum.Role)}
	outsiderActor := authz.Principal{UserID: outsider.ID, Role: string(outsider.Role)}
	unassignedTeacherActor := authz.Principal{UserID: uuid.NewString(), Role: string(users.RoleTeacher)}
	unassignedCurriculumActor := authz.Principal{UserID: uuid.NewString(), Role: string(users.RoleCurriculum)}
	adminActor := authz.Principal{UserID: admin.ID, Role: string(admin.Role)}
	principalActor := authz.Principal{UserID: principal.ID, Role: string(principal.Role)}
	superadminActor := authz.Principal{UserID: superadmin.ID, Role: string(superadmin.Role)}
	auditService := audit.NewService(db)
	courseAccess := academics.NewService(academics.NewRepository(db), auditService)
	t.Run("authenticated HTTP", func(t *testing.T) {
		engine, err := router.New(&config.Config{
			APIPrefix: "/api/v1", AppName: "Synthetic fixture",
			JWTSecret: uuid.NewString(), AllowedOrigins: []string{"https://synthetic.example"},
			AccessTokenTTL: time.Minute, RefreshTokenTTL: time.Hour,
			LoginRateLimit: 10, LoginRateWindow: time.Minute,
			LoginAccountRateLimit: 10, LoginAccountRateWindow: time.Minute,
		}, db)
		if err != nil {
			t.Fatal(err)
		}
		verifyRoleRouteMatrix(t, ctx, engine, studentPassword, []roleAccount{
			{identifier: superadmin.Identifier, role: string(superadmin.Role)},
			{identifier: principal.Identifier, role: string(principal.Role)},
			{identifier: curriculum.Identifier, role: string(curriculum.Role)},
			{identifier: admin.Identifier, role: string(admin.Role)},
			{identifier: teacher.Identifier, role: string(teacher.Role)},
			{identifier: student.Identifier, role: string(student.Role)},
		})
		verifyAuthenticatedHTTP(t, ctx, engine, student.Identifier, studentPassword)
	})
	t.Run("materials", func(t *testing.T) {
		verifyMaterialFlow(t, ctx, materials.NewService(materials.NewRepository(db), courseAccess, auditService), course.ID, teacherActor, studentActor, outsiderActor)
	})
	t.Run("assignments", func(t *testing.T) {
		verifyAssignmentFlow(t, ctx, assignments.NewService(assignments.NewRepository(db), courseAccess, auditService), course.ID, teacherActor, studentActor, outsiderActor)
	})

	createExam := func(title string) string {
		exam := exams.Exam{ID: uuid.NewString(), CourseID: course.ID, AuthorID: teacher.ID, Title: title, Status: exams.StatusPublished, StartsAt: now.Add(-time.Minute), EndsAt: now.Add(time.Hour), DurationMinutes: 30, MaxAttempts: 1, ResultPolicy: "after_publish"}
		create(&exam)
		create(&exams.Participant{ExamID: exam.ID, StudentID: student.ID, AssignedAt: now})
		return exam.ID
	}
	repo := attempts.NewRepository(db)
	type result struct {
		id  string
		err error
	}
	examID := createExam("Synthetic concurrency exam")
	secondExamID := createExam("Synthetic key conflict exam")
	t.Run("role and course ownership matrix", func(t *testing.T) {
		verifyRoleAndCourseOwnershipMatrix(t, ctx, courseAccess, course.ID, examID, teacherActor, curriculumActor, studentActor, outsiderActor, unassignedTeacherActor, unassignedCurriculumActor, adminActor, principalActor, superadminActor, auditService, db)
	})
	source := questions.Question{ID: uuid.NewString(), CourseID: course.ID, AuthorID: teacher.ID, Type: questions.TypeSingleChoice, Stem: "Synthetic question", DefaultPoints: 1, Status: "active", Version: 1}
	create(&source)
	sourceOption := questions.Option{ID: uuid.NewString(), QuestionID: source.ID, Content: "Synthetic option", IsCorrect: true, Position: 1}
	create(&sourceOption)
	question := exams.ExamQuestion{ID: uuid.NewString(), ExamID: examID, SourceQuestionID: source.ID, SourceVersion: source.Version, Type: source.Type, Stem: source.Stem, Position: 1, Points: 1}
	create(&question)
	option := exams.ExamQuestionOption{ID: uuid.NewString(), ExamQuestionID: question.ID, SourceOptionID: sourceOption.ID, Content: sourceOption.Content, IsCorrect: true, Position: 1}
	create(&option)
	const callers = 16
	ready := make(chan struct{}, callers)
	start := make(chan struct{})
	startResults := make(chan result, callers)
	for i := range callers {
		go func() {
			ready <- struct{}{}
			<-start
			attempt, _, err := repo.Start(ctx, examID, student.ID, fmt.Sprintf("fixture-key-%d", i%2), now)
			startResults <- result{attempt.ID, err}
		}()
	}
	for range callers {
		<-ready
	}
	close(start)
	var id string
	for range callers {
		got := <-startResults
		if got.err != nil || got.id == "" {
			t.Errorf("start returned id=%q, error=%v", got.id, got.err)
		} else if id == "" {
			id = got.id
		} else if id != got.id {
			t.Errorf("start returned a different attempt: %q", got.id)
		}
	}
	if t.Failed() {
		return
	}
	var stored attempts.Attempt
	if err := db.WithContext(ctx).First(&stored, "id = ?", id).Error; err != nil {
		t.Fatalf("read stored attempt: %v", err)
	}
	var events int64
	if err := db.WithContext(ctx).Model(&attempts.Event{}).Where("attempt_id = ? AND event_type = ?", id, "attempt.started").Count(&events).Error; err != nil || events != 1 {
		t.Fatalf("expected one start event, got %d, error=%v", events, err)
	}
	if _, _, err := repo.Start(ctx, secondExamID, student.ID, stored.StartIdempotencyKey, now); !errors.Is(err, attempts.ErrIdempotencyConflict) {
		t.Fatalf("expected key conflict across exams, got %v", err)
	}

	service := attempts.NewService(repo, grading.DefaultCalculator{})
	answer := attempts.SaveAnswerRequest{ExamQuestionID: question.ID, SelectedOptionID: option.ID}
	saved, err := service.SaveAnswer(ctx, studentActor, id, "fixture-answer", answer)
	if err != nil || saved.Revision != 1 || saved.SelectedOptionID != option.ID {
		t.Fatalf("save answer: revision=%d, error=%v", saved.Revision, err)
	}
	retry, err := service.SaveAnswer(ctx, studentActor, id, "fixture-answer", answer)
	if err != nil || retry.Revision != saved.Revision {
		t.Fatalf("retry answer: revision=%d, error=%v", retry.Revision, err)
	}
	resumed, err := service.Resume(ctx, studentActor, id)
	if err != nil || len(resumed.Answers) != 1 || len(resumed.Questions) != 1 || resumed.Answers[0].SelectedOptionID != option.ID {
		t.Fatalf("resume attempt: answers=%d, questions=%d, error=%v", len(resumed.Answers), len(resumed.Questions), err)
	}
	reopened, err := service.Start(ctx, studentActor, examID, stored.StartIdempotencyKey)
	if err != nil || reopened.AttemptID != id || len(reopened.Answers) != 1 || len(reopened.Questions) != 1 {
		t.Fatalf("start retry did not hydrate saved answers: %v", err)
	}
	receipt, err := service.Submit(ctx, studentActor, id, "fixture-submit")
	if err != nil || receipt.Status != attempts.StatusSubmitted || receipt.Receipt == nil {
		t.Fatalf("submit attempt: status=%q, error=%v", receipt.Status, err)
	}
	repeated, err := service.Submit(ctx, studentActor, id, "fixture-submit")
	if err != nil || repeated.Receipt == nil || *repeated.Receipt != *receipt.Receipt {
		t.Fatalf("submit retry did not return the same receipt: %v", err)
	}
	var resultRow results.Result
	err = db.WithContext(ctx).First(&resultRow, "attempt_id = ?", id).Error
	if err != nil || resultRow.Status != "draft" || resultRow.PublishedAt != nil || resultRow.Score != 1 || resultRow.MaxScore != 1 {
		t.Fatalf("result was not graded as an unpublished draft: %v", err)
	}
	resultService := results.NewService(results.NewRepository(db), courseAccess, auditService)
	if _, err := resultService.FindStudent(ctx, studentActor, resultRow.ID); err == nil {
		t.Fatal("a draft result should be invisible to the student")
	}
	published, err := resultService.PublishByExam(ctx, teacherActor, examID)
	if err != nil || published != 1 {
		t.Fatalf("publish result: published=%d, error=%v", published, err)
	}
	if _, err := resultService.FindStudent(ctx, studentActor, resultRow.ID); err != nil {
		t.Fatalf("the owning student could not read the published result: %v", err)
	}
	if _, err := resultService.FindStudent(ctx, outsiderActor, resultRow.ID); err == nil {
		t.Fatal("a published result should be private to its student")
	}
}

func copyPriorMigrations(t *testing.T, source string) (string, error) {
	t.Helper()
	entries, err := os.ReadDir(source)
	if err != nil {
		return "", err
	}
	directory := t.TempDir()
	for _, entry := range entries {
		name := entry.Name()
		version, _, hasVersion := strings.Cut(name, "_")
		if entry.IsDir() || !hasVersion || !strings.HasSuffix(name, ".up.sql") || version >= "000009" {
			continue
		}
		content, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(directory, name), content, 0o600); err != nil {
			return "", err
		}
	}
	return directory, nil
}

func verifyRoleAndCourseOwnershipMatrix(
	t *testing.T,
	ctx context.Context,
	courseAccess *academics.Service,
	courseID string,
	examID string,
	teacher, curriculum, student, outsider, unassignedTeacher, unassignedCurriculum, admin, principal, superadmin authz.Principal,
	auditRecorder audit.Recorder,
	db *gorm.DB,
) {
	t.Helper()
	managerCases := []struct {
		name    string
		actor   authz.Principal
		allowed bool
	}{
		{name: "superadmin", actor: superadmin, allowed: true},
		{name: "principal", actor: principal, allowed: true},
		{name: "admin", actor: admin, allowed: true},
		{name: "assigned teacher", actor: teacher, allowed: true},
		{name: "assigned curriculum", actor: curriculum, allowed: true},
		{name: "enrolled student", actor: student},
		{name: "unenrolled student", actor: outsider},
		{name: "unassigned teacher", actor: unassignedTeacher},
		{name: "unassigned curriculum", actor: unassignedCurriculum},
	}
	for _, testCase := range managerCases {
		t.Run("course manager/"+testCase.name, func(t *testing.T) {
			err := courseAccess.RequireCourseManager(ctx, testCase.actor, courseID)
			if (err == nil) != testCase.allowed {
				t.Fatalf("manager access allowed=%v, error=%v", testCase.allowed, err)
			}
		})
	}
	if err := courseAccess.RequireCourseManager(ctx, teacher, uuid.NewString()); err == nil {
		t.Fatal("a teacher assigned to another course received manager access")
	}
	if err := courseAccess.RequireCourseManager(ctx, curriculum, uuid.NewString()); err == nil {
		t.Fatal("a curriculum coordinator assigned to another course received manager access")
	}
	studentCases := []struct {
		name    string
		actor   authz.Principal
		allowed bool
	}{
		{name: "superadmin", actor: superadmin},
		{name: "principal", actor: principal},
		{name: "admin", actor: admin},
		{name: "assigned teacher", actor: teacher},
		{name: "assigned curriculum", actor: curriculum},
		{name: "enrolled student", actor: student, allowed: true},
		{name: "unenrolled student", actor: outsider},
		{name: "unassigned teacher", actor: unassignedTeacher},
		{name: "unassigned curriculum", actor: unassignedCurriculum},
	}
	for _, testCase := range studentCases {
		t.Run("course student/"+testCase.name, func(t *testing.T) {
			err := courseAccess.RequireCourseStudent(ctx, testCase.actor, courseID)
			if (err == nil) != testCase.allowed {
				t.Fatalf("student access allowed=%v, error=%v", testCase.allowed, err)
			}
		})
	}

	questionService := questions.NewService(questions.NewRepository(db), courseAccess, auditRecorder)
	examService := exams.NewService(exams.NewRepository(db), courseAccess, auditRecorder)
	materialService := materials.NewService(materials.NewRepository(db), courseAccess, auditRecorder)
	assignmentService := assignments.NewService(assignments.NewRepository(db), courseAccess, auditRecorder)
	resultService := results.NewService(results.NewRepository(db), courseAccess, auditRecorder)
	deniedActors := []struct {
		name  string
		actor authz.Principal
	}{
		{name: "enrolled student", actor: student},
		{name: "unenrolled student", actor: outsider},
		{name: "admin", actor: admin},
		{name: "principal", actor: principal},
		{name: "superadmin", actor: superadmin},
		{name: "unassigned teacher", actor: unassignedTeacher},
		{name: "unassigned curriculum", actor: unassignedCurriculum},
	}
	for _, testCase := range deniedActors {
		t.Run("authoring denied/"+testCase.name, func(t *testing.T) {
			assertForbidden := func(operation string, err error) {
				t.Helper()
				appErr, ok := apperror.As(err)
				if !ok || appErr.Status != http.StatusForbidden {
					t.Fatalf("%s should be forbidden, got %v", operation, err)
				}
			}
			_, err := questionService.Create(ctx, testCase.actor, questions.WriteRequest{CourseID: courseID})
			assertForbidden("question authoring", err)
			_, err = examService.Create(ctx, testCase.actor, exams.WriteRequest{CourseID: courseID})
			assertForbidden("exam authoring", err)
			_, err = materialService.Create(ctx, testCase.actor, materials.WriteRequest{CourseID: courseID})
			assertForbidden("material authoring", err)
			_, err = assignmentService.Create(ctx, testCase.actor, assignments.WriteRequest{CourseID: courseID, DueAt: time.Now().UTC().Add(time.Hour)})
			assertForbidden("assignment authoring", err)
			_, err = resultService.PublishByExam(ctx, testCase.actor, examID)
			assertForbidden("result publication", err)
		})
	}
}

func verifyMaterialFlow(t *testing.T, ctx context.Context, service *materials.Service, courseID string, teacher, student, outsider authz.Principal) {
	t.Helper()
	material, err := service.Create(ctx, teacher, materials.WriteRequest{CourseID: courseID, Title: "Synthetic Material", Content: "Synthetic content", Position: 1})
	if err != nil {
		t.Fatal(err)
	}
	var imageData bytes.Buffer
	if err := png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	attachment, err := service.UploadAttachment(ctx, teacher, material.ID, "synthetic.png", imageData.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := service.UploadAttachment(ctx, teacher, material.ID, "repeated.png", imageData.Bytes())
	if err != nil || repeated.ID != attachment.ID {
		t.Fatalf("duplicate upload did not return the same attachment: %v", err)
	}
	if _, err := service.GetAttachment(ctx, student, material.ID, attachment.ID); err == nil {
		t.Fatal("draft attachment should be invisible to the enrolled student")
	}
	if err := service.Publish(ctx, teacher, material.ID); err != nil {
		t.Fatal(err)
	}
	file, err := service.GetAttachment(ctx, student, material.ID, attachment.ID)
	if err != nil || !bytes.Equal(file.FileData, imageData.Bytes()) {
		t.Fatalf("student attachment download did not return the stored bytes: %v", err)
	}
	if _, err := service.GetAttachment(ctx, outsider, material.ID, attachment.ID); err == nil {
		t.Fatal("a student outside the course should not receive the attachment")
	}
	if err := service.Complete(ctx, student, material.ID); err != nil {
		t.Fatal(err)
	}
	items, err := service.List(ctx, student, courseID)
	if err != nil || len(items) != 1 || items[0].CompletedAt == nil || len(items[0].Attachments) != 1 {
		t.Fatalf("expected completed material with attachment metadata: %v", err)
	}
}

func verifyAssignmentFlow(t *testing.T, ctx context.Context, service *assignments.Service, courseID string, teacher, student, outsider authz.Principal) {
	t.Helper()
	assignment, err := service.Create(ctx, teacher, assignments.WriteRequest{CourseID: courseID, Title: "Synthetic Assignment", Instructions: "Synthetic instructions", DueAt: time.Now().UTC().Add(time.Hour), MaxScore: 10})
	if err != nil {
		t.Fatal(err)
	}
	submission := assignments.SubmitRequest{Content: "Synthetic submission"}
	if err := service.Submit(ctx, student, assignment.ID, submission); err == nil {
		t.Fatal("draft assignment should not accept submissions")
	}
	if err := service.Publish(ctx, teacher, assignment.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.Submit(ctx, outsider, assignment.ID, submission); err == nil {
		t.Fatal("assignment should not accept submissions from outside the course")
	}
	if err := service.Submit(ctx, student, assignment.ID, submission); err != nil {
		t.Fatal(err)
	}
	submissions, err := service.ListSubmissions(ctx, teacher, assignment.ID)
	if err != nil || len(submissions) != 1 {
		t.Fatalf("expected one teacher-visible submission: %v", err)
	}
	if err := service.Grade(ctx, teacher, submissions[0].ID, assignments.GradeRequest{Score: 8, Feedback: "Synthetic feedback"}); err != nil {
		t.Fatal(err)
	}
	items, err := service.List(ctx, student, courseID)
	if err != nil || len(items) != 1 || items[0].Submission == nil || items[0].Submission.Score == nil || *items[0].Submission.Score != 8 {
		t.Fatalf("expected student-visible graded submission: %v", err)
	}
}
