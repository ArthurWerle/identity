.PHONY: help build run test test-coverage lint fmt clean docker-build docker-up docker-down migrate-up migrate-down migrate-create swagger deps

# Variables
APP_NAME=identity-service
BINARY_NAME=server
DOCKER_IMAGE=identity-service:latest
DB_URL=postgres://identity:identity_dev_password@localhost:5432/identity_db?sslmode=disable

# Help target
help: ## Display this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Build the application
build: ## Build the application binary
	@echo "Building $(APP_NAME)..."
	@go build -o bin/$(BINARY_NAME) ./cmd/server

# Run the application
run: ## Run the application locally
	@echo "Running $(APP_NAME)..."
	@go run ./cmd/server/main.go

# Run tests
test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./... -count=1

# Run tests with coverage
test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Lint code
lint: ## Run linter
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install it from https://golangci-lint.run/usage/install/"; \
	fi

# Format code
fmt: ## Format code with gofmt
	@echo "Formatting code..."
	@gofmt -s -w .
	@go mod tidy

# Clean build artifacts
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

# Docker commands
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE) .

docker-up: ## Start all services with docker-compose
	@echo "Starting services with docker-compose..."
	@docker-compose up -d

docker-down: ## Stop all services
	@echo "Stopping services..."
	@docker-compose down

docker-logs: ## View docker-compose logs
	@docker-compose logs -f

# Database migration commands (using atlas)
migrate-up: ## Run database migrations
	@echo "Running migrations..."
	@if command -v atlas > /dev/null; then \
		atlas schema apply --env local --auto-approve; \
	else \
		echo "Atlas not installed. Install it from https://atlasgo.io/getting-started#installation"; \
	fi

migrate-down: ## Rollback last migration
	@echo "Rolling back migration..."
	@echo "Note: Atlas doesn't have a traditional 'down' migration. Use 'atlas schema apply' with a previous state."

migrate-status: ## Check migration status
	@echo "Checking migration status..."
	@if command -v atlas > /dev/null; then \
		atlas schema inspect --env local; \
	else \
		echo "Atlas not installed."; \
	fi

migrate-create: ## Create a new migration file (usage: make migrate-create NAME=migration_name)
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate-create NAME=my_migration"; \
		exit 1; \
	fi
	@TIMESTAMP=$$(date +%Y%m%d%H%M%S); \
	FILENAME="db/migrations/$${TIMESTAMP}_$(NAME).sql"; \
	touch "$$FILENAME"; \
	echo "-- Migration: $(NAME)" > "$$FILENAME"; \
	echo "-- Created: $$(date)" >> "$$FILENAME"; \
	echo "" >> "$$FILENAME"; \
	echo "-- Add your SQL statements below" >> "$$FILENAME"; \
	echo "" >> "$$FILENAME"; \
	echo "Created migration: $$FILENAME"

# Swagger documentation
swagger: ## Generate Swagger documentation
	@echo "Generating Swagger documentation..."
	@if command -v swag > /dev/null; then \
		swag init -g cmd/server/main.go -o docs; \
	else \
		echo "swag not installed. Install it with: go install github.com/swaggo/swag/cmd/swag@latest"; \
	fi

# Install dependencies
deps: ## Install project dependencies
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy
	@echo "Installing development tools..."
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Dependencies installed!"

# Database helpers
db-start: ## Start only the database
	@echo "Starting PostgreSQL database..."
	@docker-compose up -d postgres

db-stop: ## Stop the database
	@echo "Stopping PostgreSQL database..."
	@docker-compose stop postgres

db-reset: ## Reset the database (WARNING: deletes all data)
	@echo "Resetting database..."
	@docker-compose down -v
	@docker-compose up -d postgres
	@sleep 3
	@make migrate-up

# Development workflow
dev: db-start ## Start development environment
	@echo "Starting development environment..."
	@sleep 3
	@make run

# Full setup
setup: deps swagger ## Full project setup
	@echo "Project setup complete!"
	@echo "Run 'make dev' to start developing"
