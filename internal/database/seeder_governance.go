package database

import (
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"lms-cn-api/internal/modules/governance"
	"time"
)

func seedDemoGovernance(tx *gorm.DB, d demoDataset, now time.Time) error {
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
