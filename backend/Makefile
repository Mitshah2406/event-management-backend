.PHONY: \
  build \
  run \
  dev \
  setup \
  test \
  clean \
  docker-up \
  docker-down \
  prod-build \
  prod-up \
  prod-down \
  prod-restart \
  prod-logs \
  prod-logs-app \
  prod-logs-db \
  prod-logs-redis \
  prod-logs-kafka \
  prod-status \
  prod-db-info \
  prod-redis-info \
  prod-connect-db \
  prod-connect-redis \
  seed \
  help

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

# Development Docker commands
docker-up: ## Start development database services
	@echo "Starting development services..."
	docker compose -f ./deployments/docker/docker-compose.dev.yml up -d

docker-down: ## Stop development database services
	@echo "Stopping development services..."
	docker compose -f ./deployments/docker/docker-compose.dev.yml down

# Production Docker commands
prod-build: ## Build production Docker images
	@echo "Building production Docker images..."
	@if [ ! -f .env ]; then \
		echo "❌ .env file not found. Please create it first."; \
		exit 1; \
	fi
	docker compose -f ./deployments/docker/docker-compose.prod.yml build
	@echo "✅ Production images built successfully!"

prod-up: ## Start production environment
	@echo "Starting production environment..."
	@if [ ! -f .env ]; then \
		echo "❌ .env file not found. Please create it first."; \
		exit 1; \
	fi
	docker compose -f ./deployments/docker/docker-compose.prod.yml --env-file .env up -d --build
	@echo ""
	@echo "🚀 Production services started successfully!"
	@echo ""
	@echo "📋 Service URLs:"
	@echo "   🌐 App: http://localhost:9000"
	@echo "   ❤️  Health Check: http://localhost:9000/health"
	@echo "   🗄️  PostgreSQL: localhost:9001"
	@echo "   🔴 Redis: localhost:9002"
	@echo "   📨 Kafka: localhost:9003"
	@echo ""
	@echo "💡 Use 'make prod-status' to check service health"
	@echo "💡 Use 'make prod-logs' to view all logs"

prod-down: ## Stop production environment
	@echo "Stopping production environment..."
	docker compose -f ./deployments/docker/docker-compose.prod.yml down
	@echo "🛑 Production services stopped"

prod-restart: ## Restart production environment
	@echo "Restarting production environment..."
	docker compose -f ./deployments/docker/docker-compose.prod.yml restart
	@echo "🔄 Production services restarted"

# Production logging commands
prod-logs: ## View all production logs (follow mode)
	@echo "Viewing all production logs (press Ctrl+C to exit)..."
	docker compose -f ./deployments/docker/docker-compose.prod.yml logs -f

prod-logs-app: ## View application logs only
	@echo "Viewing application logs (press Ctrl+C to exit)..."
	docker compose -f ./deployments/docker/docker-compose.prod.yml logs -f app

prod-logs-db: ## View PostgreSQL logs only
	@echo "Viewing PostgreSQL logs (press Ctrl+C to exit)..."
	docker compose -f ./deployments/docker/docker-compose.prod.yml logs -f postgres

prod-logs-redis: ## View Redis logs only
	@echo "Viewing Redis logs (press Ctrl+C to exit)..."
	docker compose -f ./deployments/docker/docker-compose.prod.yml logs -f redis

prod-logs-kafka: ## View Kafka logs only
	@echo "Viewing Kafka logs (press Ctrl+C to exit)..."
	docker compose -f ./deployments/docker/docker-compose.prod.yml logs -f kafka

# Production status and health
prod-status: ## Check production services status and health
	@echo "📊 Production Services Status"
	@echo "==============================="
	@echo ""
	@echo "🔍 Container Status:"
	@docker compose -f ./deployments/docker/docker-compose.prod.yml ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}"
	@echo ""
	@echo "🏥 Health Checks:"
	@echo -n "   App Health: "
	@curl -f -s http://localhost:9000/health >/dev/null 2>&1 && echo "✅ Healthy" || echo "❌ Unhealthy"
	@echo -n "   PostgreSQL: "
	@docker exec evently_postgres pg_isready -U $$(grep DB_USER .env | cut -d'=' -f2) >/dev/null 2>&1 && echo "✅ Ready" || echo "❌ Not Ready"
	@echo -n "   Redis: "
	@if grep -q "REDIS_PASSWORD=" .env && [ -n "$$(grep REDIS_PASSWORD .env | cut -d'=' -f2)" ]; then \
		docker exec evently_redis redis-cli --pass "$$(grep REDIS_PASSWORD .env | cut -d'=' -f2)" ping >/dev/null 2>&1 && echo "✅ Responding" || echo "❌ Not Responding"; \
	else \
		docker exec evently_redis redis-cli ping >/dev/null 2>&1 && echo "✅ Responding" || echo "❌ Not Responding"; \
	fi
	@echo -n "   Kafka: "
	@docker exec evently_kafka kafka-broker-api-versions --bootstrap-server localhost:9092 >/dev/null 2>&1 && echo "✅ Ready" || echo "❌ Not Ready"

# Database connection information
prod-db-info: ## Show PostgreSQL connection details for production
	@echo "🗄️  PostgreSQL Production Connection Details"
	@echo "=============================================="
	@echo ""
	@echo "📋 Connection Information:"
	@echo "   Host: localhost (or your server IP)"
	@echo "   Port: 9001"
	@echo "   Database: $$(grep DB_NAME .env | cut -d'=' -f2 2>/dev/null || echo 'evently_prod')"
	@echo "   Username: $$(grep DB_USER .env | cut -d'=' -f2 2>/dev/null || echo 'evently_user')"
	@echo "   Password: $$(grep DB_PASSWORD .env | cut -d'=' -f2 2>/dev/null || echo 'check .env file')"
	@echo "   SSL Mode: disable"
	@echo ""
	@echo "🔧 Connection String:"
	@echo "   postgresql://$$(grep DB_USER .env | cut -d'=' -f2):[PASSWORD]@localhost:9001/$$(grep DB_NAME .env | cut -d'=' -f2)?sslmode=disable"
	@echo ""
	@echo "💻 Database Clients:"
	@echo "   • pgAdmin (Web-based)"
	@echo "   • Beekeeper Studio (Free)"
	@echo ""
	@echo "🐳 Docker CLI Access:"
	@echo "   docker exec -it evently_postgres psql -U $$(grep DB_USER .env | cut -d'=' -f2) -d $$(grep DB_NAME .env | cut -d'=' -f2)"

prod-redis-info: ## Show Redis connection details for production
	@echo "🔴 Redis Production Connection Details"
	@echo "======================================"
	@echo ""
	@echo "📋 Connection Information:"
	@echo "   Host: localhost (or your server IP)"
	@echo "   Port: 9002"
	@if grep -q "REDIS_PASSWORD=" .env && [ -n "$$(grep REDIS_PASSWORD .env | cut -d'=' -f2)" ]; then \
		echo "   Password: $$(grep REDIS_PASSWORD .env | cut -d'=' -f2)"; \
	else \
		echo "   Password: (no password set)"; \
	fi
	@echo "   Database: 0 (default)"
	@echo ""
	@echo "🔧 Connection String:"
	@if grep -q "REDIS_PASSWORD=" .env && [ -n "$$(grep REDIS_PASSWORD .env | cut -d'=' -f2)" ]; then \
		echo "   redis://:[PASSWORD]@localhost:9002/0"; \
	else \
		echo "   redis://localhost:9002/0"; \
	fi
	@echo ""
	@echo "💻 Redis Clients:"
	@echo "   • RedisInsight (Free, Official)"
	@echo ""
	@echo "🐳 Docker CLI Access:"
	@if grep -q "REDIS_PASSWORD=" .env && [ -n "$$(grep REDIS_PASSWORD .env | cut -d'=' -f2)" ]; then \
		echo "   docker exec -it evently_redis redis-cli -a $$(grep REDIS_PASSWORD .env | cut -d'=' -f2)"; \
	else \
		echo "   docker exec -it evently_redis redis-cli"; \
	fi

# Direct database connections
prod-connect-db: ## Connect to production PostgreSQL via CLI
	@echo "Connecting to PostgreSQL..."
	@docker exec -it evently_postgres psql -U $$(grep DB_USER .env | cut -d'=' -f2) -d $$(grep DB_NAME .env | cut -d'=' -f2)

prod-connect-redis: ## Connect to production Redis via CLI
	@echo "Connecting to Redis..."
	@if grep -q "REDIS_PASSWORD=" .env && [ -n "$$(grep REDIS_PASSWORD .env | cut -d'=' -f2)" ]; then \
		docker exec -it evently_redis redis-cli -a $$(grep REDIS_PASSWORD .env | cut -d'=' -f2); \
	else \
		docker exec -it evently_redis redis-cli; \
	fi

# Database seeding
seed: ## Seed the database with sample data (cleans DB first)
	@echo "🌱 Seeding database with sample data..."
	@echo ""
	@echo "⚠️  WARNING: This will clean the entire database and insert fresh seed data!"
	@echo "   This action is irreversible and will delete all existing data."
	@echo ""
	@read -p "Are you sure you want to continue? (y/N): " confirm && [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ] || exit 1
	@echo ""
	go run cmd/seed/main.go
	@echo ""
	@echo "🎉 Database seeded successfully!"
	@echo ""
	@echo "📊 Seed Data Summary:"
	@echo "   👤 Users: 3 (1 admin, 2 regular users)"
	@echo "   🏟️  Venue Templates: 2 (Small Theater, Conference Hall)"
	@echo "   🎪 Events: 6 (upcoming events with different pricing)"
	@echo "   📋 Cancellation Policies: 6 (different policy types)"
	@echo ""
	@echo "🔑 Login Credentials:"
	@echo "   Admin: admin@gmail.com / qwerty"
	@echo "   User1: mitshah2406@gmail.com / qwerty"
	@echo "   User2: mitshah2406.work@gmail.com / qwerty"

.DEFAULT_GOAL := help
