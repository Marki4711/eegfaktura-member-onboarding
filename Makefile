.PHONY: build run test clean migrate-up migrate-down docker-up docker-down dev-setup dev seed-dev

# Default database URL matching docker-compose.yml defaults.
# Override on the command line: make migrate-up DATABASE_URL=postgres://...
DATABASE_URL ?= postgres://postgres:password@localhost:5432/member_onboarding?sslmode=disable

# Build the application
build:
	go build -o bin/server ./cmd/server

# Run the application
run:
	go run ./cmd/server

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Database migrations (uses go run so no external migrate binary needed)
migrate-up:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate -direction up

migrate-down:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate -direction down

# Docker commands (requires Docker Desktop with Compose plugin)
docker-up:
	docker compose up -d

docker-down:
	docker compose down

# Start postgres and apply migrations
dev-setup: docker-up migrate-up

# Insert demo data used by the screenshot generator
# (RC123456 + one application in every status, two metering points each).
# Idempotent — safe to re-run.
seed-dev:
	docker compose exec -T postgres psql -U postgres -d member_onboarding -f - < db/seed/dev_screenshots.sql

# Full screenshot stack: Postgres + Keycloak + migrations + seed + Keycloak config.
# After this you can `npm run screenshots` fully headless.
.PHONY: screenshots-stack
screenshots-stack:
	docker compose -f docker-compose.yml -f docker-compose.screenshots.yml up -d
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate -direction up
	docker compose exec -T postgres psql -U postgres -d member_onboarding -f - < db/seed/dev_screenshots.sql
	npx tsx scripts/setup-screenshot-keycloak.ts

.PHONY: screenshots-stack-down
screenshots-stack-down:
	docker compose -f docker-compose.yml -f docker-compose.screenshots.yml down

# Full local workflow
dev: dev-setup run
