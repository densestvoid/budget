package server

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/densestvoid/budget/data"
	"github.com/densestvoid/budget/templates"

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

	// Financial accounts routes (require authentication)
	s.router.With(authHandler.AuthRequiredMiddleware).Route("/financial-accounts", func(r chi.Router) {
		r.Get("/", s.financialAccountsPageHandler)
		r.Post("/", s.createFinancialAccountHandler)
		r.Put("/{id}", s.updateFinancialAccountHandler)
		r.Delete("/{id}", s.deleteFinancialAccountHandler)
		r.Get("/{id}/edit", s.editFinancialAccountHandler)
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

	// Budget plans routes (require authentication)
	s.router.With(authHandler.AuthRequiredMiddleware).Route("/budget-plans", func(r chi.Router) {
		r.Get("/", s.budgetPlansPageHandler)
		r.Get("/list", s.budgetPlansListHandler)
		r.Post("/", s.createBudgetPlanHandler)
		r.Put("/{id}", s.updateBudgetPlanHandler)
		r.Delete("/{id}", s.deleteBudgetPlanHandler)
		r.Post("/{id}/activate", s.activateBudgetPlanHandler)
		r.Post("/{id}/copy", s.copyBudgetPlanHandler)
	})

	// Bills routes (require authentication and budget plan selection)
	s.router.With(authHandler.AuthRequiredMiddleware, authHandler.BudgetPlanSelectionMiddleware).Route("/bills", func(r chi.Router) {
		r.Get("/", s.recurringTransactionsPageHandler)
		r.Post("/", s.createRecurringTransactionHandler)
		r.Put("/{id}", s.updateRecurringTransactionHandler)
		r.Post("/{id}/archive", s.archiveRecurringTransactionHandler)
		r.Delete("/{id}", s.deleteRecurringTransactionHandler)
		r.Get("/{id}/edit", s.editRecurringTransactionHandler)
	})

	// Budgets routes (require authentication and budget plan selection)
	s.router.With(authHandler.AuthRequiredMiddleware, authHandler.BudgetPlanSelectionMiddleware).Route("/budgets", func(r chi.Router) {
		r.Get("/", s.budgetsPageHandler)
		r.Get("/new", s.newBudgetHandler)
		r.Post("/", s.createBudgetHandler)
		r.Get("/{id}/edit", s.editBudgetHandler)
		r.Put("/{id}", s.updateBudgetHandler)
		r.Delete("/{id}", s.deleteBudgetHandler)
	})

	// Paycheck summary route (require authentication and budget plan selection)
	s.router.With(authHandler.AuthRequiredMiddleware, authHandler.BudgetPlanSelectionMiddleware).Get("/paycheck-summary", s.paycheckSummaryHandler)

	// Web routes with optional authentication and budget plan selection
	s.router.With(authHandler.OptionalAuthMiddleware, authHandler.BudgetPlanSelectionMiddleware).Get("/", s.homeHandler)
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

// requireBudgetPlan checks if a budget plan is available and redirects to budget plans page if not
func (s *Server) requireBudgetPlan(w http.ResponseWriter, r *http.Request, accountID int) bool {
	budgetPlanID, ok := r.Context().Value("budget_plan_id").(int)
	if !ok || budgetPlanID == 0 {
		// Check if any budget plans exist
		plans, err := s.store.GetBudgetPlansByAccount(accountID)
		if err != nil || len(plans) == 0 {
			// No budget plans exist, redirect to create one
			http.Redirect(w, r, "/budget-plans", http.StatusSeeOther)
			return false
		}
		// Budget plans exist but none selected, redirect to select one
		http.Redirect(w, r, "/budget-plans", http.StatusSeeOther)
		return false
	}
	return true
}

func (s *Server) homeHandler(w http.ResponseWriter, r *http.Request) {
	var isAuthenticated bool
	var moneyIn, moneyOut, netMoney float64
	var monthName string
	var expenseOccurrences []data.Occurrence
	var incomeOccurrences []data.Occurrence
	var allOccurrences []data.Occurrence
	includeUpcoming := true // Default to including upcoming transactions

	// Parse year and month from query parameters (default to current month)
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			year = y
		}
	}
	if monthStr := r.URL.Query().Get("month"); monthStr != "" {
		if m, err := strconv.Atoi(monthStr); err == nil && m >= 1 && m <= 12 {
			month = m
		}
	}

	monthName = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC).Format("January 2006")

	// Calculate previous and next months
	prevMonth := month - 1
	prevYear := year
	if prevMonth < 1 {
		prevMonth = 12
		prevYear = year - 1
	}

	nextMonth := month + 1
	nextYear := year
	if nextMonth > 12 {
		nextMonth = 1
		nextYear = year + 1
	}

	// Check if user wants to include upcoming transactions (default: true)
	if includeUpcomingStr := r.URL.Query().Get("include_upcoming"); includeUpcomingStr != "" {
		includeUpcoming = includeUpcomingStr == "true" || includeUpcomingStr == "1"
	}

	if account := r.Context().Value("account"); account != nil {
		isAuthenticated = true
		acc := account.(*data.Account)

		// Check if budget plan is required (only if authenticated)
		if !s.requireBudgetPlan(w, r, acc.ID) {
			return
		}

		// Get transactions for current month
		txs, err := s.store.GetTransactionsByMonth(acc.ID, year, month)
		if err != nil {
			log.Printf("Error fetching transactions: %v", err)
			// Continue with zero values if there's an error
		} else {
			// Calculate money in, money out, and net
			for _, tx := range txs {
				amount := float64(tx.Amount) / 100.0 // Convert cents to dollars
				if amount > 0 {
					moneyIn += amount
				} else {
					moneyOut += -amount // Make positive for display
				}
				netMoney += amount
			}
		}

		// Get budget plan ID from context
		budgetPlanID, ok := r.Context().Value("budget_plan_id").(int)
		if !ok || budgetPlanID == 0 {
			log.Printf("No budget plan selected for account %d", acc.ID)
		} else {
			// Get expenses and expand into occurrences for current month
			allExpenses, err := s.store.GetRecurringExpensesByAccount(acc.ID, budgetPlanID)
			if err != nil {
				log.Printf("Error fetching expenses: %v", err)
			} else {
				currentDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
				// Filter out archived expenses that have passed their end_date
				for _, expense := range allExpenses {
					// Skip archived expenses that have passed their end_date
					if expense.EndDate != nil && expense.EndDate.Before(currentDate) {
						continue
					}

					// Get all occurrences for this month
					occurrences := expense.GetOccurrencesInMonth(year, time.Month(month))

					// Get all matched transactions for this recurring transaction in this month
					matchedTransactions, err := s.store.GetMatchedTransactionsForRecurring(expense.ID, budgetPlanID, year, month)
					if err != nil {
						log.Printf("Error getting matched transactions: %v", err)
						matchedTransactions = []time.Time{}
					}

					// Match transactions to occurrences (each transaction can only match one occurrence)
					usedTransactions := make(map[time.Time]bool)
					for i := range occurrences {
						occ := &occurrences[i]
						// Find the closest unmatched transaction to this occurrence
						var bestMatch *time.Time
						bestDiff := 7 * 24 * time.Hour // Max 7 days difference

						for _, txDate := range matchedTransactions {
							if usedTransactions[txDate] {
								continue // This transaction is already matched to another occurrence
							}

							diff := txDate.Sub(occ.ExpectedDate)
							if diff < 0 {
								diff = -diff
							}
							if diff <= bestDiff {
								bestDiff = diff
								txDateCopy := txDate
								bestMatch = &txDateCopy
							}
						}

						if bestMatch != nil {
							occ.IsMatched = true
							occ.TransactionDate = bestMatch
							usedTransactions[*bestMatch] = true
						}

						expenseOccurrences = append(expenseOccurrences, *occ)
					}
				}

				// Sort occurrences: paid first (by transaction date), then upcoming (by expected date)
				sort.Slice(expenseOccurrences, func(i, j int) bool {
					occI := expenseOccurrences[i]
					occJ := expenseOccurrences[j]

					// Paid occurrences come first
					if occI.IsMatched && !occJ.IsMatched {
						return true
					}
					if !occI.IsMatched && occJ.IsMatched {
						return false
					}

					// Within same status, sort by date
					var dateI, dateJ time.Time
					if occI.IsMatched && occI.TransactionDate != nil {
						dateI = *occI.TransactionDate
					} else {
						dateI = occI.ExpectedDate
					}
					if occJ.IsMatched && occJ.TransactionDate != nil {
						dateJ = *occJ.TransactionDate
					} else {
						dateJ = occJ.ExpectedDate
					}
					return dateI.Before(dateJ)
				})
			}

			// Get income and expand into occurrences for current month
			allIncome, err := s.store.GetRecurringIncomeByAccount(acc.ID, budgetPlanID)
			if err != nil {
				log.Printf("Error fetching income: %v", err)
			} else {
				currentDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
				// Filter out archived income that has passed their end_date
				for _, inc := range allIncome {
					// Skip archived income that has passed their end_date
					if inc.EndDate != nil && inc.EndDate.Before(currentDate) {
						continue
					}

					// Get all occurrences for this month
					occurrences := inc.GetOccurrencesInMonth(year, time.Month(month))

					// Get all matched transactions for this recurring transaction in this month
					matchedTransactions, err := s.store.GetMatchedTransactionsForRecurring(inc.ID, budgetPlanID, year, month)
					if err != nil {
						log.Printf("Error getting matched transactions: %v", err)
						matchedTransactions = []time.Time{}
					}

					// Match transactions to occurrences (each transaction can only match one occurrence)
					usedTransactions := make(map[time.Time]bool)
					for i := range occurrences {
						occ := &occurrences[i]
						// Find the closest unmatched transaction to this occurrence
						var bestMatch *time.Time
						bestDiff := 7 * 24 * time.Hour // Max 7 days difference

						for _, txDate := range matchedTransactions {
							if usedTransactions[txDate] {
								continue // This transaction is already matched to another occurrence
							}

							diff := txDate.Sub(occ.ExpectedDate)
							if diff < 0 {
								diff = -diff
							}
							if diff <= bestDiff {
								bestDiff = diff
								txDateCopy := txDate
								bestMatch = &txDateCopy
							}
						}

						if bestMatch != nil {
							occ.IsMatched = true
							occ.TransactionDate = bestMatch
							usedTransactions[*bestMatch] = true
						}

						incomeOccurrences = append(incomeOccurrences, *occ)
					}
				}

				// Sort occurrences: matched first (by transaction date), then upcoming (by expected date)
				sort.Slice(incomeOccurrences, func(i, j int) bool {
					occI := incomeOccurrences[i]
					occJ := incomeOccurrences[j]

					// Matched occurrences come first
					if occI.IsMatched && !occJ.IsMatched {
						return true
					}
					if !occI.IsMatched && occJ.IsMatched {
						return false
					}

					// Within same status, sort by date
					var dateI, dateJ time.Time
					if occI.IsMatched && occI.TransactionDate != nil {
						dateI = *occI.TransactionDate
					} else {
						dateI = occI.ExpectedDate
					}
					if occJ.IsMatched && occJ.TransactionDate != nil {
						dateJ = *occJ.TransactionDate
					} else {
						dateJ = occJ.ExpectedDate
					}
					return dateI.Before(dateJ)
				})
			}

			// Combine all occurrences and sort by date
			allOccurrences = append(allOccurrences, expenseOccurrences...)
			allOccurrences = append(allOccurrences, incomeOccurrences...)

			// Sort all occurrences by date (use transaction date if matched, expected date if not)
			sort.Slice(allOccurrences, func(i, j int) bool {
				occI := allOccurrences[i]
				occJ := allOccurrences[j]

				var dateI, dateJ time.Time
				if occI.IsMatched && occI.TransactionDate != nil {
					dateI = *occI.TransactionDate
				} else {
					dateI = occI.ExpectedDate
				}
				if occJ.IsMatched && occJ.TransactionDate != nil {
					dateJ = *occJ.TransactionDate
				} else {
					dateJ = occJ.ExpectedDate
				}
				return dateI.Before(dateJ)
			})

			// If including upcoming transactions, add them to the totals
			if includeUpcoming {
				// Add upcoming expense occurrences
				for _, occ := range expenseOccurrences {
					if !occ.IsMatched {
						amount := float64(occ.RecurringTransaction.ExpectedAmount) / 100.0
						if amount < 0 {
							moneyOut += -amount // Make positive for display
						}
						netMoney += amount
					}
				}
				// Add upcoming income occurrences
				for _, occ := range incomeOccurrences {
					if !occ.IsMatched {
						amount := float64(occ.RecurringTransaction.ExpectedAmount) / 100.0
						if amount > 0 {
							moneyIn += amount
						}
						netMoney += amount
					}
				}
			}
		}

		log.Printf("Home handler: User is authenticated - isAuthenticated=%t", isAuthenticated)

		// Get budget plans and selected plan for selector
		var budgetPlans []data.BudgetPlan
		var selectedBudgetPlanID int
		var budgetSummaries []data.BudgetSummary
		var yearlyIncome int
		if budgetPlanID, ok := r.Context().Value("budget_plan_id").(int); ok && budgetPlanID > 0 {
			selectedBudgetPlanID = budgetPlanID
			plans, err := s.store.GetBudgetPlansByAccount(acc.ID)
			if err == nil {
				budgetPlans = plans
			}
			// Get budgets with summaries for the selected month
			summaries, err := s.store.GetBudgetSummaries(acc.ID, budgetPlanID, year, month)
			if err == nil {
				budgetSummaries = summaries
			}
			// Calculate yearly income for display
			income, err := s.store.CalculateExpectedYearlyIncome(acc.ID, budgetPlanID)
			if err == nil {
				yearlyIncome = income
			}
		}

		page := templates.SummaryPage(moneyIn, moneyOut, netMoney, monthName, allOccurrences, includeUpcoming, year, month, prevYear, prevMonth, nextYear, nextMonth, budgetPlans, selectedBudgetPlanID, budgetSummaries, yearlyIncome)
		if err := templates.BaseLayoutWithAuthAndBudgetPlan("Monthly Summary - Budget App", isAuthenticated, budgetPlans, selectedBudgetPlanID, page).Render(w); err != nil {
			log.Printf("Error rendering summary page: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	} else {
		log.Printf("Home handler: User is not authenticated - isAuthenticated=%t", isAuthenticated)
		page := templates.SummaryPage(moneyIn, moneyOut, netMoney, monthName, allOccurrences, includeUpcoming, year, month, prevYear, prevMonth, nextYear, nextMonth, nil, 0, nil, 0)
		if err := templates.BaseLayoutWithAuth("Monthly Summary - Budget App", isAuthenticated, page).Render(w); err != nil {
			log.Printf("Error rendering summary page: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
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
	financialAccounts, err := s.store.GetFinancialAccountsByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load financial accounts", http.StatusInternalServerError)
		return
	}
	errMsg := r.URL.Query().Get("error")
	successMsg := r.URL.Query().Get("success")
	page := templates.TransactionsPage(txs, cats, financialAccounts, errMsg, successMsg)
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

	// Get financial account ID from form
	financialAccountIDStr := r.FormValue("financial_account_id")
	if financialAccountIDStr == "" {
		http.Error(w, "Financial account is required", http.StatusBadRequest)
		return
	}
	financialAccountID, err := strconv.Atoi(financialAccountIDStr)
	if err != nil {
		http.Error(w, "Invalid financial account ID", http.StatusBadRequest)
		return
	}

	// Load financial account and its CSV mappings
	financialAccount, err := s.store.GetFinancialAccount(account.ID, financialAccountID)
	if err != nil || financialAccount == nil {
		http.Error(w, "Financial account not found", http.StatusBadRequest)
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
	// Map CSV column names to indices using financial account's field mappings
	colIdx := make(map[string]int)
	requiredFields := map[string]string{
		"date":    financialAccount.CSVDateField,
		"payee":   financialAccount.CSVPayeeField,
		"expense": financialAccount.CSVExpenseField,
		"income":  financialAccount.CSVIncomeField,
	}
	optionalFields := make(map[string]string)
	if financialAccount.CSVCategoryField != nil {
		optionalFields["category"] = *financialAccount.CSVCategoryField
	}
	if financialAccount.CSVBalanceField != nil {
		optionalFields["balance"] = *financialAccount.CSVBalanceField
	}

	// Find column indices for required fields
	missing := []string{}
	for fieldName, mappedName := range requiredFields {
		found := false
		for i, col := range header {
			if strings.EqualFold(strings.TrimSpace(col), mappedName) {
				colIdx[fieldName] = i
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, mappedName)
		}
	}

	// Find column indices for optional fields
	for fieldName, mappedName := range optionalFields {
		for i, col := range header {
			if strings.EqualFold(strings.TrimSpace(col), mappedName) {
				colIdx[fieldName] = i
				break
			}
		}
	}

	if len(missing) > 0 {
		errMsg := "CSV error: Missing required columns: " + strings.Join(missing, ", ")
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

	// Get all categories for matching
	categories, err := s.store.GetCategoriesByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}
	categoryMap := make(map[string]int) // category name (lowercase) -> category ID
	for _, cat := range categories {
		categoryMap[strings.ToLower(cat.Name)] = cat.ID
	}

	// Get or create "Imported" category
	var importedCategoryID *int
	if importedID, found := categoryMap["imported"]; found {
		importedCategoryID = &importedID
	} else {
		// Create "Imported" category
		importedCat, err := s.store.CreateCategory(account.ID, "Imported", nil)
		if err != nil {
			log.Printf("Failed to create Imported category: %v", err)
			http.Error(w, "Failed to create Imported category", http.StatusInternalServerError)
			return
		}
		importedCategoryID = &importedCat.ID
		categoryMap["imported"] = importedCat.ID
		// Reload categories to include the new one
		categories, err = s.store.GetCategoriesByAccount(account.ID)
		if err == nil {
			for _, cat := range categories {
				categoryMap[strings.ToLower(cat.Name)] = cat.ID
			}
		}
	}

	var txs []data.Transaction
	var lastBalance *int
	var lastBalanceDate *time.Time
	var skippedCount int
	var skippedReasons []string

	for i, rec := range records[1:] {
		if len(rec) < len(header) {
			skippedCount++
			skippedReasons = append(skippedReasons, fmt.Sprintf("Row %d: missing columns", i+2))
			continue
		}

		// Parse date - skip if invalid
		dateStr := strings.TrimSpace(rec[colIdx["date"]])
		if dateStr == "" || strings.EqualFold(dateStr, "pending") {
			skippedCount++
			skippedReasons = append(skippedReasons, fmt.Sprintf("Row %d: invalid or pending date", i+2))
			continue
		}
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			// Try mm/dd/yyyy
			date, err = time.Parse("01/02/2006", dateStr)
			if err != nil {
				skippedCount++
				skippedReasons = append(skippedReasons, fmt.Sprintf("Row %d: invalid date format: %s", i+2, dateStr))
				continue
			}
		}

		// Parse payee - skip if empty
		payee := strings.TrimSpace(rec[colIdx["payee"]])
		if payee == "" {
			skippedCount++
			skippedReasons = append(skippedReasons, fmt.Sprintf("Row %d: empty payee", i+2))
			continue
		}

		// Calculate amount from expense and income fields - skip if invalid
		var amountCents int
		expenseStr := strings.TrimSpace(rec[colIdx["expense"]])
		incomeStr := strings.TrimSpace(rec[colIdx["income"]])

		if expenseStr != "" {
			expenseCents, err := parseAmountToCents(expenseStr)
			if err != nil {
				skippedCount++
				skippedReasons = append(skippedReasons, fmt.Sprintf("Row %d: invalid expense amount: %s", i+2, expenseStr))
				continue
			}
			amountCents = -expenseCents // Expenses are negative
		} else if incomeStr != "" {
			incomeCents, err := parseAmountToCents(incomeStr)
			if err != nil {
				skippedCount++
				skippedReasons = append(skippedReasons, fmt.Sprintf("Row %d: invalid income amount: %s", i+2, incomeStr))
				continue
			}
			amountCents = incomeCents // Income is positive
		} else {
			skippedCount++
			skippedReasons = append(skippedReasons, fmt.Sprintf("Row %d: both expense and income fields are empty", i+2))
			continue
		}

		// Parse optional category - create if doesn't exist under "Imported"
		var categoryID *int
		if catIdx, ok := colIdx["category"]; ok {
			categoryName := strings.TrimSpace(rec[catIdx])
			if categoryName != "" {
				categoryNameLower := strings.ToLower(categoryName)
				if catID, found := categoryMap[categoryNameLower]; found {
					categoryID = &catID
				} else {
					// Category doesn't exist, create it under "Imported"
					newCat, err := s.store.CreateCategory(account.ID, categoryName, importedCategoryID)
					if err != nil {
						log.Printf("Failed to create category %s: %v", categoryName, err)
						// Continue without category rather than failing
					} else {
						categoryID = &newCat.ID
						categoryMap[categoryNameLower] = newCat.ID
					}
				}
			}
		}

		// Check balance field if present - skip transaction if invalid/pending
		if balanceIdx, ok := colIdx["balance"]; ok {
			balanceStr := strings.TrimSpace(rec[balanceIdx])
			if balanceStr == "" || strings.EqualFold(balanceStr, "pending") {
				skippedCount++
				skippedReasons = append(skippedReasons, fmt.Sprintf("Row %d: invalid or pending balance", i+2))
				continue
			}
			// Validate balance can be parsed
			balanceCents, err := parseAmountToCents(balanceStr)
			if err != nil {
				skippedCount++
				skippedReasons = append(skippedReasons, fmt.Sprintf("Row %d: invalid balance format: %s", i+2, balanceStr))
				continue
			}
			// Track balance for account update
			if lastBalanceDate == nil || !date.Before(*lastBalanceDate) {
				lastBalance = &balanceCents
				dateCopy := date
				lastBalanceDate = &dateCopy
			}
		}

		txs = append(txs, data.Transaction{
			AccountID:          account.ID,
			FinancialAccountID: financialAccountID,
			Date:               date,
			OriginalPayee:      payee,
			Payee:              payee,
			CategoryID:         categoryID,
			Amount:             amountCents,
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

	// Insert transactions
	if err := s.store.BulkInsertTransactions(account.ID, financialAccountID, deduped); err != nil {
		http.Error(w, "Failed to import transactions", http.StatusInternalServerError)
		return
	}

	// Build success message with import stats
	var successMsg string
	if len(deduped) > 0 {
		successMsg = fmt.Sprintf("Successfully imported %d transaction(s)", len(deduped))
		if skippedCount > 0 {
			successMsg += fmt.Sprintf(". Skipped %d invalid transaction(s)", skippedCount)
		}
	} else if skippedCount > 0 {
		successMsg = fmt.Sprintf("No valid transactions to import. Skipped %d invalid transaction(s)", skippedCount)
	} else {
		successMsg = "No new transactions to import (all were duplicates)"
	}

	// Log skipped reasons for debugging
	if len(skippedReasons) > 0 {
		log.Printf("CSV import skipped transactions: %v", skippedReasons)
	}

	// Update account balance if balance field was present and we have a valid balance
	if lastBalance != nil && lastBalanceDate != nil {
		// Check if this transaction date is >= current latest transaction date for this account
		// Get the latest transaction date for this financial account
		var latestTxDate sql.NullTime
		query := `SELECT MAX(date) FROM transactions WHERE financial_account_id = $1`
		err := s.store.GetDB().QueryRow(query, financialAccountID).Scan(&latestTxDate)
		if err == nil && latestTxDate.Valid {
			// Only update if the imported transaction date is >= current latest
			if !lastBalanceDate.Before(latestTxDate.Time) {
				if err := s.store.UpdateFinancialAccountBalance(account.ID, financialAccountID, *lastBalance); err != nil {
					log.Printf("Failed to update financial account balance: %v", err)
					// Don't fail the import if balance update fails
				}
			}
		} else {
			// No existing transactions, update balance
			if err := s.store.UpdateFinancialAccountBalance(account.ID, financialAccountID, *lastBalance); err != nil {
				log.Printf("Failed to update financial account balance: %v", err)
			}
		}
	}

	// Redirect with success message
	if successMsg != "" {
		http.Redirect(w, r, "/transactions/?success="+url.QueryEscape(successMsg), http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/transactions/", http.StatusSeeOther)
	}
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
	cardID := "transaction-card-" + strconv.Itoa(transaction.ID)
	g.El("form",
		g.Attr("hx-patch", "/transactions/"+strconv.Itoa(transaction.ID)),
		g.Attr("hx-target", "#"+cardID),
		g.Attr("hx-swap", "outerHTML"),
		g.Attr("hx-on::after-request", `if (event.detail.successful) { const modal = bootstrap.Modal.getInstance(document.querySelector('#editTransactionModal')); if (modal) modal.hide(); }`),
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

	// Check if run_on_existing was requested
	updatedCount := 0
	if r.FormValue("run_on_existing") == "on" || r.FormValue("run_on_existing") == "true" {
		// Get the updated rule to apply it
		rules, err := s.store.GetRulesByAccount(account.ID)
		if err == nil {
			for _, rule := range rules {
				if rule.ID == ruleID {
					count, err := s.store.ApplyRuleToAllTransactions(account.ID, rule)
					if err == nil {
						updatedCount = count
					}
					break
				}
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

// RecurringTransaction management handlers

func (s *Server) recurringTransactionsPageHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}
	if !s.requireBudgetPlan(w, r, account.ID) {
		return
	}
	budgetPlanID, _ := r.Context().Value("budget_plan_id").(int)
	rts, err := s.store.GetRecurringTransactionsByAccount(account.ID, budgetPlanID)
	if err != nil {
		http.Error(w, "Failed to load recurring transactions", http.StatusInternalServerError)
		return
	}
	categories, err := s.store.GetCategoriesByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}
	financialAccounts, err := s.store.GetFinancialAccountsByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load financial accounts", http.StatusInternalServerError)
		return
	}
	budgetPlans, err := s.store.GetBudgetPlansByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load budget plans", http.StatusInternalServerError)
		return
	}
	page := templates.RecurringTransactionsPage(rts, categories, financialAccounts, budgetPlans, budgetPlanID)
	_ = templates.BaseLayoutWithAuthAndBudgetPlan("Recurring Transactions - Budget App", true, budgetPlans, budgetPlanID, page).Render(w)
}

func (s *Server) createRecurringTransactionHandler(w http.ResponseWriter, r *http.Request) {
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
	counterparty := r.FormValue("counterparty")
	categoryIDStr := r.FormValue("category_id")
	expectedAmountStr := r.FormValue("expected_amount")
	toleranceStr := r.FormValue("tolerance")
	startDateStr := r.FormValue("start_date")
	recurrenceUnitStr := r.FormValue("recurrence_unit")
	recurrenceValueStr := r.FormValue("recurrence_value")

	if name == "" || counterparty == "" || expectedAmountStr == "" || toleranceStr == "" || startDateStr == "" || recurrenceUnitStr == "" || recurrenceValueStr == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
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
			}
		}
	}

	expectedAmount, err := parseAmountToCents(expectedAmountStr)
	if err != nil {
		http.Error(w, "Invalid expected amount", http.StatusBadRequest)
		return
	}

	// Type is generated from amount sign by the database
	// Store amount as-is: negative = expense, positive = income
	// No conversion needed - user enters the signed value

	tolerance, err := parseAmountToCents(toleranceStr)
	if err != nil {
		http.Error(w, "Invalid tolerance", http.StatusBadRequest)
		return
	}
	if tolerance < 0 {
		tolerance = -tolerance // Make positive
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		http.Error(w, "Invalid start date", http.StatusBadRequest)
		return
	}

	recurrenceUnit := recurrenceUnitStr
	if recurrenceUnit != "week" && recurrenceUnit != "month" && recurrenceUnit != "year" {
		http.Error(w, "Recurrence unit must be 'week', 'month', or 'year'", http.StatusBadRequest)
		return
	}

	recurrenceValue, err := strconv.Atoi(recurrenceValueStr)
	if err != nil {
		http.Error(w, "Invalid recurrence value", http.StatusBadRequest)
		return
	}
	if recurrenceValue < 1 {
		http.Error(w, "Recurrence value must be greater than 0", http.StatusBadRequest)
		return
	}

	financialAccountIDStr := r.FormValue("financial_account_id")
	if financialAccountIDStr == "" {
		http.Error(w, "Financial account is required", http.StatusBadRequest)
		return
	}
	financialAccountID, err := strconv.Atoi(financialAccountIDStr)
	if err != nil {
		http.Error(w, "Invalid financial account ID", http.StatusBadRequest)
		return
	}
	// Verify the financial account belongs to this account
	_, err = s.store.GetFinancialAccount(account.ID, financialAccountID)
	if err != nil || err == sql.ErrNoRows {
		http.Error(w, "Financial account not found", http.StatusBadRequest)
		return
	}

	budgetPlanID, ok := r.Context().Value("budget_plan_id").(int)
	if !ok || budgetPlanID == 0 {
		http.Error(w, "No budget plan selected", http.StatusBadRequest)
		return
	}

	_, err = s.store.CreateRecurringTransaction(account.ID, budgetPlanID, financialAccountID, name, counterparty, categoryID, expectedAmount, tolerance, startDate, recurrenceUnit, recurrenceValue)
	if err != nil {
		http.Error(w, "Failed to create recurring transaction", http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		// Return updated recurring transactions list
		budgetPlanID, ok := r.Context().Value("budget_plan_id").(int)
		if !ok || budgetPlanID == 0 {
			http.Error(w, "No budget plan selected", http.StatusBadRequest)
			return
		}
		rts, err := s.store.GetRecurringTransactionsByAccount(account.ID, budgetPlanID)
		if err != nil {
			http.Error(w, "Failed to load recurring transactions", http.StatusInternalServerError)
			return
		}
		categories, err := s.store.GetCategoriesByAccount(account.ID)
		if err != nil {
			http.Error(w, "Failed to load categories", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		templates.RenderRecurringTransactionsList(w, rts, categories)
	} else {
		http.Redirect(w, r, "/bills/", http.StatusSeeOther)
	}
}

func (s *Server) updateRecurringTransactionHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	rtID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid recurring transaction ID", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	counterparty := r.FormValue("counterparty")
	categoryIDStr := r.FormValue("category_id")
	expectedAmountStr := r.FormValue("expected_amount")
	toleranceStr := r.FormValue("tolerance")
	startDateStr := r.FormValue("start_date")
	recurrenceUnitStr := r.FormValue("recurrence_unit")
	recurrenceValueStr := r.FormValue("recurrence_value")

	if name == "" || counterparty == "" || expectedAmountStr == "" || toleranceStr == "" || startDateStr == "" || recurrenceUnitStr == "" || recurrenceValueStr == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
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
			}
		}
	}

	expectedAmount, err := parseAmountToCents(expectedAmountStr)
	if err != nil {
		http.Error(w, "Invalid expected amount", http.StatusBadRequest)
		return
	}

	// Type is generated from amount sign by the database
	// Store amount as-is: negative = expense, positive = income
	// User enters the signed value directly, no conversion needed

	tolerance, err := parseAmountToCents(toleranceStr)
	if err != nil {
		http.Error(w, "Invalid tolerance", http.StatusBadRequest)
		return
	}
	if tolerance < 0 {
		tolerance = -tolerance // Make positive
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		http.Error(w, "Invalid start date", http.StatusBadRequest)
		return
	}

	recurrenceUnit := recurrenceUnitStr
	if recurrenceUnit != "week" && recurrenceUnit != "month" && recurrenceUnit != "year" {
		http.Error(w, "Recurrence unit must be 'week', 'month', or 'year'", http.StatusBadRequest)
		return
	}

	recurrenceValue, err := strconv.Atoi(recurrenceValueStr)
	if err != nil {
		http.Error(w, "Invalid recurrence value", http.StatusBadRequest)
		return
	}
	if recurrenceValue < 1 {
		http.Error(w, "Recurrence value must be greater than 0", http.StatusBadRequest)
		return
	}

	financialAccountIDStr := r.FormValue("financial_account_id")
	if financialAccountIDStr == "" {
		http.Error(w, "Financial account is required", http.StatusBadRequest)
		return
	}
	financialAccountID, err := strconv.Atoi(financialAccountIDStr)
	if err != nil {
		http.Error(w, "Invalid financial account ID", http.StatusBadRequest)
		return
	}
	// Verify the financial account belongs to this account
	_, err = s.store.GetFinancialAccount(account.ID, financialAccountID)
	if err != nil {
		http.Error(w, "Financial account not found", http.StatusBadRequest)
		return
	}

	budgetPlanID, ok := r.Context().Value("budget_plan_id").(int)
	if !ok || budgetPlanID == 0 {
		http.Error(w, "No budget plan selected", http.StatusBadRequest)
		return
	}

	err = s.store.UpdateRecurringTransaction(account.ID, rtID, budgetPlanID, financialAccountID, name, counterparty, categoryID, expectedAmount, tolerance, startDate, recurrenceUnit, recurrenceValue)
	if err != nil {
		http.Error(w, "Failed to update recurring transaction", http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		// Return updated recurring transactions list
		rts, err := s.store.GetRecurringTransactionsByAccount(account.ID, budgetPlanID)
		if err != nil {
			http.Error(w, "Failed to load recurring transactions", http.StatusInternalServerError)
			return
		}
		categories, err := s.store.GetCategoriesByAccount(account.ID)
		if err != nil {
			http.Error(w, "Failed to load categories", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		templates.RenderRecurringTransactionsList(w, rts, categories)
	} else {
		http.Redirect(w, r, "/bills/", http.StatusSeeOther)
	}
}

func (s *Server) archiveRecurringTransactionHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	rtID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid recurring transaction ID", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	endDateStr := r.FormValue("end_date")
	if endDateStr == "" {
		http.Error(w, "Missing end_date", http.StatusBadRequest)
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		http.Error(w, "Invalid end date", http.StatusBadRequest)
		return
	}

	err = s.store.ArchiveRecurringTransaction(account.ID, rtID, endDate)
	if err != nil {
		http.Error(w, "Failed to archive recurring transaction", http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		// Return updated recurring transactions list
		budgetPlanID, ok := r.Context().Value("budget_plan_id").(int)
		if !ok || budgetPlanID == 0 {
			http.Error(w, "No budget plan selected", http.StatusBadRequest)
			return
		}
		rts, err := s.store.GetRecurringTransactionsByAccount(account.ID, budgetPlanID)
		if err != nil {
			http.Error(w, "Failed to load recurring transactions", http.StatusInternalServerError)
			return
		}
		categories, err := s.store.GetCategoriesByAccount(account.ID)
		if err != nil {
			http.Error(w, "Failed to load categories", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		templates.RenderRecurringTransactionsList(w, rts, categories)
	} else {
		http.Redirect(w, r, "/bills/", http.StatusSeeOther)
	}
}

func (s *Server) deleteRecurringTransactionHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	rtID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid recurring transaction ID", http.StatusBadRequest)
		return
	}

	err = s.store.DeleteRecurringTransaction(account.ID, rtID)
	if err != nil {
		http.Error(w, "Failed to delete recurring transaction", http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		// Return updated recurring transactions list
		budgetPlanID, ok := r.Context().Value("budget_plan_id").(int)
		if !ok || budgetPlanID == 0 {
			http.Error(w, "No budget plan selected", http.StatusBadRequest)
			return
		}
		rts, err := s.store.GetRecurringTransactionsByAccount(account.ID, budgetPlanID)
		if err != nil {
			http.Error(w, "Failed to load recurring transactions", http.StatusInternalServerError)
			return
		}
		categories, err := s.store.GetCategoriesByAccount(account.ID)
		if err != nil {
			http.Error(w, "Failed to load categories", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		templates.RenderRecurringTransactionsList(w, rts, categories)
	} else {
		http.Redirect(w, r, "/bills/", http.StatusSeeOther)
	}
}

func (s *Server) editRecurringTransactionHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	rtID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid recurring transaction ID", http.StatusBadRequest)
		return
	}

	// Get the recurring transaction
	rt, err := s.store.GetRecurringTransaction(account.ID, rtID)
	if err != nil {
		http.Error(w, "Failed to load recurring transaction", http.StatusInternalServerError)
		return
	}

	if rt == nil {
		http.Error(w, "Recurring transaction not found", http.StatusNotFound)
		return
	}

	// Get categories and financial accounts for the form
	categories, err := s.store.GetCategoriesByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}
	financialAccounts, err := s.store.GetFinancialAccountsByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load financial accounts", http.StatusInternalServerError)
		return
	}

	budgetPlans, err := s.store.GetBudgetPlansByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load budget plans", http.StatusInternalServerError)
		return
	}

	budgetPlanID, ok := r.Context().Value("budget_plan_id").(int)
	if !ok || budgetPlanID == 0 {
		http.Error(w, "No budget plan selected", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html")

	// Render the edit form
	g.El("form",
		g.Attr("hx-put", "/bills/"+strconv.Itoa(rt.ID)),
		g.Attr("hx-target", "#recurring-transactions-list"),
		g.Attr("hx-swap", "outerHTML"),
		g.Attr("hx-on::after-request", `if (event.target === this) { bootstrap.Modal.getInstance(this.closest('.modal')).hide(); }`),
		templates.RecurringTransactionFormFields(categories, financialAccounts, budgetPlans, budgetPlanID, rt),
		html.Div(
			html.Class("d-flex justify-content-end gap-2"),
			html.Button(html.Type("button"), html.Class("btn btn-secondary"), html.DataAttr("bs-dismiss", "modal"), g.Text("Cancel")),
			html.Button(html.Type("submit"), html.Class("btn btn-primary"), g.Text("Save Changes")),
		),
	).Render(w)
}

func (s *Server) paycheckSummaryHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}
	if !s.requireBudgetPlan(w, r, account.ID) {
		return
	}

	// Get all financial accounts for selection
	financialAccounts, err := s.store.GetFinancialAccountsByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load financial accounts", http.StatusInternalServerError)
		return
	}

	// Get selected financial account from query parameter
	var selectedFinancialAccount *data.FinancialAccount
	financialAccountIDStr := r.URL.Query().Get("financial_account_id")
	if financialAccountIDStr != "" {
		financialAccountID, err := strconv.Atoi(financialAccountIDStr)
		if err == nil {
			fa, err := s.store.GetFinancialAccount(account.ID, financialAccountID)
			if err == nil && fa != nil {
				selectedFinancialAccount = fa
			}
		}
	}
	// If no account selected, use the first one if available
	if selectedFinancialAccount == nil && len(financialAccounts) > 0 {
		selectedFinancialAccount = &financialAccounts[0]
	}

	// Get budget plan ID from context
	budgetPlanID, ok := r.Context().Value("budget_plan_id").(int)
	if !ok || budgetPlanID == 0 {
		http.Error(w, "No budget plan selected", http.StatusBadRequest)
		return
	}

	// Parse offset parameter (default 0 for current period, must be >= 0)
	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Find base last received income date (for offset 0)
	baseLastIncomeDate, err := s.store.GetLastReceivedIncomeOccurrence(account.ID, budgetPlanID)
	if err != nil {
		log.Printf("Error getting last received income: %v", err)
		http.Error(w, "Failed to load paycheck summary data", http.StatusInternalServerError)
		return
	}

	// If no last income date but we have recurring income, use the earliest start date as baseline
	if baseLastIncomeDate == nil {
		incomeRts, err := s.store.GetRecurringIncomeByAccount(account.ID, budgetPlanID)
		if err == nil && len(incomeRts) > 0 {
			// Find the earliest start date
			earliestStart := incomeRts[0].StartDate
			for _, rt := range incomeRts {
				if rt.StartDate.Before(earliestStart) {
					earliestStart = rt.StartDate
				}
			}
			// Use the earliest start date if it's today or in the past
			// If it's in the future, use today as the baseline
			now := time.Now().Truncate(24 * time.Hour)
			if earliestStart.Before(now) || earliestStart.Equal(now) {
				baseLastIncomeDate = &earliestStart
			} else {
				// Start date is in the future, use today
				baseLastIncomeDate = &now
			}
		}
	}

	// Calculate lastIncomeDate based on offset (offset >= 0, where 0 is current period)
	var lastIncomeDate *time.Time
	if offset == 0 {
		lastIncomeDate = baseLastIncomeDate
	} else {
		// Go forwards: find next income occurrences
		currentDate := baseLastIncomeDate
		if currentDate == nil {
			// No base date, start from now
			now := time.Now()
			currentDate = &now
		}
		for i := 0; i < offset; i++ {
			next, err := s.store.GetNextIncomeOccurrenceAfter(account.ID, budgetPlanID, *currentDate)
			if err != nil || next == nil {
				// No more next income found
				lastIncomeDate = nil
				break
			}
			lastIncomeDate = next
			currentDate = next
		}
	}

	// Calculate balance for the period being viewed
	var balanceCents *int
	if offset == 0 {
		// Current period: use actual account balance
		if selectedFinancialAccount != nil {
			balanceCents = &selectedFinancialAccount.Balance
		} else {
			// Fallback to last income amount if no financial account
			lastIncomeAmount, err := s.store.GetLastReceivedIncomeAmount(account.ID, budgetPlanID)
			if err == nil && lastIncomeAmount != nil {
				balanceCents = lastIncomeAmount
			}
		}
	} else {
		// Future period: calculate balance by applying all recurring transactions
		// Start with the current period balance
		var startingBalance int
		if selectedFinancialAccount != nil {
			startingBalance = selectedFinancialAccount.Balance
		} else {
			lastIncomeAmount, err := s.store.GetLastReceivedIncomeAmount(account.ID, budgetPlanID)
			if err == nil && lastIncomeAmount != nil {
				startingBalance = *lastIncomeAmount
			}
		}

		// Apply all recurring transactions between baseLastIncomeDate and lastIncomeDate
		if baseLastIncomeDate != nil && lastIncomeDate != nil {
			// Iterate through each paycheck period between base and target
			currentPeriodStart := *baseLastIncomeDate

			for currentPeriodStart.Before(*lastIncomeDate) {
				// Find the end of this period (next income occurrence)
				periodEnd, err := s.store.GetNextIncomeOccurrenceAfter(account.ID, budgetPlanID, currentPeriodStart)
				if err != nil || periodEnd == nil {
					break
				}
				if !periodEnd.Before(*lastIncomeDate) && !periodEnd.Equal(*lastIncomeDate) {
					// We've reached or passed the target period
					break
				}

				// Process all recurring transactions in this period
				periodStart := currentPeriodStart
				periodEndDate := *periodEnd

				// Get all expense occurrences in this period
				expenseOccs, err := s.store.GetUpcomingExpenseOccurrencesBetweenDates(account.ID, budgetPlanID, periodStart, periodEndDate)
				if err == nil {
					for _, occ := range expenseOccs {
						// Filter by financial account if selected
						if selectedFinancialAccount == nil || occ.RecurringTransaction.FinancialAccountID == selectedFinancialAccount.ID {
							// Apply expense (negative amount)
							startingBalance += occ.RecurringTransaction.ExpectedAmount
						}
					}
				}

				// Apply only the income that starts this period (at periodStart)
				// The income at periodEndDate starts the next period, so we don't apply it here
				incomeRts, err := s.store.GetRecurringIncomeByAccount(account.ID, budgetPlanID)
				if err == nil {
					for _, rt := range incomeRts {
						if selectedFinancialAccount == nil || rt.FinancialAccountID == selectedFinancialAccount.ID {
							// Check if this recurring income has an occurrence at periodStart
							occDate := rt.NextOccurrence(periodStart.AddDate(0, 0, -1))
							if !occDate.IsZero() && occDate.Equal(periodStart) {
								// This income starts the period, apply it
								startingBalance += rt.ExpectedAmount
							}
						}
					}
				}

				// Move to next period
				currentPeriodStart = periodEndDate
			}
		}

		balanceCents = &startingBalance
	}

	// Find next upcoming income date based on lastIncomeDate
	var nextIncomeDate *time.Time
	if lastIncomeDate != nil {
		nextIncomeDate, err = s.store.GetNextIncomeOccurrenceAfter(account.ID, budgetPlanID, *lastIncomeDate)
		if err != nil {
			log.Printf("Error getting next upcoming income: %v", err)
			http.Error(w, "Failed to load paycheck summary data", http.StatusInternalServerError)
			return
		}
	} else {
		// Fallback to original method if no lastIncomeDate
		nextIncomeDate, err = s.store.GetNextUpcomingIncomeOccurrence(account.ID, budgetPlanID)
		if err != nil {
			log.Printf("Error getting next upcoming income: %v", err)
			http.Error(w, "Failed to load paycheck summary data", http.StatusInternalServerError)
			return
		}
	}

	// Calculate previous and next paycheck periods for pagination
	// Previous = go back towards offset 0 (only if offset > 0)
	// Next = go forward to next period (offset + 1)
	var prevLastIncomeDate, nextLastIncomeDate *time.Time
	// Previous period: only if we're not at offset 0 (current period)
	if offset > 0 {
		// Calculate what lastIncomeDate would be at offset - 1
		if offset == 1 {
			// Going back to offset 0 means using baseLastIncomeDate
			prevLastIncomeDate = baseLastIncomeDate
		} else {
			// Calculate the period at offset - 1
			currentDate := baseLastIncomeDate
			if currentDate == nil {
				now := time.Now()
				currentDate = &now
			}
			for i := 0; i < offset-1; i++ {
				next, err := s.store.GetNextIncomeOccurrenceAfter(account.ID, budgetPlanID, *currentDate)
				if err != nil || next == nil {
					prevLastIncomeDate = nil
					break
				}
				prevLastIncomeDate = next
				currentDate = next
			}
		}
	}
	if nextIncomeDate != nil {
		// Next period: find the income occurrence after nextIncomeDate
		nextLastIncomeDate, _ = s.store.GetNextIncomeOccurrenceAfter(account.ID, budgetPlanID, *nextIncomeDate)
	}

	// Get budget plans for selector
	budgetPlans, err := s.store.GetBudgetPlansByAccount(account.ID)
	if err != nil {
		log.Printf("Error getting budget plans: %v", err)
		budgetPlans = []data.BudgetPlan{}
	}

	// If we still don't have dates, show the page with no data message
	if lastIncomeDate == nil || nextIncomeDate == nil {
		page := templates.PaycheckSummaryPage(nil, nil, nil, nil, financialAccounts, selectedFinancialAccount, nil, 0, nil, prevLastIncomeDate, nextLastIncomeDate, offset, budgetPlans, budgetPlanID)
		if err := templates.BaseLayoutWithAuthAndBudgetPlan("Paycheck Summary - Budget App", true, budgetPlans, budgetPlanID, page).Render(w); err != nil {
			log.Printf("Error rendering paycheck summary page: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Get all transactions in the period (between last income and next income)
	// Filter by financial account if one is selected
	var transactions []data.Transaction
	allTransactions, err := s.store.GetTransactionsBetweenDates(account.ID, *lastIncomeDate, *nextIncomeDate)
	if err != nil {
		log.Printf("Error getting transactions between dates: %v", err)
		http.Error(w, "Failed to load transactions", http.StatusInternalServerError)
		return
	}

	// Filter by financial account if selected
	if selectedFinancialAccount != nil {
		for _, tx := range allTransactions {
			if tx.FinancialAccountID == selectedFinancialAccount.ID {
				transactions = append(transactions, tx)
			}
		}
	} else {
		transactions = allTransactions
	}

	// Get upcoming expense occurrences in the period
	expenseOccurrences, err := s.store.GetUpcomingExpenseOccurrencesBetweenDates(account.ID, budgetPlanID, *lastIncomeDate, *nextIncomeDate)
	if err != nil {
		log.Printf("Error getting expense occurrences: %v", err)
		http.Error(w, "Failed to load expense occurrences", http.StatusInternalServerError)
		return
	}

	// Calculate total expenses
	// Count actual transactions (which includes matched recurring transactions)
	var totalExpensesCents int64
	for _, tx := range transactions {
		// Only count negative amounts (expenses)
		if tx.Amount < 0 {
			totalExpensesCents += int64(-tx.Amount) // Make positive for calculation
		}
	}
	// Only count unmatched expense occurrences to avoid double-counting
	// Matched occurrences are already counted as actual transactions above
	for _, occ := range expenseOccurrences {
		// Only count expenses (negative expected amounts) that haven't been matched yet
		if occ.RecurringTransaction.ExpectedAmount < 0 && !occ.IsMatched {
			totalExpensesCents += int64(-occ.RecurringTransaction.ExpectedAmount) // Make positive
		}
	}

	// Calculate expected remaining balance
	var expectedRemainingCents *int64
	if balanceCents != nil {
		remaining := int64(*balanceCents) - totalExpensesCents
		expectedRemainingCents = &remaining
	}

	page := templates.PaycheckSummaryPage(
		lastIncomeDate,
		nextIncomeDate,
		transactions,
		expenseOccurrences,
		financialAccounts,
		selectedFinancialAccount,
		balanceCents,
		totalExpensesCents,
		expectedRemainingCents,
		prevLastIncomeDate,
		nextLastIncomeDate,
		offset,
		budgetPlans,
		budgetPlanID,
	)
	if err := templates.BaseLayoutWithAuthAndBudgetPlan("Paycheck Summary - Budget App", true, budgetPlans, budgetPlanID, page).Render(w); err != nil {
		log.Printf("Error rendering paycheck summary page: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Budget plan management handlers

func (s *Server) budgetPlansListHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	plans, err := s.store.GetBudgetPlansByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load budget plans", http.StatusInternalServerError)
		return
	}
	// Get transaction counts for each plan
	plansWithCounts := make([]templates.PlanWithCount, len(plans))
	for i, p := range plans {
		var count int
		query := `SELECT COUNT(*) FROM recurring_transactions WHERE budget_plan_id = $1`
		err := s.store.GetDB().QueryRow(query, p.ID).Scan(&count)
		if err != nil {
			count = 0
		}
		plansWithCounts[i] = templates.PlanWithCount{
			BudgetPlan:       p,
			TransactionCount: count,
		}
	}
	w.Header().Set("Content-Type", "text/html")
	templates.RenderBudgetPlansList(w, plansWithCounts)
}

func (s *Server) budgetPlansPageHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}
	plans, err := s.store.GetBudgetPlansByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load budget plans", http.StatusInternalServerError)
		return
	}
	// Get transaction counts for each plan
	plansWithCounts := make([]templates.PlanWithCount, len(plans))
	for i, p := range plans {
		var count int
		query := `SELECT COUNT(*) FROM recurring_transactions WHERE budget_plan_id = $1`
		err := s.store.GetDB().QueryRow(query, p.ID).Scan(&count)
		if err != nil {
			count = 0
		}
		plansWithCounts[i] = templates.PlanWithCount{
			BudgetPlan:       p,
			TransactionCount: count,
		}
	}
	// Get budget plans for sidebar selector
	var budgetPlans []data.BudgetPlan
	for _, pwc := range plansWithCounts {
		budgetPlans = append(budgetPlans, pwc.BudgetPlan)
	}
	// Get selected budget plan ID from context
	var selectedBudgetPlanID int
	if budgetPlanID, ok := r.Context().Value("budget_plan_id").(int); ok {
		selectedBudgetPlanID = budgetPlanID
	}
	page := templates.BudgetPlansPage(plansWithCounts)
	_ = templates.BaseLayoutWithAuthAndBudgetPlan("Budget Plans - Budget App", true, budgetPlans, selectedBudgetPlanID, page).Render(w)
}

func (s *Server) createBudgetPlanHandler(w http.ResponseWriter, r *http.Request) {
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
	if name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}
	_, err := s.store.CreateBudgetPlan(account.ID, name)
	if err != nil {
		http.Error(w, "Failed to create budget plan", http.StatusInternalServerError)
		return
	}
	if isHTMX(r) {
		// Return updated budget plans list
		plans, err := s.store.GetBudgetPlansByAccount(account.ID)
		if err != nil {
			http.Error(w, "Failed to load budget plans", http.StatusInternalServerError)
			return
		}
		plansWithCounts := make([]templates.PlanWithCount, len(plans))
		for i, p := range plans {
			var count int
			query := `SELECT COUNT(*) FROM recurring_transactions WHERE budget_plan_id = $1`
			err := s.store.GetDB().QueryRow(query, p.ID).Scan(&count)
			if err != nil {
				count = 0
			}
			plansWithCounts[i] = templates.PlanWithCount{
				BudgetPlan:       p,
				TransactionCount: count,
			}
		}
		w.Header().Set("Content-Type", "text/html")
		templates.RenderBudgetPlansList(w, plansWithCounts)
	} else {
		http.Redirect(w, r, "/budget-plans/", http.StatusSeeOther)
	}
}

func (s *Server) updateBudgetPlanHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	planID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid budget plan ID", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}
	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}
	err = s.store.UpdateBudgetPlan(account.ID, planID, name)
	if err != nil {
		http.Error(w, "Failed to update budget plan", http.StatusInternalServerError)
		return
	}
	if isHTMX(r) {
		// Return updated budget plans list
		plans, err := s.store.GetBudgetPlansByAccount(account.ID)
		if err != nil {
			http.Error(w, "Failed to load budget plans", http.StatusInternalServerError)
			return
		}
		plansWithCounts := make([]templates.PlanWithCount, len(plans))
		for i, p := range plans {
			var count int
			query := `SELECT COUNT(*) FROM recurring_transactions WHERE budget_plan_id = $1`
			err := s.store.GetDB().QueryRow(query, p.ID).Scan(&count)
			if err != nil {
				count = 0
			}
			plansWithCounts[i] = templates.PlanWithCount{
				BudgetPlan:       p,
				TransactionCount: count,
			}
		}
		w.Header().Set("Content-Type", "text/html")
		templates.RenderBudgetPlansList(w, plansWithCounts)
	} else {
		http.Redirect(w, r, "/budget-plans/", http.StatusSeeOther)
	}
}

func (s *Server) deleteBudgetPlanHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	planID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid budget plan ID", http.StatusBadRequest)
		return
	}
	err = s.store.DeleteBudgetPlan(account.ID, planID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if isHTMX(r) {
		// Return updated budget plans list
		plans, err := s.store.GetBudgetPlansByAccount(account.ID)
		if err != nil {
			http.Error(w, "Failed to load budget plans", http.StatusInternalServerError)
			return
		}
		plansWithCounts := make([]templates.PlanWithCount, len(plans))
		for i, p := range plans {
			var count int
			query := `SELECT COUNT(*) FROM recurring_transactions WHERE budget_plan_id = $1`
			err := s.store.GetDB().QueryRow(query, p.ID).Scan(&count)
			if err != nil {
				count = 0
			}
			plansWithCounts[i] = templates.PlanWithCount{
				BudgetPlan:       p,
				TransactionCount: count,
			}
		}
		w.Header().Set("Content-Type", "text/html")
		templates.RenderBudgetPlansList(w, plansWithCounts)
	} else {
		http.Redirect(w, r, "/budget-plans/", http.StatusSeeOther)
	}
}

func (s *Server) activateBudgetPlanHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	planID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid budget plan ID", http.StatusBadRequest)
		return
	}
	err = s.store.SetActiveBudgetPlan(account.ID, planID)
	if err != nil {
		http.Error(w, "Failed to activate budget plan", http.StatusInternalServerError)
		return
	}
	// Update cookie to reflect new active plan
	http.SetCookie(w, &http.Cookie{
		Name:     "selected_budget_plan_id",
		Value:    idStr,
		Path:     "/",
		HttpOnly: false,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(30 * 24 * time.Hour.Seconds()),
	})
	if isHTMX(r) {
		// Return updated budget plans list
		plans, err := s.store.GetBudgetPlansByAccount(account.ID)
		if err != nil {
			http.Error(w, "Failed to load budget plans", http.StatusInternalServerError)
			return
		}
		plansWithCounts := make([]templates.PlanWithCount, len(plans))
		for i, p := range plans {
			var count int
			query := `SELECT COUNT(*) FROM recurring_transactions WHERE budget_plan_id = $1`
			err := s.store.GetDB().QueryRow(query, p.ID).Scan(&count)
			if err != nil {
				count = 0
			}
			plansWithCounts[i] = templates.PlanWithCount{
				BudgetPlan:       p,
				TransactionCount: count,
			}
		}
		w.Header().Set("Content-Type", "text/html")
		templates.RenderBudgetPlansList(w, plansWithCounts)
	} else {
		http.Redirect(w, r, "/budget-plans/", http.StatusSeeOther)
	}
}

func (s *Server) copyBudgetPlanHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	sourcePlanID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid budget plan ID", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}
	newName := r.FormValue("name")
	if newName == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}
	_, err = s.store.CopyBudgetPlan(account.ID, sourcePlanID, newName)
	if err != nil {
		http.Error(w, "Failed to copy budget plan", http.StatusInternalServerError)
		return
	}
	if isHTMX(r) {
		// Return updated budget plans list
		plans, err := s.store.GetBudgetPlansByAccount(account.ID)
		if err != nil {
			http.Error(w, "Failed to load budget plans", http.StatusInternalServerError)
			return
		}
		plansWithCounts := make([]templates.PlanWithCount, len(plans))
		for i, p := range plans {
			var count int
			query := `SELECT COUNT(*) FROM recurring_transactions WHERE budget_plan_id = $1`
			err := s.store.GetDB().QueryRow(query, p.ID).Scan(&count)
			if err != nil {
				count = 0
			}
			plansWithCounts[i] = templates.PlanWithCount{
				BudgetPlan:       p,
				TransactionCount: count,
			}
		}
		w.Header().Set("Content-Type", "text/html")
		templates.RenderBudgetPlansList(w, plansWithCounts)
	} else {
		http.Redirect(w, r, "/budget-plans/", http.StatusSeeOther)
	}
}

// Budget management handlers

func (s *Server) budgetsPageHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	// Check if budget plan is required
	if !s.requireBudgetPlan(w, r, account.ID) {
		return
	}

	budgetPlanID, ok := r.Context().Value("budget_plan_id").(int)
	if !ok || budgetPlanID == 0 {
		http.Redirect(w, r, "/budget-plans", http.StatusSeeOther)
		return
	}

	// Parse year and month from query parameters (default to current month)
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			year = y
		}
	}
	if monthStr := r.URL.Query().Get("month"); monthStr != "" {
		if m, err := strconv.Atoi(monthStr); err == nil && m >= 1 && m <= 12 {
			month = m
		}
	}

	// Get budgets with summaries
	budgetSummaries, err := s.store.GetBudgetSummaries(account.ID, budgetPlanID, year, month)
	if err != nil {
		log.Printf("Error fetching budgets: %v", err)
		budgetSummaries = []data.BudgetSummary{}
	}

	// Get categories for the form
	categories, err := s.store.GetCategoriesByAccount(account.ID)
	if err != nil {
		log.Printf("Error fetching categories: %v", err)
		categories = []data.Category{}
	}

	// Calculate yearly income for display
	yearlyIncome, err := s.store.CalculateExpectedYearlyIncome(account.ID, budgetPlanID)
	if err != nil {
		log.Printf("Error calculating yearly income: %v", err)
		yearlyIncome = 0
	}

	// Get budget plans for sidebar selector
	budgetPlans, err := s.store.GetBudgetPlansByAccount(account.ID)
	if err != nil {
		log.Printf("Error fetching budget plans: %v", err)
		budgetPlans = []data.BudgetPlan{}
	}

	page := templates.BudgetsPage(budgetSummaries, categories, yearlyIncome, year, month)
	_ = templates.BaseLayoutWithAuthAndBudgetPlan("Budgets - Budget App", true, budgetPlans, budgetPlanID, page).Render(w)
}

func (s *Server) createBudgetHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if budget plan is required
	if !s.requireBudgetPlan(w, r, account.ID) {
		return
	}

	budgetPlanID, ok := r.Context().Value("budget_plan_id").(int)
	if !ok || budgetPlanID == 0 {
		http.Error(w, "Budget plan required", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	categoryIDStr := r.FormValue("category_id")
	amountType := r.FormValue("amount_type")
	amountStr := r.FormValue("amount")

	if amountType == "" || amountStr == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	var categoryID *int
	if categoryIDStr != "" && categoryIDStr != "0" {
		id, err := strconv.Atoi(categoryIDStr)
		if err == nil {
			categoryID = &id
		}
	}

	var amount int
	if amountType == "fixed" {
		// Parse as dollar amount (e.g., "100.50" -> 10050 cents)
		parsed, err := parseAmountToCents(amountStr)
		if err != nil {
			http.Error(w, "Invalid amount format", http.StatusBadRequest)
			return
		}
		amount = parsed
	} else if amountType == "percentage" {
		// Parse as percentage (e.g., "10.5" -> 1050 for 10.5%)
		percentage, err := strconv.ParseFloat(amountStr, 64)
		if err != nil || percentage < 0 || percentage > 100 {
			http.Error(w, "Invalid percentage (must be 0-100)", http.StatusBadRequest)
			return
		}
		// Store as percentage * 100 (e.g., 10.5% = 1050)
		amount = int(percentage * 100)
	} else {
		http.Error(w, "Invalid amount type", http.StatusBadRequest)
		return
	}

	_, err := s.store.CreateBudget(account.ID, budgetPlanID, categoryID, amountType, amount)
	if err != nil {
		log.Printf("Error creating budget: %v", err)
		http.Error(w, "Failed to create budget", http.StatusInternalServerError)
		return
	}

	// Parse year and month for redirect
	now := time.Now()
	year := now.Year()
	month := int(now.Month())
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			year = y
		}
	}
	if monthStr := r.URL.Query().Get("month"); monthStr != "" {
		if m, err := strconv.Atoi(monthStr); err == nil && m >= 1 && m <= 12 {
			month = m
		}
	}

	if isHTMX(r) {
		// Return updated budgets list
		budgetSummaries, err := s.store.GetBudgetSummaries(account.ID, budgetPlanID, year, month)
		if err != nil {
			http.Error(w, "Failed to load budgets", http.StatusInternalServerError)
			return
		}
		yearlyIncome, _ := s.store.CalculateExpectedYearlyIncome(account.ID, budgetPlanID)
		w.Header().Set("Content-Type", "text/html")
		templates.BudgetTable(budgetSummaries, yearlyIncome).Render(w)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/budgets?year=%d&month=%d", year, month), http.StatusSeeOther)
	}
}

func (s *Server) updateBudgetHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	budgetID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid budget ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	categoryIDStr := r.FormValue("category_id")
	amountType := r.FormValue("amount_type")
	amountStr := r.FormValue("amount")

	if amountType == "" || amountStr == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	var categoryID *int
	if categoryIDStr != "" && categoryIDStr != "0" {
		id, err := strconv.Atoi(categoryIDStr)
		if err == nil {
			categoryID = &id
		}
	}

	var amount int
	if amountType == "fixed" {
		parsed, err := parseAmountToCents(amountStr)
		if err != nil {
			http.Error(w, "Invalid amount format", http.StatusBadRequest)
			return
		}
		amount = parsed
	} else if amountType == "percentage" {
		percentage, err := strconv.ParseFloat(amountStr, 64)
		if err != nil || percentage < 0 || percentage > 100 {
			http.Error(w, "Invalid percentage (must be 0-100)", http.StatusBadRequest)
			return
		}
		amount = int(percentage * 100)
	} else {
		http.Error(w, "Invalid amount type", http.StatusBadRequest)
		return
	}

	err = s.store.UpdateBudget(account.ID, budgetID, categoryID, amountType, amount)
	if err != nil {
		log.Printf("Error updating budget: %v", err)
		http.Error(w, "Failed to update budget", http.StatusInternalServerError)
		return
	}

	// Get budget plan ID for redirect
	budget, err := s.store.GetBudget(account.ID, budgetID)
	if err != nil || budget == nil {
		http.Error(w, "Budget not found", http.StatusNotFound)
		return
	}

	// Parse year and month for redirect
	now := time.Now()
	year := now.Year()
	month := int(now.Month())
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			year = y
		}
	}
	if monthStr := r.URL.Query().Get("month"); monthStr != "" {
		if m, err := strconv.Atoi(monthStr); err == nil && m >= 1 && m <= 12 {
			month = m
		}
	}

	if isHTMX(r) {
		// Return updated budgets list
		budgetSummaries, err := s.store.GetBudgetSummaries(account.ID, budget.BudgetPlanID, year, month)
		if err != nil {
			http.Error(w, "Failed to load budgets", http.StatusInternalServerError)
			return
		}
		yearlyIncome, _ := s.store.CalculateExpectedYearlyIncome(account.ID, budget.BudgetPlanID)
		w.Header().Set("Content-Type", "text/html")
		templates.BudgetTable(budgetSummaries, yearlyIncome).Render(w)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/budgets?year=%d&month=%d", year, month), http.StatusSeeOther)
	}
}

func (s *Server) deleteBudgetHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	budgetID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid budget ID", http.StatusBadRequest)
		return
	}

	// Get budget to get budget plan ID
	budget, err := s.store.GetBudget(account.ID, budgetID)
	if err != nil || budget == nil {
		http.Error(w, "Budget not found", http.StatusNotFound)
		return
	}

	err = s.store.DeleteBudget(account.ID, budgetID)
	if err != nil {
		log.Printf("Error deleting budget: %v", err)
		http.Error(w, "Failed to delete budget", http.StatusInternalServerError)
		return
	}

	// Parse year and month for redirect
	now := time.Now()
	year := now.Year()
	month := int(now.Month())
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			year = y
		}
	}
	if monthStr := r.URL.Query().Get("month"); monthStr != "" {
		if m, err := strconv.Atoi(monthStr); err == nil && m >= 1 && m <= 12 {
			month = m
		}
	}

	if isHTMX(r) {
		// Return updated budgets list
		budgetSummaries, err := s.store.GetBudgetSummaries(account.ID, budget.BudgetPlanID, year, month)
		if err != nil {
			http.Error(w, "Failed to load budgets", http.StatusInternalServerError)
			return
		}
		yearlyIncome, _ := s.store.CalculateExpectedYearlyIncome(account.ID, budget.BudgetPlanID)
		w.Header().Set("Content-Type", "text/html")
		templates.BudgetTable(budgetSummaries, yearlyIncome).Render(w)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/budgets?year=%d&month=%d", year, month), http.StatusSeeOther)
	}
}

func (s *Server) editBudgetHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	budgetID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid budget ID", http.StatusBadRequest)
		return
	}

	budget, err := s.store.GetBudget(account.ID, budgetID)
	if err != nil || budget == nil {
		http.Error(w, "Budget not found", http.StatusNotFound)
		return
	}

	categories, err := s.store.GetCategoriesByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}

	budgetPlanID, _ := r.Context().Value("budget_plan_id").(int)
	yearlyIncome, _ := s.store.CalculateExpectedYearlyIncome(account.ID, budgetPlanID)

	w.Header().Set("Content-Type", "text/html")
	templates.BudgetForm(budget, categories, yearlyIncome, true).Render(w)
}

func (s *Server) newBudgetHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	categories, err := s.store.GetCategoriesByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}

	budgetPlanID, _ := r.Context().Value("budget_plan_id").(int)
	yearlyIncome, _ := s.store.CalculateExpectedYearlyIncome(account.ID, budgetPlanID)

	w.Header().Set("Content-Type", "text/html")
	templates.BudgetForm(nil, categories, yearlyIncome, false).Render(w)
}

// Financial accounts management handlers

func (s *Server) financialAccountsPageHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}
	accounts, err := s.store.GetFinancialAccountsByAccount(account.ID)
	if err != nil {
		http.Error(w, "Failed to load financial accounts", http.StatusInternalServerError)
		return
	}
	page := templates.FinancialAccountsPage(accounts)
	_ = templates.BaseLayoutWithAuth("Financial Accounts - Budget App", true, page).Render(w)
}

func (s *Server) createFinancialAccountHandler(w http.ResponseWriter, r *http.Request) {
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
	accountType := r.FormValue("type")
	csvDateField := r.FormValue("csv_date_field")
	csvPayeeField := r.FormValue("csv_payee_field")
	csvExpenseField := r.FormValue("csv_expense_field")
	csvIncomeField := r.FormValue("csv_income_field")
	csvCategoryField := r.FormValue("csv_category_field")
	csvBalanceField := r.FormValue("csv_balance_field")

	if name == "" || accountType == "" || csvDateField == "" || csvPayeeField == "" || csvExpenseField == "" || csvIncomeField == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	var csvCategoryFieldPtr *string
	if csvCategoryField != "" {
		csvCategoryFieldPtr = &csvCategoryField
	}
	var csvBalanceFieldPtr *string
	if csvBalanceField != "" {
		csvBalanceFieldPtr = &csvBalanceField
	}

	_, err := s.store.CreateFinancialAccount(account.ID, name, accountType, csvDateField, csvPayeeField, csvExpenseField, csvIncomeField, csvCategoryFieldPtr, csvBalanceFieldPtr)
	if err != nil {
		http.Error(w, "Failed to create financial account", http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		// Return updated financial accounts list
		accounts, err := s.store.GetFinancialAccountsByAccount(account.ID)
		if err != nil {
			http.Error(w, "Failed to load financial accounts", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		templates.RenderFinancialAccountsList(w, accounts)
	} else {
		http.Redirect(w, r, "/financial-accounts/", http.StatusSeeOther)
	}
}

func (s *Server) updateFinancialAccountHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	financialAccountID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid financial account ID", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	accountType := r.FormValue("type")
	csvDateField := r.FormValue("csv_date_field")
	csvPayeeField := r.FormValue("csv_payee_field")
	csvExpenseField := r.FormValue("csv_expense_field")
	csvIncomeField := r.FormValue("csv_income_field")
	csvCategoryField := r.FormValue("csv_category_field")
	csvBalanceField := r.FormValue("csv_balance_field")
	balanceStr := r.FormValue("balance")

	if name == "" || accountType == "" || csvDateField == "" || csvPayeeField == "" || csvExpenseField == "" || csvIncomeField == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	var csvCategoryFieldPtr *string
	if csvCategoryField != "" {
		csvCategoryFieldPtr = &csvCategoryField
	}
	var csvBalanceFieldPtr *string
	if csvBalanceField != "" {
		csvBalanceFieldPtr = &csvBalanceField
	}
	var balancePtr *int
	if balanceStr != "" {
		balance, err := parseAmountToCents(balanceStr)
		if err == nil {
			balancePtr = &balance
		}
	}

	err = s.store.UpdateFinancialAccount(account.ID, financialAccountID, name, accountType, csvDateField, csvPayeeField, csvExpenseField, csvIncomeField, csvCategoryFieldPtr, csvBalanceFieldPtr, balancePtr)
	if err != nil {
		http.Error(w, "Failed to update financial account", http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		// Return updated financial accounts list
		accounts, err := s.store.GetFinancialAccountsByAccount(account.ID)
		if err != nil {
			http.Error(w, "Failed to load financial accounts", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		templates.RenderFinancialAccountsList(w, accounts)
	} else {
		http.Redirect(w, r, "/financial-accounts/", http.StatusSeeOther)
	}
}

func (s *Server) deleteFinancialAccountHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	financialAccountID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid financial account ID", http.StatusBadRequest)
		return
	}

	err = s.store.DeleteFinancialAccount(account.ID, financialAccountID)
	if err != nil {
		http.Error(w, "Failed to delete financial account", http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		// Return updated financial accounts list
		accounts, err := s.store.GetFinancialAccountsByAccount(account.ID)
		if err != nil {
			http.Error(w, "Failed to load financial accounts", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		templates.RenderFinancialAccountsList(w, accounts)
	} else {
		http.Redirect(w, r, "/financial-accounts/", http.StatusSeeOther)
	}
}

func (s *Server) editFinancialAccountHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := r.Context().Value("account").(*data.Account)
	if !ok || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	financialAccountID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid financial account ID", http.StatusBadRequest)
		return
	}

	// Get the financial account
	fa, err := s.store.GetFinancialAccount(account.ID, financialAccountID)
	if err != nil {
		http.Error(w, "Failed to load financial account", http.StatusInternalServerError)
		return
	}

	if fa == nil {
		http.Error(w, "Financial account not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html")

	// Render the edit form
	templates.FinancialAccountFormFields(fa, "edit").Render(w)
}
