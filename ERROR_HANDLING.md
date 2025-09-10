# Error Handling and Logging System

This document describes the comprehensive error handling and logging system implemented in the Budget App.

## Overview

The application now includes:
- **Structured Logging**: JSON-formatted logs with context and metadata
- **Centralized Error Handling**: Consistent error responses across all endpoints
- **Comprehensive HTTP Error Tracking**: All 4xx and 5xx responses are logged with details
- **Enhanced Migration Handling**: Robust database migration with proper error handling
- **Terraform Integration**: Migration jobs that properly fail deployments on error

## Components

### 1. Error Handler (`server/errors.go`)

The `ErrorHandler` provides centralized error processing with:

- **Structured Error Logging**: All errors are logged as JSON with context
- **Request Context Capture**: Includes request details, user info, and form data
- **Stack Traces**: Automatically included for 5xx errors
- **Client-Safe Responses**: Internal error details are hidden from clients
- **Error Levels**: Automatic categorization (info, warning, error, critical)

**Usage in handlers:**
```go
// Simple error
HTTPError(w, r, http.StatusBadRequest, "Invalid input", nil)

// Error with underlying cause
HTTPError(w, r, http.StatusInternalServerError, "Database error", err)

// Formatted error message
HTTPErrorf(w, r, http.StatusNotFound, "User %s not found", userID)
```

### 2. Structured Logger (`server/logger.go`)

The `Logger` provides JSON-formatted logging with:

- **Multiple Log Levels**: Debug, Info, Warn, Error, Fatal
- **Structured Fields**: Key-value pairs for searchable logs
- **Environment-Based Configuration**: Log level set via `LOG_LEVEL` env var
- **Helper Functions**: Specialized logging for HTTP requests, DB operations, auth events

**Usage examples:**
```go
// Basic logging
Info("User logged in", map[string]interface{}{
    "user_id": 123,
    "email": "user@example.com",
})

// Error logging
Error("Database connection failed", err, map[string]interface{}{
    "operation": "user_lookup",
    "user_id": 123,
})

// HTTP request logging (automatic)
LogHTTPRequest("POST", "/auth/login", userAgent, remoteAddr, duration, 200)

// Authentication events
LogAuthentication("login", "user@example.com", "success", map[string]interface{}{
    "account_id": 123,
})
```

### 3. Enhanced Middleware

**Error Middleware**: Automatically captures all HTTP responses ≥400 and logs them with full context.

**Logging Middleware**: Logs all HTTP requests with timing and response codes.

**Integration**: Both middleware are automatically applied to all routes.

## Error Response Format

All error responses follow a consistent JSON format:

```json
{
  "error": true,
  "code": 400,
  "message": "Invalid input data",
  "timestamp": "2025-09-10T12:34:56Z",
  "request_id": "abc123def456"
}
```

**Note**: Internal error details (stack traces, database errors) are only included in logs, not client responses.

## Log Format

All logs are structured JSON for easy parsing by log aggregation systems:

```json
{
  "level": "ERROR",
  "message": "Registration failed - database error",
  "timestamp": "2025-09-10T12:34:56.789Z",
  "fields": {
    "event": "registration",
    "email": "user@example.com",
    "result": "database_error",
    "operation": "create_account",
    "error": "connection refused"
  },
  "error": "dial tcp [::1]:5432: connect: connection refused"
}
```

## Database Migration Error Handling

### Enhanced Migration Script (`scripts/run-migrations.sh`)

Features:
- **Prerequisite Checking**: Validates environment and binary before running
- **Database Connectivity Testing**: Tests connection before attempting migrations
- **Comprehensive Error Messages**: Detailed failure explanations
- **Timeout Protection**: Prevents hanging migrations
- **Status Verification**: Confirms successful completion

**Usage:**
```bash
# Run migrations
./scripts/run-migrations.sh run

# Check status only
./scripts/run-migrations.sh status

# Check prerequisites
./scripts/run-migrations.sh check
```

### Terraform Integration (`terraform/migration_job.tf`)

The Terraform configuration includes:
- **Pre-Deploy Migration Job**: Runs before the main application
- **Failure Propagation**: Application deployment fails if migrations fail
- **Enhanced Logging**: Comprehensive migration logging in deployment
- **Resource Limits**: Prevents runaway migration jobs
- **Validation Steps**: Confirms migration success before proceeding

**Key Features:**
- Uses `PRE_DEPLOY` job kind to ensure migrations complete before app deployment
- Includes `depends_on` relationships to enforce proper ordering
- Migration failures will stop the entire deployment
- Comprehensive logging for debugging deployment issues

## Error Scenarios Covered

### HTTP Errors (4xx/5xx)
- ✅ 400 Bad Request (invalid form data, missing fields)
- ✅ 401 Unauthorized (invalid credentials, missing auth)
- ✅ 403 Forbidden (insufficient permissions)
- ✅ 404 Not Found (missing resources, invalid routes)
- ✅ 405 Method Not Allowed (wrong HTTP method)
- ✅ 409 Conflict (duplicate accounts, constraint violations)
- ✅ 413 Payload Too Large (oversized requests)
- ✅ 415 Unsupported Media Type (wrong content-type)
- ✅ 500 Internal Server Error (database errors, system failures)
- ✅ 503 Service Unavailable (database unavailable, system overload)

### Database Errors
- ✅ Connection failures (network issues, wrong credentials)
- ✅ Migration failures (schema conflicts, permission issues)
- ✅ Query timeouts (long-running operations)
- ✅ Constraint violations (unique key conflicts, foreign key errors)
- ✅ Transaction failures (deadlocks, rollbacks)

### Authentication Errors
- ✅ Invalid credentials (wrong email/password)
- ✅ Missing session tokens (expired or invalid sessions)
- ✅ Account creation failures (duplicate emails, database errors)
- ✅ Session creation failures (database issues)

## Testing Error Handling

Use the included test script to validate error handling:

```bash
# Test HTTP error scenarios (requires running server)
./scripts/test-error-scenarios.sh http

# Test database scenarios
./scripts/test-error-scenarios.sh database

# Run all tests
./scripts/test-error-scenarios.sh all
```

## Configuration

### Environment Variables

- `LOG_LEVEL`: Set log level (DEBUG, INFO, WARN, ERROR, FATAL)
- `BUDGET_ENV`: Environment (development, production)
- `BUDGET_DATABASE_URL`: Database connection string

### Development vs Production

**Development:**
- Stack traces included in error logs
- More verbose logging (DEBUG level available)
- Detailed error context in logs

**Production:**
- Stack traces only for critical errors
- INFO level logging by default
- Sanitized error responses to clients
- Structured JSON logs for aggregation

## Log Aggregation

The structured JSON logs are designed for easy integration with:
- **ELK Stack** (Elasticsearch, Logstash, Kibana)
- **Fluentd/Fluent Bit**
- **Grafana Loki**
- **Datadog**
- **New Relic**
- **AWS CloudWatch**

Example Elasticsearch mapping:
```json
{
  "mappings": {
    "properties": {
      "level": { "type": "keyword" },
      "message": { "type": "text" },
      "timestamp": { "type": "date" },
      "fields": { "type": "object" },
      "error": { "type": "text" }
    }
  }
}
```

## Monitoring and Alerting

Recommended alerts based on log data:
- **High Error Rate**: >5% of requests returning 5xx in 5 minutes
- **Authentication Failures**: >10 failed logins per minute
- **Database Errors**: Any database connection failures
- **Migration Failures**: Any migration job failures
- **Critical Errors**: Any log entry with level "CRITICAL"

## Troubleshooting

### Common Issues

**1. Registration Returns 500 Error**
- Check logs for database connection errors
- Verify `BUDGET_DATABASE_URL` is correct
- Ensure database migrations have run successfully
- Check if `sessions` table exists

**2. Migration Job Fails in Terraform**
- Check DigitalOcean App Platform logs
- Verify database is accessible from VPC
- Confirm environment variables are set correctly
- Check migration script output in deployment logs

**3. Missing Error Context in Logs**
- Ensure handlers use `HTTPError()` instead of `http.Error()`
- Check that error middleware is properly configured
- Verify log level allows the error level being logged

### Log Analysis Queries

**Find all authentication failures:**
```bash
grep '"event":"login"' logs.json | grep '"result":"authentication_failed"'
```

**Find all 5xx errors:**
```bash
grep '"level":"ERROR"' logs.json | grep '"code":[5][0-9][0-9]'
```

**Find database connection issues:**
```bash
grep '"error":".*connection.*refused"' logs.json
```

## Best Practices

1. **Always use `HTTPError()` for error responses** instead of `http.Error()`
2. **Include relevant context** in error logs (user ID, operation, etc.)
3. **Use appropriate log levels** (ERROR for failures, WARN for recoverable issues)
4. **Don't log sensitive data** (passwords, tokens, etc.)
5. **Test error scenarios** regularly with the provided test script
6. **Monitor error rates** and set up appropriate alerts
7. **Review error logs** regularly to identify patterns and issues