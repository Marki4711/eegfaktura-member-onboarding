package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	Server        ServerConfig
	Database      DatabaseConfig
	CORS          CORSConfig
	SMTP          SMTPConfig
	Keycloak      KeycloakConfig
	Turnstile     TurnstileConfig
	CentralPolicy CentralPolicyConfig
	Core          CoreConfig
	AdminBaseURL  string
	// PublicBaseURL is the externally-reachable URL of the public-facing
	// frontend (e.g. https://onboarding.eeg-x.at). Used to build absolute
	// URLs for outgoing e-mails — notably the e-mail-confirmation link.
	// Falls back to "" when unset; the e-mail-confirmation feature will
	// log a warning and skip the link in that case.
	PublicBaseURL string
	// MetricsPort, when non-empty, starts a separate HTTP server on that port
	// that serves /metrics (Prometheus exposition format). Default "9090".
	// Set to "" to disable the metrics endpoint entirely.
	MetricsPort string
	// TrustedProxyCIDRs is a comma-separated list of CIDR ranges (IPv4 or IPv6)
	// from which X-Real-IP / X-Forwarded-For headers are honoured. Anything
	// outside these ranges has its proxy headers ignored. Empty (dev default)
	// = trust nothing → realIP() falls back to RemoteAddr.
	TrustedProxyCIDRs string
}

// CoreConfig holds connection settings for the eegFaktura core service used
// by the import endpoint (PROJ-4) and the EEG-master-data sync (PROJ-32).
//
// BaseURL — hostname of the eegFaktura core (e.g. `https://eegfaktura.at`).
// Path prefixes are appended by the coreclient at the call site:
//   - REST       — {BaseURL}/api/participant, {BaseURL}/api/eeg/tariff, …
//   - GraphQL    — {BaseURL}/api/query (PROJ-32)
// Phase 2 (logo) will append `/cash/api/billingConfigs/...` to the same
// hostname. Empty BaseURL disables every core-dependent feature.
//
// AuthMode — selects how REST-Calls to the core are authenticated:
//   - "direct"   (default) — the admin's Onboarding-Keycloak access-token is
//     forwarded verbatim. Works iff the Faktura backend whitelists the
//     `eegfaktura-member-onboarding` Keycloak-client as a legitimate `azp`.
//   - "exchange" — the Frontend obtains a second access-token via silent
//     SSO against the Faktura-Frontend Keycloak-client (`at.ourproject.vfeeg.app`)
//     and forwards it in the `X-Core-Authorization` header. The Onboarding-
//     Backend uses that token for REST-Calls instead of the admin's session
//     token. Workaround for a stealth `azp`-filter in the Faktura backend
//     that returns 200+empty for non-whitelisted clients.
//
// The mode is a global helm-deploy decision (not per-EEG) because the
// underlying constraint is the Faktura backend's filter, which is shared
// across all tenants. Switch to "direct" once the Faktura maintainer
// extends the filter.
type CoreConfig struct {
	BaseURL        string
	TimeoutSeconds int
	AuthMode       string
}

// CentralPolicyConfig holds title and URL of the operator's central privacy policy.
// Always shown as a required document in the public registration form.
type CentralPolicyConfig struct {
	Title string
	URL   string
}

// TurnstileConfig holds Cloudflare Turnstile settings.
// When SecretKey is empty, server-side verification is disabled (dev mode).
type TurnstileConfig struct {
	SecretKey string
}

// KeycloakConfig holds Keycloak JWT validation settings.
// When JWKSUrl is empty, admin auth middleware is disabled (dev mode).
type KeycloakConfig struct {
	JWKSUrl string
	Issuer  string
}

// SMTPConfig holds SMTP mail configuration.
// When Host is empty the mail sender is disabled.
type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
	// FromName is the display name shown before the address in mail clients
	// (`"eegFaktura …" <noreply@…>`). Empty falls back to the bare address.
	// Inbox providers count a present display name as a legitimacy signal.
	FromName string
}

// CORSConfig holds allowed origins for CORS.
type CORSConfig struct {
	AllowedOrigins []string
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			ReadTimeout:  getDurationEnv("READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDurationEnv("WRITE_TIMEOUT", 15*time.Second),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getIntEnv("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "password"),
			DBName:   getEnv("DB_NAME", "member_onboarding"),
			SSLMode:  getEnv("DB_SSLMODE", "require"),
		},
		CORS: CORSConfig{
			AllowedOrigins: strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"), ","),
		},
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", ""),
			Port:     getIntEnv("SMTP_PORT", 587),
			User:     getEnv("SMTP_USER", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
			From:     getEnv("SMTP_FROM", ""),
			FromName: getEnv("SMTP_FROM_NAME", "eegFaktura Mitglieder-Onboarding"),
		},
		Keycloak: KeycloakConfig{
			JWKSUrl: getEnv("KEYCLOAK_JWKS_URL", ""),
			Issuer:  getEnv("KEYCLOAK_ISSUER", ""),
		},
		Turnstile: TurnstileConfig{
			SecretKey: getEnv("TURNSTILE_SECRET_KEY", ""),
		},
		CentralPolicy: CentralPolicyConfig{
			Title: getEnv("CENTRAL_POLICY_TITLE", "Datenschutzerklärung"),
			URL:   getEnv("CENTRAL_POLICY_URL", ""),
		},
		Core: CoreConfig{
			BaseURL:        getEnv("CORE_BASE_URL", ""),
			TimeoutSeconds: getIntEnv("CORE_TIMEOUT_SECONDS", 30),
			AuthMode:       getEnv("CORE_AUTH_MODE", "direct"),
		},
		AdminBaseURL:      getEnv("ADMIN_BASE_URL", ""),
		PublicBaseURL:     getEnv("PUBLIC_BASE_URL", ""),
		TrustedProxyCIDRs: getEnv("TRUSTED_PROXY_CIDRS", ""),
		MetricsPort:       getEnv("METRICS_PORT", "9090"),
	}

	return config, nil
}

// DSN returns the database connection string
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// getEnv gets an environment variable with a fallback value
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getIntEnv gets an integer environment variable with a fallback value
func getIntEnv(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return fallback
}

// getDurationEnv gets a duration environment variable with a fallback value
func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return fallback
}