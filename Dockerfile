# Development stage with Air for hot reloading
FROM golang:1.25-alpine

WORKDIR /app

# Install git, ca-certificates, and Air for hot reloading
RUN go install github.com/air-verse/air@latest

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Expose port
EXPOSE 8080

# Use Air for development with hot reloading
CMD ["air"]