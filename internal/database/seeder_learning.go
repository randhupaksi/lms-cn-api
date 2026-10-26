package database

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"lms-cn-api/internal/modules/assignments"
	"lms-cn-api/internal/modules/materials"
	"lms-cn-api/internal/modules/users"
)

func seedDemoLearningContent(tx *gorm.DB, dataset *demoDataset, now time.Time) error {
	for courseIndex := range dataset.courses {
		course := &dataset.courses[courseIndex]
		if err := seedDemoMaterials(tx, course, now); err != nil {
			return err
		}
		if err := seedDemoAssignments(tx, course, now); err != nil {
			return err
		}
	}
	return nil
}

func seedDemoMaterials(tx *gorm.DB, course *demoCourse, now time.Time) error {
	materialsForCourse := demoMaterialsForSubject(course.subject.Name)
	course.materials = make([]materials.Material, 0, len(materialsForCourse))
	for index, definition := range materialsForCourse {
		id := 2000 + course.index*10 + index
		if course.index == 0 {
			id = 1000 + index
		}
		material := materials.Material{
			ID:          seedID(id),
			CourseID:    course.course.ID,
			AuthorID:    course.teachers[0].ID,
			Title:       definition.title,
			Description: definition.description,
			Content:     definition.content,
			Position:    uint(index + 1),
			Status:      definition.status,
		}
		if definition.status == materials.StatusPublished {
			publishedAt := now.AddDate(0, 0, -(16 - index*2))
			material.PublishedAt = &publishedAt
		}
		if err := seedRecord(tx, &material); err != nil {
			return fmt.Errorf("material %s: %w", material.Title, err)
		}
		course.materials = append(course.materials, material)

		if material.Status != materials.StatusPublished {
			continue
		}
		for _, student := range course.students {
			if !demoMaterialCompleted(course.index, student.classNumber, index) {
				continue
			}
			daysAgo := 1 + (course.index+student.classNumber*2+index*3)%13
			progress := materials.Progress{
				MaterialID:  material.ID,
				StudentID:   student.user.ID,
				CompletedAt: now.AddDate(0, 0, -daysAgo),
			}
			if err := seedRecord(tx, &progress); err != nil {
				return fmt.Errorf("material progress for %s: %w", material.Title, err)
			}
		}
	}
	return nil
}

type demoMaterialDefinition struct {
	title       string
	description string
	content     string
	status      string
}

func demoMaterialsForSubject(subject string) []demoMaterialDefinition {
	return []demoMaterialDefinition{
		{
			title:       "Konsep Dasar " + subject,
			description: "Tujuan pembelajaran, istilah kunci, dan gambaran materi.",
			content:     fmt.Sprintf("Materi ini memperkenalkan konsep inti %s. Mulailah dengan memahami istilah penting, hubungan antar konsep, lalu catat contoh yang paling dekat dengan kegiatan sehari-hari.", subject),
			status:      materials.StatusPublished,
		},
		{
			title:       "Contoh dan Pembahasan " + subject,
			description: "Contoh langkah demi langkah untuk menerapkan konsep.",
			content:     fmt.Sprintf("Gunakan contoh pada bagian ini untuk melihat bagaimana konsep %s diterapkan. Periksa setiap langkah, alasan pemilihannya, dan cara memeriksa kembali hasil akhir.", subject),
			status:      materials.StatusPublished,
		},
		{
			title:       "Latihan Mandiri " + subject,
			description: "Pertanyaan refleksi dan latihan sebelum asesmen.",
			content:     fmt.Sprintf("Coba kerjakan latihan %s secara mandiri. Jelaskan alasan di balik jawabanmu, tandai bagian yang belum dipahami, kemudian bandingkan dengan catatan pembelajaran.", subject),
			status:      materials.StatusPublished,
		},
		{
			title:       "Pengayaan " + subject,
			description: "Draf materi lanjutan yang sedang disiapkan pengajar.",
			content:     fmt.Sprintf("Catatan pengayaan %s ini masih disiapkan oleh pengajar dan belum tersedia untuk siswa.", subject),
			status:      materials.StatusDraft,
		},
	}
}

func demoMaterialCompleted(courseIndex, studentPosition, materialIndex int) bool {
	return (courseIndex+studentPosition+materialIndex)%4 != 0
}

func seedDemoAssignments(tx *gorm.DB, course *demoCourse, now time.Time) error {
	definitions := []struct {
		title     string
		brief     string
		status    string
		dueAt     time.Time
		published *time.Time
	}{
		{
			title:     "Proyek Penerapan " + course.subject.Name,
			brief:     fmt.Sprintf("Susun sebuah hasil kerja singkat yang menerapkan konsep %s. Sertakan tujuan, langkah yang dilakukan, dan alasan pemilihan pendekatan.", course.subject.Name),
			status:    assignments.StatusPublished,
			dueAt:     now.AddDate(0, 0, 7),
			published: timePointer(now.AddDate(0, 0, -14)),
		},
		{
			title:     "Refleksi Pembelajaran " + course.subject.Name,
			brief:     fmt.Sprintf("Tuliskan rangkuman konsep %s yang sudah dikuasai dan satu hal yang ingin kamu pelajari lebih lanjut.", course.subject.Name),
			status:    assignments.StatusClosed,
			dueAt:     now.AddDate(0, 0, -7),
			published: timePointer(now.AddDate(0, 0, -20)),
		},
		{
			title:     "Tugas Penguatan " + course.subject.Name,
			brief:     fmt.Sprintf("Jawab pertanyaan penguatan %s dengan menyertakan proses berpikir dan contoh pendukung.", course.subject.Name),
			status:    assignments.StatusPublished,
			dueAt:     now.AddDate(0, 0, -3),
			published: timePointer(now.AddDate(0, 0, -12)),
		},
		{
			title:  "Rencana Presentasi " + course.subject.Name,
			brief:  fmt.Sprintf("Draf arahan presentasi untuk membagikan temuan belajar terkait %s.", course.subject.Name),
			status: assignments.StatusDraft,
			dueAt:  now.AddDate(0, 0, 14),
		},
	}

	course.assignments = make([]assignments.Assignment, 0, len(definitions))
	for index, definition := range definitions {
		id := 1100 + course.index*10 + index
		assignment := assignments.Assignment{
			ID:           seedID(id),
			CourseID:     course.course.ID,
			AuthorID:     course.teachers[0].ID,
			Title:        definition.title,
			Instructions: definition.brief,
			DueAt:        definition.dueAt,
			MaxScore:     100,
			Status:       definition.status,
			PublishedAt:  definition.published,
		}
		if err := seedRecord(tx, &assignment); err != nil {
			return fmt.Errorf("assignment %s: %w", assignment.Title, err)
		}
		course.assignments = append(course.assignments, assignment)
		if index == 3 {
			continue
		}
		if err := seedAssignmentSubmissions(tx, *course, assignment, index, now); err != nil {
			return err
		}
	}
	return nil
}

func seedAssignmentSubmissions(tx *gorm.DB, course demoCourse, assignment assignments.Assignment, assignmentKind int, now time.Time) error {
	for _, student := range course.students {
		status := demoSubmissionStatus(assignmentKind, student.classNumber)
		if status == "" {
			continue
		}
		submittedAt := demoSubmissionTime(course.index, assignmentKind, student.classNumber, now)
		graderID := course.teachers[0].ID
		score := 58 + float64((student.classNumber*7+course.index*11+assignmentKind*5)%41)
		var scoreValue *float64
		var gradedAt *time.Time
		var gradedBy *string
		feedback := ""
		if status == "graded" || status == "returned" {
			scoreValue = pointer(score)
			gradeTime := submittedAt.Add(time.Duration(2+student.classNumber%5) * time.Hour)
			gradedAt = &gradeTime
			gradedBy = stringPointer(graderID)
			feedback = "Contoh dan alasan sudah tersusun baik. Periksa kembali ketelitian langkah sebelum mengumpulkan tugas berikutnya."
		}
		id := demoSubmissionIDSuffix(course.index, assignmentKind, student.classNumber)
		submission := assignments.Submission{
			ID:           seedID(id),
			AssignmentID: assignment.ID,
			StudentID:    student.user.ID,
			Content:      demoSubmissionContent(course.subject.Name, assignmentKind, student.user.FullName),
			Status:       status,
			Score:        scoreValue,
			Feedback:     feedback,
			SubmittedAt:  submittedAt,
			GradedAt:     gradedAt,
			GradedBy:     gradedBy,
		}
		if err := seedRecord(tx, &submission); err != nil {
			return fmt.Errorf("submission for %s: %w", assignment.Title, err)
		}
	}
	return nil
}

func demoSubmissionIDSuffix(courseIndex, assignmentKind, studentPosition int) int {
	if courseIndex == 0 && assignmentKind == 0 && studentPosition == 0 {
		return 1120
	}
	if courseIndex == 0 && assignmentKind == 1 && studentPosition == 1 {
		return 1121
	}
	return 12000 + courseIndex*100 + assignmentKind*10 + studentPosition
}

func demoSubmissionStatus(kind, studentPosition int) string {
	statuses := [][]string{
		{"submitted", "graded", "returned", "", "submitted", ""},
		{"graded", "graded", "returned", "", "graded", ""},
		{"submitted", "", "graded", "", "returned", ""},
	}
	return statuses[kind][studentPosition%6]
}

func demoSubmissionTime(courseIndex, assignmentKind, studentPosition int, now time.Time) time.Time {
	daysAgo := 1 + (courseIndex+studentPosition*2)%12
	if assignmentKind == 1 {
		daysAgo = 8 + (courseIndex+studentPosition)%8
	}
	if assignmentKind == 2 {
		daysAgo = 4 + (courseIndex+studentPosition)%8
	}
	return now.AddDate(0, 0, -daysAgo).Add(time.Duration((studentPosition+courseIndex)%8) * time.Hour)
}

func demoSubmissionContent(subject string, kind int, studentName string) string {
	work := []string{
		fmt.Sprintf("Saya menyusun contoh penerapan %s, menjelaskan tujuan tiap langkah, lalu memeriksa kembali hasilnya dengan kriteria yang diberikan.", subject),
		fmt.Sprintf("Bagian %s yang paling saya pahami adalah hubungan antara konsep dan contoh. Saya akan berlatih lagi pada langkah yang masih perlu ketelitian.", subject),
		fmt.Sprintf("Saya membandingkan dua cara menyelesaikan latihan %s dan menuliskan alasan mengapa cara yang dipilih lebih sesuai.", subject),
	}
	return fmt.Sprintf("Catatan belajar %s: %s", studentName, work[kind])
}

var _ = users.RoleStudent
