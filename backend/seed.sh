#!/bin/bash

# Database Seed Script Runner
# This script provides a convenient way to run the database seeder

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_color() {
    printf "${1}%s${NC}\n" "$2"
}

# Function to print header
print_header() {
    echo ""
    print_color $BLUE "=================================="
    print_color $BLUE "$1"
    print_color $BLUE "=================================="
    echo ""
}

# Function to check if database is accessible
check_database() {
    print_color $YELLOW "🔍 Checking database connection..."
    
    # Check if PostgreSQL is accessible (basic check)
    if ! command -v psql &> /dev/null; then
        print_color $YELLOW "   psql command not found, skipping connection check"
        return 0
    fi
    
    # Try to connect to database (using environment variables or defaults)
    DB_HOST=${DB_HOST:-localhost}
    DB_PORT=${DB_PORT:-5432}
    DB_NAME=${DB_NAME:-evently_db}
    DB_USER=${DB_USER:-evently_user}
    
    if PGPASSWORD="${DB_PASSWORD:-evently_password}" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" &> /dev/null; then
        print_color $GREEN "   ✅ Database connection successful"
    else
        print_color $YELLOW "   ⚠️  Could not verify database connection, but proceeding anyway"
        print_color $YELLOW "      Make sure your database is running and environment variables are set"
    fi
}

# Function to show environment info
show_environment() {
    print_color $BLUE "📊 Environment Information:"
    echo "   Database Host: ${DB_HOST:-localhost (default)}"
    echo "   Database Port: ${DB_PORT:-5432 (default)}"
    echo "   Database Name: ${DB_NAME:-evently_db (default)}"
    echo "   Database User: ${DB_USER:-evently_user (default)}"
    echo "   Redis Host: ${REDIS_HOST:-localhost (default)}"
    echo "   Redis Port: ${REDIS_PORT:-6379 (default)}"
}

# Main script
main() {
    print_header "🌱 Evently Database Seeder"
    
    # Check if we're in the right directory
    if [[ ! -f "cmd/seed/main.go" ]]; then
        print_color $RED "❌ Error: This script must be run from the backend directory"
        print_color $YELLOW "   Current directory: $(pwd)"
        print_color $YELLOW "   Expected file: cmd/seed/main.go"
        echo ""
        print_color $BLUE "💡 Try running: cd backend && ./seed.sh"
        exit 1
    fi
    
    # Show environment info
    show_environment
    echo ""
    
    # Check database connection
    check_database
    echo ""
    
    # Show warning
    print_color $RED "⚠️  WARNING: DATABASE CLEANUP"
    print_color $YELLOW "   This script will:"
    print_color $YELLOW "   • Clean ALL existing data from the database"
    print_color $YELLOW "   • Insert fresh seed data for testing"
    print_color $YELLOW "   • Clear Redis cache"
    echo ""
    print_color $RED "   This action is IRREVERSIBLE!"
    echo ""
    
    # Ask for confirmation
    read -p "$(print_color $BLUE "🤔 Are you sure you want to continue? (y/N): ")" -n 1 -r
    echo ""
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_color $YELLOW "   Operation cancelled by user."
        exit 0
    fi
    
    echo ""
    print_header "🚀 Starting Database Seeding Process"
    
    # Build the seeder first
    print_color $YELLOW "📦 Building seeder..."
    if go build -o bin/seed cmd/seed/main.go; then
        print_color $GREEN "   ✅ Build successful"
    else
        print_color $RED "   ❌ Build failed"
        exit 1
    fi
    
    echo ""
    
    # Run the seeder
    print_color $YELLOW "🌱 Running seeder..."
    echo ""
    
    if go run cmd/seed/main.go; then
        echo ""
        print_header "🎉 Seeding Completed Successfully!"
        
        print_color $GREEN "📊 What was created:"
        print_color $GREEN "   👤 Users: 3 (1 admin, 2 regular users)"
        print_color $GREEN "   🏟️  Venue Templates: 2 with sections and seats"
        print_color $GREEN "   🎪 Events: 6 upcoming events with pricing"
        print_color $GREEN "   📋 Cancellation Policies: 6 different policies"
        print_color $GREEN "   🏷️  Tags: 6 event categories"
        
        echo ""
        print_color $BLUE "🔑 Test Credentials:"
        print_color $BLUE "   Admin: admin@gmail.com / qwerty"
        print_color $BLUE "   User1: mitshah2406@gmail.com / qwerty"
        print_color $BLUE "   User2: mitshah2406.work@gmail.com / qwerty"
        
        echo ""
        print_color $GREEN "💡 Next Steps:"
        print_color $GREEN "   • Start your server: make run or go run server/main.go"
        print_color $GREEN "   • Test authentication with the credentials above"
        print_color $GREEN "   • Browse events: GET /api/v1/events"
        print_color $GREEN "   • Test booking flow with available seats"
        
        echo ""
        print_color $BLUE "📖 For detailed information, see: SEED_README.md"
        
    else
        echo ""
        print_color $RED "❌ Seeding failed!"
        print_color $YELLOW "💡 Troubleshooting tips:"
        print_color $YELLOW "   • Check database connection and credentials"
        print_color $YELLOW "   • Ensure database migrations have been run"
        print_color $YELLOW "   • Verify environment variables are set correctly"
        print_color $YELLOW "   • Check logs above for specific error messages"
        exit 1
    fi
}

# Run main function
main "$@"