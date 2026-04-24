package main

import (
	"database/sql"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"

	"github.com/your-org/eegfaktura-member-onboarding/internal/application"
	"github.com/your-org/eegfaktura-member-onboarding/internal/config"
	internalhttp "github.com/your-org/eegfaktura-member-onboarding/internal/http"
	"github.com/your-org/eegfaktura-member-onboarding/internal/mail"
	"github.com/your-org/eegfaktura-member-onboarding/internal/pdf"
)

func main() {
	// Structured JSON logging — level configurable via LOG_LEVEL env var
	logLevel := slog.LevelInfo
	switch os.Getenv("LOG_LEVEL") {
	case "DEBUG", "debug":
		logLevel = slog.LevelDebug
	case "WARN", "warn":
		logLevel = slog.LevelWarn
	case "ERROR", "error":
		logLevel = slog.LevelError
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})))

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to database
	db, err := sql.Open("postgres", cfg.Database.DSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	slog.Info("connected to database")

	// Initialize repositories
	appRepo := application.NewApplicationRepository(db)
	meteringRepo := application.NewMeteringPointRepository(db)
	statusLogRepo := application.NewStatusLogRepository(db)
	entrypointRepo := application.NewRegistrationEntrypointRepository(db)
	fieldConfigRepo := application.NewFieldConfigRepository(db)
	apiKeyRepo := application.NewExternalAPIKeyRepository(db)

	// Initialize mail service
	var mailService mail.MailService = &mail.NoOpMailService{}
	if cfg.SMTP.Host != "" {
		if cfg.SMTP.From == "" {
			log.Fatalf("SMTP_FROM must be set when SMTP_HOST is configured")
		}
		mailer := mail.NewMailer(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.User, cfg.SMTP.Password, cfg.SMTP.From)
		svc, err := mail.NewSMTPMailService(mailer)
		if err != nil {
			log.Fatalf("Failed to initialize mail service: %v", err)
		}
		mailService = svc
		slog.Info("mail service enabled", "smtp_host", cfg.SMTP.Host)
	}

	// Initialize services
	pdfGenerator := pdf.NewFPDFGenerator()
	registrationService := application.NewRegistrationService(entrypointRepo, fieldConfigRepo)
	applicationService := application.NewApplicationService(db, appRepo, meteringRepo, statusLogRepo, entrypointRepo, fieldConfigRepo, mailService, pdfGenerator)
	adminService := application.NewAdminApplicationService(db, appRepo, meteringRepo, statusLogRepo, fieldConfigRepo, mailService)

	// Initialize handlers
	registrationHandler := internalhttp.NewRegistrationHandler(registrationService)
	applicationHandler := internalhttp.NewApplicationHandler(applicationService)
	adminHandler := internalhttp.NewAdminHandler(adminService, entrypointRepo, apiKeyRepo)
	externalHandler := internalhttp.NewExternalHandler(applicationService)
	healthHandler := internalhttp.NewHealthHandler(db)

	// Setup routes
	r := chi.NewRouter()

	// Middleware
	r.Use(internalhttp.CORSMiddleware(cfg.CORS.AllowedOrigins))
	r.Use(middleware.RequestID)
	r.Use(internalhttp.SlogRequestLogger)
	r.Use(middleware.Recoverer)

	// Health check
	r.Get("/health", healthHandler.Health)

	// API routes
	r.Route("/api/public", func(r chi.Router) {
		r.Route("/registration/{rc_number}", func(r chi.Router) {
			r.Get("/", registrationHandler.GetRegistrationConfig)
		})

		r.Route("/applications", func(r chi.Router) {
			r.Post("/", applicationHandler.CreateApplication)
			r.Route("/{id}", func(r chi.Router) {
				r.Put("/", applicationHandler.UpdateApplication)
				r.Route("/submit", func(r chi.Router) {
					r.Post("/", applicationHandler.SubmitApplication)
				})
			})
		})
	})

	// Admin routes — protected by Keycloak JWT middleware
	r.Route("/api/admin", func(r chi.Router) {
		r.Use(internalhttp.KeycloakAuthMiddleware(cfg.Keycloak.JWKSUrl, cfg.Keycloak.Issuer))
		r.Post("/sync", adminHandler.SyncEntrypoints)
		r.Route("/applications", func(r chi.Router) {
			r.Get("/", adminHandler.ListApplications)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", adminHandler.GetApplicationDetail)
				r.Put("/", adminHandler.UpdateApplication)
				r.Delete("/", adminHandler.DeleteApplication)
				r.Post("/status", adminHandler.ChangeStatus)
				r.Post("/resend-confirmation", adminHandler.ResendMemberConfirmation)
			})
		})
		r.Route("/settings", func(r chi.Router) {
			r.Get("/fields", adminHandler.GetFieldConfig)
			r.Put("/fields", adminHandler.SaveFieldConfig)
			r.Get("/intro-text", adminHandler.GetIntroText)
			r.Put("/intro-text", adminHandler.SaveIntroText)
			r.Get("/eeg", adminHandler.GetEEGSettings)
			r.Put("/eeg", adminHandler.SaveEEGSettings)
			r.Get("/api-key", adminHandler.GetAPIKeyStatus)
			r.Post("/api-key", adminHandler.GenerateAPIKey)
			r.Delete("/api-key", adminHandler.RevokeAPIKey)
		})
	})

	// External API routes — authenticated via API key middleware (no Keycloak)
	r.Route("/api/external", func(r chi.Router) {
		r.Use(internalhttp.APIKeyMiddleware(apiKeyRepo))
		r.Post("/v1/applications", externalHandler.SubmitExternalApplication)
	})

	// Start server
	addr := ":" + cfg.Server.Port
	slog.Info("starting server", "addr", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
}