// @title           eegfaktura Member Onboarding API
// @version         1.0
// @description     REST API for EEG (Energiegemeinschaft) member self-service registration and admin review.
// @description
// @description     **Auth schemes:**
// @description     - **Public** endpoints (`/api/public/*`): no auth required. Rate-limited + optional Turnstile CAPTCHA.
// @description     - **Admin** endpoints (`/api/admin/*`): Keycloak Bearer JWT (`Authorization: Bearer <token>`).
// @description     - **External** endpoints (`/api/external/*`): API key (`Authorization: Bearer moak_<key>`).
//
// @host            member-onboarding.eegfaktura.at
// @BasePath        /
// @schemes         https
//
// @securityDefinitions.apikey  BearerAuth
// @in              header
// @name            Authorization
// @description     Keycloak JWT. Format: "Bearer <token>"
//
// @securityDefinitions.apikey  ApiKeyAuth
// @in              header
// @name            Authorization
// @description     EEG API key. Format: "Bearer moak_<key>"

package main

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpswagger "github.com/swaggo/http-swagger/v2"

	"github.com/your-org/eegfaktura-member-onboarding/internal/application"
	"github.com/your-org/eegfaktura-member-onboarding/internal/config"
	"github.com/your-org/eegfaktura-member-onboarding/internal/coreclient"
	_ "github.com/your-org/eegfaktura-member-onboarding/docs"
	internalhttp "github.com/your-org/eegfaktura-member-onboarding/internal/http"
	"github.com/your-org/eegfaktura-member-onboarding/internal/importing"
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

	// In production, Keycloak must be configured — fail fast rather than serving
	// admin routes unprotected when JWKSUrl is missing.
	if os.Getenv("ENVIRONMENT") == "production" && cfg.Keycloak.JWKSUrl == "" {
		log.Fatalf("KEYCLOAK_JWKS_URL must be set in production")
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
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	slog.Info("connected to database")

	// Initialize repositories
	appRepo := application.NewApplicationRepository(db)
	meteringRepo := application.NewMeteringPointRepository(db)
	statusLogRepo := application.NewStatusLogRepository(db)
	entrypointRepo := application.NewRegistrationEntrypointRepository(db)
	fieldConfigRepo := application.NewFieldConfigRepository(db)
	apiKeyRepo := application.NewExternalAPIKeyRepository(db)
	legalDocumentRepo := application.NewLegalDocumentRepository(db)
	consentRepo := application.NewDocumentConsentRepository(db)

	// Initialize mail service
	var mailService mail.MailService = &mail.NoOpMailService{}
	if cfg.SMTP.Host != "" {
		if cfg.SMTP.From == "" {
			log.Fatalf("SMTP_FROM must be set when SMTP_HOST is configured")
		}
		mailer := mail.NewMailer(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.User, cfg.SMTP.Password, cfg.SMTP.From, cfg.SMTP.FromName)
		svc, err := mail.NewSMTPMailService(mailer, cfg.AdminBaseURL)
		if err != nil {
			log.Fatalf("Failed to initialize mail service: %v", err)
		}
		mailService = svc
		slog.Info("mail service enabled", "smtp_host", cfg.SMTP.Host)
	}

	// Initialize services
	pdfGenerator := pdf.NewFPDFGenerator()
	approvalPDFGenerator := pdf.NewFPDFApprovalGenerator()
	registrationService := application.NewRegistrationService(entrypointRepo, fieldConfigRepo, legalDocumentRepo, cfg.CentralPolicy.Title, cfg.CentralPolicy.URL)
	applicationService := application.NewApplicationService(db, appRepo, meteringRepo, statusLogRepo, entrypointRepo, fieldConfigRepo, consentRepo, legalDocumentRepo, mailService, pdfGenerator, cfg.PublicBaseURL)
	adminService := application.NewAdminApplicationService(db, appRepo, meteringRepo, statusLogRepo, fieldConfigRepo, entrypointRepo, consentRepo, mailService, approvalPDFGenerator, pdfGenerator, cfg.PublicBaseURL)

	// PROJ-4 (REST) + PROJ-32 (GraphQL) share one HTTPCoreClient. Both
	// features are enabled together by setting CORE_BASE_URL — the
	// production deployment serves /api/participant, /api/query, etc.
	// under the same hostname.
	var coreHTTPClient *coreclient.HTTPCoreClient
	var importService *importing.ImportService
	if cfg.Core.BaseURL != "" {
		coreHTTPClient = coreclient.NewHTTPCoreClient(cfg.Core.BaseURL, time.Duration(cfg.Core.TimeoutSeconds)*time.Second)
		importService = importing.NewImportService(db, appRepo, meteringRepo, statusLogRepo, coreHTTPClient)
		slog.Info("core integration enabled", "core_base_url", cfg.Core.BaseURL)
	}

	// Initialize handlers
	registrationHandler := internalhttp.NewRegistrationHandler(registrationService)
	applicationHandler := internalhttp.NewApplicationHandler(applicationService, cfg.Turnstile.SecretKey)
	adminHandler := internalhttp.NewAdminHandler(adminService, entrypointRepo, apiKeyRepo, legalDocumentRepo, importService, coreHTTPClient)
	externalHandler := internalhttp.NewExternalHandler(applicationService)
	healthHandler := internalhttp.NewHealthHandler(db)

	// Configure trusted-proxy CIDRs before realIP() is used by any middleware.
	// Empty value = trust nothing → r.RemoteAddr wins.
	internalhttp.SetTrustedProxyCIDRs(cfg.TrustedProxyCIDRs)

	// Setup routes
	r := chi.NewRouter()

	// Middleware
	r.Use(internalhttp.CORSMiddleware(cfg.CORS.AllowedOrigins))
	r.Use(internalhttp.SecurityHeadersMiddleware)
	r.Use(middleware.RequestID)
	r.Use(internalhttp.SlogRequestLogger)
	r.Use(middleware.Recoverer)

	// Request body size limits per route group. Public/external endpoints get
	// a tight cap because their payloads are bounded (form data + a few
	// metering points); admin endpoints get a larger budget for intro_text
	// and admin_note. Limits are applied before handler decode so an oversize
	// body surfaces as a clean 400 from the json decoder.
	const (
		publicBodyMax   int64 = 256 * 1024  // 256 KiB
		externalBodyMax int64 = 256 * 1024  // 256 KiB
		adminBodyMax    int64 = 1024 * 1024 // 1 MiB
	)

	// Swagger UI — publicly accessible, no auth required
	r.Get("/api/docs/*", httpswagger.Handler(
		httpswagger.URL("/api/docs/doc.json"),
	))

	// Health checks
	// /health stays for compatibility; new K8s probes use the split endpoints:
	// - /livez:  process alive — does NOT touch DB, so a Postgres blip doesn't
	//   kill the pod via livenessProbe and amplify the outage.
	// - /readyz: DB reachable — readinessProbe drops the pod from Service
	//   endpoints during a DB outage instead of restarting it.
	r.Get("/health", healthHandler.Health)
	r.Get("/livez", healthHandler.Livez)
	r.Get("/readyz", healthHandler.Readyz)

	// API routes
	r.Route("/api/public", func(r chi.Router) {
		r.Use(internalhttp.MaxBodySize(publicBodyMax))
		r.Route("/registration/{rc_number}", func(r chi.Router) {
			r.Get("/", registrationHandler.GetRegistrationConfig)
		})

		r.Route("/applications", func(r chi.Router) {
			r.With(internalhttp.PublicSubmitRateLimitMiddleware).Post("/", applicationHandler.CreateApplication)
			// PROJ-31: e-mail confirmation endpoint. Rate-limited via the same
			// public-submit middleware so an attacker can't brute-force tokens
			// (the 32-byte entropy already makes that astronomical, but the
			// rate-limit is a cheap defence-in-depth).
			r.With(internalhttp.PublicSubmitRateLimitMiddleware).Post("/confirm-email", applicationHandler.ConfirmEmail)
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
		r.Use(internalhttp.MaxBodySize(adminBodyMax))
		r.Use(internalhttp.KeycloakAuthMiddleware(cfg.Keycloak.JWKSUrl, cfg.Keycloak.Issuer))
		r.Post("/sync", adminHandler.SyncEntrypoints)
		r.Get("/tariffs", adminHandler.ListTariffs)
		r.Route("/applications", func(r chi.Router) {
			r.Get("/", adminHandler.ListApplications)
			r.Delete("/drafts", adminHandler.DeleteDraftApplications)
			r.Post("/bulk-action", adminHandler.BulkAction)
			r.Post("/check-activation", adminHandler.CheckActivation)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", adminHandler.GetApplicationDetail)
				r.Put("/", adminHandler.UpdateApplication)
				r.Delete("/", adminHandler.DeleteApplication)
				r.Post("/status", adminHandler.ChangeStatus)
				r.Post("/resend-confirmation", adminHandler.ResendMemberConfirmation)
				r.Post("/resend-email-confirmation", adminHandler.ResendEmailConfirmation)
				r.Get("/export/excel", adminHandler.ExportApplicationExcel)
				r.Get("/approval-pdf", adminHandler.DownloadApprovalPDF)
				r.Post("/import", adminHandler.ImportApplication)
				r.Post("/reset-import", adminHandler.ResetImport)
				r.Post("/mark-imported-manually", adminHandler.MarkImportedManually)
				r.Post("/clear-import-lock", adminHandler.ClearImportLock)
				r.Post("/reassign-eeg", adminHandler.ReassignEEG)
				r.Patch("/admin-note", adminHandler.UpdateAdminNote)
				r.Get("/next-member-number", adminHandler.SuggestNextMemberNumber)
			})
		})
		r.Route("/settings", func(r chi.Router) {
			r.Get("/fields", adminHandler.GetFieldConfig)
			r.Put("/fields", adminHandler.SaveFieldConfig)
			r.Get("/intro-text", adminHandler.GetIntroText)
			r.Put("/intro-text", adminHandler.SaveIntroText)
			r.Get("/eeg", adminHandler.GetEEGSettings)
			r.Put("/eeg", adminHandler.SaveEEGSettings)
			r.Get("/eeg/core-comparison", adminHandler.CompareEEGSettingsWithCore)
			r.Post("/eeg/sync", adminHandler.SyncEEGSettingsFromCore)
			r.Get("/eeg/logo", adminHandler.GetEEGLogo)
			r.Get("/api-key", adminHandler.GetAPIKeyStatus)
			r.Post("/api-key", adminHandler.GenerateAPIKey)
			r.Delete("/api-key", adminHandler.RevokeAPIKey)
		})
		r.Route("/legal-documents", func(r chi.Router) {
			r.Get("/", adminHandler.ListLegalDocuments)
			r.Post("/", adminHandler.CreateLegalDocument)
			r.Put("/reorder", adminHandler.ReorderLegalDocuments)
			r.Put("/{id}", adminHandler.UpdateLegalDocument)
			r.Delete("/{id}", adminHandler.DeleteLegalDocument)
		})
	})

	// External API routes — authenticated via API key middleware (no Keycloak)
	r.Route("/api/external", func(r chi.Router) {
		r.Use(internalhttp.MaxBodySize(externalBodyMax))
		r.Use(internalhttp.APIKeyMiddleware(apiKeyRepo))
		r.Post("/v1/applications", externalHandler.SubmitExternalApplication)
	})

	// Start IP bucket cleanup goroutine
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	internalhttp.StartIPBucketCleanup(cleanupCtx)

	// PROJ-31: scan once a day for applications stuck on an expired
	// e-mail-confirmation token and auto-reject them. Uses cleanupCtx so it
	// stops together with the rate-limit bucket cleanup on shutdown.
	go adminService.RunAutoRejectLoop(cleanupCtx, 24*time.Hour)

	// Start main HTTP server
	addr := ":" + cfg.Server.Port
	slog.Info("starting server", "addr", addr)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	// Metrics server — separate port so /metrics never gets exposed via the
	// public ingress. The Helm Service for this port is ClusterIP-only and
	// only the in-cluster Prometheus pod can reach it. Disabled when
	// METRICS_PORT="" (no metrics endpoint at all).
	var metricsSrv *http.Server
	if cfg.MetricsPort != "" {
		metricsMux := http.NewServeMux()
		metricsMux.Handle("/metrics", promhttp.Handler())
		metricsSrv = &http.Server{
			Addr:         ":" + cfg.MetricsPort,
			Handler:      metricsMux,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		}
		slog.Info("starting metrics server", "addr", metricsSrv.Addr)
		go func() {
			if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("metrics server failed", "error", err)
			}
		}()
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	slog.Info("shutdown signal received, draining requests...")
	cleanupCancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if metricsSrv != nil {
		_ = metricsSrv.Shutdown(shutdownCtx)
	}
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	slog.Info("server stopped")
}