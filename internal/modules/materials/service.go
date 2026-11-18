package materials

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"lms-cn-api/internal/authz"
	"lms-cn-api/internal/modules/academics"
	"lms-cn-api/internal/modules/audit"
	"lms-cn-api/internal/modules/users"
	"lms-cn-api/pkg/apperror"
	"lms-cn-api/pkg/pagination"

	"github.com/google/uuid"
)

type Service struct {
	repository *Repository
	academics  academics.CourseAccess
	audit      audit.Recorder
	now        func() time.Time
}

func NewService(repository *Repository, academicsService academics.CourseAccess, auditRecorder audit.Recorder) *Service {
	return &Service{repository: repository, academics: academicsService, audit: auditRecorder, now: time.Now}
}

func (s *Service) List(ctx context.Context, actor authz.Principal, courseID string) ([]Response, error) {
	studentID := ""
	publishedOnly := false
	if actor.Role == "student" {
		if err := s.academics.RequireCourseStudent(ctx, actor, courseID); err != nil {
			return nil, err
		}
		studentID, publishedOnly = actor.UserID, true
	} else if err := s.academics.RequireCourseManager(ctx, actor, courseID); err != nil {
		return nil, err
	}
	rows, err := s.repository.List(ctx, courseID, studentID, publishedOnly)
	if err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, "MATERIALS_READ_FAILED", "Gagal memuat materi", err)
	}
	if err := s.includeAttachments(ctx, rows); err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, "MATERIALS_READ_FAILED", "Gagal memuat materi", err)
	}
	result := make([]Response, len(rows))
	for index, row := range rows {
		result[index] = toResponse(row)
	}
	return result, nil
}

func (s *Service) ListPaged(ctx context.Context, actor authz.Principal, courseID string, page pagination.Request, filter ListFilter) ([]Response, int64, error) {
	if courseID != "" {
		if actor.Role == string(users.RoleStudent) {
			if err := s.academics.RequireCourseStudent(ctx, actor, courseID); err != nil {
				return nil, 0, err
			}
		} else if err := s.academics.RequireCourseManager(ctx, actor, courseID); err != nil {
			return nil, 0, err
		}
	} else if err := actor.RequireRole(string(users.RoleSuperadmin), string(users.RolePrincipal), string(users.RoleCurriculum), string(users.RoleAdmin), string(users.RoleTeacher), string(users.RoleStudent)); err != nil {
		return nil, 0, err
	}
	filter.Search = strings.TrimSpace(filter.Search)
	if filter.Status != "" && filter.Status != StatusDraft && filter.Status != StatusPublished {
		return nil, 0, apperror.New(http.StatusBadRequest, "MATERIAL_STATUS_INVALID", "Status materi tidak valid")
	}
	if actor.Role == string(users.RoleStudent) && filter.Status != "" && filter.Status != StatusPublished {
		return nil, 0, apperror.New(http.StatusBadRequest, "MATERIAL_STATUS_INVALID", "Status materi tidak valid")
	}
	rows, total, err := s.repository.ListPaged(ctx, courseID, actor.Role, actor.UserID, page, filter)
	if err != nil {
		return nil, 0, apperror.Wrap(http.StatusInternalServerError, "MATERIALS_READ_FAILED", "Gagal memuat materi", err)
	}
	if err := s.includeAttachments(ctx, rows); err != nil {
		return nil, 0, apperror.Wrap(http.StatusInternalServerError, "MATERIALS_READ_FAILED", "Gagal memuat materi", err)
	}
	result := make([]Response, len(rows))
	for index, row := range rows {
		result[index] = toResponse(row)
	}
	return result, total, nil
}

func (s *Service) includeAttachments(ctx context.Context, rows []materialRow) error {
	materialIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		materialIDs = append(materialIDs, row.ID)
	}
	attachments, err := s.repository.ListAttachments(ctx, materialIDs)
	if err != nil {
		return err
	}
	byMaterial := make(map[string][]Attachment, len(rows))
	for _, attachment := range attachments {
		byMaterial[attachment.MaterialID] = append(byMaterial[attachment.MaterialID], attachment)
	}
	for index := range rows {
		rows[index].Attachments = byMaterial[rows[index].ID]
	}
	return nil
}

func (s *Service) Create(ctx context.Context, actor authz.Principal, request WriteRequest) (Response, error) {
	if !actor.IsTeachingStaff() {
		return Response{}, apperror.New(http.StatusForbidden, "MATERIAL_WRITE_DENIED", "Hanya guru yang ditugaskan dapat mengelola materi")
	}
	if err := s.academics.RequireCourseManager(ctx, actor, request.CourseID); err != nil {
		return Response{}, err
	}
	material := Material{ID: uuid.NewString(), CourseID: request.CourseID, AuthorID: actor.UserID, Title: strings.TrimSpace(request.Title), Description: strings.TrimSpace(request.Description), Content: strings.TrimSpace(request.Content), Position: request.Position, Status: StatusDraft}
	if err := s.repository.Create(ctx, &material); err != nil {
		return Response{}, apperror.Wrap(http.StatusInternalServerError, "MATERIAL_CREATE_FAILED", "Gagal membuat materi", err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "material.created", EntityType: "material", EntityID: material.ID})
	return toResponse(materialRow{Material: material}), nil
}

func (s *Service) Update(ctx context.Context, actor authz.Principal, id string, request WriteRequest) (Response, error) {
	material, err := s.requireTeacherOwner(ctx, actor, id, request.CourseID)
	if err != nil {
		return Response{}, err
	}
	updated, err := s.repository.UpdateDraft(ctx, material.ID, func(value *Material) {
		value.Title = strings.TrimSpace(request.Title)
		value.Description = strings.TrimSpace(request.Description)
		value.Content = strings.TrimSpace(request.Content)
		value.Position = request.Position
	})
	if err != nil {
		return Response{}, mapError(err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "material.updated", EntityType: "material", EntityID: id})
	rows := []materialRow{{Material: updated}}
	if err := s.includeAttachments(ctx, rows); err != nil {
		return Response{}, apperror.Wrap(http.StatusInternalServerError, "MATERIALS_READ_FAILED", "Gagal memuat materi", err)
	}
	return toResponse(rows[0]), nil
}

func (s *Service) Publish(ctx context.Context, actor authz.Principal, id string) error {
	material, err := s.requireTeacherOwner(ctx, actor, id, "")
	if err != nil {
		return err
	}
	if err := s.repository.Publish(ctx, material.ID, s.now().UTC()); err != nil {
		return mapError(err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "material.published", EntityType: "material", EntityID: id})
	return nil
}

func (s *Service) Complete(ctx context.Context, actor authz.Principal, id string) error {
	if actor.Role != "student" {
		return apperror.New(http.StatusForbidden, "MATERIAL_PROGRESS_DENIED", "Progress materi hanya tersedia untuk siswa")
	}
	material, err := s.repository.Find(ctx, id)
	if err != nil {
		return mapError(err)
	}
	if material.Status != StatusPublished {
		return apperror.New(http.StatusNotFound, "MATERIAL_NOT_FOUND", "Materi tidak ditemukan")
	}
	if err := s.academics.RequireCourseStudent(ctx, actor, material.CourseID); err != nil {
		return err
	}
	if err := s.repository.Complete(ctx, id, actor.UserID, s.now().UTC()); err != nil {
		return apperror.Wrap(http.StatusInternalServerError, "MATERIAL_PROGRESS_FAILED", "Gagal menyimpan progress materi", err)
	}
	return nil
}

func (s *Service) UploadAttachment(ctx context.Context, actor authz.Principal, materialID, fileName string, data []byte) (AttachmentInfo, error) {
	if _, err := s.requireTeacherOwner(ctx, actor, materialID, ""); err != nil {
		return AttachmentInfo{}, err
	}
	name, contentType, err := validateAttachment(fileName, data)
	if err != nil {
		return AttachmentInfo{}, err
	}
	checksum := sha256.Sum256(data)
	attachment := Attachment{
		ID:             uuid.NewString(),
		MaterialID:     materialID,
		FileName:       name,
		ContentType:    contentType,
		SizeBytes:      uint64(len(data)),
		ChecksumSHA256: hex.EncodeToString(checksum[:]),
		FileData:       data,
		CreatedAt:      s.now().UTC(),
	}
	saved, err := s.repository.CreateAttachment(ctx, &attachment)
	if err != nil {
		return AttachmentInfo{}, mapError(err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "material.attachment.uploaded", EntityType: "material", EntityID: materialID})
	return toAttachmentInfo(saved), nil
}

func (s *Service) DeleteAttachment(ctx context.Context, actor authz.Principal, materialID, attachmentID string) error {
	if _, err := s.requireTeacherOwner(ctx, actor, materialID, ""); err != nil {
		return err
	}
	if err := s.repository.DeleteAttachment(ctx, materialID, attachmentID); err != nil {
		return mapError(err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "material.attachment.deleted", EntityType: "material", EntityID: materialID})
	return nil
}

func (s *Service) GetAttachment(ctx context.Context, actor authz.Principal, materialID, attachmentID string) (Attachment, error) {
	material, err := s.repository.Find(ctx, materialID)
	if err != nil {
		return Attachment{}, mapError(err)
	}
	switch {
	case actor.Role == string(users.RoleStudent):
		if material.Status != StatusPublished {
			return Attachment{}, apperror.New(http.StatusNotFound, "MATERIAL_NOT_FOUND", "Materi tidak ditemukan")
		}
		if err := s.academics.RequireCourseStudent(ctx, actor, material.CourseID); err != nil {
			return Attachment{}, err
		}
	case actor.IsTeachingStaff():
		if err := s.academics.RequireCourseManager(ctx, actor, material.CourseID); err != nil {
			return Attachment{}, err
		}
	case actor.IsSchoolAdmin():
	default:
		return Attachment{}, apperror.New(http.StatusForbidden, "MATERIAL_ATTACHMENT_READ_DENIED", "Kamu tidak memiliki izin untuk membuka lampiran ini")
	}
	attachment, err := s.repository.FindAttachment(ctx, materialID, attachmentID)
	if err != nil {
		return Attachment{}, mapError(err)
	}
	return attachment, nil
}

func (s *Service) requireTeacherOwner(ctx context.Context, actor authz.Principal, id, requestedCourseID string) (Material, error) {
	if !actor.IsTeachingStaff() {
		return Material{}, apperror.New(http.StatusForbidden, "MATERIAL_WRITE_DENIED", "Hanya guru yang ditugaskan dapat mengelola materi")
	}
	material, err := s.repository.Find(ctx, id)
	if err != nil {
		return Material{}, mapError(err)
	}
	if requestedCourseID != "" && requestedCourseID != material.CourseID {
		return Material{}, apperror.New(http.StatusConflict, "MATERIAL_COURSE_IMMUTABLE", "Course materi tidak dapat dipindahkan")
	}
	if err := s.academics.RequireCourseManager(ctx, actor, material.CourseID); err != nil {
		return Material{}, err
	}
	return material, nil
}

func mapError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return apperror.New(http.StatusNotFound, "MATERIAL_NOT_FOUND", "Materi tidak ditemukan")
	}
	if errors.Is(err, ErrLocked) {
		return apperror.New(http.StatusConflict, "MATERIAL_LOCKED", "Materi yang telah dipublikasikan tidak dapat diubah")
	}
	if errors.Is(err, ErrAttachmentNotFound) {
		return apperror.New(http.StatusNotFound, "MATERIAL_ATTACHMENT_NOT_FOUND", "Lampiran materi tidak ditemukan")
	}
	if errors.Is(err, ErrAttachmentLimit) {
		return apperror.New(http.StatusConflict, "MATERIAL_ATTACHMENT_LIMIT", "Lampiran materi maksimal 5 file dengan total ukuran 20 MiB")
	}
	return apperror.Wrap(http.StatusInternalServerError, "MATERIAL_WRITE_FAILED", "Gagal memproses materi", err)
}
