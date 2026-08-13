package cmd

// defaultLocalDSN is a dev-only fallback with no credentials (password via DATABASE_URL).
const defaultLocalDSN = "postgres://postgres@localhost:5432/budget?sslmode=disable"
