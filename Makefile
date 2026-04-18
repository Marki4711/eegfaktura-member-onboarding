.PHONY: build run test clean migrate-up migrate-down docker-up docker-down

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

# Database migrations
migrate-up:
	migrate -path db/migrations -database "$(shell go run ./cmd/server --print-dsn)" up

migrate-down:
	migrate -path db/migrations -database "$(shell go run ./cmd/server --print-dsn)" down 1

# Docker commands
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

# Development setup
dev-setup: docker-up migrate-up

# Full development workflow
dev: dev-setup build run