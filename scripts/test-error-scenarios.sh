#!/bin/bash

# Test script for error handling and logging scenarios
# This script can be used to validate that our error handling works correctly

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

print_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

# Base URL for testing (can be overridden)
BASE_URL="${BASE_URL:-http://localhost:8080}"

# Test counter
TESTS_RUN=0
TESTS_PASSED=0

# Function to run a test
run_test() {
    local test_name="$1"
    local test_command="$2"
    local expected_status="$3"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    print_status "Running: $test_name"
    
    # Run the test command and capture both status and output
    if response=$(eval "$test_command" 2>&1); then
        actual_status=0
    else
        actual_status=$?
    fi
    
    # Check if status matches expectation
    if [ "$actual_status" -eq "$expected_status" ]; then
        print_success "$test_name"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo "  Response: $response"
    else
        print_error "$test_name (expected status $expected_status, got $actual_status)"
        echo "  Response: $response"
    fi
    echo ""
}

# Function to test HTTP endpoints
test_http_endpoint() {
    local endpoint="$1"
    local method="$2"
    local expected_status="$3"
    local data="$4"
    local description="$5"
    
    local curl_cmd="curl -s -w '%{http_code}' -X $method"
    
    if [ ! -z "$data" ]; then
        curl_cmd="$curl_cmd -H 'Content-Type: application/x-www-form-urlencoded' -d '$data'"
    fi
    
    curl_cmd="$curl_cmd '$BASE_URL$endpoint'"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    print_status "Testing: $description"
    
    # Execute curl and capture response
    response=$(eval "$curl_cmd" 2>/dev/null || echo "CURL_FAILED")
    
    if [ "$response" = "CURL_FAILED" ]; then
        print_error "$description (curl failed - server might not be running)"
    else
        # Extract status code (last 3 characters)
        actual_status="${response: -3}"
        response_body="${response%???}"
        
        if [ "$actual_status" = "$expected_status" ]; then
            print_success "$description (Status: $actual_status)"
            TESTS_PASSED=$((TESTS_PASSED + 1))
            if [ ! -z "$response_body" ] && [ "$response_body" != "null" ]; then
                echo "  Response body: ${response_body:0:200}..."
            fi
        else
            print_error "$description (expected $expected_status, got $actual_status)"
            echo "  Response body: ${response_body:0:200}..."
        fi
    fi
    echo ""
}

# Main test function
run_error_tests() {
    echo "🧪 Budget App Error Handling Test Suite"
    echo "======================================="
    echo "Testing against: $BASE_URL"
    echo ""
    
    # Test 1: Health endpoint (should work)
    test_http_endpoint "/health" "GET" "200" "" "Health check endpoint"
    
    # Test 2: Non-existent endpoint (404)
    test_http_endpoint "/nonexistent" "GET" "404" "" "Non-existent endpoint"
    
    # Test 3: Registration with missing data (400)
    test_http_endpoint "/auth/register" "POST" "400" "" "Registration with no data"
    
    # Test 4: Registration with incomplete data (400)
    test_http_endpoint "/auth/register" "POST" "400" "email=test@example.com" "Registration with incomplete data"
    
    # Test 5: Login with missing data (400)
    test_http_endpoint "/auth/login" "POST" "400" "" "Login with no data"
    
    # Test 6: Login with incomplete data (400)
    test_http_endpoint "/auth/login" "POST" "400" "email=test@example.com" "Login with incomplete data"
    
    # Test 7: Protected endpoint without authentication (401 or redirect)
    test_http_endpoint "/categories" "GET" "303" "" "Protected endpoint without auth"
    
    # Test 8: Invalid method on endpoint (405)
    test_http_endpoint "/health" "DELETE" "405" "" "Invalid HTTP method"
    
    # Test 9: Large request body (413 - if configured)
    large_data=$(printf 'a%.0s' {1..10000})  # 10KB of 'a' characters
    test_http_endpoint "/auth/register" "POST" "400" "email=test@example.com&password=pass&name=$large_data" "Large request body"
    
    # Test 10: Invalid content type
    TESTS_RUN=$((TESTS_RUN + 1))
    print_status "Testing: Invalid content type"
    response=$(curl -s -w '%{http_code}' -X POST -H 'Content-Type: application/json' -d '{"invalid": "json"}' "$BASE_URL/auth/register" 2>/dev/null || echo "CURL_FAILED")
    
    if [ "$response" != "CURL_FAILED" ]; then
        actual_status="${response: -3}"
        if [ "$actual_status" = "400" ] || [ "$actual_status" = "415" ]; then
            print_success "Invalid content type (Status: $actual_status)"
            TESTS_PASSED=$((TESTS_PASSED + 1))
        else
            print_error "Invalid content type (expected 400 or 415, got $actual_status)"
        fi
    else
        print_error "Invalid content type (curl failed)"
    fi
    echo ""
    
    # Summary
    echo "🏁 Test Results"
    echo "==============="
    echo "Tests run: $TESTS_RUN"
    echo "Tests passed: $TESTS_PASSED"
    echo "Tests failed: $((TESTS_RUN - TESTS_PASSED))"
    
    if [ $TESTS_PASSED -eq $TESTS_RUN ]; then
        print_success "All tests passed! 🎉"
        exit 0
    else
        print_error "Some tests failed. Check the output above for details."
        exit 1
    fi
}

# Function to test database connection scenarios
test_database_scenarios() {
    echo "🗄️  Database Connection Test Scenarios"
    echo "====================================="
    
    # Test migration script with various scenarios
    print_status "Testing migration script prerequisites check"
    
    # Save original DATABASE_URL if it exists
    ORIGINAL_DB_URL="$BUDGET_DATABASE_URL"
    
    # Test 1: Missing DATABASE_URL
    unset BUDGET_DATABASE_URL
    run_test "Migration without DATABASE_URL" "./scripts/run-migrations.sh check" 1
    
    # Test 2: Invalid DATABASE_URL
    export BUDGET_DATABASE_URL="invalid://url"
    run_test "Migration with invalid DATABASE_URL" "./scripts/run-migrations.sh check" 0  # Should pass prereq check
    
    # Test 3: Missing binary
    export BUDGET_DATABASE_URL="postgres://user:pass@localhost:5432/db"
    mv budget budget.backup 2>/dev/null || true
    run_test "Migration without binary" "./scripts/run-migrations.sh check" 1
    mv budget.backup budget 2>/dev/null || true
    
    # Restore original DATABASE_URL
    if [ ! -z "$ORIGINAL_DB_URL" ]; then
        export BUDGET_DATABASE_URL="$ORIGINAL_DB_URL"
    else
        unset BUDGET_DATABASE_URL
    fi
    
    echo ""
}

# Function to show help
show_help() {
    echo "Budget App Error Testing Script"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  http       Test HTTP error scenarios (default)"
    echo "  database   Test database connection scenarios"
    echo "  all        Run all test scenarios"
    echo "  help       Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  BASE_URL   Base URL for HTTP tests (default: http://localhost:8080)"
    echo ""
    exit 0
}

# Check if server is running
check_server() {
    if ! curl -s "$BASE_URL/health" > /dev/null 2>&1; then
        print_error "Server is not running at $BASE_URL"
        print_status "Please start the server first with: go run main.go serve"
        print_status "Or set BASE_URL to point to a running server"
        exit 1
    fi
    print_success "Server is running at $BASE_URL"
    echo ""
}

# Main function
main() {
    case "${1:-http}" in
        "http")
            check_server
            run_error_tests
            ;;
        "database")
            test_database_scenarios
            ;;
        "all")
            check_server
            run_error_tests
            echo ""
            test_database_scenarios
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