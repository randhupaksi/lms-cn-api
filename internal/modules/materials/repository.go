package materials

import (
	"context"
	"errors"
	"strings"
	"time"

	"lms-cn-api/pkg/pagination"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNotFound           = errors.New("material not found")
	ErrLocked             = errors.New("material locked")
	ErrAttachmentNotFound = errors.New("material attachment not found")
	ErrAttachmentLimit    = errors.New("material attachment limit reached")
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, material *Material) error {
	return r.db.WithContext(ctx).Create(material).Error
}

func (r *Repository) Find(ctx context.Context, id string) (Material, error) {
	var material Material
	if err := r.db.WithContext(ctx).First(&material, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Material{}, ErrNotFound
		}
		return Material{}, err
	}
	return material, nil
}

func (r *Repository) ListAttachments(ctx context.Context, materialIDs []string) ([]Attachment, error) {
	attachments := make([]Attachment, 0)
	if len(materialIDs) == 0 {
		return attachments, nil
	}
	err := r.db.WithContext(ctx).
		Select("id", "material_id", "file_name", "content_type", "size_bytes", "checksum_sha256", "created_at").
		Where("material_id IN ?", materialIDs).
		Order("created_at ASC, id ASC").
		Find(&attachments).Error
	return attachments, err
}

func (r *Repository) FindAttachment(ctx context.Context, materialID, attachmentID string) (Attachment, error) {
	var attachment Attachment
	err := r.db.WithContext(ctx).Where("material_id = ? AND id = ?", materialID, attachmentID).First(&attachment).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Attachment{}, ErrAttachmentNotFound
	}
	return attachment, err
}

func (r *Repository) CreateAttachment(ctx context.Context, attachment *Attachment) (Attachment, error) {
	var saved Attachment
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var material Material
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&material, "id = ?", attachment.MaterialID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if material.Status != StatusDraft {
			return ErrLocked
		}

		err := tx.Select("id", "material_id", "file_name", "content_type", "size_bytes", "checksum_sha256", "created_at").
			Where("material_id = ? AND checksum_sha256 = ?", attachment.MaterialID, attachment.ChecksumSHA256).
			First(&saved).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var count int64
		if err := tx.Model(&Attachment{}).Where("material_id = ?", attachment.MaterialID).Count(&count).Error; err != nil {
			return err
		}
		var totalBytes uint64
		if err := tx.Model(&Attachment{}).Where("material_id = ?", attachment.MaterialID).Select("COALESCE(SUM(size_bytes), 0)").Scan(&totalBytes).Error; err != nil {
			return err
		}
		if count >= maxMaterialAttachments || totalBytes+attachment.SizeBytes > maxMaterialAttachmentTotal {
			return ErrAttachmentLimit
		}
		if err := tx.Create(attachment).Error; err != nil {
			return err
		}
		saved = *attachment
		return nil
	})
	return saved, err
}

func (r *Repository) DeleteAttachment(ctx context.Context, materialID, attachmentID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var material Material
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&material, "id = ?", materialID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if material.Status != StatusDraft {
			return ErrLocked
		}
		result := tx.Where("material_id = ? AND id = ?", materialID, attachmentID).Delete(&Attachment{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrAttachmentNotFound
		}
		return nil
	})
}

func (r *Repository) ListPaged(ctx context.Context, courseID, role, userID string, page pagination.Request, filter ListFilter) ([]materialRow, int64, error) {
	query := r.db.WithContext(ctx).Table("course_materials m").
		Select("m.*, c.name AS course_name").
		Joins("JOIN courses c ON c.id = m.course_id")
	switch role {
	case "teacher", "curriculum":
		query = query.Joins("JOIN course_teachers ct ON ct.course_id = m.course_id AND ct.teacher_id = ?", userID)
	case "student":
		query = query.Joins("JOIN course_students cs ON cs.course_id = m.course_id AND cs.student_id = ?", userID).
			Select("m.*, c.name AS course_name, mp.completed_at").
			Joins("LEFT JOIN material_progress mp ON mp.material_id = m.id AND mp.student_id = ?", userID).
			Where("m.status = ?", StatusPublished)
	case "admin", "superadmin", "principal":
	default:
		return nil, 0, errors.New("unsupported material list role")
	}
	if courseID != "" {
		query = query.Where("m.course_id = ?", courseID)
	}
	if filter.Search != "" {
		pattern := "%" + strings.TrimSpace(filter.Search) + "%"
		query = query.Where("m.title LIKE ? OR m.description LIKE ?", pattern, pattern)
	}
	if filter.Status != "" {
		query = query.Where("m.status = ?", filter.Status)
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Distinct("m.id").Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []materialRow
	err := query.Order("c.name ASC, m.position ASC, m.created_at ASC").
		Offset(page.Offset()).Limit(page.PerPage).Scan(&rows).Error
	return rows, total, err
}

func (r *Repository) UpdateDraft(ctx context.Context, id string, apply func(*Material)) (Material, error) {
	var result Material
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var material Material
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&material, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if material.Status != StatusDraft {
			return ErrLocked
		}
		apply(&material)
		if err := tx.Save(&material).Error; err != nil {
			return err
		}
		result = material
		return nil
	})
	return result, err
}

func (r *Repository) Publish(ctx context.Context, id string, now time.Time) error {
	result := r.db.WithContext(ctx).Model(&Material{}).Where("id = ? AND status = ?", id, StatusDraft).Updates(map[string]any{"status": StatusPublished, "published_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrLocked
	}
	return nil
}

func (r *Repository) Complete(ctx context.Context, materialID, studentID string, now time.Time) error {
	progress := Progress{MaterialID: materialID, StudentID: studentID, CompletedAt: now}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "material_id"}, {Name: "student_id"}}, DoUpdates: clause.AssignmentColumns([]string{"completed_at"})}).Create(&progress).Error
}
