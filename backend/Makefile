.PHONY: \
  db-info \
  build \
  run \
  dev \
  setup \
  test \
  clean \
  docker-up \
  docker-down \
  prod-up-simple \
  prod-down-simple \
  prod-logs-simple \
  prod-health-simple \
  generate-secrets \
  prod-clean-simple \
  prod-backup-simple \
  prod-restore-simple \
  update-env-template \
  switch-to-dev \
  switch-to-prod \
  prepare-deploy \
  check-deploy-ready \
  test-prod-build \
  help

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


# Start simplified production environment
prod-up-simple: ## Start simplified production services (App, DB, Redis only)
	@echo "Starting simplified production environment..."
	docker compose -f ./deployments/docker/docker-compose.prod.yml --env-file .env up -d --build
	@echo "🚀 Simplified production services started!"
	@echo "📋 Service URLs:"
	@echo "   App: http://localhost:9000"
	@echo "   Health Check: http://localhost:9000/health"
	@echo "   PostgreSQL: localhost:9001"
	@echo "   Redis: localhost:9002"

# Stop simplified production services
prod-down-simple: ## Stop simplified production services
	@echo "Stopping simplified production services..."
	docker compose -f ./deployments/docker/docker-compose.prod.yml down
	@echo "🛑 Simplified production services stopped"

# View simplified production logs
prod-logs-simple: ## View simplified production logs
	@echo "Viewing simplified production logs (press Ctrl+C to exit)..."
	docker compose -f ./deployments/docker/docker-compose.prod.yml logs -f

# Health check for simplified production
prod-health-simple: ## Check health of simplified production services
	@echo "Checking simplified production service health..."
	@echo ""
	@echo "🔍 Service Status:"
	@docker compose -f ./deployments/docker/docker-compose.prod.yml ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}"
	@echo ""
	@echo "🏥 Health Checks:"
	@echo -n "App Health: "
	@curl -f http://localhost:9000/health >/dev/null 2>&1 && echo "✅ Healthy" || echo "❌ Unhealthy"
	@echo -n "PostgreSQL: "
	@docker exec evently_postgres pg_isready -U $$(grep DB_USER .env | cut -d'=' -f2) >/dev/null 2>&1 && echo "✅ Ready" || echo "❌ Not Ready"
	@echo -n "Redis: "
	@if grep -q "REDIS_PASSWORD=" .env && [ -n "$(grep REDIS_PASSWORD .env | cut -d'=' -f2)" ]; then \
		docker exec evently_redis redis-cli --pass "$(grep REDIS_PASSWORD .env | cut -d'=' -f2)" ping >/dev/null 2>&1 && echo "✅ Responding" || echo "❌ Not Responding"; \
	else \
		docker exec evently_redis redis-cli ping >/dev/null 2>&1 && echo "✅ Responding" || echo "❌ Not Responding"; \
	fi

	@echo "🧪 Testing production build..."
	@if [ ! -f .env ]; then \
		echo "❌ .env not found"; \
		exit 1; \
	fi
	@echo "Building production image..."
	@docker build -f ./deployments/docker/Dockerfile.prod -t evently-test:latest .
	@echo "✅ Production build test successful!"

.DEFAULT_GOAL := help
