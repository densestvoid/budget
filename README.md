# Budget App

[![CI](https://github.com/densestvoid/budget/actions/workflows/ci.yml/badge.svg)](https://github.com/densestvoid/budget/actions/workflows/ci.yml)
[![Deploy to Production](https://github.com/densestvoid/budget/actions/workflows/deploy-production.yml/badge.svg)](https://github.com/densestvoid/budget/actions/workflows/deploy-production.yml)

A modern budget management web application built with Go, featuring Chi router, PostgreSQL, HTMX, Alpine.js, Bootstrap 5, and Gomponents.

## Features

- **Backend**: Go with Chi router for HTTP routing
- **Database**: PostgreSQL with embedded Goose migrations
- **Frontend**: HTMX for dynamic interactions, Alpine.js for reactive UI
- **Styling**: Bootstrap 5 for responsive design
- **HTML Generation**: Gomponents for type-safe HTML
- **CLI**: Cobra and Viper for configuration management
- **Development**: Go Task for build automation
- **Quality**: Comprehensive linting and security tools
- **Architecture**: Clean separation of concerns with layered architecture

## Prerequisites

- Go 1.26 or later
- PostgreSQL (for local development without Docker)
- Docker (recommended for local development)

## Quick Start

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd budget
   ```

2. **Start the full dev stack (recommended)**
   ```bash
   go tool task dev
   ```

   This starts PostgreSQL, runs Goose migrations, and launches the app via Docker Compose.

3. **Or run locally without Docker**
   ```bash
   # Start PostgreSQL, then run migrations and the app on the host
   go tool task migrate-up
   go tool task dev-local
   ```

4. **Access the application**
   - Web interface: http://localhost:8080
   - Health check: http://localhost:8080/api/health

## Development

### Available Tasks

```bash
# Build and run
go tool task build      # Build the application
go tool task run        # Run the built application
go tool task dev        # Run with Docker Compose (postgres, migrations, app)
go tool task dev-local  # Run locally without Docker (requires local postgres)
go tool task watch      # Run with live reload using Air

# Database operations
go tool task migrate-up     # Run migrations up
go tool task migrate-down   # Rollback migrations
go tool task migrate-status # Show migration status

# Code quality and security
go tool task lint           # Run all linting and security checks
go tool task lint-golangci  # Run golangci-lint
go tool task lint-staticcheck # Run staticcheck
go tool task lint-gosec     # Run gosec security checks
go tool task lint-govulncheck # Run govulncheck

# Testing
go tool task test       # Run tests

# Docker
go tool task docker-build # Build production Docker image
go tool task docker-run   # Alias for dev (Docker Compose)

# Utilities
go tool task clean      # Clean build artifacts
go tool task help       # Show available tasks
go tool task cli-help   # Show CLI help
```

### Code Quality Tools

The project includes several code quality and security tools:

- **golangci-lint**: Comprehensive Go linter with multiple rules
- **staticcheck**: Advanced static analysis for Go
- **gosec**: Security linter for Go code
- **govulncheck**: Vulnerability scanner for Go dependencies

Run all checks with:
```bash
go tool task lint
```

### CLI Commands

The application provides a rich CLI interface:

```bash
# Show help
go run main.go --help

# Start the web server
go run main.go serve [--port 8080] [--env development]

# Database migrations (using embedded migration files)
go run main.go migrate up      # Apply migrations
go run main.go migrate down    # Rollback migrations
go run main.go migrate status  # Show migration status
```

## Configuration

The application uses Viper for configuration management. Configuration can be provided via:

1. **Config file**: `config.yaml` (default) or `~/.budget.yaml`
2. **Environment variables**: Prefixed with `BUDGET_`
3. **Command line flags**: `--port`, `--env`, etc.

### Example Configuration

```yaml
# config.yaml
port: 8080
env: development
database:
  url: postgres://postgres:password@localhost:5432/budget?sslmode=disable
log:
  level: info
```

### Environment Variables

```bash
export BUDGET_PORT=8080
export BUDGET_ENV=production
export BUDGET_DATABASE_URL="postgres://user:pass@host:5432/db"
```

## Project Structure

The project follows a clean layered architecture with clear separation of concerns:

```
budget/
├── cmd/                    # CLI commands and entry points
│   ├── migrate.go         # Database migration commands
│   ├── root.go            # Root command and configuration
│   └── serve.go           # Web server command
├── app/                   # Application logic and business layer
│   └── app.go             # Main application with dependencies
├── server/                # Web server and HTTP handling
│   └── server.go          # HTTP server, routes, and middleware
├── data/                  # Data access and persistence layer
│   ├── storage.go         # Database connection and storage interface
│   ├── migrations.go      # Embedded migration functionality
│   ├── 000001_create_users_table.up.sql
│   └── 000001_create_users_table.down.sql
├── templates/             # HTML templates using Gomponents
│   ├── layout.go          # Base layout template
│   └── pages.go           # Page-specific templates
├── assets/                # Static assets (CSS, JS, images)
│   └── css/
│       └── app.css
├── config.yaml            # Default configuration
├── docker-compose.yml     # Local dev stack (postgres, migrations, app)
├── Dockerfile             # Production Docker image definition
├── Dockerfile.migrate     # Docker image for running migrations in compose
├── go.mod                 # Go module file
├── go.sum                 # Go module checksums
├── main.go                # Application entry point
├── Taskfile.yml           # Build automation tasks
└── README.md              # This file
```

### Architecture Layers

1. **CLI Layer** (`cmd/`): Command-line interface and application entry points
2. **Application Layer** (`app/`): Business logic and application services
3. **Server Layer** (`server/`): HTTP server, routing, and middleware
4. **Data Layer** (`data/`): Database operations and data persistence
5. **Presentation Layer** (`templates/`, `assets/`): Templates, static assets, and frontend code

## Database Migrations

The application uses embedded Goose migrations, which means:

- **No external dependencies**: Migration files are embedded in the binary
- **Self-contained**: The application can run migrations without external tools
- **Version controlled**: Migration files are part of the source code
- **Easy deployment**: No need to install or configure external migration tools

### Migration Commands

```bash
# Apply all pending migrations
go run main.go migrate

# Rollback the last migration
go run main.go migrate down

# Show migration status
go run main.go migrate status
```

### Adding New Migrations

1. Create new `.sql` files in the `data/` directory
2. Follow the naming convention: `{version}_{description}.{up|down}.sql`
3. The migrations will be automatically embedded and available

## Docker

### Using Docker Compose

```bash
# Start all services (PostgreSQL, migrations, app)
go tool task dev

# Or manually
docker-compose up --build
```

The compose stack runs in order: Postgres becomes healthy, the migrate service applies pending Goose migrations, then the app starts. The app service uses the `builder` stage from `Dockerfile` so `go run` works with the mounted source tree.

### Building Docker Image

```bash
go tool task docker-build
```

## Technologies

- **Go 1.26+**: Backend language
- **Chi**: HTTP router and middleware
- **PostgreSQL**: Database
- **Goose**: Embedded database migrations
- **HTMX**: Dynamic web interactions
- **Alpine.js**: Reactive UI components
- **Bootstrap 5**: CSS framework
- **Gomponents**: Type-safe HTML generation
- **Cobra**: CLI framework
- **Viper**: Configuration management
- **Go Task**: Build automation
- **Docker**: Containerization

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run the linting tools: `go tool task lint`
5. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
