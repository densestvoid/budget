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

func (s *Server) homeHandler(w http.ResponseWriter, r *http.Request) {
	var isAuthenticated bool
	if account := r.Context().Value("account"); account != nil {
		isAuthenticated = true
		log.Printf("Home handler: User is authenticated - isAuthenticated=%t", isAuthenticated)
	} else {
		log.Printf("Home handler: User is not authenticated - isAuthenticated=%t", isAuthenticated)
	}
	page := templates.HomePage()
	if err := templates.BaseLayoutWithAuth("Home - Budget App", isAuthenticated, page).Render(w); err != nil {
		log.Printf("Error rendering home page: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (s *Server) aboutHandler(w http.ResponseWriter, r *http.Request) {
	var isAuthenticated bool
	if account := r.Context().Value("account"); account != nil {
		isAuthenticated = true
		log.Printf("About handler: User is authenticated - isAuthenticated=%t", isAuthenticated)
	} else {
		log.Printf("About handler: User is not authenticated - isAuthenticated=%t", isAuthenticated)
	}
	page := templates.AboutPage()
	if err := templates.BaseLayoutWithAuth("About - Budget App", isAuthenticated, page).Render(w); err != nil {
		log.Printf("Error rendering about page: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
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
		_ = content.Render(w)
	} else {
		// Return full page for regular requests
		page := templates.CategoriesDirectoryPage(visibleCategories, categories, currentParent, breadcrumb)
		_ = templates.BaseLayoutWithAuth("Categories - Budget App", true, page).Render(w)
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
		_ = content.Render(w)
	} else {
		http.Redirect(w, r, "/categories/", http.StatusSeeOther)
	}
}

func (s *Server) updateCategoryHandler(w http.ResponseWriter, r *http.Request) {
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
	err = s.store.UpdateCategory(account.ID, categoryID, name, parentID)
	if err != nil {
		http.Error(w, "Failed to update category", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/categories/", http.StatusSeeOther)
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
		w.Write([]byte(""))
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
	_ = templates.BaseLayoutWithAuth("Transactions - Budget App", true, page).Render(w)
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
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("csv")
	if err != nil {
		http.Error(w, "Failed to get uploaded file", http.StatusBadRequest)
		return
	}
	defer file.Close()
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		http.Error(w, "Failed to read CSV", http.StatusBadRequest)
		return
	}
	if len(records) == 0 {
		http.Error(w, "CSV file is empty", http.StatusBadRequest)
		return
	}
	header := records[0]
	required := map[string]bool{"date": false, "payee": false, "amount": false}
	unrecognized := []string{}
	colIdx := map[string]int{}
	for i, col := range header {
		colNorm := strings.ToLower(strings.TrimSpace(col))
		switch colNorm {
		case "date", "payee", "amount":
			required[colNorm] = true
			colIdx[colNorm] = i
		default:
			unrecognized = append(unrecognized, col)
		}
	}
	missing := []string{}
	for k, v := range required {
		if !v {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 || len(unrecognized) > 0 {
		errMsg := "CSV error: "
		if len(missing) > 0 {
			errMsg += "Missing columns: " + strings.Join(missing, ", ")
		}
		if len(unrecognized) > 0 {
			if len(missing) > 0 {
				errMsg += ". "
			}
			errMsg += "Unrecognized columns: " + strings.Join(unrecognized, ", ")
		}
		if isHTMX(r) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`<div class="alert alert-danger" role="alert">` + errMsg + `</div>`))
			return
		} else {
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
	}
	var txs []data.Transaction
	for i, rec := range records[1:] {
		if len(rec) < len(header) {
			http.Error(w, "Row "+strconv.Itoa(i+2)+" is missing columns", http.StatusBadRequest)
			return
		}
		dateStr := rec[colIdx["date"]]
		payee := rec[colIdx["payee"]]
		amountStr := rec[colIdx["amount"]]
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			// Try mm/dd/yyyy
			date, err = time.Parse("01/02/2006", dateStr)
			if err != nil {
				http.Error(w, "Invalid date in row "+strconv.Itoa(i+2)+": "+dateStr, http.StatusBadRequest)
				return
			}
		}
		amountCents, err := parseAmountToCents(amountStr)
		if err != nil {
			http.Error(w, "Invalid amount in row "+strconv.Itoa(i+2)+": "+amountStr, http.StatusBadRequest)
			return
		}
		txs = append(txs, data.Transaction{
			AccountID:     account.ID,
			Date:          date,
			OriginalPayee: payee,
			Payee:         payee,
			Amount:        amountCents,
		})
	}
	// Deduplicate: fetch all existing transactions for this account (including reviewed ones)
	existingTxs, err := s.store.GetAllTransactionsByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to check for duplicates", http.StatusInternalServerError)
		return
	}
	txKey := func(date time.Time, originalPayee string, amount int) string {
		return date.Format("2006-01-02") + "|" + strings.ToLower(strings.TrimSpace(originalPayee)) + "|" + strconv.Itoa(amount)
	}
	existing := make(map[string]struct{})
	for _, t := range existingTxs {
		existing[txKey(t.Date, t.OriginalPayee, t.Amount)] = struct{}{}
	}
	var deduped []data.Transaction
	for _, t := range txs {
		key := txKey(t.Date, t.OriginalPayee, t.Amount)
		if _, found := existing[key]; !found {
			deduped = append(deduped, t)
			existing[key] = struct{}{} // prevent duplicates within the same upload
		}
	}
	if err := s.store.BulkInsertTransactions(account.ID, deduped); err != nil {
		http.Error(w, "Failed to import transactions", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/transactions/", http.StatusSeeOther)
}

// parseAmountToCents parses a decimal string to int cents
func parseAmountToCents(s string) (int, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "") // Remove thousands separator if present
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	} else if strings.HasPrefix(s, "+") {
		s = s[1:]
	}
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "$") {
		s = s[1:]
	}
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
	} else if r.Form.Has("category_id") {
		categoryID = nil // explicitly set to null if empty string is sent
	}

	reviewed := r.FormValue("reviewed") == "on"

	// Try to update the transaction
	var updateErr error
	if payee != "" {
		updateErr = s.store.UpdateTransactionPayee(account.ID, transactionID, payee)
	}
	if r.Form.Has("category_id") {
		if updateErr == nil { // Only try if previous update succeeded
			updateErr = s.store.UpdateTransactionCategory(account.ID, transactionID, categoryID)
		}
	}
	if updateErr == nil {
		updateErr = s.store.UpdateTransactionReviewed(account.ID, transactionID, reviewed)
	}

	// Fetch updated transaction list and categories for rendering
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
			templates.RenderTransactionListWithSelectableCards(w, txs, cats)
			return
		}
		// Render the modal with the error message
		w.WriteHeader(http.StatusBadRequest)
		templates.TransactionCardWithModalSelectable(*updatedTx, cats, "Failed to update transaction: "+updateErr.Error()).Render(w)
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
		templates.TransactionCardWithModalSelectable(*updatedTx, cats, "").Render(w)
	} else {
		// Transaction was marked as reviewed and is now hidden, return empty content to remove the card
		w.Write([]byte(""))
	}
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
	w.Write([]byte(`<form hx-patch="/transactions/` + idStr + `" hx-trigger="blur from:input, submit" hx-target="this" hx-swap="outerHTML" style="display:inline;">
		<input type="text" name="payee" value="` + htmlEscape(payee) + `" class="form-control form-control-sm w-auto d-inline" autofocus onblur="this.form.requestSubmit()">
	</form>`))
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
	w.Write([]byte(sb.String()))
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
	g.El("form",
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
	).Render(w)
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
	templates.RenderTransactionListWithSelectableCards(w, txs, cats)
}

func (s *Server) editCategoryHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	categoryID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
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

	// Get the category before updating to check if parent changed
	categories, err := s.store.GetCategoriesByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}

	var oldParentID *int
	for _, c := range categories {
		if c.ID == categoryID {
			oldParentID = c.ParentID
			break
		}
	}

	// Validate that parentID doesn't create a cyclic reference
	if parentID != nil {
		// Build a map for quick lookups
		catMap := make(map[int]*data.Category)
		for i := range categories {
			catMap[categories[i].ID] = &categories[i]
		}

		// Check if the new parent is a descendant of the category being edited (would create cyclic reference)
		descendants := s.store.GetDescendants(catMap, categoryID)
		if descendants[*parentID] {
			http.Error(w, "Cannot create cyclic reference", http.StatusBadRequest)
			return
		}
	}

	err = s.store.UpdateCategory(account.ID, categoryID, name, parentID)
	if err != nil {
		http.Error(w, "Failed to update category", http.StatusInternalServerError)
		return
	}

	// Check if this is an HTMX request
	if isHTMX(r) {
		// Check if parent changed - if so, remove the card, otherwise update it
		parentChanged := (oldParentID == nil && parentID != nil) ||
			(oldParentID != nil && parentID == nil) ||
			(oldParentID != nil && parentID != nil && *oldParentID != *parentID)

		if parentChanged {
			// Parent changed, remove the card
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(""))
		} else {
			// Parent didn't change, update the card
			// Get updated categories
			categories, err := s.store.GetCategoriesByAccount(account.ID)
			if err != nil {
				http.Error(w, "Failed to load categories", http.StatusInternalServerError)
				return
			}

			// Find the updated category
			var updatedCategory *data.Category
			for _, c := range categories {
				if c.ID == categoryID {
					updatedCategory = &c
					break
				}
			}

			if updatedCategory != nil {
				w.Header().Set("Content-Type", "text/html")
				card := templates.CategoryCardOnly(*updatedCategory, categories)
				_ = card.Render(w)
			} else {
				http.Error(w, "Category not found", http.StatusNotFound)
			}
		}
	} else {
		http.Redirect(w, r, "/categories/", http.StatusSeeOther)
	}
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
	_ = templates.BaseLayoutWithAuth("Rules - Budget App", true, page).Render(w)
}

func (s *Server) createRuleHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
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

	createdRule, err := s.store.CreateRule(account.ID, name, newPayeePtr, categoryID, priority, conditions)
	if err != nil {
		http.Error(w, "Failed to create rule", http.StatusInternalServerError)
		return
	}

	// Check if run_immediately was requested
	updatedCount := 0
	if r.FormValue("run_immediately") == "on" || r.FormValue("run_immediately") == "true" {
		if createdRule != nil {
			count, err := s.store.ApplyRuleToAllTransactions(account.ID, *createdRule)
			if err == nil {
				updatedCount = count
			}
		}
	}

	if isHTMX(r) {
		// Return updated rules list with banner if needed
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
		if updatedCount > 0 {
			w.Write([]byte(`<div id="rule-banner" class="alert alert-success position-fixed top-0 start-50 translate-middle-x mt-3" style="z-index:2000; min-width:300px; text-align:center;">Updated ` + strconv.Itoa(updatedCount) + ` transactions</div><script>setTimeout(function(){ var b=document.getElementById('rule-banner'); if(b){b.remove();}}, 3500);</script>`))
		}
		templates.RenderRulesList(w, rules, categories)
	} else {
		http.Redirect(w, r, "/rules/", http.StatusSeeOther)
	}
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
		templates.RenderRulesList(w, rules, categories)
	} else {
		http.Redirect(w, r, "/rules/", http.StatusSeeOther)
	}
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
		templates.RenderRulesList(w, rules, categories)
	} else {
		http.Redirect(w, r, "/rules/", http.StatusSeeOther)
	}
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
		templates.RuleCard(*updatedRule, categories).Render(w)
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
	g.El("form",
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
	).Render(w)
}
