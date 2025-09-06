# Budget App

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

- Go 1.24 or later
- PostgreSQL
- Docker (optional)

## Quick Start

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd budget
   ```

2. **Set up the database**
   ```bash
   # Using Docker Compose (recommended)
   go tool task docker-run
   
   # Or manually start PostgreSQL and run migrations
   go tool task migrate-up
   ```

3. **Run the application**
   ```bash
   # Development mode with live reload
   go tool task watch
   
   # Or standard development mode
   go tool task dev
   
   # Or build and run
   go tool task run
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
go tool task dev        # Run in development mode
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
go tool task docker-build # Build Docker image
go tool task docker-run   # Run with Docker Compose

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
├── docker-compose.yml     # Docker Compose setup
├── Dockerfile             # Docker image definition
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
# Start all services (PostgreSQL + App)
go tool task docker-run

# Or manually
docker-compose up --build
```

### Building Docker Image

```bash
go tool task docker-build
```

## Technologies

- **Go 1.24+**: Backend language
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
