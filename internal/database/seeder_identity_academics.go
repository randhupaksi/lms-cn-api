package database

import (
	"errors"
	"fmt"
	"time"

	"lms-cn-api/internal/modules/academics"
	"lms-cn-api/internal/modules/users"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type demoUserDefinition struct {
	suffix      int
	identifier  string
	fullName    string
	role        users.Role
	previousIDs []string
}

func seedDemoUsers(tx *gorm.DB) (demoDataset, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(demoPassword), bcrypt.DefaultCost)
	if err != nil {
		return demoDataset{}, fmt.Errorf("hash demo password: %w", err)
	}

	definitions := []demoUserDefinition{
		{suffix: 1, identifier: "admin", fullName: "Admin Akademik Citra Negara", role: users.RoleAdmin},
		{suffix: 2, identifier: "rina.maharani", fullName: "Rina Maharani, S.Kom.", role: users.RoleTeacher, previousIDs: []string{"teacher"}},
		{suffix: 3, identifier: "adi.pratama", fullName: "Adi Pratama, S.Pd.", role: users.RoleTeacher, previousIDs: []string{"teacher.adi"}},
		{suffix: 4, identifier: "sari.wulandari", fullName: "Sari Wulandari, S.Pd.", role: users.RoleTeacher},
		{suffix: 5, identifier: "budi.santoso", fullName: "Budi Santoso, S.Pd.", role: users.RoleTeacher},
	}
	studentDefinitions := []demoUserDefinition{
		{suffix: 10, identifier: "randhu.paksi", fullName: "Randhu Paksi Membumi", role: users.RoleStudent, previousIDs: []string{"student", "nadia.lestari"}},
		{suffix: 11, identifier: "andi.saputra", fullName: "Andi Saputra", role: users.RoleStudent, previousIDs: []string{"student.andi"}},
		{suffix: 12, identifier: "salsa.nuraini", fullName: "Salsa Nuraini", role: users.RoleStudent, previousIDs: []string{"student.salsa"}},
		{suffix: 13, identifier: "bagas.ramadhan", fullName: "Bagas Ramadhan", role: users.RoleStudent, previousIDs: []string{"student.bagas"}},
		{suffix: 14, identifier: "dimas.kurniawan", fullName: "Dimas Kurniawan", role: users.RoleStudent, previousIDs: []string{"student.dimas"}},
		{suffix: 15, identifier: "zahra.aisyah", fullName: "Zahra Aisyah Putri", role: users.RoleStudent},
		{suffix: 16, identifier: "raka.pratama", fullName: "Raka Pratama", role: users.RoleStudent},
		{suffix: 17, identifier: "putri.anggraini", fullName: "Putri Anggraini", role: users.RoleStudent},
		{suffix: 18, identifier: "fajar.hidayat", fullName: "Fajar Hidayat", role: users.RoleStudent},
		{suffix: 19, identifier: "intan.permatasari", fullName: "Intan Permatasari", role: users.RoleStudent},
		{suffix: 20, identifier: "reza.maulana", fullName: "Reza Maulana", role: users.RoleStudent},
		{suffix: 21, identifier: "citra.lestari", fullName: "Citra Lestari", role: users.RoleStudent},
		{suffix: 22, identifier: "dewi.anggraini", fullName: "Dewi Anggraini", role: users.RoleStudent},
		{suffix: 23, identifier: "farhan.akbar", fullName: "Farhan Akbar", role: users.RoleStudent},
		{suffix: 24, identifier: "maya.safitri", fullName: "Maya Safitri", role: users.RoleStudent},
		{suffix: 25, identifier: "ilham.fauzi", fullName: "Ilham Fauzi", role: users.RoleStudent},
		{suffix: 26, identifier: "nabila.putri", fullName: "Nabila Putri", role: users.RoleStudent},
		{suffix: 27, identifier: "yusuf.ramadhan", fullName: "Yusuf Ramadhan", role: users.RoleStudent},
	}

	seededUsers := make([]users.User, 0, len(definitions)+len(studentDefinitions))
	for _, definition := range append(definitions, studentDefinitions...) {
		seeded, seedErr := seedDemoUser(tx, definition, string(passwordHash))
		if seedErr != nil {
			return demoDataset{}, fmt.Errorf("seed account %d: %w", definition.suffix, seedErr)
		}
		seededUsers = append(seededUsers, seeded)
	}

	dataset := demoDataset{
		admin:    seededUsers[0],
		teachers: append([]users.User(nil), seededUsers[1:5]...),
		students: make([]demoStudent, 0, len(studentDefinitions)),
	}
	for index, student := range seededUsers[5:] {
		dataset.students = append(dataset.students, demoStudent{
			user:        student,
			classIndex:  index / 6,
			classNumber: index % 6,
		})
	}
	dataset.superadmin, err = seedDemoUser(tx, demoUserDefinition{suffix: 6, identifier: "arif.wicaksana", fullName: "Arif Wicaksana, M.Pd.", role: users.RoleSuperadmin}, string(passwordHash))
	if err != nil {
		return demoDataset{}, err
	}
	for _, definition := range []demoUserDefinition{
		{suffix: 7, identifier: "dewi.kusuma", fullName: "Dewi Kusuma, S.Kom.", role: users.RoleAdmin},
		{suffix: 8, identifier: "nanda.prasetyo", fullName: "Nanda Prasetyo, S.Pd.", role: users.RoleTeacher},
		{suffix: 9, identifier: "alya.ramadhani", fullName: "Alya Ramadhani", role: users.RoleStudent},
		{suffix: 28, identifier: "kepala.sekolah", fullName: "Kepala Sekolah Citra Negara", role: users.RolePrincipal},
		{suffix: 29, identifier: "kurikulum", fullName: "Guru Kurikulum Citra Negara", role: users.RoleCurriculum},
	} {
		if _, err = seedDemoUser(tx, definition, string(passwordHash)); err != nil {
			return demoDataset{}, err
		}
	}
	return dataset, nil
}

func seedDemoUser(tx *gorm.DB, definition demoUserDefinition, passwordHash string) (users.User, error) {
	id := seedID(definition.suffix)
	var existing users.User
	err := tx.First(&existing, "id = ?", id).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return users.User{}, err
	}
	if err == nil {
		// Preserve account governance and changed credentials across development restarts.
		legacyIdentity := false
		for _, previousID := range definition.previousIDs {
			legacyIdentity = legacyIdentity || existing.Identifier == previousID
		}
		if !legacyIdentity {
			return existing, nil
		}
		var conflicting users.User
		conflictErr := tx.Where("identifier = ? AND id <> ?", definition.identifier, id).First(&conflicting).Error
		if conflictErr == nil {
			return users.User{}, fmt.Errorf("demo login identifier %q is already in use", definition.identifier)
		}
		if !errors.Is(conflictErr, gorm.ErrRecordNotFound) {
			return users.User{}, conflictErr
		}

		updates := map[string]any{
			"identifier": definition.identifier,
			"full_name":  definition.fullName,
		}
		if updateErr := tx.Model(&existing).Updates(updates).Error; updateErr != nil {
			return users.User{}, updateErr
		}
		existing.Identifier = definition.identifier
		existing.FullName = definition.fullName
		return existing, nil
	}

	var conflicting users.User
	conflictErr := tx.Where("identifier = ?", definition.identifier).First(&conflicting).Error
	if conflictErr == nil {
		return users.User{}, fmt.Errorf("demo login identifier %q is already in use", definition.identifier)
	}
	if !errors.Is(conflictErr, gorm.ErrRecordNotFound) {
		return users.User{}, conflictErr
	}

	seeded := users.User{
		ID:                 id,
		Identifier:         definition.identifier,
		FullName:           definition.fullName,
		Role:               definition.role,
		Status:             users.StatusActive,
		PasswordHash:       passwordHash,
		MustChangePassword: false,
	}
	if createErr := tx.Create(&seeded).Error; createErr != nil {
		return users.User{}, createErr
	}
	return seeded, nil
}

func seedDemoAcademics(tx *gorm.DB, dataset *demoDataset, now time.Time) error {
	schoolNow := now.Add(7 * time.Hour)
	startYear := schoolNow.Year()
	if schoolNow.Month() < time.July {
		startYear--
	}
	dataset.year = academics.AcademicYear{
		ID:       seedID(100),
		Name:     fmt.Sprintf("%d/%d", startYear, startYear+1),
		StartsOn: date(startYear, time.July, 1),
		EndsOn:   date(startYear+1, time.June, 30),
		Status:   "active",
	}
	if err := seedRecord(tx, &dataset.year); err != nil {
		return fmt.Errorf("academic year: %w", err)
	}

	dataset.classes = []academics.ClassGroup{
		{ID: seedID(101), AcademicYearID: dataset.year.ID, Name: "XII RPL 1", GradeLevel: 12},
		{ID: seedID(102), AcademicYearID: dataset.year.ID, Name: "XII RPL 2", GradeLevel: 12},
		{ID: seedID(103), AcademicYearID: dataset.year.ID, Name: "XI RPL 1", GradeLevel: 11},
	}
	for index := range dataset.classes {
		if err := seedRecord(tx, &dataset.classes[index]); err != nil {
			return fmt.Errorf("class group %s: %w", dataset.classes[index].Name, err)
		}
	}

	dataset.subjects = []academics.Subject{
		{ID: seedID(110), Code: "PWEB", Name: "Pemrograman Web"},
		{ID: seedID(111), Code: "MTK", Name: "Matematika Terapan"},
		{ID: seedID(112), Code: "BIN", Name: "Bahasa Indonesia"},
		{ID: seedID(113), Code: "BD", Name: "Basis Data"},
		{ID: seedID(114), Code: "BING", Name: "Bahasa Inggris"},
	}
	for index := range dataset.subjects {
		if err := seedRecord(tx, &dataset.subjects[index]); err != nil {
			return fmt.Errorf("subject %s: %w", dataset.subjects[index].Code, err)
		}
	}

	type courseDefinition struct {
		id             int
		classIndex     int
		subjectIndex   int
		teacherIndexes []int
	}
	courseDefinitions := []courseDefinition{
		{id: 120, classIndex: 0, subjectIndex: 0, teacherIndexes: []int{0, 1}},
		{id: 121, classIndex: 0, subjectIndex: 1, teacherIndexes: []int{0}},
		{id: 123, classIndex: 0, subjectIndex: 2, teacherIndexes: []int{2}},
		{id: 122, classIndex: 1, subjectIndex: 2, teacherIndexes: []int{1}},
		{id: 124, classIndex: 1, subjectIndex: 0, teacherIndexes: []int{2}},
		{id: 125, classIndex: 1, subjectIndex: 3, teacherIndexes: []int{1, 3}},
		{id: 126, classIndex: 2, subjectIndex: 0, teacherIndexes: []int{3}},
		{id: 127, classIndex: 2, subjectIndex: 3, teacherIndexes: []int{3}},
		{id: 128, classIndex: 2, subjectIndex: 4, teacherIndexes: []int{0}},
	}
	dataset.courses = make([]demoCourse, 0, len(courseDefinitions))
	for index, definition := range courseDefinitions {
		classGroup := dataset.classes[definition.classIndex]
		subject := dataset.subjects[definition.subjectIndex]
		course := academics.Course{
			ID:             seedID(definition.id),
			AcademicYearID: dataset.year.ID,
			ClassGroupID:   classGroup.ID,
			SubjectID:      subject.ID,
			Name:           fmt.Sprintf("%s · %s", subject.Name, classGroup.Name),
			Status:         "active",
		}
		if err := seedRecord(tx, &course); err != nil {
			return fmt.Errorf("course %s: %w", course.Name, err)
		}
		seededCourse := demoCourse{index: index, class: classGroup, subject: subject, course: course}
		for _, teacherIndex := range definition.teacherIndexes {
			teacher := dataset.teachers[teacherIndex]
			seededCourse.teachers = append(seededCourse.teachers, teacher)
			membership := academics.CourseTeacher{CourseID: course.ID, TeacherID: teacher.ID, AssignedAt: now.AddDate(0, 0, -30)}
			if err := seedRecord(tx, &membership); err != nil {
				return fmt.Errorf("teacher membership in %s: %w", course.Name, err)
			}
		}
		for _, student := range dataset.students {
			if student.classIndex != definition.classIndex {
				continue
			}
			seededCourse.students = append(seededCourse.students, student)
			membership := academics.CourseStudent{CourseID: course.ID, StudentID: student.user.ID, EnrolledAt: now.AddDate(0, -2, 0)}
			if err := seedRecord(tx, &membership); err != nil {
				return fmt.Errorf("student enrollment in %s: %w", course.Name, err)
			}
		}
		dataset.courses = append(dataset.courses, seededCourse)
	}
	return nil
}
