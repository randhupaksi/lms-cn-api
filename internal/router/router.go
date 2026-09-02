package router

import (
	"net/http"
	"time"

	"lms-cn-api/internal/config"
	"lms-cn-api/internal/middleware"
	"lms-cn-api/internal/modules/academics"
	"lms-cn-api/internal/modules/analytics"
	"lms-cn-api/internal/modules/assignments"
	"lms-cn-api/internal/modules/attempts"
	"lms-cn-api/internal/modules/audit"
	"lms-cn-api/internal/modules/auth"
	"lms-cn-api/internal/modules/exams"
	"lms-cn-api/internal/modules/grading"
	"lms-cn-api/internal/modules/materials"
	"lms-cn-api/internal/modules/monitoring"
	"lms-cn-api/internal/modules/questions"
	"lms-cn-api/internal/modules/results"
	"lms-cn-api/internal/modules/users"
	"lms-cn-api/pkg/response"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Engine struct {
	*gin.Engine
	ExpiryWorker *attempts.ExpiryWorker
}

func New(cfg *config.Config, db *gorm.DB) (*Engine, error) {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(gin.Recovery(), middleware.RequestID(), middleware.RequestLogger(), middleware.CORS(cfg.AllowedOrigins), gzip.Gzip(gzip.DefaultCompression))

	auditService := audit.NewService(db)
	usersRepository := users.NewRepository(db)
	usersService := users.NewService(usersRepository, auditService)
	authRepository := auth.NewRepository(db)
	tokenManager := auth.NewTokenManager(cfg.JWTSecret, cfg.AppName, cfg.AccessTokenTTL)
	loginGuard := auth.NewLoginGuard(cfg.LoginAccountRateLimit, cfg.LoginAccountRateWindow)
	authService := auth.NewService(authRepository, usersRepository, tokenManager, auditService, cfg.AccessTokenTTL, cfg.RefreshTokenTTL, loginGuard)
	authHandler := auth.NewHandler(authService, cfg)

	academicsService := academics.NewService(academics.NewRepository(db), auditService)
	questionsService := questions.NewService(questions.NewRepository(db), academicsService, auditService)
	examsService := exams.NewService(exams.NewRepository(db), academicsService, auditService)
	attemptsService := attempts.NewService(attempts.NewRepository(db), grading.DefaultCalculator{})
	resultsService := results.NewService(results.NewRepository(db), academicsService, auditService)
	materialsService := materials.NewService(materials.NewRepository(db), academicsService, auditService)
	assignmentsService := assignments.NewService(assignments.NewRepository(db), academicsService, auditService)
	monitoringService := monitoring.NewService(monitoring.NewRepository(db), academicsService)
	analyticsService := analytics.NewService(analytics.NewRepository(db), academicsService)

	api := engine.Group(cfg.APIPrefix)
	api.GET("/health", func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil || sqlDB.PingContext(c.Request.Context()) != nil {
			response.Error(c, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "Layanan belum siap menerima permintaan")
			return
		}
		response.Success(c, http.StatusOK, "Citra Negara LMS API is ready", gin.H{"service": cfg.AppName, "environment": cfg.AppEnv, "status": "ready"})
	})
	api.GET("/health/live", func(c *gin.Context) {
		response.Success(c, http.StatusOK, "Citra Negara LMS API is alive", gin.H{"status": "alive"})
	})
	authPublic := api.Group("/auth")
	authHandler.RegisterPublicRoutes(authPublic, middleware.NewRateLimiter(cfg.LoginRateLimit, cfg.LoginRateWindow).Middleware())

	protected := api.Group("")
	protected.Use(middleware.Authenticate(authService))
	authHandler.RegisterProtectedRoutes(protected.Group("/auth"))

	adminUsers := protected.Group("/users")
	adminUsers.Use(middleware.RequireRoles(string(users.RoleAdmin)))
	users.NewHandler(usersService).RegisterRoutes(adminUsers)
	academics.NewHandler(academicsService).RegisterRoutes(protected)
	questions.NewHandler(questionsService).RegisterRoutes(protected.Group("/questions"))
	exams.NewHandler(examsService).RegisterRoutes(protected.Group("/exams"))
	materials.NewHandler(materialsService).RegisterRoutes(protected.Group("/materials"))
	assignments.NewHandler(assignmentsService).RegisterRoutes(protected.Group("/assignments"))
	analytics.NewHandler(analyticsService).RegisterRoutes(protected.Group("/analytics"))

	student := protected.Group("/student")
	student.Use(middleware.RequireRoles(string(users.RoleStudent)))
	attempts.NewHandler(attemptsService).RegisterRoutes(student)
	results.NewHandler(resultsService).RegisterStudentRoutes(student.Group("/results"))

	staffResults := protected.Group("/results")
	staffResults.Use(middleware.RequireRoles(string(users.RoleAdmin), string(users.RoleTeacher)))
	results.NewHandler(resultsService).RegisterStaffRoutes(staffResults)

	staffMonitoring := protected.Group("/monitoring")
	staffMonitoring.Use(middleware.RequireRoles(string(users.RoleAdmin), string(users.RoleTeacher)))
	monitoring.NewHandler(monitoringService).RegisterRoutes(staffMonitoring)

	adminAudit := protected.Group("/audit-logs")
	adminAudit.Use(middleware.RequireRoles(string(users.RoleAdmin)))
	audit.NewHandler(auditService).RegisterRoutes(adminAudit)

	return &Engine{Engine: engine, ExpiryWorker: attempts.NewExpiryWorker(attemptsService, 15*time.Second)}, nil
}
