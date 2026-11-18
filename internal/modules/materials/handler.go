package materials

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"lms-cn-api/internal/authz"
	"lms-cn-api/internal/middleware"
	"lms-cn-api/pkg/apperror"
	"lms-cn-api/pkg/pagination"
	"lms-cn-api/pkg/request"
	"lms-cn-api/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("", h.list)
	group.POST("", h.create)
	group.PUT("/:materialID", h.update)
	group.POST("/:materialID/attachments", h.uploadAttachment)
	group.GET("/:materialID/attachments/:attachmentID", h.getAttachment)
	group.DELETE("/:materialID/attachments/:attachmentID", h.deleteAttachment)
	group.POST("/:materialID/publish", h.publish)
	group.POST("/:materialID/complete", h.complete)
}

func (h *Handler) uploadAttachment(c *gin.Context) {
	principal, _ := middleware.Principal(c)
	if !principal.IsTeachingStaff() {
		response.FromError(c, apperror.New(http.StatusForbidden, "MATERIAL_WRITE_DENIED", "Hanya guru yang ditugaskan dapat mengelola materi"))
		return
	}
	maxRequestBytes := int64(maxMaterialAttachmentBytes + (1 << 20))
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBytes)
	fileHeader, err := c.FormFile("file")
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			response.FromError(c, apperror.New(http.StatusRequestEntityTooLarge, "REQUEST_BODY_TOO_LARGE", "Ukuran unggahan melebihi batas"))
			return
		}
		response.FromError(c, apperror.New(http.StatusBadRequest, "ATTACHMENT_REQUIRED", "Pilih satu file lampiran untuk diunggah"))
		return
	}
	if fileHeader.Size > maxMaterialAttachmentBytes {
		response.FromError(c, apperror.New(http.StatusRequestEntityTooLarge, "ATTACHMENT_SIZE_INVALID", "Ukuran lampiran maksimal 8 MiB"))
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		response.FromError(c, apperror.New(http.StatusBadRequest, "ATTACHMENT_READ_FAILED", "File lampiran tidak dapat dibaca"))
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxMaterialAttachmentBytes+1))
	if err != nil {
		response.FromError(c, apperror.New(http.StatusBadRequest, "ATTACHMENT_READ_FAILED", "File lampiran tidak dapat dibaca"))
		return
	}
	if len(data) > maxMaterialAttachmentBytes {
		response.FromError(c, apperror.New(http.StatusRequestEntityTooLarge, "ATTACHMENT_SIZE_INVALID", "Ukuran lampiran maksimal 8 MiB"))
		return
	}
	materialID, err := request.RequireID(c, "materialID")
	if err != nil {
		response.FromError(c, err)
		return
	}
	attachment, err := h.service.UploadAttachment(c.Request.Context(), principal, materialID, fileHeader.Filename, data)
	respond(c, http.StatusCreated, "Lampiran berhasil diunggah", attachment, err)
}

func (h *Handler) getAttachment(c *gin.Context) {
	materialID, err := request.RequireID(c, "materialID")
	if err != nil {
		response.FromError(c, err)
		return
	}
	attachmentID, err := request.RequireID(c, "attachmentID")
	if err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	attachment, err := h.service.GetAttachment(c.Request.Context(), principal, materialID, attachmentID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	disposition := "attachment"
	if strings.HasPrefix(attachment.ContentType, "image/") {
		disposition = "inline"
	}
	if value := mime.FormatMediaType(disposition, map[string]string{"filename": attachment.FileName}); value != "" {
		c.Header("Content-Disposition", value)
	}
	c.Header("Cache-Control", "private, no-store")
	c.Data(http.StatusOK, attachment.ContentType, attachment.FileData)
}

func (h *Handler) deleteAttachment(c *gin.Context) {
	materialID, err := request.RequireID(c, "materialID")
	if err != nil {
		response.FromError(c, err)
		return
	}
	attachmentID, err := request.RequireID(c, "attachmentID")
	if err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	if err := h.service.DeleteAttachment(c.Request.Context(), principal, materialID, attachmentID); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Lampiran berhasil dihapus", nil)
}

func (h *Handler) list(c *gin.Context) {
	courseID := c.Query("course_id")
	principal, _ := middleware.Principal(c)
	page := pagination.FromContext(c)
	filter := ListFilter{Search: c.Query("search"), Status: c.Query("status")}
	data, total, err := h.service.ListPaged(c.Request.Context(), principal, courseID, page, filter)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.SuccessWithMeta(c, http.StatusOK, "Materi berhasil dimuat", data, page.Meta(total))
}

func (h *Handler) create(c *gin.Context) { h.write(c, true) }
func (h *Handler) update(c *gin.Context) { h.write(c, false) }

func (h *Handler) write(c *gin.Context, create bool) {
	var body WriteRequest
	if err := request.BindJSON(c, &body); err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	if create {
		data, err := h.service.Create(c.Request.Context(), principal, body)
		respond(c, http.StatusCreated, "Materi berhasil dibuat", data, err)
		return
	}
	id, err := request.RequireID(c, "materialID")
	if err != nil {
		response.FromError(c, err)
		return
	}
	data, err := h.service.Update(c.Request.Context(), principal, id, body)
	respond(c, http.StatusOK, "Materi berhasil diperbarui", data, err)
}

func (h *Handler) publish(c *gin.Context) {
	h.action(c, h.service.Publish, "Materi berhasil dipublikasikan")
}
func (h *Handler) complete(c *gin.Context) {
	h.action(c, h.service.Complete, "Progress materi berhasil disimpan")
}

func (h *Handler) action(c *gin.Context, action func(ctx context.Context, actor authz.Principal, id string) error, message string) {
	id, err := request.RequireID(c, "materialID")
	if err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	if err := action(c.Request.Context(), principal, id); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusOK, message, nil)
}

func respond(c *gin.Context, status int, message string, data any, err error) {
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, status, message, data)
}
