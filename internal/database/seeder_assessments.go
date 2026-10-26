package database

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"lms-cn-api/internal/modules/attempts"
	"lms-cn-api/internal/modules/exams"
	"lms-cn-api/internal/modules/questions"
	"lms-cn-api/internal/modules/results"
)

type demoQuestionDefinition struct {
	stem     string
	category string
	tags     []string
	options  []string
	correct  int
}

const (
	demoExamActive = iota
	demoExamHistorical
	demoExamUpcoming
	demoExamDraft
)

func seedDemoQuestionBanks(tx *gorm.DB, dataset *demoDataset) error {
	for courseIndex := range dataset.courses {
		course := &dataset.courses[courseIndex]
		definitions := demoQuestionsForSubject(course.subject.Code)
		course.questions = make([]questions.Question, 0, len(definitions))
		for questionIndex, definition := range definitions {
			question := questions.Question{
				ID:            seedID(200 + course.index*10 + questionIndex),
				CourseID:      course.course.ID,
				AuthorID:      course.teachers[0].ID,
				Type:          questions.TypeSingleChoice,
				Stem:          definition.stem,
				DefaultPoints: 20,
				Category:      definition.category,
				Tags:          definition.tags,
				Status:        "active",
				Version:       1,
			}
			if err := seedRecord(tx, &question); err != nil {
				return fmt.Errorf("question %d for %s: %w", questionIndex+1, course.course.Name, err)
			}
			for optionIndex, content := range definition.options {
				option := questions.Option{
					ID:         seedID(2100 + course.index*100 + questionIndex*10 + optionIndex),
					QuestionID: question.ID,
					Content:    content,
					IsCorrect:  optionIndex == definition.correct,
					Position:   uint(optionIndex + 1),
				}
				if err := seedRecord(tx, &option); err != nil {
					return fmt.Errorf("option %d for question %s: %w", optionIndex+1, question.ID, err)
				}
				question.Options = append(question.Options, option)
			}
			course.questions = append(course.questions, question)
		}
	}
	return nil
}

func demoQuestionsForSubject(code string) []demoQuestionDefinition {
	switch code {
	case "MTK":
		return []demoQuestionDefinition{
			{"Harga sebuah buku Rp150.000 mendapat diskon 20%. Berapa harga setelah diskon?", "Aritmetika Sosial", []string{"persentase", "diskon"}, []string{"Rp120.000", "Rp125.000", "Rp130.000", "Rp135.000"}, 0},
			{"Nilai x yang memenuhi 3x + 7 = 25 adalah...", "Aljabar", []string{"persamaan", "variabel"}, []string{"4", "5", "6", "7"}, 2},
			{"Rata-rata dari data 72, 80, 84, dan 88 adalah...", "Statistika", []string{"mean", "data"}, []string{"78", "80", "81", "82"}, 2},
			{"Sebuah kelas memiliki 18 siswa dari total 30 siswa. Pecahan paling sederhana yang menyatakan bagian tersebut adalah...", "Pecahan", []string{"rasio", "pecahan"}, []string{"3/5", "2/3", "5/8", "3/4"}, 0},
			{"Luas persegi panjang dengan panjang 12 cm dan lebar 7 cm adalah...", "Geometri", []string{"luas", "bangun datar"}, []string{"72 cm²", "78 cm²", "84 cm²", "96 cm²"}, 2},
		}
	case "BIN":
		return []demoQuestionDefinition{
			{"Kalimat yang paling efektif untuk laporan kegiatan adalah...", "Kalimat Efektif", []string{"menulis", "laporan"}, []string{"Para siswa-siswa mengikuti kegiatan literasi.", "Siswa mengikuti kegiatan literasi di perpustakaan.", "Kegiatan literasi adalah merupakan kegiatan membaca.", "Banyak siswa-siswa sedang melakukan kegiatan."}, 1},
			{"Gagasan utama sebuah paragraf biasanya menjelaskan...", "Pemahaman Teks", []string{"gagasan utama", "membaca"}, []string{"inti pembahasan paragraf", "contoh terkecil dalam bacaan", "identitas penulis", "daftar sumber bacaan"}, 0},
			{"Penulisan kata baku yang tepat adalah...", "Ejaan", []string{"kata baku", "ejaan"}, []string{"resiko", "aktifitas", "analisis", "ijin"}, 2},
			{"Dalam teks prosedur, bagian yang memuat urutan tindakan disebut...", "Teks Prosedur", []string{"struktur teks", "prosedur"}, []string{"orientasi", "langkah-langkah", "resolusi", "interpretasi"}, 1},
			{"Kalimat yang menyatakan pendapat disertai alasan dan data termasuk...", "Teks Argumentasi", []string{"argumen", "teks"}, []string{"kalimat definisi", "kalimat argumentatif", "kalimat perintah", "kalimat langsung"}, 1},
		}
	case "BD":
		return []demoQuestionDefinition{
			{"Kolom yang nilainya membedakan setiap baris dalam tabel disebut...", "Relasi Basis Data", []string{"primary-key", "tabel"}, []string{"foreign key", "primary key", "index gabungan", "view"}, 1},
			{"Tabel Peserta memiliki kolom id_kelas yang merujuk tabel Kelas. Kolom id_kelas pada Peserta berperan sebagai...", "Kunci Relasi", []string{"foreign-key", "relasi"}, []string{"primary key", "foreign key", "candidate key", "composite key"}, 1},
			{"Tujuan utama normalisasi tabel adalah...", "Normalisasi", []string{"normalisasi", "integritas"}, []string{"mengurangi duplikasi dan anomali data", "menambah jumlah kolom tanpa aturan", "menghapus seluruh relasi", "mengganti basis data dengan berkas teks"}, 0},
			{"Perintah SQL untuk mengambil data dari tabel adalah...", "SQL", []string{"select", "query"}, []string{"INSERT", "UPDATE", "SELECT", "DELETE"}, 2},
			{"Relasi satu kelas yang memiliki banyak peserta paling tepat digambarkan sebagai...", "Pemodelan Data", []string{"kardinalitas", "one-to-many"}, []string{"one-to-one", "one-to-many", "many-to-many", "tanpa relasi"}, 1},
		}
	case "BING":
		return []demoQuestionDefinition{
			{"Choose the correct past form: “Yesterday, our class ___ the science exhibition.”", "Grammar", []string{"past-tense", "verb"}, []string{"visit", "visits", "visited", "visiting"}, 2},
			{"The word “carefully” is closest in meaning to...", "Vocabulary", []string{"adverb", "meaning"}, []string{"in a cautious way", "without a plan", "very quickly", "in a loud voice"}, 0},
			{"Which sentence gives a clear instruction?", "Functional Text", []string{"instruction", "imperative"}, []string{"The room is bright.", "Please submit your work before Friday.", "I submitted the work yesterday.", "The students are in the library."}, 1},
			{"Choose the correct connector: “We reviewed the draft ___ we sent it to the teacher.”", "Grammar", []string{"conjunction", "sequence"}, []string{"before", "because", "although", "unless"}, 0},
			{"A short text that retells events in the order they happened is a...", "Reading", []string{"text-type", "narrative"}, []string{"procedure", "recount", "advertisement", "report"}, 1},
		}
	default:
		return []demoQuestionDefinition{
			{"Dalam alur request web, komponen yang mengirim permintaan ke server adalah...", "Arsitektur Web", []string{"client", "request"}, []string{"client", "database", "migration", "worker"}, 0},
			{"Method HTTP yang lazim dipakai untuk membaca sebuah resource adalah...", "HTTP", []string{"http-method", "rest"}, []string{"POST", "GET", "PATCH", "DELETE"}, 1},
			{"Status HTTP yang sesuai ketika resource berhasil dibuat adalah...", "HTTP", []string{"status-code", "rest"}, []string{"200 OK", "201 Created", "304 Not Modified", "404 Not Found"}, 1},
			{"Validasi input penting dilakukan di server karena...", "Keamanan Aplikasi", []string{"validation", "security"}, []string{"input dari client tidak boleh dianggap tepercaya", "browser selalu mengirim data yang benar", "database tidak memerlukan batasan", "validasi hanya berpengaruh pada warna UI"}, 0},
			{"Format response API yang konsisten membantu client untuk...", "Kontrak API", []string{"response", "api"}, []string{"menafsirkan hasil dan error dengan pola yang dapat diprediksi", "mengakses database tanpa server", "menghapus kebutuhan otorisasi", "menyimpan semua password pengguna"}, 0},
		}
	}
}

func seedDemoExamsAndAttempts(tx *gorm.DB, dataset *demoDataset, now time.Time) error {
	for courseIndex := range dataset.courses {
		course := &dataset.courses[courseIndex]
		course.exams = make([]demoExam, 0, 4)
		for kind := demoExamActive; kind <= demoExamDraft; kind++ {
			seededExam, err := seedDemoExam(tx, *course, kind, now)
			if err != nil {
				return err
			}
			course.exams = append(course.exams, seededExam)
		}
		if err := seedDemoHistoricalAttempts(tx, *course, course.exams[demoExamHistorical]); err != nil {
			return err
		}
		if err := seedDemoUnpublishedAttempt(tx, *course, course.exams[demoExamActive], now); err != nil {
			return err
		}
		if course.index == 0 {
			if err := seedDemoActiveAttempt(tx, *course, course.exams[demoExamActive], now); err != nil {
				return err
			}
		}
	}
	return nil
}

func seedDemoUnpublishedAttempt(tx *gorm.DB, course demoCourse, exam demoExam, now time.Time) error {
	student := course.students[2]
	startedAt := now.Add(-25 * time.Minute)
	submittedAt := now.Add(-10 * time.Minute)
	receipt := seedID(760 + course.index)
	attempt := attempts.Attempt{
		ID:                  seedID(740 + course.index),
		ExamID:              exam.exam.ID,
		StudentID:           student.user.ID,
		Status:              attempts.StatusSubmitted,
		StartIdempotencyKey: fmt.Sprintf("seed-pending-result-c%d", course.index),
		StartedAt:           startedAt,
		DeadlineAt:          startedAt.Add(time.Duration(exam.exam.DurationMinutes) * time.Minute),
		SubmittedAt:         &submittedAt,
		SubmissionReceipt:   &receipt,
	}
	if err := seedRecord(tx, &attempt); err != nil {
		return fmt.Errorf("attempt awaiting result in %s: %w", course.course.Name, err)
	}

	correctCount := 3
	for questionIndex, question := range course.questions {
		wantsCorrect := questionIndex < correctCount
		selectedPosition := 0
		for optionIndex, option := range question.Options {
			if option.IsCorrect == wantsCorrect {
				selectedPosition = optionIndex
				break
			}
		}
		answer := attempts.Answer{
			ID:               seedID(870 + course.index*10 + questionIndex),
			AttemptID:        attempt.ID,
			ExamQuestionID:   seedID(exam.questionIDStart + questionIndex),
			SelectedOptionID: seedID(exam.questionIDStart*10 + questionIndex*10 + selectedPosition),
			Revision:         1,
			SavedAt:          submittedAt.Add(-time.Duration(len(course.questions)-questionIndex) * time.Minute),
		}
		if err := seedRecord(tx, &answer); err != nil {
			return fmt.Errorf("answer awaiting result in %s: %w", course.course.Name, err)
		}
	}
	if err := seedDemoAttemptEvent(tx, 860+course.index, attempt.ID, student.user.ID, "attempt.submitted", submittedAt); err != nil {
		return err
	}

	gradedAt := submittedAt.Add(5 * time.Minute)
	result := results.Result{
		ID:        seedID(940 + course.index),
		AttemptID: attempt.ID,
		ExamID:    exam.exam.ID,
		StudentID: student.user.ID,
		Status:    "draft",
		Score:     float64(correctCount * 20),
		MaxScore:  100,
		GradedAt:  gradedAt,
	}
	if err := seedRecord(tx, &result); err != nil {
		return fmt.Errorf("result awaiting publication in %s: %w", course.course.Name, err)
	}
	return nil
}

func seedDemoExam(tx *gorm.DB, course demoCourse, kind int, now time.Time) (demoExam, error) {
	startsAt, endsAt, status, publishedAt := demoExamSchedule(course.index, kind, now)
	duration := uint(75)
	if kind == demoExamUpcoming {
		duration = 90
	} else if kind == demoExamActive {
		duration = 60
	}
	exam := exams.Exam{
		ID:                  seedID(300 + course.index*10 + kind),
		CourseID:            course.course.ID,
		AuthorID:            course.teachers[0].ID,
		Title:               demoExamTitle(course, kind),
		Description:         demoExamDescription(course, kind),
		Status:              status,
		StartsAt:            startsAt,
		EndsAt:              endsAt,
		DurationMinutes:     duration,
		MaxAttempts:         1,
		AllowBackNavigation: kind != demoExamHistorical,
		RandomizeQuestions:  course.index%2 == 0,
		RandomizeOptions:    course.index%3 == 0,
		ResultPolicy:        "after_publish",
		PublishedAt:         publishedAt,
	}
	if err := seedRecord(tx, &exam); err != nil {
		return demoExam{}, fmt.Errorf("exam %s: %w", exam.Title, err)
	}

	questionIDStart := demoExamQuestionIDStart(course.index, kind)
	for questionIndex, question := range course.questions {
		examQuestion := exams.ExamQuestion{
			ID:               seedID(questionIDStart + questionIndex),
			ExamID:           exam.ID,
			SourceQuestionID: question.ID,
			SourceVersion:    question.Version,
			Type:             question.Type,
			Stem:             question.Stem,
			Position:         uint(questionIndex + 1),
			Points:           question.DefaultPoints,
		}
		if err := seedRecord(tx, &examQuestion); err != nil {
			return demoExam{}, fmt.Errorf("exam question for %s: %w", exam.Title, err)
		}
		for optionIndex, option := range question.Options {
			examOption := exams.ExamQuestionOption{
				ID:             seedID(questionIDStart*10 + questionIndex*10 + optionIndex),
				ExamQuestionID: examQuestion.ID,
				SourceOptionID: option.ID,
				Content:        option.Content,
				IsCorrect:      option.IsCorrect,
				Position:       uint(optionIndex + 1),
			}
			if err := seedRecord(tx, &examOption); err != nil {
				return demoExam{}, fmt.Errorf("exam option for %s: %w", exam.Title, err)
			}
		}
	}

	if status != exams.StatusDraft {
		for _, student := range course.students {
			participant := exams.Participant{
				ExamID:     exam.ID,
				StudentID:  student.user.ID,
				AssignedAt: startsAt.AddDate(0, 0, -14),
			}
			if err := seedRecord(tx, &participant); err != nil {
				return demoExam{}, fmt.Errorf("participant for %s: %w", exam.Title, err)
			}
		}
	}
	return demoExam{exam: exam, questionIDStart: questionIDStart}, nil
}

func demoExamSchedule(courseIndex, kind int, now time.Time) (time.Time, time.Time, string, *time.Time) {
	switch kind {
	case demoExamActive:
		start := now.Add(-35 * time.Minute)
		return start, now.Add(90 * time.Minute), exams.StatusPublished, timePointer(now.AddDate(0, 0, -1))
	case demoExamHistorical:
		start := now.AddDate(0, 0, -(3 + courseIndex)).Truncate(time.Minute).Add(-2 * time.Hour)
		end := start.Add(75 * time.Minute)
		published := end.Add(3 * time.Hour)
		return start, end, exams.StatusClosed, &published
	case demoExamUpcoming:
		start := now.AddDate(0, 0, 2+courseIndex%5).Truncate(time.Minute)
		return start, start.Add(90 * time.Minute), exams.StatusPublished, timePointer(now.AddDate(0, 0, -2))
	default:
		start := now.AddDate(0, 0, 14+courseIndex%4).Truncate(time.Minute)
		return start, start.Add(75 * time.Minute), exams.StatusDraft, nil
	}
}

func demoExamTitle(course demoCourse, kind int) string {
	grade := course.class.Name
	switch kind {
	case demoExamActive:
		return fmt.Sprintf("Uji Kompetensi %s · %s", course.subject.Name, grade)
	case demoExamHistorical:
		return fmt.Sprintf("Evaluasi Tengah Semester %s · %s", course.subject.Name, grade)
	case demoExamUpcoming:
		return fmt.Sprintf("Asesmen Bab Berikutnya %s · %s", course.subject.Name, grade)
	default:
		return fmt.Sprintf("Simulasi Akhir Semester %s · %s", course.subject.Name, grade)
	}
}

func demoExamDescription(course demoCourse, kind int) string {
	switch kind {
	case demoExamActive:
		return fmt.Sprintf("Asesmen latihan yang sedang berlangsung untuk mengukur pemahaman %s.", course.subject.Name)
	case demoExamHistorical:
		return fmt.Sprintf("Rekap latihan %s dengan hasil yang sudah dinilai dan dipublikasikan.", course.subject.Name)
	case demoExamUpcoming:
		return fmt.Sprintf("Jadwal asesmen berikutnya untuk kelas %s.", course.class.Name)
	default:
		return fmt.Sprintf("Draf simulasi untuk materi lanjutan %s.", course.subject.Name)
	}
}

func demoExamQuestionIDStart(courseIndex, kind int) int {
	if courseIndex == 0 {
		return [...]int{400, 500, 600, 700}[kind]
	}
	return 800 + courseIndex*100 + kind*10
}

func seedDemoActiveAttempt(tx *gorm.DB, course demoCourse, exam demoExam, now time.Time) error {
	student := course.students[0].user
	startedAt := now.Add(-18 * time.Minute)
	attempt := attempts.Attempt{
		ID:                  seedID(700),
		ExamID:              exam.exam.ID,
		StudentID:           student.ID,
		Status:              attempts.StatusInProgress,
		StartIdempotencyKey: "seed-active-attempt",
		StartedAt:           startedAt,
		DeadlineAt:          now.Add(42 * time.Minute),
	}
	if err := seedRecord(tx, &attempt); err != nil {
		return fmt.Errorf("active attempt: %w", err)
	}
	firstQuestion := seedID(exam.questionIDStart)
	firstOption := seedID(exam.questionIDStart * 10)
	answer := attempts.Answer{
		ID:               seedID(701),
		AttemptID:        attempt.ID,
		ExamQuestionID:   firstQuestion,
		SelectedOptionID: firstOption,
		Revision:         1,
		SavedAt:          now.Add(-12 * time.Minute),
	}
	if err := seedRecord(tx, &answer); err != nil {
		return fmt.Errorf("active answer: %w", err)
	}
	return seedDemoAttemptEvent(tx, 702, attempt.ID, student.ID, "answer.saved", answer.SavedAt)
}

func seedDemoHistoricalAttempts(tx *gorm.DB, course demoCourse, exam demoExam) error {
	for _, student := range course.students {
		studentID := student.user.ID
		position := student.classNumber
		attemptSuffix := demoHistoricalAttemptIDSuffix(course.index, position)
		answerSuffix := 9000 + course.index*100 + position*10
		receiptSuffix := 6000 + course.index*100 + position
		resultSuffix := demoHistoricalResultIDSuffix(course.index, position)
		eventSuffix := 8000 + course.index*100 + position
		idempotencyKey := fmt.Sprintf("seed-historical-c%d-s%d", course.index, position)
		if course.index == 0 {
			answerSuffix = 800 + position*10
			receiptSuffix = 750 + position
			eventSuffix = 850 + position
			idempotencyKey = fmt.Sprintf("seed-historical-%d", position)
		}

		startedAt := exam.exam.StartsAt.Add(time.Duration(position) * time.Minute)
		submittedAt := startedAt.Add(55 * time.Minute)
		receipt := seedID(receiptSuffix)
		attempt := attempts.Attempt{
			ID:                  seedID(attemptSuffix),
			ExamID:              exam.exam.ID,
			StudentID:           studentID,
			Status:              attempts.StatusSubmitted,
			StartIdempotencyKey: idempotencyKey,
			StartedAt:           startedAt,
			DeadlineAt:          startedAt.Add(time.Duration(exam.exam.DurationMinutes) * time.Minute),
			SubmittedAt:         &submittedAt,
			SubmissionReceipt:   &receipt,
		}
		if err := seedRecord(tx, &attempt); err != nil {
			return fmt.Errorf("historical attempt for %s: %w", studentID, err)
		}

		correctCount := 1 + (position*3+course.index*2)%len(course.questions)
		for questionIndex, question := range course.questions {
			wantsCorrect := questionIndex < correctCount
			selectedPosition := 0
			for optionIndex, option := range question.Options {
				if option.IsCorrect == wantsCorrect {
					selectedPosition = optionIndex
					break
				}
			}
			answer := attempts.Answer{
				ID:               seedID(answerSuffix + questionIndex),
				AttemptID:        attempt.ID,
				ExamQuestionID:   seedID(exam.questionIDStart + questionIndex),
				SelectedOptionID: seedID(exam.questionIDStart*10 + questionIndex*10 + selectedPosition),
				Revision:         1,
				SavedAt:          submittedAt.Add(-time.Duration(len(course.questions)-questionIndex) * time.Minute),
			}
			if err := seedRecord(tx, &answer); err != nil {
				return fmt.Errorf("historical answer for %s: %w", studentID, err)
			}
		}
		if err := seedDemoAttemptEvent(tx, eventSuffix, attempt.ID, studentID, "attempt.submitted", submittedAt); err != nil {
			return err
		}

		gradedAt := exam.exam.EndsAt.Add(4*time.Hour + time.Duration(position)*3*time.Minute)
		result := results.Result{
			ID:          seedID(resultSuffix),
			AttemptID:   attempt.ID,
			ExamID:      exam.exam.ID,
			StudentID:   studentID,
			Status:      "published",
			Score:       float64(correctCount * 20),
			MaxScore:    float64(len(course.questions) * 20),
			GradedAt:    gradedAt,
			PublishedAt: &gradedAt,
		}
		if err := seedRecord(tx, &result); err != nil {
			return fmt.Errorf("historical result for %s: %w", studentID, err)
		}
	}
	return nil
}

func seedDemoAttemptEvent(tx *gorm.DB, suffix int, attemptID, actorID, eventType string, createdAt time.Time) error {
	event := attempts.Event{
		ID:        seedID(suffix),
		AttemptID: attemptID,
		ActorID:   actorID,
		EventType: eventType,
		Metadata:  []byte(`{"source":"development_seed"}`),
		CreatedAt: createdAt,
	}
	if err := seedRecord(tx, &event); err != nil {
		return fmt.Errorf("attempt event %s: %w", eventType, err)
	}
	return nil
}

func timePointer(value time.Time) *time.Time { return &value }

func demoHistoricalAttemptIDSuffix(courseIndex, studentPosition int) int {
	if courseIndex == 0 {
		return 710 + studentPosition
	}
	return 2000 + courseIndex*100 + studentPosition
}

func demoHistoricalResultIDSuffix(courseIndex, studentPosition int) int {
	if courseIndex == 0 {
		return 900 + studentPosition
	}
	return 7000 + courseIndex*100 + studentPosition
}
