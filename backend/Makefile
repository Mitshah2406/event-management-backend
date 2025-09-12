.PHONY: db-info build run dev setup test clean docker-up docker-down help


# Build the application
db-info: ## Show database connection info
	@echo "PostgreSQL (Beekeeper Studio):"
	@echo "  Host: localhost"
	@echo "  Port: 5432"
	@echo "  Database: evently_db"
	@echo "  Username: evently_user"
	@echo "  Password: evently_password"
	@echo ""
	@echo "Redis (RedisInsight):"
	@echo "  Host: localhost"
	@echo "  Port: 6379"
	@echo "  Password: (empty)"

build: ## Build the application for production
	@echo "Building application..."
	@mkdir -p bin
	go build -o bin/server server/main.go

# Run the application
run: ## Run the application
	@echo "Running application..."
	go run server/main.go

# Development server with live reload
dev: ## Start development server with live reload
	@echo "Starting development server..."
	@if ! command -v air > /dev/null; then \
		echo "Installing Air for live reload..."; \
		go install github.com/air-verse/air@latest; \
	fi
	air

# Setup development environment
setup: ## Setup development environment
	@echo "Setting up development environment..."
	go mod tidy
	go mod download
	@if ! command -v air > /dev/null; then \
		go install github.com/air-verse/air@latest; \
	fi
	docker compose -f ./deployments/docker/docker-compose.dev.yml up -d
	@if [ ! -f .env ]; then \
		echo "Creating .env file..."; \
		echo "PORT=8080" > .env; \
		echo "DB_HOST=localhost" >> .env; \
		echo "DB_PORT=5432" >> .env; \
		echo "DB_NAME=evently_db" >> .env; \
		echo "DB_USER=evently_user" >> .env; \
		echo "DB_PASSWORD=evently_password" >> .env; \
		echo "JWT_SECRET=your-super-secret-jwt-key" >> .env; \
		echo "Created .env file with defaults"; \
	fi
	@echo "Setup complete! Run 'make dev' to start developing"

# Run tests
test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

# Clean build artifacts
clean: ## Clean build artifacts and stop containers
	@echo "Cleaning up..."
	rm -rf bin/ tmp/ coverage.out
	docker compose -f ./deployments/docker/docker-compose.dev.yml down
	go clean -cache

# Docker commands
docker-up: ## Start development database services
	@echo "Starting development services..."
	docker compose -f ./deployments/docker/docker-compose.dev.yml up -d

docker-down: ## Stop development database services
	@echo "Stopping development services..."
	docker compose -f ./deployments/docker/docker-compose.dev.yml down

# Help
help: ## Show available commands
	@echo "Available commands:"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Quick start: make setup && make dev"

.DEFAULT_GOAL := help
