#!/bin/bash
set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to run migration with comprehensive logging
run_migrations() {
    print_status "🚀 Budget App Migration Starting"
    echo "====================================="
    
    # Check if binary exists
    if [ ! -f "./budget" ]; then
        print_error "Budget binary not found in current directory"
        exit 1
    fi
    
    # Log environment for debugging (without sensitive data)
    print_status "🔍 Environment Variables:"
    env | grep BUDGET | sed 's/=.*PASSWORD.*=.*$/=[REDACTED]/' | sort
    echo ""
    
    # Test database connectivity first
    print_status "🔍 Testing database connectivity..."
    if ! timeout 30 ./budget migrate status; then
        print_error "Failed to connect to database or check migration status"
        print_error "This could indicate:"
        print_error "  - Database is not accessible"
        print_error "  - Database credentials are incorrect" 
        print_error "  - Database is not ready yet"
        print_error "  - Migration binary is not working"
        
        # Try to show database connection details (without password)
        if [ ! -z "$BUDGET_DATABASE_URL" ]; then
            # Extract host, port, database from URL (safely)
            DB_INFO=$(echo "$BUDGET_DATABASE_URL" | sed 's/:\/\/.*@/:\/\/[USER:PASS]@/')
            print_error "Database URL (sanitized): $DB_INFO"
        fi
        
        exit 1
    fi
    
    print_success "✅ Database connection successful"
    echo ""
    
    # Show current migration status
    print_status "📊 Current Migration Status:"
    ./budget migrate status
    echo ""
    
    # Run migrations with error handling
    print_status "🔄 Running database migrations..."
    if timeout 300 ./budget migrate; then  # 5 minute timeout
        print_success "✅ Migrations completed successfully"
    else
        print_error "❌ Migration failed or timed out"
        print_error "Checking migration status after failure:"
        ./budget migrate status || true
        
        print_error "Migration failure could be due to:"
        print_error "  - Database schema conflicts"
        print_error "  - Insufficient database permissions"
        print_error "  - Database connection lost during migration"
        print_error "  - Migration timeout (>5 minutes)"
        
        exit 1
    fi
    
    # Verify final state
    echo ""
    print_status "📊 Final Migration Status:"
    ./budget migrate status
    
    echo ""
    print_success "🎉 Migration completed successfully!"
    echo "====================================="
}

# Function to check migration prerequisites
check_prerequisites() {
    print_status "🔍 Checking migration prerequisites..."
    
    # Check required environment variables
    if [ -z "$BUDGET_DATABASE_URL" ]; then
        print_error "BUDGET_DATABASE_URL environment variable is not set"
        exit 1
    fi
    
    # Check if budget binary exists and is executable
    if [ ! -f "./budget" ]; then
        print_error "Budget binary not found in current directory"
        exit 1
    fi
    
    if [ ! -x "./budget" ]; then
        print_error "Budget binary is not executable"
        exit 1
    fi
    
    print_success "✅ Prerequisites check passed"
}

# Function to show help
show_help() {
    echo "Budget App Migration Script"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  run      Run database migrations (default)"
    echo "  status   Show current migration status"
    echo "  check    Check prerequisites only"
    echo "  help     Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  BUDGET_DATABASE_URL    Database connection URL (required)"
    echo "  BUDGET_LOG_LEVEL       Log level (optional, default: info)"
    echo ""
    exit 0
}

# Main function
main() {
    case "${1:-run}" in
        "run")
            check_prerequisites
            run_migrations
            ;;
        "status")
            check_prerequisites
            print_status "📊 Current Migration Status:"
            ./budget migrate status
            ;;
        "check")
            check_prerequisites
            print_success "✅ All prerequisites met"
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            print_error "Unknown command: $1"
            show_help
            ;;
    esac
}

# Run main function with all arguments
main "$@"