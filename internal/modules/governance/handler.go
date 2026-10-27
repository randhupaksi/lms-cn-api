package governance

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"github.com/gin-gonic/gin"
	"lms-cn-api/internal/middleware"
	"lms-cn-api/pkg/apperror"
	"lms-cn-api/pkg/pagination"
	"lms-cn-api/pkg/request"
	"lms-cn-api/pkg/response"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

func (h *Handler) RegisterPublicRoutes(group *gin.RouterGroup) {
	group.GET("/school-profile", func(c *gin.Context) { value, err := h.service.PublicProfile(c.Request.Context()); reply(c, value, err) })
}
func (h *Handler) RegisterRoutes(group *gin.RouterGroup) {
	group.Use(middleware.RequireRoles("superadmin", "admin", "teacher"))
	group.GET("/requests", h.requests)
	group.POST("/requests", h.createRequest)
	group.POST("/requests/:id/decision", h.decide)
	group.POST("/requests/:id/cancel", h.cancel)
	group.GET("/request-exams", h.exams)
	owner := group.Group("")
	owner.Use(middleware.RequireRoles("superadmin"))
	owner.GET("/overview", h.overview)
	owner.GET("/exams", h.exams)
	owner.GET("/quality", h.quality)
	owner.GET("/reports", h.reports)
	owner.GET("/reports/export", h.export)
	owner.GET("/settings", h.settings)
	owner.PUT("/settings", h.saveSettings)
	owner.GET("/settings/history", h.history)
	owner.GET("/events", h.events)
	owner.POST("/events", h.createEvent)
	owner.PUT("/events/:id", h.updateEvent)
	owner.POST("/events/:id/archive", h.archive)
}
func reply(c *gin.Context, data any, err error) {
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Data tata kelola berhasil diproses", data)
}
func (h *Handler) overview(c *gin.Context) {
	a, _ := middleware.Principal(c)
	v, e := h.service.Overview(c.Request.Context(), a)
	reply(c, v, e)
}
func (h *Handler) quality(c *gin.Context) {
	a, _ := middleware.Principal(c)
	v, e := h.service.Quality(c.Request.Context(), a)
	reply(c, v, e)
}
func (h *Handler) exams(c *gin.Context) {
	a, _ := middleware.Principal(c)
	p := pagination.FromContext(c)
	v, n, e := h.service.Exams(c.Request.Context(), a, p.Page, p.PerPage, c.Query("search"))
	if e != nil {
		response.FromError(c, e)
		return
	}
	response.SuccessWithMeta(c, 200, "Kesiapan ujian berhasil dimuat", v, p.Meta(n))
}
func (h *Handler) requests(c *gin.Context) {
	a, _ := middleware.Principal(c)
	p := pagination.FromContext(c)
	v, n, e := h.service.Requests(c.Request.Context(), a, c.Query("status"), p.Page, p.PerPage)
	if e != nil {
		response.FromError(c, e)
		return
	}
	response.SuccessWithMeta(c, 200, "Pengajuan berhasil dimuat", v, p.Meta(n))
}
func (h *Handler) createRequest(c *gin.Context) {
	var body ScheduleInput
	if e := request.BindJSON(c, &body); e != nil {
		response.FromError(c, e)
		return
	}
	a, _ := middleware.Principal(c)
	reply(c, nil, h.service.CreateRequest(c.Request.Context(), a, body))
}
func (h *Handler) decide(c *gin.Context) {
	var body DecisionInput
	if e := request.BindJSON(c, &body); e != nil {
		response.FromError(c, e)
		return
	}
	a, _ := middleware.Principal(c)
	reply(c, nil, h.service.Decide(c.Request.Context(), a, c.Param("id"), body))
}
func (h *Handler) cancel(c *gin.Context) {
	a, _ := middleware.Principal(c)
	reply(c, nil, h.service.Cancel(c.Request.Context(), a, c.Param("id")))
}
func (h *Handler) settings(c *gin.Context) {
	a, _ := middleware.Principal(c)
	v, e := h.service.Settings(c.Request.Context(), a)
	reply(c, v, e)
}
func (h *Handler) saveSettings(c *gin.Context) {
	var body SettingsInput
	if e := request.BindJSON(c, &body); e != nil {
		response.FromError(c, e)
		return
	}
	a, _ := middleware.Principal(c)
	reply(c, nil, h.service.SaveSettings(c.Request.Context(), a, body))
}
func (h *Handler) history(c *gin.Context) {
	a, _ := middleware.Principal(c)
	v, e := h.service.History(c.Request.Context(), a)
	reply(c, v, e)
}
func (h *Handler) events(c *gin.Context) {
	zone := time.FixedZone("Asia/Jakarta", 7*3600)
	now := time.Now().In(zone)
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, zone)
	to := from.AddDate(0, 1, 0)
	var err error
	if c.Query("from") != "" {
		from, err = time.ParseInLocation(time.DateOnly, c.Query("from"), zone)
		if err != nil {
			reply(c, nil, apperror.New(422, "INVALID_DATE", "Tanggal awal tidak valid"))
			return
		}
	}
	if c.Query("to") != "" {
		to, err = time.ParseInLocation(time.DateOnly, c.Query("to"), zone)
		if err != nil {
			reply(c, nil, apperror.New(422, "INVALID_DATE", "Tanggal akhir tidak valid"))
			return
		}
	}
	a, _ := middleware.Principal(c)
	v, e := h.service.Agenda(c.Request.Context(), a, from.UTC(), to.UTC())
	reply(c, v, e)
}
func (h *Handler) createEvent(c *gin.Context) { h.saveEvent(c, "") }
func (h *Handler) updateEvent(c *gin.Context) { h.saveEvent(c, c.Param("id")) }
func (h *Handler) saveEvent(c *gin.Context, id string) {
	var body EventInput
	if e := request.BindJSON(c, &body); e != nil {
		response.FromError(c, e)
		return
	}
	a, _ := middleware.Principal(c)
	reply(c, nil, h.service.SaveEvent(c.Request.Context(), a, id, body))
}
func (h *Handler) archive(c *gin.Context) {
	var body struct {
		Version int `json:"version" binding:"required,min=1"`
	}
	if e := request.BindJSON(c, &body); e != nil {
		response.FromError(c, e)
		return
	}
	a, _ := middleware.Principal(c)
	reply(c, nil, h.service.ArchiveEvent(c.Request.Context(), a, c.Param("id"), body.Version))
}
func (h *Handler) reports(c *gin.Context) {
	a, _ := middleware.Principal(c)
	p := pagination.FromContext(c)
	v, n, e := h.service.Reports(c.Request.Context(), a, c.Query("year_id"), c.Query("class_id"), p.Page, p.PerPage)
	if e != nil {
		response.FromError(c, e)
		return
	}
	response.SuccessWithMeta(c, 200, "Laporan berhasil dimuat", v, p.Meta(n))
}
func safeCSV(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) {
		return "'" + value
	}
	return value
}
func (h *Handler) export(c *gin.Context) {
	a, _ := middleware.Principal(c)
	rows, total, err := h.service.Reports(c.Request.Context(), a, c.Query("year_id"), c.Query("class_id"), 1, 10000)
	if err != nil {
		response.FromError(c, err)
		return
	}
	if total > 10000 {
		reply(c, nil, apperror.New(422, "EXPORT_TOO_LARGE", "Persempit filter agar laporan tidak melebihi 10.000 course"))
		return
	}
	var output bytes.Buffer
	w := csv.NewWriter(&output)
	_ = w.Write([]string{"Tahun ajaran", "Kelas", "Course", "Siswa", "Guru", "Ujian", "Attempt selesai", "Hasil publik", "Rerata hasil publik (%)", "Di bawah referensi", "Referensi ketuntasan (%)", "Tugas", "Perlu dinilai"})
	for _, row := range rows {
		avg := ""
		if row.AverageScore != nil {
			avg = fmt.Sprintf("%.2f", *row.AverageScore)
		}
		_ = w.Write([]string{safeCSV(row.YearName), safeCSV(row.ClassName), safeCSV(row.Name), strconv.FormatInt(row.Students, 10), strconv.FormatInt(row.Teachers, 10), strconv.FormatInt(row.Exams, 10), strconv.FormatInt(row.SubmittedAttempts, 10), strconv.FormatInt(row.PublishedResults, 10), avg, strconv.FormatInt(row.BelowReference, 10), fmt.Sprintf("%.2f", row.PassingReference), strconv.FormatInt(row.Assignments, 10), strconv.FormatInt(row.PendingGrading, 10)})
	}
	w.Flush()
	if err = w.Error(); err != nil {
		reply(c, nil, failure(err))
		return
	}
	c.Header("Content-Disposition", "attachment; filename=citra-negara-course-report.csv")
	c.Data(200, "text/csv; charset=utf-8", output.Bytes())
}
