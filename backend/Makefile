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

# Add these commands to your existing Makefile

# ============================================================================
# SIMPLIFIED PRODUCTION COMMANDS (Using only essential services)
# ============================================================================

# Start simplified production environment
prod-up-simple: ## Start simplified production services (App, DB, Redis only)
	@echo "Starting simplified production environment..."
	@if [ ! -f .env.prod ]; then \
		echo "❌ .env.prod file not found!"; \
		echo "📋 Copy .env.prod template and configure it first"; \
		exit 1; \
	fi
	docker compose -f ./deployments/docker/docker-compose.prod.yml --env-file .env.prod up -d --build
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
	@docker exec evently_postgres pg_isready -U $$(grep DB_USER .env.prod | cut -d'=' -f2) >/dev/null 2>&1 && echo "✅ Ready" || echo "❌ Not Ready"
	@echo -n "Redis: "
	@if grep -q "REDIS_PASSWORD=" .env.prod && [ -n "$(grep REDIS_PASSWORD .env.prod | cut -d'=' -f2)" ]; then \
		docker exec evently_redis redis-cli --pass "$(grep REDIS_PASSWORD .env.prod | cut -d'=' -f2)" ping >/dev/null 2>&1 && echo "✅ Responding" || echo "❌ Not Responding"; \
	else \
		docker exec evently_redis redis-cli ping >/dev/null 2>&1 && echo "✅ Responding" || echo "❌ Not Responding"; \
	fi

# Generate production secrets
generate-secrets: ## Generate strong passwords and secrets for production
	@echo "🔐 Generating production secrets..."
	@echo ""
	@echo "DB_PASSWORD=$(shell openssl rand -base64 32)"
	@echo "REDIS_PASSWORD=$(shell openssl rand -base64 32)"
	@echo "JWT_SECRET=$(shell openssl rand -base64 64)"
	@echo ""
	@echo "📋 Copy these values to your .env.prod file"
	@echo "⚠️  Keep these secrets secure and never commit them to version control!"

# Clean production environment (DANGEROUS)
prod-clean-simple: ## Clean simplified production environment (REMOVES ALL DATA!)
	@echo "⚠️  WARNING: This will DELETE ALL production data!"
	@echo "⚠️  This includes database, cache, and uploaded files!"
	@read -p "Type 'DELETE-ALL-DATA' to continue: " confirm; \
	if [ "$confirm" = "DELETE-ALL-DATA" ]; then \
		echo "🧹 Cleaning simplified production environment..."; \
		docker compose -f ./deployments/docker/docker-compose.prod.yml down -v; \
		docker volume rm evently_postgres_data evently_redis_data evently_app_uploads evently_app_logs 2>/dev/null || true; \
		echo "✅ Simplified production environment cleaned"; \
	else \
		echo "❌ Operation cancelled"; \
	fi

# Backup simplified production database
prod-backup-simple: ## Create backup of simplified production database
	@echo "Creating simplified production database backup..."
	@mkdir -p ./backups
	@DB_USER=$(grep DB_USER .env.prod | cut -d'=' -f2); \
	DB_NAME=$(grep DB_NAME .env.prod | cut -d'=' -f2); \
	BACKUP_FILE="./backups/evently-simple-prod-$(date +%Y%m%d-%H%M%S).sql"; \
	docker exec evently_postgres pg_dump -U "$DB_USER" "$DB_NAME" > "$BACKUP_FILE"; \
	gzip "$BACKUP_FILE"; \
	echo "✅ Database backup created: $BACKUP_FILE.gz"

# Restore simplified production database
prod-restore-simple: ## Restore simplified production database (requires BACKUP_FILE)
	@if [ -z "$(BACKUP_FILE)" ]; then \
		echo "❌ Please specify BACKUP_FILE variable:"; \
		echo "   make prod-restore-simple BACKUP_FILE=./backups/your-backup.sql.gz"; \
		exit 1; \
	fi
	@if [ ! -f "$(BACKUP_FILE)" ]; then \
		echo "❌ Backup file not found: $(BACKUP_FILE)"; \
		exit 1; \
	fi
	@echo "⚠️  WARNING: This will replace the current production database!"
	@read -p "Continue? Type 'yes': " confirm; \
	if [ "$confirm" = "yes" ]; then \
		echo "🔄 Restoring database from $(BACKUP_FILE)..."; \
		DB_USER=$(grep DB_USER .env.prod | cut -d'=' -f2); \
		DB_NAME=$(grep DB_NAME .env.prod | cut -d'=' -f2); \
		if [[ "$(BACKUP_FILE)" == *.gz ]]; then \
			gunzip -c "$(BACKUP_FILE)" | docker exec -i evently_postgres psql -U "$DB_USER" -d "$DB_NAME"; \
		else \
			docker exec -i evently_postgres psql -U "$DB_USER" -d "$DB_NAME" < "$(BACKUP_FILE)"; \
		fi; \
		echo "✅ Database restored successfully"; \
	else \
		echo "❌ Restore cancelled"; \
	fi

# Update environment template
update-env-template: ## Update .env.prod.template from current .env
	@echo "Updating .env.prod.template..."
	@cp .env .env.prod.template
	@sed -i 's/evently_db/evently_prod/g' .env.prod.template
	@sed -i 's/localhost/postgres/g' .env.prod.template
	@sed -i 's/GIN_MODE=debug/GIN_MODE=release/g' .env.prod.template
	@sed -i 's/LOG_LEVEL=debug/LOG_LEVEL=info/g' .env.prod.template
	@echo "✅ .env.prod.template updated"

# ============================================================================
# DEVELOPMENT HELPERS
# ============================================================================

# Switch between development and production configs
switch-to-dev: ## Switch Docker configs to development mode
	@echo "Switching to development configuration..."
	@if [ -f .env.backup ]; then rm .env.backup; fi
	@cp .env .env.backup
	@sed -i.bak 's/DB_HOST=postgres/DB_HOST=localhost/g' .env
	@sed -i.bak 's/REDIS_HOST=redis/REDIS_HOST=localhost/g' .env
	@sed -i.bak 's/GIN_MODE=release/GIN_MODE=debug/g' .env
	@sed -i.bak 's/LOG_LEVEL=info/LOG_LEVEL=debug/g' .env
	@rm .env.bak
	@echo "✅ Switched to development configuration"

switch-to-prod: ## Switch Docker configs to production mode
	@echo "Switching to production configuration..."
	@if [ -f .env.backup ]; then rm .env.backup; fi
	@cp .env .env.backup
	@sed -i.bak 's/DB_HOST=localhost/DB_HOST=postgres/g' .env
	@sed -i.bak 's/REDIS_HOST=localhost/REDIS_HOST=redis/g' .env
	@sed -i.bak 's/GIN_MODE=debug/GIN_MODE=release/g' .env
	@sed -i.bak 's/LOG_LEVEL=debug/LOG_LEVEL=info/g' .env
	@rm .env.bak
	@echo "✅ Switched to production configuration"

# ============================================================================
# DEPLOYMENT HELPERS
# ============================================================================

# Prepare for deployment
prepare-deploy: ## Prepare files for deployment
	@echo "📦 Preparing deployment package..."
	@if [ ! -f .env.prod ]; then \
		echo "❌ .env.prod not found. Creating template..."; \
		cp .env.prod.template .env.prod; \
		echo "📝 Please edit .env.prod with your production values before deploying"; \
		exit 1; \
	fi
	@echo "✅ Deployment preparation complete"

# Check deployment readiness
check-deploy-ready: ## Check if ready for production deployment
	@echo "🔍 Checking deployment readiness..."
	@errors=0; \
	if [ ! -f .env.prod ]; then \
		echo "❌ .env.prod file missing"; \
		errors=$((errors + 1)); \
	else \
		if grep -q "CHANGE_THIS" .env.prod; then \
			echo "❌ .env.prod contains placeholder values (CHANGE_THIS)"; \
			errors=$((errors + 1)); \
		fi; \
		if grep -q "your-" .env.prod; then \
			echo "❌ .env.prod contains template values (your-*)"; \
			errors=$((errors + 1)); \
		fi; \
	fi; \
	if [ ! -f deployments/docker/docker-compose.prod.yml ]; then \
		echo "❌ docker-compose.prod.yml missing"; \
		errors=$((errors + 1)); \
	fi; \
	if [ ! -f deployments/docker/Dockerfile.prod ]; then \
		echo "❌ Dockerfile.prod missing"; \
		errors=$((errors + 1)); \
	fi; \
	if [ $errors -eq 0 ]; then \
		echo "✅ Ready for production deployment!"; \
	else \
		echo "❌ $errors issues found. Please fix before deploying."; \
		exit 1; \
	fi

# Test production build locally
test-prod-build: ## Test production build locally
	@echo "🧪 Testing production build..."
	@if [ ! -f .env.prod ]; then \
		echo "❌ .env.prod not found"; \
		exit 1; \
	fi
	@echo "Building production image..."
	@docker build -f ./deployments/docker/Dockerfile.prod -t evently-test:latest .
	@echo "✅ Production build test successful!"

.DEFAULT_GOAL := help
