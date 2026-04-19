package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"

	"github.com/your-org/eegfaktura-member-onboarding/internal/application"
	"github.com/your-org/eegfaktura-member-onboarding/internal/config"
	internalhttp "github.com/your-org/eegfaktura-member-onboarding/internal/http"
)

func main() {
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

	log.Println("Connected to database")

	// Initialize repositories
	appRepo := application.NewApplicationRepository(db)
	meteringRepo := application.NewMeteringPointRepository(db)
	statusLogRepo := application.NewStatusLogRepository(db)
	entrypointRepo := application.NewRegistrationEntrypointRepository(db)

	// Initialize services
	registrationService := application.NewRegistrationService(entrypointRepo)
	applicationService := application.NewApplicationService(db, appRepo, meteringRepo, statusLogRepo, entrypointRepo)
	adminService := application.NewAdminApplicationService(db, appRepo, meteringRepo, statusLogRepo)

	// Initialize handlers
	registrationHandler := internalhttp.NewRegistrationHandler(registrationService)
	applicationHandler := internalhttp.NewApplicationHandler(applicationService)
	adminHandler := internalhttp.NewAdminHandler(adminService)

	// Setup routes
	r := chi.NewRouter()

	// Middleware
	r.Use(internalhttp.CORSMiddleware(cfg.CORS.AllowedOrigins))
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

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

	// Admin routes (authentication added in PROJ-4 via Keycloak middleware)
	r.Route("/api/admin", func(r chi.Router) {
		r.Route("/applications", func(r chi.Router) {
			r.Get("/", adminHandler.ListApplications)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", adminHandler.GetApplicationDetail)
				r.Put("/", adminHandler.UpdateApplication)
				r.Post("/status", adminHandler.ChangeStatus)
			})
		})
	})

	// Start server
	addr := ":" + cfg.Server.Port
	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}