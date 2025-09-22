package server

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"budget/data"
	"budget/templates"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	_ "github.com/lib/pq"
	g "github.com/maragudk/gomponents"
	"github.com/maragudk/gomponents/html"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type Server struct {
	router *chi.Mux
	db     *sql.DB
	store  *data.Storage
	port   string
}

func NewServer(port string) *Server {
	return &Server{
		router: chi.NewRouter(),
		port:   port,
	}
}

func (s *Server) SetupMiddleware() {
	// Basic middleware
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.Timeout(60 * time.Second))

	// CORS middleware
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
}

func (s *Server) SetupRoutes() {
	// Create auth handler
	authHandler := NewAuthHandler(s.store)

	// Static files
	s.router.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("assets"))))

	// Health check endpoint (no auth required)
	s.router.Get("/health", s.healthHandler)

	// API routes
	s.router.Route("/api", func(r chi.Router) {
		r.Get("/health", s.healthHandler)
	})

	// Auth routes (form-based)
	s.router.Get("/auth/register", authHandler.RegisterPage)
	s.router.Post("/auth/register", authHandler.Register)
	s.router.Post("/auth/login", authHandler.Login)
	s.router.Post("/auth/logout", authHandler.Logout)

	// Category management routes (require authentication)
	s.router.With(authHandler.AuthRequiredMiddleware).Route("/categories", func(r chi.Router) {
		r.Get("/", s.categoriesPageHandler)
		r.Post("/", s.createCategoryHandler)
		r.Put("/{id}", s.editCategoryHandler)
		r.Delete("/{id}", s.deleteCategoryHandler)
	})

	// Transactions routes (require authentication)
	s.router.With(authHandler.AuthRequiredMiddleware).Route("/transactions", func(r chi.Router) {
		r.Get("/", s.transactionsPageHandler)
		r.Put("/{id}", s.updateTransactionHandler)
		r.Patch("/{id}", s.patchTransactionHandler)
		r.Patch("/{id}/reviewed", s.markTransactionReviewedHandler)
		r.Get("/{id}/payee/edit", s.editPayeeInlineHandler)
		r.Get("/{id}/category/edit", s.editCategoryInlineHandler)
		r.Get("/{id}/edit", s.editTransactionModalHandler)
		r.Post("/upload", s.uploadTransactionsHandler)
		r.Post("/mark-reviewed", s.markTransactionsReviewedHandler)
	})

	// Rules routes (require authentication)
	s.router.With(authHandler.AuthRequiredMiddleware).Route("/rules", func(r chi.Router) {
		r.Get("/", s.rulesPageHandler)
		r.Post("/", s.createRuleHandler)
		r.Put("/{id}", s.updateRuleHandler)
		r.Delete("/{id}", s.deleteRuleHandler)
		r.Patch("/{id}/toggle", s.toggleRuleHandler)
		r.Get("/{id}/edit", s.editRuleHandler)
	})

	// Web routes with optional authentication
	s.router.With(authHandler.OptionalAuthMiddleware).Get("/", s.homeHandler)
	s.router.With(authHandler.OptionalAuthMiddleware).Get("/about", s.aboutHandler)
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := `{"status":"ok","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`
	if _, err := w.Write([]byte(response)); err != nil {
		log.Printf("Error writing health response: %v", err)
	}
}

// renderPageWithAuth is a helper function to render pages with authentication check
func (s *Server) renderPageWithAuth(w http.ResponseWriter, r *http.Request, title string, pageFunc func() g.Node, handlerName string) {
	var isAuthenticated bool
	if account := r.Context().Value("account"); account != nil {
		isAuthenticated = true
		log.Printf("%s handler: User is authenticated - isAuthenticated=%t", handlerName, isAuthenticated)
	} else {
		log.Printf("%s handler: User is not authenticated - isAuthenticated=%t", handlerName, isAuthenticated)
	}
	page := pageFunc()
	if err := templates.BaseLayoutWithAuth(title, isAuthenticated, page).Render(w); err != nil {
		log.Printf("Error rendering %s page: %v", handlerName, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// renderRulesListResponse is a helper function to render rules list for HTMX responses
func (s *Server) renderRulesListResponse(w http.ResponseWriter, r *http.Request, account *data.Account) {
	if isHTMX(r) {
		// Return updated rules list
		rules, err := s.store.GetRulesByAccount(account.ID)
		if err != nil {
			http.Error(w, "Failed to load rules", http.StatusInternalServerError)
			return
		}
		categories, err := s.store.GetCategoriesByAccount(account.ID)
		if err != nil {
			http.Error(w, "Failed to load categories", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		if err := templates.RenderRulesList(w, rules, categories); err != nil {
			log.Printf("Error rendering rules list: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	} else {
		http.Redirect(w, r, "/rules/", http.StatusSeeOther)
	}
}

// parseCategoryFormData parses form data for category operations
func parseCategoryFormData(r *http.Request) (name string, parentID *int, err error) {
	if err := r.ParseForm(); err != nil {
		return "", nil, err
	}
	name = r.FormValue("name")
	parentIDStr := r.FormValue("parent_id")
	if parentIDStr != "" {
		pid, err := strconv.Atoi(parentIDStr)
		if err == nil {
			parentID = &pid
		}
	}
	return name, parentID, nil
}

// validateCategoryParent validates that a parent category exists and doesn't create cycles
func validateCategoryParent(categories []data.Category, categoryID int, parentID *int) error {
	if parentID == nil {
		return nil // No parent is valid
	}

	// Check if parent exists
	var parentExists bool
	for _, cat := range categories {
		if cat.ID == *parentID {
			parentExists = true
			break
		}
	}
	if !parentExists {
		return fmt.Errorf("parent category not found")
	}

	// Check for circular reference
	if *parentID == categoryID {
		return fmt.Errorf("category cannot be its own parent")
	}

	// Check if parent is a descendant of this category
	descendants := getDescendants(categories, categoryID)
	if descendants[*parentID] {
		return fmt.Errorf("cannot set parent to a descendant category")
	}

	return nil
}

// getDescendants returns a set of all descendant category IDs (including the category itself)
func getDescendants(categories []data.Category, categoryID int) map[int]bool {
	descendants := make(map[int]bool)
	descendants[categoryID] = true

	// Build a map for quick lookups
	catMap := make(map[int]*data.Category)
	for i := range categories {
		catMap[categories[i].ID] = &categories[i]
	}

	// Use a queue to traverse all descendants
	queue := []int{categoryID}
	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]

		// Find all children of the current category
		for _, cat := range categories {
			if cat.ParentID != nil && *cat.ParentID == currentID {
				descendants[cat.ID] = true
				queue = append(queue, cat.ID)
			}
		}
	}

	return descendants
}

func (s *Server) homeHandler(w http.ResponseWriter, r *http.Request) {
	s.renderPageWithAuth(w, r, "Home - Budget App", templates.HomePage, "Home")
}

func (s *Server) aboutHandler(w http.ResponseWriter, r *http.Request) {
	s.renderPageWithAuth(w, r, "About - Budget App", templates.AboutPage, "About")
}

// Helper to build the breadcrumb path for a category by ID
func buildCategoryBreadcrumbByID(categories []data.Category, parentID *int) []data.Category {
	catMap := map[int]data.Category{}
	for _, c := range categories {
		catMap[c.ID] = c
	}
	var path []data.Category
	currentID := parentID
	for currentID != nil {
		cat := catMap[*currentID]
		path = append([]data.Category{cat}, path...)
		currentID = cat.ParentID
	}
	return path
}

func (s *Server) categoriesPageHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}
	categories, err := s.store.GetCategoriesByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}
	// Get parent_id from query param
	var parentID *int
	if pidStr := r.URL.Query().Get("parent_id"); pidStr != "" {
		pid, err := strconv.Atoi(pidStr)
		if err == nil {
			parentID = &pid
		}
	}
	// Find current parent category (if any)
	var currentParent *data.Category
	if parentID != nil {
		for _, c := range categories {
			if c.ID == *parentID {
				currentParent = &c
				break
			}
		}
	}
	// Filter categories to only those whose ParentID == parentID
	var visibleCategories []data.Category
	for _, c := range categories {
		if (c.ParentID == nil && parentID == nil) || (c.ParentID != nil && parentID != nil && *c.ParentID == *parentID) {
			visibleCategories = append(visibleCategories, c)
		}
	}
	// Build breadcrumb path
	breadcrumb := buildCategoryBreadcrumbByID(categories, parentID)

	// Check if this is an HTMX request
	if isHTMX(r) {
		// Return only the directory navigation content for HTMX requests
		w.Header().Set("Content-Type", "text/html")
		content := templates.CategoriesDirectoryNavigation(visibleCategories, categories, currentParent, breadcrumb)
		if err := content.Render(w); err != nil {
			log.Printf("Error rendering categories navigation: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	} else {
		// Return full page for regular requests
		page := templates.CategoriesDirectoryPage(visibleCategories, categories, currentParent, breadcrumb)
		if err := templates.BaseLayoutWithAuth("Categories - Budget App", true, page).Render(w); err != nil {
			log.Printf("Error rendering categories page: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
}

func (s *Server) createCategoryHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}
	name := r.FormValue("name")
	parentIDStr := r.FormValue("parent_id")
	var parentID *int
	if parentIDStr != "" {
		pid, err := strconv.Atoi(parentIDStr)
		if err == nil {
			parentID = &pid
		}
	}

	_, err := s.store.CreateCategory(account.ID, name, parentID)
	if err != nil {
		http.Error(w, "Failed to create category", http.StatusInternalServerError)
		return
	}

	// Check if this is an HTMX request
	if isHTMX(r) {
		// Get the current parent_id from the form or referer to refresh the correct page
		var currentParentID *int
		if parentID != nil {
			currentParentID = parentID
		} else {
			// If no parent specified, check referer for current directory
			referer := r.Header.Get("Referer")
			if referer != "" {
				if u, err := url.Parse(referer); err == nil {
					if pidStr := u.Query().Get("parent_id"); pidStr != "" {
						if pid, err := strconv.Atoi(pidStr); err == nil {
							currentParentID = &pid
						}
					}
				}
			}
		}

		// Get updated categories
		categories, err := s.store.GetCategoriesByAccount(account.ID)
		if err != nil {
			http.Error(w, "Failed to load categories", http.StatusInternalServerError)
			return
		}

		// Find current parent category (if any)
		var currentParent *data.Category
		if currentParentID != nil {
			for _, c := range categories {
				if c.ID == *currentParentID {
					currentParent = &c
					break
				}
			}
		}

		// Filter categories to only those whose ParentID == currentParentID
		var visibleCategories []data.Category
		for _, c := range categories {
			if (c.ParentID == nil && currentParentID == nil) || (c.ParentID != nil && currentParentID != nil && *c.ParentID == *currentParentID) {
				visibleCategories = append(visibleCategories, c)
			}
		}

		// Build breadcrumb path
		breadcrumb := buildCategoryBreadcrumbByID(categories, currentParentID)

		// Return the updated directory navigation content
		w.Header().Set("Content-Type", "text/html")
		content := templates.CategoriesDirectoryNavigation(visibleCategories, categories, currentParent, breadcrumb)
		if err := content.Render(w); err != nil {
			log.Printf("Error rendering categories navigation: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	} else {
		http.Redirect(w, r, "/categories/", http.StatusSeeOther)
	}
}

func (s *Server) deleteCategoryHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}
	idStr := chi.URLParam(r, "id")
	categoryID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}
	err = s.store.DeleteCategory(account.ID, categoryID)
	if err != nil {
		http.Error(w, "Failed to delete category", http.StatusInternalServerError)
		return
	}

	// Check if this is an HTMX request
	if isHTMX(r) {
		// For delete, just return empty content to remove the card
		w.Header().Set("Content-Type", "text/html")
		if _, err := w.Write([]byte("")); err != nil {
			log.Printf("Error writing response: %v", err)
		}
	} else {
		http.Redirect(w, r, "/categories/", http.StatusSeeOther)
	}
}

// Handler stubs for transactions
func (s *Server) transactionsPageHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}
	txs, err := s.store.GetTransactionsByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load transactions", http.StatusInternalServerError)
		return
	}
	cats, err := s.store.GetCategoriesByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}
	errMsg := r.URL.Query().Get("error")
	page := templates.TransactionsPage(txs, cats, errMsg)
	if err := templates.BaseLayoutWithAuth("Transactions - Budget App", true, page).Render(w); err != nil {
		log.Printf("Error rendering transactions page: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (s *Server) updateTransactionHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}
	idStr := chi.URLParam(r, "id")
	transactionID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid transaction ID", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}
	payee := r.FormValue("payee")
	categoryIDStr := r.FormValue("category_id")
	var categoryID *int
	if categoryIDStr != "" {
		cid, err := strconv.Atoi(categoryIDStr)
		if err == nil {
			categoryID = &cid
		}
	}
	err = s.store.UpdateTransactionPayeeCategory(account.ID, transactionID, payee, categoryID)
	if err != nil {
		http.Error(w, "Failed to update transaction", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func (s *Server) uploadTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	// Parse CSV file
	records, err := parseCSVFile(r)
	if err != nil {
		s.handleCSVError(w, r, err.Error())
		return
	}

	// Validate headers
	colIdx, err := validateCSVHeaders(records[0])
	if err != nil {
		s.handleCSVError(w, r, err.Error())
		return
	}

	// Process transactions
	newCount, duplicateCount, err := s.processUploadedTransactions(account.ID, records, colIdx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect with success message
	http.Redirect(w, r, fmt.Sprintf("/transactions/?uploaded=%d&duplicates=%d", newCount, duplicateCount), http.StatusSeeOther)
}

// handleCSVError handles CSV parsing errors with appropriate response format
func (s *Server) handleCSVError(w http.ResponseWriter, r *http.Request, errMsg string) {
	if isHTMX(r) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`<div class="alert alert-danger" role="alert">` + errMsg + `</div>`)); err != nil {
			log.Printf("Error writing error message: %v", err)
		}
	} else {
		http.Error(w, errMsg, http.StatusBadRequest)
	}
}

// parseRuleFormData parses form data for rule creation
func parseRuleFormData(r *http.Request) (name, newPayee, categoryIDStr, priorityStr string, err error) {
	if err := r.ParseForm(); err != nil {
		return "", "", "", "", err
	}
	return r.FormValue("name"), r.FormValue("new_payee"), r.FormValue("category_id"), r.FormValue("priority"), nil
}

// validateRuleCategory validates that a category belongs to the account
func (s *Server) validateRuleCategory(accountID int, categoryIDStr string) (*int, error) {
	if categoryIDStr == "" {
		return nil, nil
	}

	categoryID, err := strconv.Atoi(categoryIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid category ID")
	}

	categories, err := s.store.GetCategoriesByAccount(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to load categories")
	}

	for _, cat := range categories {
		if cat.ID == categoryID {
			return &categoryID, nil
		}
	}

	return nil, fmt.Errorf("category not found")
}

// parseRulePriority parses and validates rule priority
func parseRulePriority(priorityStr string) (int, error) {
	if priorityStr == "" {
		return 0, nil
	}

	priority, err := strconv.Atoi(priorityStr)
	if err != nil {
		return 0, fmt.Errorf("invalid priority")
	}

	if priority < 0 || priority > 100 {
		return 0, fmt.Errorf("priority must be between 0 and 100")
	}

	return priority, nil
}

// createRuleFromFormData creates a rule from form data
func (s *Server) createRuleFromFormData(accountID int, name, newPayee, categoryIDStr, priorityStr string) (*data.Rule, error) {
	var newPayeePtr *string
	if newPayee != "" {
		newPayeePtr = &newPayee
	}

	categoryID, err := s.validateRuleCategory(accountID, categoryIDStr)
	if err != nil {
		return nil, err
	}

	priority, err := parseRulePriority(priorityStr)
	if err != nil {
		return nil, err
	}

	return &data.Rule{
		AccountID:  accountID,
		Name:       name,
		NewPayee:   newPayeePtr,
		CategoryID: categoryID,
		Priority:   priority,
	}, nil
}

// parseRuleConditions parses rule conditions from form data
func parseRuleConditions(r *http.Request) []data.RuleCondition {
	var conditions []data.RuleCondition
	conditionCount := 0

	for {
		operator := r.FormValue(fmt.Sprintf("conditions[%d][operator]", conditionCount))
		value := r.FormValue(fmt.Sprintf("conditions[%d][value]", conditionCount))
		if operator == "" || value == "" {
			break
		}
		conditions = append(conditions, data.RuleCondition{
			Field:    "payee", // Conditions are always applied to payee
			Operator: operator,
			Value:    value,
		})
		conditionCount++
	}

	return conditions
}

// handleRuleCreationResponse handles the response after creating a rule
func (s *Server) handleRuleCreationResponse(w http.ResponseWriter, r *http.Request, accountID int, createdRule *data.Rule, runImmediately bool) {
	updatedCount := 0
	if runImmediately && createdRule != nil {
		count, err := s.store.ApplyRuleToAllTransactions(accountID, *createdRule)
		if err == nil {
			updatedCount = count
		}
	}

	if !isHTMX(r) {
		http.Redirect(w, r, "/rules/", http.StatusSeeOther)
		return
	}

	// Return updated rules list with banner if needed
	rules, err := s.store.GetRulesByAccount(accountID)
	if err != nil {
		http.Error(w, "Failed to load rules", http.StatusInternalServerError)
		return
	}
	categories, err := s.store.GetCategoriesByAccount(accountID)
	if err != nil {
		http.Error(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if updatedCount > 0 {
		banner := fmt.Sprintf(`<div id="rule-banner" class="alert alert-success position-fixed top-0 start-50 translate-middle-x mt-3" style="z-index:2000; min-width:300px; text-align:center;">Updated %d transactions</div><script>setTimeout(function(){ var b=document.getElementById('rule-banner'); if(b){b.remove();}}, 3500);</script>`, updatedCount)
		if _, err := w.Write([]byte(banner)); err != nil {
			log.Printf("Error writing rule banner: %v", err)
		}
	}
	if err := templates.RenderRulesList(w, rules, categories); err != nil {
		log.Printf("Error rendering rules list: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// parseTransactionPatchData parses form data for transaction patching
func parseTransactionPatchData(r *http.Request) (payee string, categoryID *int, reviewed bool, hasCategoryID bool, err error) {
	if err := r.ParseForm(); err != nil {
		return "", nil, false, false, err
	}

	payee = r.FormValue("payee")
	categoryIDStr := r.FormValue("category_id")
	hasCategoryID = r.Form.Has("category_id")

	if categoryIDStr != "" {
		cid, err := strconv.Atoi(categoryIDStr)
		if err == nil {
			categoryID = &cid
		}
	} else if hasCategoryID {
		categoryID = nil // explicitly set to null if empty string is sent
	}

	reviewed = r.FormValue("reviewed") == "on"

	return payee, categoryID, reviewed, hasCategoryID, nil
}

// updateTransactionFields updates transaction fields in the database
func (s *Server) updateTransactionFields(accountID, transactionID int, payee string, categoryID *int, reviewed bool, hasCategoryID bool) error {
	if payee != "" {
		if err := s.store.UpdateTransactionPayee(accountID, transactionID, payee); err != nil {
			return fmt.Errorf("failed to update payee: %w", err)
		}
	}

	if categoryID != nil || hasCategoryID {
		if err := s.store.UpdateTransactionCategory(accountID, transactionID, categoryID); err != nil {
			return fmt.Errorf("failed to update category: %w", err)
		}
	}

	if reviewed {
		if err := s.store.MarkTransactionReviewed(accountID, transactionID); err != nil {
			return fmt.Errorf("failed to mark as reviewed: %w", err)
		}
	}

	return nil
}

// handleTransactionPatchResponse handles the response after patching a transaction
func (s *Server) handleTransactionPatchResponse(w http.ResponseWriter, r *http.Request, accountID, transactionID int, updateErr error) {
	if !isHTMX(r) {
		http.Redirect(w, r, "/transactions/", http.StatusSeeOther)
		return
	}

	// Fetch updated transaction list and categories for rendering
	txs, err := s.store.GetTransactionsByAccount(accountID)
	if err != nil {
		http.Error(w, "Failed to load transactions", http.StatusInternalServerError)
		return
	}
	cats, err := s.store.GetCategoriesByAccount(accountID)
	if err != nil {
		http.Error(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if updateErr != nil {
		// Find the specific transaction for error display
		var updatedTx *data.Transaction
		for _, t := range txs {
			if t.ID == transactionID {
				updatedTx = &t
				break
			}
		}
		if updatedTx == nil {
			// If transaction not found in list, it might have been reviewed and hidden
			// In this case, just return the updated list
			if err := templates.RenderTransactionListWithSelectableCards(w, txs, cats); err != nil {
				log.Printf("Error rendering transaction list: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			return
		}
		// Render the modal with the error message
		w.WriteHeader(http.StatusBadRequest)
		if err := templates.TransactionCardWithModalSelectable(*updatedTx, cats, "Failed to update transaction: "+updateErr.Error()).Render(w); err != nil {
			log.Printf("Error rendering transaction card: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		return
	}

	// Check if the transaction was marked as reviewed and should be hidden
	var transactionStillVisible bool
	for _, t := range txs {
		if t.ID == transactionID {
			transactionStillVisible = true
			break
		}
	}

	if transactionStillVisible {
		// Transaction is still visible, return just the updated card
		var updatedTx *data.Transaction
		for _, t := range txs {
			if t.ID == transactionID {
				updatedTx = &t
				break
			}
		}
		if err := templates.TransactionCardWithModalSelectable(*updatedTx, cats, "").Render(w); err != nil {
			log.Printf("Error rendering transaction card: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	} else {
		// Transaction was marked as reviewed and is now hidden, return empty content to remove the card
		if _, err := w.Write([]byte("")); err != nil {
			log.Printf("Error writing response: %v", err)
		}
	}
}

// parseAmountToCents parses a decimal string to int cents
func parseAmountToCents(s string) (int, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "") // Remove thousands separator if present
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
	}
	s = strings.TrimPrefix(s, "-")
	s = strings.TrimPrefix(s, "+")
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "$")
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	cents := int(f * 100)
	if neg {
		cents = -cents
	}
	return cents, nil
}

func (s *Server) SetDB(db *sql.DB) {
	s.db = db
	s.store = data.NewStorage(db)
}

func (s *Server) Run() error {
	log.Printf("Server starting on port %s", s.port)

	// Create HTTP/2 server
	h2s := &http2.Server{}
	server := &http.Server{
		Addr:              ":" + s.port,
		Handler:           h2c.NewHandler(s.router, h2s),
		ReadHeaderTimeout: 20 * time.Second, // Prevent Slowloris attacks
	}

	return server.ListenAndServe()
}

// PATCH handler for transactions
func (s *Server) patchTransactionHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	transactionID, err := s.parseTransactionID(r)
	if err != nil {
		http.Error(w, "Invalid transaction ID", http.StatusBadRequest)
		return
	}

	// Parse form data
	payee, categoryID, reviewed, hasCategoryID, err := parseTransactionPatchData(r)
	if err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Update transaction fields
	updateErr := s.updateTransactionFields(account.ID, transactionID, payee, categoryID, reviewed, hasCategoryID)

	// Handle response
	s.handleTransactionPatchResponse(w, r, account.ID, transactionID, updateErr)
}

// parseTransactionID extracts and parses the transaction ID from the request
func (s *Server) parseTransactionID(r *http.Request) (int, error) {
	idStr := chi.URLParam(r, "id")
	return strconv.Atoi(idStr)
}

// GET /transactions/{id}/payee/edit returns an inline text input for payee
func (s *Server) editPayeeInlineHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	transactionID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid transaction ID", http.StatusBadRequest)
		return
	}
	txns, err := s.store.GetTransactionsByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load transaction", http.StatusInternalServerError)
		return
	}
	var payee string
	for _, t := range txns {
		if t.ID == transactionID {
			payee = t.Payee
			break
		}
	}
	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write([]byte(`<form hx-patch="/transactions/` + idStr + `" hx-trigger="blur from:input, submit" hx-target="this" hx-swap="outerHTML" style="display:inline;">
		<input type="text" name="payee" value="` + htmlEscape(payee) + `" class="form-control form-control-sm w-auto d-inline" autofocus onblur="this.form.requestSubmit()">
</form>`)); err != nil {
		log.Printf("Error writing form response: %v", err)
	}
}

// GET /transactions/{id}/category/edit returns an inline select for category
func (s *Server) editCategoryInlineHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	transactionID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid transaction ID", http.StatusBadRequest)
		return
	}
	txns, err := s.store.GetTransactionsByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load transaction", http.StatusInternalServerError)
		return
	}
	var currentCatID *int
	for _, t := range txns {
		if t.ID == transactionID {
			currentCatID = t.CategoryID
			break
		}
	}
	cats, err := s.store.GetCategoriesByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}
	var sb strings.Builder
	sb.WriteString(`<select name="category_id" class="form-select form-select-sm w-auto" 
		hx-patch="/transactions/` + idStr + `" 
		hx-trigger="blur,change" 
		hx-target="this" 
		hx-swap="outerHTML" autofocus>`)
	sb.WriteString(`<option value="">Uncategorized</option>`)
	for _, c := range cats {
		sel := ""
		if currentCatID != nil && *currentCatID == c.ID {
			sel = " selected"
		}
		sb.WriteString(`<option value="` + strconv.Itoa(c.ID) + `"` + sel + `>` + htmlEscape(c.Name) + `</option>`)
	}
	sb.WriteString(`</select>`)
	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write([]byte(sb.String())); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

// Helper to escape HTML
func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, `'`, "&#39;")
	return s
}

// GET /transactions/{id}/edit returns the edit form for the modal
func (s *Server) editTransactionModalHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	transactionID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid transaction ID", http.StatusBadRequest)
		return
	}

	// Get the specific transaction
	txs, err := s.store.GetTransactionsByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load transaction", http.StatusInternalServerError)
		return
	}
	var transaction *data.Transaction
	for _, t := range txs {
		if t.ID == transactionID {
			transaction = &t
			break
		}
	}
	if transaction == nil {
		http.Error(w, "Transaction not found", http.StatusNotFound)
		return
	}

	// Get categories for the form
	cats, err := s.store.GetCategoriesByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")

	// Render the edit form
	form := g.El("form",
		g.Attr("hx-patch", "/transactions/"+strconv.Itoa(transaction.ID)),
		g.Attr("hx-target", "closest .mb-2"),
		g.Attr("hx-swap", "outerHTML"),
		g.Attr("hx-on::after-request", `if (event.target === this) { bootstrap.Modal.getInstance(this.closest('.modal')).hide(); }`),
		html.Div(
			html.Class("mb-3"),
			html.Label(html.Class("form-label"), html.For("edit-payee-"+strconv.Itoa(transaction.ID)), g.Text("Payee")),
			html.Input(
				html.Type("text"),
				html.Name("payee"),
				html.ID("edit-payee-"+strconv.Itoa(transaction.ID)),
				html.Value(transaction.Payee),
				html.Class("form-control"),
				html.Required(),
			),
		),
		html.Div(
			html.Class("mb-3"),
			html.Label(html.Class("form-label"), html.For("edit-category-"+strconv.Itoa(transaction.ID)), g.Text("Category")),
			templates.CategoryBootstrapDropdown(cats, "category_id", "edit-category-"+strconv.Itoa(transaction.ID), transaction.CategoryID, nil, false, nil),
		),
		html.Div(
			html.Class("form-check mb-3"),
			html.Input(
				html.Class("form-check-input"),
				html.Type("checkbox"),
				html.ID("edit-reviewed-"+strconv.Itoa(transaction.ID)),
				html.Name("reviewed"),
				func() g.Node {
					if transaction.Reviewed {
						return g.Attr("checked", "checked")
					}
					return nil
				}(),
			),
			html.Label(
				html.Class("form-check-label"),
				html.For("edit-reviewed-"+strconv.Itoa(transaction.ID)),
				g.Text("Reviewed"),
			),
		),
		html.Div(
			html.Class("d-flex justify-content-end gap-2"),
			html.Button(html.Type("button"), html.Class("btn btn-secondary"), html.DataAttr("bs-dismiss", "modal"), g.Text("Cancel")),
			html.Button(html.Type("submit"), html.Class("btn btn-primary"), g.Text("Save Changes")),
		),
	)
	if err := form.Render(w); err != nil {
		log.Printf("Error rendering transaction edit form: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// Handler to mark a transaction as reviewed
func (s *Server) markTransactionReviewedHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	transactionID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid transaction ID", http.StatusBadRequest)
		return
	}
	err = s.store.MarkTransactionReviewed(account.ID, transactionID)
	if err != nil {
		http.Error(w, "Failed to mark as reviewed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) markTransactionsReviewedHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}
	ids := r.Form["transaction_ids"]
	for _, idStr := range ids {
		id, err := strconv.Atoi(idStr)
		if err == nil {
			_ = s.store.UpdateTransactionReviewed(account.ID, id, true)
		}
	}
	// Return the updated transaction list
	txs, err := s.store.GetTransactionsByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load transactions", http.StatusInternalServerError)
		return
	}
	cats, err := s.store.GetCategoriesByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	// Render the transaction list with selectable cards
	if err := templates.RenderTransactionListWithSelectableCards(w, txs, cats); err != nil {
		log.Printf("Error rendering transaction list: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (s *Server) editCategoryHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	categoryID, err := s.parseCategoryID(r)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	name, parentID, err := parseCategoryFormData(r)
	if err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Get the category before updating to check if parent changed
	categories, err := s.store.GetCategoriesByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}

	oldParentID := s.getCategoryParentID(categories, categoryID)

	// Validate parent relationship
	if err := validateCategoryParent(categories, categoryID, parentID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = s.store.UpdateCategory(account.ID, categoryID, name, parentID)
	if err != nil {
		http.Error(w, "Failed to update category", http.StatusInternalServerError)
		return
	}

	s.handleCategoryUpdateResponse(w, r, account.ID, categoryID, oldParentID, parentID)
}

// parseCategoryID extracts and parses the category ID from the request
func (s *Server) parseCategoryID(r *http.Request) (int, error) {
	idStr := chi.URLParam(r, "id")
	return strconv.Atoi(idStr)
}

// getCategoryParentID finds the parent ID of a category
func (s *Server) getCategoryParentID(categories []data.Category, categoryID int) *int {
	for _, c := range categories {
		if c.ID == categoryID {
			return c.ParentID
		}
	}
	return nil
}

// handleCategoryUpdateResponse handles the response after updating a category
func (s *Server) handleCategoryUpdateResponse(w http.ResponseWriter, r *http.Request, accountID, categoryID int, oldParentID, parentID *int) {
	if !isHTMX(r) {
		http.Redirect(w, r, "/categories/", http.StatusSeeOther)
		return
	}

	parentChanged := s.hasParentChanged(oldParentID, parentID)
	if parentChanged {
		// Parent changed, remove the card
		w.Header().Set("Content-Type", "text/html")
		if _, err := w.Write([]byte("")); err != nil {
			log.Printf("Error writing response: %v", err)
		}
		return
	}

	// Parent didn't change, update the card
	s.renderUpdatedCategoryCard(w, accountID, categoryID)
}

// hasParentChanged checks if the parent category has changed
func (s *Server) hasParentChanged(oldParentID, parentID *int) bool {
	return (oldParentID == nil && parentID != nil) ||
		(oldParentID != nil && parentID == nil) ||
		(oldParentID != nil && parentID != nil && *oldParentID != *parentID)
}

// renderUpdatedCategoryCard renders the updated category card
func (s *Server) renderUpdatedCategoryCard(w http.ResponseWriter, accountID, categoryID int) {
	categories, err := s.store.GetCategoriesByAccount(accountID)
	if err != nil {
		http.Error(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}

	updatedCategory := s.findCategoryByID(categories, categoryID)
	if updatedCategory == nil {
		http.Error(w, "Category not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	card := templates.CategoryCardOnly(*updatedCategory, categories)
	if err := card.Render(w); err != nil {
		log.Printf("Error rendering category card: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// findCategoryByID finds a category by its ID
func (s *Server) findCategoryByID(categories []data.Category, categoryID int) *data.Category {
	for _, c := range categories {
		if c.ID == categoryID {
			return &c
		}
	}
	return nil
}

// parseCSVFile parses the uploaded CSV file and returns records
func parseCSVFile(r *http.Request) ([][]string, error) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		return nil, fmt.Errorf("failed to parse form: %w", err)
	}

	file, _, err := r.FormFile("csv")
	if err != nil {
		return nil, fmt.Errorf("failed to get uploaded file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	return records, nil
}

// validateCSVHeaders validates CSV headers and returns column indices
func validateCSVHeaders(header []string) (map[string]int, error) {
	required := map[string]bool{"date": false, "payee": false, "amount": false}
	colIdx := map[string]int{}

	for i, col := range header {
		col = strings.ToLower(strings.TrimSpace(col))
		switch col {
		case "date":
			required["date"] = true
			colIdx["date"] = i
		case "payee", "description", "memo":
			required["payee"] = true
			colIdx["payee"] = i
		case "amount", "debit", "credit":
			required["amount"] = true
			colIdx["amount"] = i
		default:
			// Ignore unrecognized columns
		}
	}

	for field, found := range required {
		if !found {
			return nil, fmt.Errorf("missing required column: %s", field)
		}
	}

	return colIdx, nil
}

// parseTransactionFromRecord parses a transaction from a CSV record
func parseTransactionFromRecord(record []string, colIdx map[string]int) (*data.Transaction, error) {
	dateStr := strings.TrimSpace(record[colIdx["date"]])
	payee := strings.TrimSpace(record[colIdx["payee"]])
	amountStr := strings.TrimSpace(record[colIdx["amount"]])

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		// Try other common date formats
		formats := []string{"01/02/2006", "1/2/2006", "01-02-2006", "1-2-2006"}
		for _, format := range formats {
			if date, err = time.Parse(format, dateStr); err == nil {
				break
			}
		}
		if err != nil {
			return nil, fmt.Errorf("invalid date format: %s", dateStr)
		}
	}

	amount, err := parseAmountToCents(amountStr)
	if err != nil {
		return nil, fmt.Errorf("invalid amount format: %s", amountStr)
	}

	return &data.Transaction{
		Date:          date,
		OriginalPayee: payee,
		Payee:         payee,
		Amount:        amount,
		Reviewed:      false,
	}, nil
}

// processUploadedTransactions processes the uploaded transactions and returns results
func (s *Server) processUploadedTransactions(accountID int, records [][]string, colIdx map[string]int) (int, int, error) {
	var newCount, duplicateCount int
	var txs []data.Transaction

	// Parse all transactions first
	for i := 1; i < len(records); i++ { // Skip header
		record := records[i]
		if len(record) <= colIdx["amount"] {
			continue // Skip incomplete rows
		}

		tx, err := parseTransactionFromRecord(record, colIdx)
		if err != nil {
			continue // Skip invalid rows
		}

		tx.AccountID = accountID
		txs = append(txs, *tx)
	}

	// Get existing transactions for duplicate checking
	existingTxs, err := s.store.GetAllTransactionsByAccount(accountID)
	if err != nil {
		return newCount, duplicateCount, fmt.Errorf("failed to check for duplicates: %w", err)
	}

	// Create a map of existing transactions for quick lookup
	existing := make(map[string]struct{})
	for _, t := range existingTxs {
		key := fmt.Sprintf("%s|%s|%d", t.Date.Format("2006-01-02"), strings.ToLower(strings.TrimSpace(t.OriginalPayee)), t.Amount)
		existing[key] = struct{}{}
	}

	// Filter out duplicates and prepare for bulk insert
	var deduped []data.Transaction
	for _, tx := range txs {
		key := fmt.Sprintf("%s|%s|%d", tx.Date.Format("2006-01-02"), strings.ToLower(strings.TrimSpace(tx.OriginalPayee)), tx.Amount)
		if _, found := existing[key]; found {
			duplicateCount++
		} else {
			deduped = append(deduped, tx)
			existing[key] = struct{}{} // Prevent duplicates within the same upload
		}
	}

	// Bulk insert new transactions
	if len(deduped) > 0 {
		if err := s.store.BulkInsertTransactions(accountID, deduped); err != nil {
			return newCount, duplicateCount, fmt.Errorf("failed to import transactions: %w", err)
		}
		newCount = len(deduped)
	}

	return newCount, duplicateCount, nil
}

// Rule management handlers

func (s *Server) rulesPageHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}
	rules, err := s.store.GetRulesByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load rules", http.StatusInternalServerError)
		return
	}
	categories, err := s.store.GetCategoriesByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}
	page := templates.RulesPage(rules, categories)
	if err := templates.BaseLayoutWithAuth("Rules - Budget App", true, page).Render(w); err != nil {
		log.Printf("Error rendering rules page: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (s *Server) createRuleHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse form data
	name, newPayee, categoryIDStr, priorityStr, err := parseRuleFormData(r)
	if err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Create rule from form data
	rule, err := s.createRuleFromFormData(account.ID, name, newPayee, categoryIDStr, priorityStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Parse conditions
	conditions := parseRuleConditions(r)

	// Create the rule in the database
	createdRule, err := s.store.CreateRule(account.ID, rule.Name, rule.NewPayee, rule.CategoryID, rule.Priority, conditions)
	if err != nil {
		http.Error(w, "Failed to create rule", http.StatusInternalServerError)
		return
	}

	// Check if run_immediately was requested
	runImmediately := r.FormValue("run_immediately") == "on" || r.FormValue("run_immediately") == "true"

	// Handle response
	s.handleRuleCreationResponse(w, r, account.ID, createdRule, runImmediately)
}

func (s *Server) updateRuleHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	ruleID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid rule ID", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	newPayee := r.FormValue("new_payee")
	categoryIDStr := r.FormValue("category_id")
	priorityStr := r.FormValue("priority")

	var newPayeePtr *string
	if newPayee != "" {
		newPayeePtr = &newPayee
	}

	var categoryID *int
	if categoryIDStr != "" {
		cid, err := strconv.Atoi(categoryIDStr)
		if err == nil {
			// Validate that the category belongs to the current account
			categories, err := s.store.GetCategoriesByAccount(account.ID)
			if err != nil {
				http.Error(w, "Failed to load categories", http.StatusInternalServerError)
				return
			}

			// Check if the category ID exists for this account
			categoryExists := false
			for _, cat := range categories {
				if cat.ID == cid {
					categoryExists = true
					break
				}
			}

			if categoryExists {
				categoryID = &cid
			} else {
				// Category doesn't belong to this account, ignore it
				categoryID = nil
			}
		}
	}

	priority := 0
	if priorityStr != "" {
		if p, err := strconv.Atoi(priorityStr); err == nil {
			priority = p
		}
	}

	// Parse conditions from form
	var conditions []data.RuleCondition
	conditionCount := 0
	for {
		operator := r.FormValue(fmt.Sprintf("conditions[%d][operator]", conditionCount))
		value := r.FormValue(fmt.Sprintf("conditions[%d][value]", conditionCount))
		if operator == "" || value == "" {
			break
		}
		conditions = append(conditions, data.RuleCondition{
			Field:    "payee", // Conditions are always applied to payee
			Operator: operator,
			Value:    value,
		})
		conditionCount++
	}

	err = s.store.UpdateRule(account.ID, ruleID, name, newPayeePtr, categoryID, priority, conditions)
	if err != nil {
		http.Error(w, "Failed to update rule", http.StatusInternalServerError)
		return
	}

	s.renderRulesListResponse(w, r, account)
}

func (s *Server) deleteRuleHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	ruleID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid rule ID", http.StatusBadRequest)
		return
	}

	err = s.store.DeleteRule(account.ID, ruleID)
	if err != nil {
		http.Error(w, "Failed to delete rule", http.StatusInternalServerError)
		return
	}

	s.renderRulesListResponse(w, r, account)
}

func (s *Server) toggleRuleHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	ruleID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid rule ID", http.StatusBadRequest)
		return
	}

	err = s.store.ToggleRuleActive(account.ID, ruleID)
	if err != nil {
		http.Error(w, "Failed to toggle rule", http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		// Get the updated rule
		rules, err := s.store.GetRulesByAccount(account.ID)
		if err != nil {
			http.Error(w, "Failed to load rules", http.StatusInternalServerError)
			return
		}
		categories, err := s.store.GetCategoriesByAccount(account.ID)
		if err != nil {
			http.Error(w, "Failed to load categories", http.StatusInternalServerError)
			return
		}

		// Find the specific rule that was toggled
		var updatedRule *data.Rule
		for _, rule := range rules {
			if rule.ID == ruleID {
				updatedRule = &rule
				break
			}
		}

		if updatedRule == nil {
			http.Error(w, "Rule not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		// Return only the updated rule card
		if err := templates.RuleCard(*updatedRule, categories).Render(w); err != nil {
			log.Printf("Error rendering rule card: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	} else {
		http.Redirect(w, r, "/rules/", http.StatusSeeOther)
	}
}

func (s *Server) editRuleHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	ruleID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid rule ID", http.StatusBadRequest)
		return
	}

	// Get the rule
	rules, err := s.store.GetRulesByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load rules", http.StatusInternalServerError)
		return
	}

	var rule *data.Rule
	for _, r := range rules {
		if r.ID == ruleID {
			rule = &r
			break
		}
	}

	if rule == nil {
		http.Error(w, "Rule not found", http.StatusNotFound)
		return
	}

	// Get categories for the form
	categories, err := s.store.GetCategoriesByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")

	// Render the edit form
	form := g.El("form",
		g.Attr("hx-put", "/rules/"+strconv.Itoa(rule.ID)),
		g.Attr("hx-target", "#rules-list"),
		g.Attr("hx-swap", "outerHTML"),
		g.Attr("hx-on::after-request", `if (event.target === this) { bootstrap.Modal.getInstance(this.closest('.modal')).hide(); }`),
		templates.RuleFormFields(categories, rule, "edit"),
		html.Div(
			html.Class("d-flex justify-content-end gap-2"),
			html.Button(html.Type("button"), html.Class("btn btn-secondary"), html.DataAttr("bs-dismiss", "modal"), g.Text("Cancel")),
			html.Button(html.Type("submit"), html.Class("btn btn-primary"), g.Text("Save Changes")),
		),
		html.Script(g.Raw(`
			// Ensure the function is available globally for edit modal
			window.addConditionRow = function(modalType) {
				const container = document.getElementById('conditions-container-' + modalType);
				if (!container) {
					console.error('Conditions container not found for modal type:', modalType);
					return;
				}
				const conditionCount = container.querySelectorAll('.card').length;
				const template = document.getElementById('condition-card-template-' + modalType);
				if (!template) {
					console.error('Condition template not found for modal type:', modalType);
					return;
				}
				const clone = template.content.cloneNode(true);
				clone.querySelectorAll('[name]').forEach(el => {
					el.name = el.name.replace('__INDEX__', conditionCount);
				});
				clone.querySelectorAll('[for]').forEach(el => {
					el.setAttribute('for', el.getAttribute('for').replace('__INDEX__', conditionCount));
				});
				container.insertBefore(clone, container.lastElementChild);
			};
		`)),
	)
	if err := form.Render(w); err != nil {
		log.Printf("Error rendering rule edit form: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
