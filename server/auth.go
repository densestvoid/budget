package server

import (
	"budget/data"
	"budget/templates"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// Context key types to avoid collisions
type contextKey string

const (
	accountKey contextKey = "account"
)

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	store *data.Storage
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(store *data.Storage) *AuthHandler {
	return &AuthHandler{store: store}
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	Token string        `json:"token"`
	User  *data.Account `json:"user"`
}

// RegisterPage handles the registration page display
func (h *AuthHandler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	if err := templates.BaseLayoutWithAuth("Register - Budget App", false, templates.RegisterPage()).Render(w); err != nil {
		log.Printf("Error rendering register page: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// Register handles user registration from form submission
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	name := r.FormValue("name")

	// Validate input
	if email == "" || password == "" || name == "" {
		http.Error(w, "Email, password, and name are required", http.StatusBadRequest)
		return
	}

	// Check if account already exists
	existing, err := h.store.GetAccountByEmail(email)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if existing != nil {
		http.Error(w, "Account already exists", http.StatusConflict)
		return
	}

	// Create account
	account, err := h.store.CreateAccount(email, password, name)
	if err != nil {
		http.Error(w, "Failed to create account", http.StatusInternalServerError)
		return
	}

	// Create session
	session, err := h.store.CreateSession(account.ID, 24*time.Hour)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    session.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(24 * time.Hour.Seconds()),
	})

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Login handles user login from form submission
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	// Validate input
	if email == "" || password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	// Authenticate account
	account, err := h.store.AuthenticateAccount(email, password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Create session
	session, err := h.store.CreateSession(account.ID, 24*time.Hour)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    session.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(24 * time.Hour.Seconds()),
	})

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Logout handles user logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Get session token from cookie
	cookie, err := r.Cookie("session_token")
	if err == nil && cookie.Value != "" {
		// Delete session from database
		if err := h.store.DeleteSession(cookie.Value); err != nil {
			log.Printf("Error deleting session: %v", err)
		}
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// GetCurrentUser returns the current authenticated user
func (h *AuthHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	account := r.Context().Value("account").(*data.Account)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(account); err != nil {
		log.Printf("Error encoding account to JSON: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// AuthMiddleware authenticates requests using session tokens
func (h *AuthHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get session token from cookie
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get account from session
		account, err := h.store.GetAccountBySession(cookie.Value)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if account == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Add account to request context
		ctx := r.Context()
		ctx = context.WithValue(ctx, accountKey, account)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuthMiddleware optionally authenticates requests
func (h *AuthHandler) OptionalAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get session token from cookie
		cookie, err := r.Cookie("session_token")
		if err != nil {
			log.Printf("No session cookie found: %v", err)
		} else if cookie.Value == "" {
			log.Printf("Session cookie is empty")
		} else {
			log.Printf("Found session token: %s", cookie.Value[:10]+"...")
			// Get account from session
			account, err := h.store.GetAccountBySession(cookie.Value)
			if err != nil {
				log.Printf("Error getting account from session: %v", err)
			} else if account == nil {
				log.Printf("No account found for session token")
			} else {
				log.Printf("Found authenticated account: %s (%s)", account.Name, account.Email)
				// Add account to request context
				ctx := r.Context()
				ctx = context.WithValue(ctx, accountKey, account)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// Continue without authentication
		log.Printf("Continuing without authentication")
		next.ServeHTTP(w, r)
	})
}

// AuthRequiredMiddleware enforces authentication and adds the account to context
func (h *AuthHandler) AuthRequiredMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get session token from cookie
		cookie, err := r.Cookie("session_token")
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}

		// Get account from session
		account, err := h.store.GetAccountBySession(cookie.Value)
		if err != nil || account == nil {
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}

		// Add account to request context
		ctx := r.Context()
		ctx = context.WithValue(ctx, accountKey, account)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
