package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/densestvoid/budget/data"
	"github.com/densestvoid/budget/templates"
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
	templates.BaseLayoutWithAuth("Register - Budget App", false, templates.RegisterPage()).Render(w)
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
		h.store.DeleteSession(cookie.Value)
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
	json.NewEncoder(w).Encode(account)
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
		ctx = context.WithValue(ctx, "account", account)
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
				ctx = context.WithValue(ctx, "account", account)
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
		ctx = context.WithValue(ctx, "account", account)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// BudgetPlanSelectionMiddleware adds the selected budget plan to context
// Checks query param first, then cookie, then defaults to active plan
func (h *AuthHandler) BudgetPlanSelectionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		account, ok := r.Context().Value("account").(*data.Account)
		if !ok || account == nil {
			next.ServeHTTP(w, r)
			return
		}

		var budgetPlanID int

		// Check query parameter first
		if planIDStr := r.URL.Query().Get("budget_plan_id"); planIDStr != "" {
			if planID, err := strconv.Atoi(planIDStr); err == nil {
				// Verify plan belongs to account
				plan, err := h.store.GetBudgetPlan(account.ID, planID)
				if err == nil && plan != nil {
					budgetPlanID = planID
					// Set cookie to persist selection
					http.SetCookie(w, &http.Cookie{
						Name:     "selected_budget_plan_id",
						Value:    planIDStr,
						Path:     "/",
						HttpOnly: false, // Allow JS to read it
						Secure:   false,
						SameSite: http.SameSiteStrictMode,
						MaxAge:   int(30 * 24 * time.Hour.Seconds()), // 30 days
					})
				}
			}
		}

		// If not in query param, check cookie
		if budgetPlanID == 0 {
			if cookie, err := r.Cookie("selected_budget_plan_id"); err == nil && cookie.Value != "" {
				if planID, err := strconv.Atoi(cookie.Value); err == nil {
					// Verify plan belongs to account
					plan, err := h.store.GetBudgetPlan(account.ID, planID)
					if err == nil && plan != nil {
						budgetPlanID = planID
					}
					// If plan not found or doesn't belong to account, fall through to active plan
				}
			}
		}

		// Always fall back to active plan if no plan selected or if selected plan not found
		if budgetPlanID == 0 {
			activePlan, err := h.store.GetActiveBudgetPlan(account.ID)
			if err == nil && activePlan != nil {
				budgetPlanID = activePlan.ID
				// Update cookie to reflect active plan
				http.SetCookie(w, &http.Cookie{
					Name:     "selected_budget_plan_id",
					Value:    strconv.Itoa(activePlan.ID),
					Path:     "/",
					HttpOnly: false,
					Secure:   false,
					SameSite: http.SameSiteStrictMode,
					MaxAge:   int(30 * 24 * time.Hour.Seconds()),
				})
			}
		}

		// Add budget plan ID to context
		ctx := r.Context()
		ctx = context.WithValue(ctx, "budget_plan_id", budgetPlanID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
