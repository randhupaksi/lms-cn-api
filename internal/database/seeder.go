package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"lms-cn-api/internal/modules/academics"
	"lms-cn-api/internal/modules/assignments"
	"lms-cn-api/internal/modules/attempts"
	"lms-cn-api/internal/modules/audit"
	"lms-cn-api/internal/modules/exams"
	"lms-cn-api/internal/modules/materials"
	"lms-cn-api/internal/modules/questions"
	"lms-cn-api/internal/modules/results"
	"lms-cn-api/internal/modules/users"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const demoPassword = "12121212"

// SeedDemoData creates a connected development dataset that exercises the
// principal LMS workflows. It is intentionally idempotent and must never run
// in production; the configuration layer enforces that boundary.
func SeedDemoData(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return errors.New("cannot seed demo data with a nil database")
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		admin, err := seedUser(tx, 1, "admin", "Administrator Citra Negara", users.RoleAdmin)
		if err != nil {
			return fmt.Errorf("seed admin: %w", err)
		}
		teacher, err := seedUser(tx, 2, "teacher", "Rina Maharani, S.Kom.", users.RoleTeacher)
		if err != nil {
			return fmt.Errorf("seed teacher: %w", err)
		}
		teacherAssistant, err := seedUser(tx, 3, "teacher.adi", "Adi Pratama, S.Pd.", users.RoleTeacher)
		if err != nil {
			return fmt.Errorf("seed assistant teacher: %w", err)
		}

		studentNames := []string{"student|Nadia Putri Lestari", "student.andi|Andi Saputra", "student.salsa|Salsa Nuraini", "student.bagas|Bagas Ramadhan", "student.dimas|Dimas Kurniawan"}
		students := make([]users.User, 0, len(studentNames))
		for index, entry := range studentNames {
			parts := splitSeedUser(entry)
			identifier, fullName := parts[0], parts[1]
			student, seedErr := seedUser(tx, 10+index, identifier, fullName, users.RoleStudent)
			if seedErr != nil {
				return fmt.Errorf("seed student %s: %w", identifier, seedErr)
			}
			students = append(students, student)
		}

		year := academics.AcademicYear{ID: seedID(100), Name: "2025/2026", StartsOn: date(2025, time.July, 1), EndsOn: date(2026, time.June, 30), Status: "active"}
		if err := seedRecord(tx, &year); err != nil {
			return fmt.Errorf("seed academic year: %w", err)
		}
		classRPL1 := academics.ClassGroup{ID: seedID(101), AcademicYearID: year.ID, Name: "XII RPL 1", GradeLevel: 12}
		classRPL2 := academics.ClassGroup{ID: seedID(102), AcademicYearID: year.ID, Name: "XII RPL 2", GradeLevel: 12}
		for _, classGroup := range []*academics.ClassGroup{&classRPL1, &classRPL2} {
			if err := seedRecord(tx, classGroup); err != nil {
				return fmt.Errorf("seed class group: %w", err)
			}
		}

		subjects := []academics.Subject{
			{ID: seedID(110), Code: "PWEB", Name: "Pemrograman Web"},
			{ID: seedID(111), Code: "MTK", Name: "Matematika Terapan"},
			{ID: seedID(112), Code: "BIN", Name: "Bahasa Indonesia"},
		}
		for index := range subjects {
			if err := seedRecord(tx, &subjects[index]); err != nil {
				return fmt.Errorf("seed subject %s: %w", subjects[index].Code, err)
			}
		}

		courses := []academics.Course{
			{ID: seedID(120), AcademicYearID: year.ID, ClassGroupID: classRPL1.ID, SubjectID: subjects[0].ID, Name: "Pemrograman Web XII RPL 1", Status: "active"},
			{ID: seedID(121), AcademicYearID: year.ID, ClassGroupID: classRPL1.ID, SubjectID: subjects[1].ID, Name: "Matematika Terapan XII RPL 1", Status: "active"},
			{ID: seedID(122), AcademicYearID: year.ID, ClassGroupID: classRPL2.ID, SubjectID: subjects[2].ID, Name: "Bahasa Indonesia XII RPL 2", Status: "active"},
		}
		for index := range courses {
			if err := seedRecord(tx, &courses[index]); err != nil {
				return fmt.Errorf("seed course %s: %w", courses[index].Name, err)
			}
		}

		memberships := []any{
			&academics.CourseTeacher{CourseID: courses[0].ID, TeacherID: teacher.ID, AssignedAt: now},
			&academics.CourseTeacher{CourseID: courses[0].ID, TeacherID: teacherAssistant.ID, AssignedAt: now},
			&academics.CourseTeacher{CourseID: courses[1].ID, TeacherID: teacher.ID, AssignedAt: now},
			&academics.CourseTeacher{CourseID: courses[2].ID, TeacherID: teacherAssistant.ID, AssignedAt: now},
		}
		for _, student := range students {
			memberships = append(memberships,
				&academics.CourseStudent{CourseID: courses[0].ID, StudentID: student.ID, EnrolledAt: now},
				&academics.CourseStudent{CourseID: courses[1].ID, StudentID: student.ID, EnrolledAt: now},
			)
		}
		for _, membership := range memberships {
			if err := seedRecord(tx, membership); err != nil {
				return fmt.Errorf("seed course membership: %w", err)
			}
		}

		questionSet, err := seedQuestionBank(tx, courses[0].ID, teacher.ID)
		if err != nil {
			return err
		}
		activeExam, historicalExam, err := seedExams(tx, courses[0].ID, teacher.ID, questionSet, students, now)
		if err != nil {
			return err
		}
		if err := seedExamActivity(tx, activeExam, historicalExam, questionSet, students, now); err != nil {
			return err
		}
		if err := seedLearningContent(tx, courses[0].ID, teacher.ID, students, now); err != nil {
			return err
		}
		if err := seedAuditTrail(tx, admin.ID, teacher.ID, activeExam.ID, now); err != nil {
			return err
		}
		return nil
	})
}

func seedUser(tx *gorm.DB, suffix int, identifier, fullName string, role users.Role) (users.User, error) {
	var value users.User
	if err := tx.Where("identifier = ?", identifier).First(&value).Error; err == nil {
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(demoPassword), bcrypt.DefaultCost)
		if hashErr != nil {
			return users.User{}, hashErr
		}
		value.FullName = fullName
		value.Role = role
		value.Status = users.StatusActive
		value.PasswordHash = string(hash)
		value.MustChangePassword = false
		if err := tx.Model(&value).Updates(map[string]any{
			"full_name":            value.FullName,
			"role":                 value.Role,
			"status":               value.Status,
			"password_hash":        value.PasswordHash,
			"must_change_password": value.MustChangePassword,
		}).Error; err != nil {
			return users.User{}, err
		}
		return value, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return users.User{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(demoPassword), bcrypt.DefaultCost)
	if err != nil {
		return users.User{}, err
	}
	value = users.User{ID: seedID(suffix), Identifier: identifier, FullName: fullName, Role: role, Status: users.StatusActive, PasswordHash: string(hash), MustChangePassword: false}
	if err := tx.Create(&value).Error; err != nil {
		return users.User{}, err
	}
	return value, nil
}

func seedQuestionBank(tx *gorm.DB, courseID, authorID string) ([]questions.Question, error) {
	definitions := []struct {
		stem     string
		category string
		tags     []string
		options  []string
		correct  int
	}{
		{"Dalam arsitektur web modern, fungsi utama REST API adalah...", "Web Development", []string{"api", "http"}, []string{"Menyediakan komunikasi terstruktur antara client dan server", "Menggantikan database", "Menyimpan password di browser", "Mengatur desain warna aplikasi"}, 0},
		{"HTTP status code yang tepat untuk resource berhasil dibuat adalah...", "Web Development", []string{"http", "backend"}, []string{"200 OK", "201 Created", "302 Found", "404 Not Found"}, 1},
		{"Prinsip yang menjaga module tetap memiliki satu tanggung jawab utama disebut...", "Software Engineering", []string{"clean-code", "architecture"}, []string{"Open-closed principle", "Single responsibility principle", "Dependency inversion", "Interface segregation"}, 1},
		{"Query database yang menggunakan parameter terikat membantu mencegah...", "Database", []string{"security", "sql"}, []string{"Layout shift", "SQL injection", "Memory leak", "Cache miss"}, 1},
		{"HTTP method yang umum digunakan untuk mengambil data tanpa mengubah resource adalah...", "Web Development", []string{"http", "fundamental"}, []string{"GET", "POST", "PATCH", "DELETE"}, 0},
	}
	result := make([]questions.Question, 0, len(definitions))
	for index, definition := range definitions {
		question := questions.Question{ID: seedID(200 + index), CourseID: courseID, AuthorID: authorID, Type: questions.TypeSingleChoice, Stem: definition.stem, DefaultPoints: 20, Category: definition.category, Tags: definition.tags, Status: "active", Version: 1}
		if err := seedRecord(tx, &question); err != nil {
			return nil, fmt.Errorf("seed question %d: %w", index+1, err)
		}
		for optionIndex, content := range definition.options {
			option := questions.Option{ID: seedID(2100 + index*10 + optionIndex), QuestionID: question.ID, Content: content, IsCorrect: optionIndex == definition.correct, Position: uint(optionIndex + 1)}
			if err := seedRecord(tx, &option); err != nil {
				return nil, fmt.Errorf("seed option %d: %w", optionIndex+1, err)
			}
			question.Options = append(question.Options, option)
		}
		result = append(result, question)
	}
	return result, nil
}

func seedExams(tx *gorm.DB, courseID, authorID string, questionSet []questions.Question, students []users.User, now time.Time) (exams.Exam, exams.Exam, error) {
	publishedAt := now.Add(-2 * time.Hour)
	active := exams.Exam{ID: seedID(300), CourseID: courseID, AuthorID: authorID, Title: "Ujian Diagnostik Pemrograman Web", Description: "Ujian latihan untuk memetakan kesiapan siswa sebelum ujian kelulusan.", Status: exams.StatusPublished, StartsAt: now.Add(-30 * time.Minute), EndsAt: now.Add(3 * time.Hour), DurationMinutes: 60, MaxAttempts: 1, AllowBackNavigation: true, RandomizeQuestions: true, RandomizeOptions: true, ResultPolicy: "after_publish", PublishedAt: &publishedAt}
	historicalPublishedAt := now.Add(-7 * 24 * time.Hour)
	historical := exams.Exam{ID: seedID(301), CourseID: courseID, AuthorID: authorID, Title: "Ujian Tengah Semester Pemrograman Web", Description: "Simulasi ujian tengah semester dengan hasil yang sudah dipublikasikan.", Status: exams.StatusClosed, StartsAt: now.Add(-10 * 24 * time.Hour), EndsAt: now.Add(-9 * 24 * time.Hour), DurationMinutes: 75, MaxAttempts: 1, AllowBackNavigation: false, RandomizeQuestions: false, RandomizeOptions: false, ResultPolicy: "after_publish", PublishedAt: &historicalPublishedAt}
	for _, exam := range []*exams.Exam{&active, &historical} {
		if err := seedRecord(tx, exam); err != nil {
			return exams.Exam{}, exams.Exam{}, fmt.Errorf("seed exam %s: %w", exam.Title, err)
		}
	}
	for _, exam := range []*exams.Exam{&active, &historical} {
		base := 400
		if exam.ID == historical.ID {
			base = 500
		}
		for index, question := range questionSet {
			examQuestion := exams.ExamQuestion{ID: seedID(base + index), ExamID: exam.ID, SourceQuestionID: question.ID, SourceVersion: question.Version, Type: question.Type, Stem: question.Stem, Position: uint(index + 1), Points: question.DefaultPoints}
			if err := seedRecord(tx, &examQuestion); err != nil {
				return exams.Exam{}, exams.Exam{}, fmt.Errorf("seed exam question: %w", err)
			}
			for optionIndex, option := range question.Options {
				examOption := exams.ExamQuestionOption{ID: seedID(base*10 + index*10 + optionIndex), ExamQuestionID: examQuestion.ID, SourceOptionID: option.ID, Content: option.Content, IsCorrect: option.IsCorrect, Position: uint(optionIndex + 1)}
				if err := seedRecord(tx, &examOption); err != nil {
					return exams.Exam{}, exams.Exam{}, fmt.Errorf("seed exam option: %w", err)
				}
			}
		}
		for _, student := range students {
			participant := exams.Participant{ExamID: exam.ID, StudentID: student.ID, AssignedAt: exam.StartsAt.Add(-24 * time.Hour)}
			if err := seedRecord(tx, &participant); err != nil {
				return exams.Exam{}, exams.Exam{}, fmt.Errorf("seed exam participant: %w", err)
			}
		}
	}
	return active, historical, nil
}

func seedExamActivity(tx *gorm.DB, activeExam, historicalExam exams.Exam, questionSet []questions.Question, students []users.User, now time.Time) error {
	activeAttempt := attempts.Attempt{ID: seedID(700), ExamID: activeExam.ID, StudentID: students[0].ID, Status: attempts.StatusInProgress, StartIdempotencyKey: "seed-active-attempt", StartedAt: now.Add(-18 * time.Minute), DeadlineAt: now.Add(42 * time.Minute)}
	if err := seedRecord(tx, &activeAttempt); err != nil {
		return fmt.Errorf("seed active attempt: %w", err)
	}
	activeAnswer := attempts.Answer{ID: seedID(701), AttemptID: activeAttempt.ID, ExamQuestionID: seedID(400), SelectedOptionID: seedID(4000), Revision: 1, SavedAt: now.Add(-12 * time.Minute)}
	if err := seedRecord(tx, &activeAnswer); err != nil {
		return fmt.Errorf("seed active answer: %w", err)
	}
	if err := seedAttemptEvent(tx, seedID(702), activeAttempt.ID, students[0].ID, "answer.saved", now.Add(-12*time.Minute)); err != nil {
		return err
	}

	correctAnswers := []int{3, 5, 2, 4, 1}
	for index := 0; index < len(students); index++ {
		startedAt := historicalExam.StartsAt.Add(time.Duration(index) * time.Minute)
		submittedAt := startedAt.Add(55 * time.Minute)
		receipt := seedID(750 + index)
		attempt := attempts.Attempt{ID: seedID(710 + index), ExamID: historicalExam.ID, StudentID: students[index].ID, Status: attempts.StatusSubmitted, StartIdempotencyKey: fmt.Sprintf("seed-historical-%d", index), StartedAt: startedAt, DeadlineAt: startedAt.Add(75 * time.Minute), SubmittedAt: &submittedAt, SubmissionReceipt: &receipt}
		if err := seedRecord(tx, &attempt); err != nil {
			return fmt.Errorf("seed historical attempt: %w", err)
		}
		for questionIndex, question := range questionSet {
			selectedPosition := 0
			if questionIndex < correctAnswers[index] {
				for optionIndex, option := range question.Options {
					if option.IsCorrect {
						selectedPosition = optionIndex
						break
					}
				}
			} else {
				for optionIndex, option := range question.Options {
					if !option.IsCorrect {
						selectedPosition = optionIndex
						break
					}
				}
			}
			answer := attempts.Answer{ID: seedID(800 + index*10 + questionIndex), AttemptID: attempt.ID, ExamQuestionID: seedID(500 + questionIndex), SelectedOptionID: seedID(5000 + questionIndex*10 + selectedPosition), Revision: 1, SavedAt: submittedAt.Add(-time.Duration(len(questionSet)-questionIndex) * time.Minute)}
			if err := seedRecord(tx, &answer); err != nil {
				return fmt.Errorf("seed historical answer: %w", err)
			}
		}
		if err := seedAttemptEvent(tx, seedID(850+index), attempt.ID, students[index].ID, "attempt.submitted", submittedAt); err != nil {
			return err
		}
		gradedAt := submittedAt.Add(10 * time.Minute)
		result := results.Result{ID: seedID(900 + index), AttemptID: attempt.ID, ExamID: historicalExam.ID, StudentID: students[index].ID, Status: "published", Score: float64(correctAnswers[index] * 20), MaxScore: 100, GradedAt: gradedAt, PublishedAt: &gradedAt}
		if err := seedRecord(tx, &result); err != nil {
			return fmt.Errorf("seed result: %w", err)
		}
	}
	return nil
}

func seedLearningContent(tx *gorm.DB, courseID, teacherID string, students []users.User, now time.Time) error {
	publishedAt := now.Add(-72 * time.Hour)
	materialsData := []materials.Material{
		{ID: seedID(1000), CourseID: courseID, AuthorID: teacherID, Title: "Mengenal Arsitektur Web", Description: "Konsep client, server, API, dan database dalam aplikasi modern.", Content: "Pelajari alur request dari browser menuju API, proses bisnis, database, hingga response kembali ke pengguna.", Position: 1, Status: materials.StatusPublished, PublishedAt: &publishedAt},
		{ID: seedID(1001), CourseID: courseID, AuthorID: teacherID, Title: "HTTP dan REST API", Description: "Method, status code, resource, dan kontrak endpoint.", Content: "Gunakan HTTP method sesuai tujuan operasi dan selalu kembalikan response yang konsisten serta aman.", Position: 2, Status: materials.StatusPublished, PublishedAt: &publishedAt},
		{ID: seedID(1002), CourseID: courseID, AuthorID: teacherID, Title: "Validasi dan Keamanan Input", Description: "Mengapa validasi harus dilakukan di server.", Content: "Jangan pernah mempercayai input dari browser. Validasi bentuk data, kepemilikan resource, dan permission di API.", Position: 3, Status: materials.StatusPublished, PublishedAt: &publishedAt},
	}
	for index := range materialsData {
		if err := seedRecord(tx, &materialsData[index]); err != nil {
			return fmt.Errorf("seed material: %w", err)
		}
	}
	for index := 0; index < 2; index++ {
		progress := materials.Progress{MaterialID: materialsData[index].ID, StudentID: students[0].ID, CompletedAt: now.Add(-time.Duration(index+1) * 24 * time.Hour)}
		if err := seedRecord(tx, &progress); err != nil {
			return fmt.Errorf("seed material progress: %w", err)
		}
	}

	dueAt := now.Add(7 * 24 * time.Hour)
	assignmentPublishedAt := now.Add(-24 * time.Hour)
	assignment := assignments.Assignment{ID: seedID(1100), CourseID: courseID, AuthorID: teacherID, Title: "Mini Project: Rancang Endpoint LMS", Instructions: "Buat rancangan endpoint untuk modul pengumpulan tugas. Sertakan method, payload, response sukses, dan minimal dua error case.", DueAt: dueAt, MaxScore: 100, Status: assignments.StatusPublished, PublishedAt: &assignmentPublishedAt}
	if err := seedRecord(tx, &assignment); err != nil {
		return fmt.Errorf("seed assignment: %w", err)
	}
	closedDueAt := now.Add(-3 * 24 * time.Hour)
	closedAssignmentPublishedAt := now.Add(-10 * 24 * time.Hour)
	closedAssignment := assignments.Assignment{ID: seedID(1101), CourseID: courseID, AuthorID: teacherID, Title: "Refleksi Praktik Clean Code", Instructions: "Tuliskan tiga contoh perbaikan clean code yang kamu terapkan selama mengerjakan project.", DueAt: closedDueAt, MaxScore: 100, Status: assignments.StatusClosed, PublishedAt: &closedAssignmentPublishedAt}
	if err := seedRecord(tx, &closedAssignment); err != nil {
		return fmt.Errorf("seed closed assignment: %w", err)
	}

	submittedAt := now.Add(-6 * time.Hour)
	gradedAt := now.Add(-2 * time.Hour)
	graderID := teacherID
	submissions := []assignments.Submission{
		{ID: seedID(1120), AssignmentID: assignment.ID, StudentID: students[0].ID, Content: "Saya merancang GET /courses/{courseId}/assignments dan POST /assignments/{id}/submissions dengan response envelope yang konsisten.", Status: "submitted", SubmittedAt: submittedAt},
		{ID: seedID(1121), AssignmentID: closedAssignment.ID, StudentID: students[1].ID, Content: "Saya memisahkan handler, service, dan repository agar tanggung jawab setiap layer jelas.", Status: "graded", Score: pointer(88), Feedback: "Struktur dan alasan teknis sudah jelas. Tambahkan contoh negative case pada revisi berikutnya.", SubmittedAt: closedDueAt.Add(-12 * time.Hour), GradedAt: &gradedAt, GradedBy: &graderID},
	}
	for index := range submissions {
		if err := seedRecord(tx, &submissions[index]); err != nil {
			return fmt.Errorf("seed assignment submission: %w", err)
		}
	}
	return nil
}

func seedAuditTrail(tx *gorm.DB, adminID, teacherID, examID string, now time.Time) error {
	entries := []audit.Event{
		{ID: seedID(1200), ActorID: stringPointer(adminID), Action: "course.created", EntityType: "course", EntityID: stringPointer(seedID(120)), Metadata: metadata(map[string]any{"source": "development_seed"}), CreatedAt: now.Add(-14 * 24 * time.Hour)},
		{ID: seedID(1201), ActorID: stringPointer(teacherID), Action: "exam.published", EntityType: "exam", EntityID: stringPointer(examID), Metadata: metadata(map[string]any{"source": "development_seed"}), CreatedAt: now.Add(-2 * time.Hour)},
		{ID: seedID(1202), ActorID: stringPointer(teacherID), Action: "material.published", EntityType: "material", EntityID: stringPointer(seedID(1000)), Metadata: metadata(map[string]any{"source": "development_seed"}), CreatedAt: now.Add(-72 * time.Hour)},
		{ID: seedID(1203), ActorID: stringPointer(teacherID), Action: "assignment.graded", EntityType: "assignment_submission", EntityID: stringPointer(seedID(1121)), Metadata: metadata(map[string]any{"score": 88, "source": "development_seed"}), CreatedAt: now.Add(-2 * time.Hour)},
	}
	for index := range entries {
		if err := seedRecord(tx, &entries[index]); err != nil {
			return fmt.Errorf("seed audit event: %w", err)
		}
	}
	return nil
}

func seedAttemptEvent(tx *gorm.DB, id, attemptID, actorID, eventType string, createdAt time.Time) error {
	event := attempts.Event{ID: id, AttemptID: attemptID, ActorID: actorID, EventType: eventType, Metadata: []byte(`{"source":"development_seed"}`), CreatedAt: createdAt}
	if err := seedRecord(tx, &event); err != nil {
		return fmt.Errorf("seed attempt event: %w", err)
	}
	return nil
}

func seedRecord(tx *gorm.DB, value any) error {
	if err := tx.First(value).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return tx.Create(value).Error
	}
	return nil
}

func seedID(suffix int) string { return fmt.Sprintf("00000000-0000-4000-8000-%012d", suffix) }

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func pointer(value float64) *float64 { return &value }

func stringPointer(value string) *string { return &value }

func metadata(value map[string]any) json.RawMessage {
	encoded, _ := json.Marshal(value)
	return encoded
}

func splitSeedUser(value string) [2]string {
	for index, character := range value {
		if character == '|' {
			return [2]string{value[:index], value[index+1:]}
		}
	}
	return [2]string{value, value}
}
