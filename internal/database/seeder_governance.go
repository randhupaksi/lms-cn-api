package database

import (
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"lms-cn-api/internal/modules/governance"
	"time"
)

func seedDemoGovernance(tx *gorm.DB, d demoDataset, now time.Time) error {
	zone := time.FixedZone("Asia/Jakarta", 7*3600)
	today := now.In(zone)
	for i, definition := range []struct {
		title, description, category string
		day                          int
	}{
		{"Rapat kesiapan asesmen akhir", "Koordinasi guru pengampu, operator, dan tim kurikulum.", "meeting", 1},
		{"Pekan pengayaan kelas XII", "Pendampingan persiapan ujian kelulusan dan penyelesaian tugas.", "academic", 3},
		{"Evaluasi hasil belajar semester", "Tinjauan ketercapaian pembelajaran bersama wali kelas.", "academic", 7},
		{"Layanan bantuan LMS", "Sesi konsultasi akses akun dan penggunaan LMS di laboratorium.", "announcement", 10},
		{"Libur kegiatan akademik", "Tidak ada kegiatan belajar terjadwal pada hari ini.", "holiday", 14},
	} {
		start := time.Date(today.Year(), today.Month(), today.Day()+definition.day, 9, 0, 0, 0, zone).UTC()
		event := governance.Event{ID: seedID(900001 + i), Title: definition.title, Description: definition.description, Category: definition.category, StartsAt: start, EndsAt: start.Add(2 * time.Hour), Status: "active", CreatedBy: d.superadmin.ID, Version: 1, CreatedAt: now, UpdatedAt: now}
		var previous governance.Event
		if err := tx.First(&previous, "id=?", event.ID).Error; err == nil {
			if previous.Version != 1 || previous.Status != "active" {
				continue
			}
			event.CreatedAt = previous.CreatedAt
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Save(&event).Error; err != nil {
			return err
		}
	}
	for i := 0; i < 3 && i < len(d.courses); i++ {
		exam := d.courses[i].exams[demoExamUpcoming].exam
		value := governance.ScheduleRequest{ID: seedID(900100 + i), ExamID: exam.ID, RequesterID: d.courses[i].teachers[0].ID, Reason: []string{"Penyesuaian jadwal penggunaan laboratorium untuk ujian praktik.", "Menghindari benturan kegiatan kelas dengan agenda pendampingan industri.", "Penyesuaian waktu agar seluruh peserta dapat mengikuti asesmen."}[i], OriginalStartsAt: exam.StartsAt, OriginalEndsAt: exam.EndsAt, ProposedStartsAt: exam.StartsAt.Add(24 * time.Hour), ProposedEndsAt: exam.EndsAt.Add(24 * time.Hour), Status: "pending", CreatedAt: now.Add(-time.Duration(3+i) * time.Hour)}
		if i == 2 {
			value.Status = "rejected"
			value.ReviewerID = stringPointer(d.superadmin.ID)
			value.DecisionNote = "Jadwal awal tetap digunakan setelah koordinasi dengan tim kurikulum."
			decided := now.Add(-time.Hour)
			value.DecidedAt = &decided
		}
		var existing governance.ScheduleRequest
		err := tx.First(&existing, "id=?", value.ID).Error
		if err == nil {
			if existing.Status != "pending" {
				continue
			}
			value.CreatedAt = existing.CreatedAt
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err = tx.Save(&value).Error; err != nil {
			return err
		}
	}
	var settings governance.Settings
	if err := tx.First(&settings, 1).Error; err != nil {
		return err
	}
	var historyCount int64
	if err := tx.Model(&governance.SettingChange{}).Where("id=?", seedID(900300)).Count(&historyCount).Error; err != nil {
		return err
	}
	if historyCount == 0 {
		snapshot, err := json.Marshal(settings)
		if err != nil {
			return err
		}
		if err = tx.Create(&governance.SettingChange{ID: seedID(900300), ActorID: d.superadmin.ID, Version: settings.Version, Reason: "Konfigurasi awal profil dan layanan sekolah untuk lingkungan development.", Snapshot: snapshot, CreatedAt: now}).Error; err != nil {
			return err
		}
	}
	return nil
}
