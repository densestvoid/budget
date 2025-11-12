package data

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Account represents a user account
type Account struct {
	ID           int       `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Session represents a user session
type Session struct {
	ID        int       `json:"id"`
	AccountID int       `json:"account_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Category represents a budget category
// Supports nesting via ParentID and is owned by an Account
type Category struct {
	ID        int       `json:"id"`
	AccountID int       `json:"account_id"`
	Name      string    `json:"name"`
	ParentID  *int      `json:"parent_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Transaction represents a financial transaction
type Transaction struct {
	ID            int       `json:"id"`
	AccountID     int       `json:"account_id"`
	Date          time.Time `json:"date"`
	OriginalPayee string    `json:"original_payee"`
	Payee         string    `json:"payee"`
	CategoryID    *int      `json:"category_id"`
	Amount        int       `json:"amount"` // cents
	Reviewed      bool      `json:"reviewed"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// RuleCondition represents a condition for a transaction processing rule
type RuleCondition struct {
	ID       int    `json:"id"`
	RuleID   int    `json:"rule_id"`
	Field    string `json:"field"`    // 'payee' for now
	Operator string `json:"operator"` // 'equals', 'contains', 'begins', 'ends'
	Value    string `json:"value"`
}

// Rule represents a transaction processing rule
type Rule struct {
	ID         int             `json:"id"`
	AccountID  int             `json:"account_id"`
	Name       string          `json:"name"`
	NewPayee   *string         `json:"new_payee"`
	CategoryID *int            `json:"category_id"`
	Priority   int             `json:"priority"`
	Active     bool            `json:"active"`
	Conditions []RuleCondition `json:"conditions"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// RecurringTransaction represents a recurring expense or income
type RecurringTransaction struct {
	ID              int        `json:"id"`
	AccountID       int        `json:"account_id"`
	Name            string     `json:"name"`
	Counterparty    string     `json:"counterparty"` // payee for expenses, payer for income
	CategoryID      *int       `json:"category_id"`
	ExpectedAmount  int        `json:"expected_amount"` // cents, negative for expenses, positive for income
	Tolerance       int        `json:"tolerance"`       // cents
	StartDate       time.Time  `json:"start_date"`
	RecurrenceUnit  string     `json:"recurrence_unit"` // 'week', 'month', 'year'
	RecurrenceValue int        `json:"recurrence_value"`
	EndDate         *time.Time `json:"end_date"` // nullable, NULL = active
	Type            string     `json:"type"`     // 'expense' or 'income' (generated column)
	Archived        bool       `json:"archived"` // computed: end_date IS NOT NULL
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// AmountDecimal returns the amount as a decimal string (e.g., 1234 -> "12.34")
func (t Transaction) AmountDecimal() string {
	return fmt.Sprintf("%.2f", float64(t.Amount)/100.0)
}

// ExpectedAmountDecimal returns the expected amount as a decimal string (absolute value)
func (rt RecurringTransaction) ExpectedAmountDecimal() string {
	amount := rt.ExpectedAmount
	if amount < 0 {
		amount = -amount
	}
	return fmt.Sprintf("%.2f", float64(amount)/100.0)
}

// ToleranceDecimal returns the tolerance as a decimal string
func (rt RecurringTransaction) ToleranceDecimal() string {
	return fmt.Sprintf("%.2f", float64(rt.Tolerance)/100.0)
}

// RecurrenceDisplay returns a human-readable string for the recurrence
func (rt RecurringTransaction) RecurrenceDisplay() string {
	unit := rt.RecurrenceUnit
	if rt.RecurrenceValue > 1 {
		unit = unit + "s"
	}
	return fmt.Sprintf("Every %d %s starting %s", rt.RecurrenceValue, unit, rt.StartDate.Format("Jan 2, 2006"))
}

// IsExpense returns true if this is an expense
func (rt RecurringTransaction) IsExpense() bool {
	return rt.Type == "expense"
}

// IsIncome returns true if this is income
func (rt RecurringTransaction) IsIncome() bool {
	return rt.Type == "income"
}

// NextOccurrence calculates the next occurrence date after the given date
func (rt RecurringTransaction) NextOccurrence(after time.Time) time.Time {
	// If start_date is after 'after', return start_date
	if rt.StartDate.After(after) {
		// Check if archived
		if rt.EndDate != nil && rt.StartDate.After(*rt.EndDate) {
			return time.Time{}
		}
		return rt.StartDate
	}

	// Calculate how many periods have passed since start_date
	var periods int
	switch rt.RecurrenceUnit {
	case "week":
		daysDiff := int(after.Sub(rt.StartDate).Hours() / 24)
		weeksDiff := daysDiff / 7
		periods = (weeksDiff / rt.RecurrenceValue) + 1
	case "month":
		yearsDiff := after.Year() - rt.StartDate.Year()
		monthsDiff := yearsDiff*12 + int(after.Month()) - int(rt.StartDate.Month())
		periods = (monthsDiff / rt.RecurrenceValue) + 1
	case "year":
		yearsDiff := after.Year() - rt.StartDate.Year()
		periods = (yearsDiff / rt.RecurrenceValue) + 1
	}

	// Calculate next occurrence
	var next time.Time
	switch rt.RecurrenceUnit {
	case "week":
		next = rt.StartDate.AddDate(0, 0, periods*rt.RecurrenceValue*7)
	case "month":
		next = rt.StartDate.AddDate(0, periods*rt.RecurrenceValue, 0)
	case "year":
		next = rt.StartDate.AddDate(periods*rt.RecurrenceValue, 0, 0)
	}

	// Ensure next is after 'after'
	if !next.After(after) {
		// Add one more period
		switch rt.RecurrenceUnit {
		case "week":
			next = next.AddDate(0, 0, rt.RecurrenceValue*7)
		case "month":
			next = next.AddDate(0, rt.RecurrenceValue, 0)
		case "year":
			next = next.AddDate(rt.RecurrenceValue, 0, 0)
		}
	}

	// Check if recurring transaction is archived and next occurrence is after end_date
	if rt.EndDate != nil && next.After(*rt.EndDate) {
		// Recurring transaction is archived and this occurrence is after end_date
		// Return zero time to indicate no more occurrences
		return time.Time{}
	}

	return next
}

// Occurrence represents a single occurrence of a recurring transaction
type Occurrence struct {
	RecurringTransaction RecurringTransaction
	ExpectedDate         time.Time
	IsMatched            bool
	TransactionDate      *time.Time
}

// GetOccurrencesInMonth returns all occurrences of a recurring transaction within a given month
func (rt RecurringTransaction) GetOccurrencesInMonth(year int, month time.Month) []Occurrence {
	var occurrences []Occurrence
	
	// Calculate month boundaries
	monthStart := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	// Calculate last day of month: first day of next month minus 1 day
	nextMonth := monthStart.AddDate(0, 1, 0)
	lastDayOfMonth := nextMonth.AddDate(0, 0, -1)
	monthEnd := time.Date(year, month, lastDayOfMonth.Day(), 23, 59, 59, 999999999, time.UTC)
	
	// If archived and start_date is after end_date, no occurrences
	if rt.EndDate != nil && rt.StartDate.After(*rt.EndDate) {
		return occurrences
	}
	
	// Find the first occurrence on or after monthStart
	current := rt.StartDate
	
	// If start_date is before monthStart, calculate the first occurrence in the month
	if rt.StartDate.Before(monthStart) {
		var periods int
		switch rt.RecurrenceUnit {
		case "week":
			daysDiff := int(monthStart.Sub(rt.StartDate).Hours() / 24)
			weeksDiff := daysDiff / 7
			periods = weeksDiff / rt.RecurrenceValue
			current = rt.StartDate.AddDate(0, 0, periods*rt.RecurrenceValue*7)
			// If current is still before monthStart, add one more period
			for current.Before(monthStart) {
				current = current.AddDate(0, 0, rt.RecurrenceValue*7)
			}
		case "month":
			yearsDiff := monthStart.Year() - rt.StartDate.Year()
			monthsDiff := yearsDiff*12 + int(monthStart.Month()) - int(rt.StartDate.Month())
			periods = monthsDiff / rt.RecurrenceValue
			current = rt.StartDate.AddDate(0, periods*rt.RecurrenceValue, 0)
			// If current is still before monthStart, add one more period
			for current.Before(monthStart) {
				current = current.AddDate(0, rt.RecurrenceValue, 0)
			}
		case "year":
			yearsDiff := monthStart.Year() - rt.StartDate.Year()
			periods = yearsDiff / rt.RecurrenceValue
			current = rt.StartDate.AddDate(periods*rt.RecurrenceValue, 0, 0)
			// If current is still before monthStart, add one more period
			for current.Before(monthStart) {
				current = current.AddDate(rt.RecurrenceValue, 0, 0)
			}
		}
	}
	
	// Collect all occurrences in the month
	// Keep iterating until we've passed the end of the month
	for !current.After(monthEnd) {
		// Check if archived and this occurrence is after end_date
		if rt.EndDate != nil && current.After(*rt.EndDate) {
			break
		}
		
		// Only add if occurrence is within the month (on or after monthStart, on or before monthEnd)
		if !current.Before(monthStart) && !current.After(monthEnd) {
			// Add this occurrence
			occurrences = append(occurrences, Occurrence{
				RecurringTransaction: rt,
				ExpectedDate:         current,
			})
		}
		
		// Calculate next occurrence
		switch rt.RecurrenceUnit {
		case "week":
			current = current.AddDate(0, 0, rt.RecurrenceValue*7)
		case "month":
			current = current.AddDate(0, rt.RecurrenceValue, 0)
		case "year":
			current = current.AddDate(rt.RecurrenceValue, 0, 0)
		}
		
		// Safety check to prevent infinite loops
		if current.Before(rt.StartDate) {
			break
		}
	}
	
	return occurrences
}

// CreateAccount creates a new account with hashed password
func (s *Storage) CreateAccount(email, password, name string) (*Account, error) {
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Insert the account
	var account Account
	query := `
		INSERT INTO accounts (email, password_hash, name)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, name, created_at, updated_at
	`
	err = s.db.QueryRow(query, email, string(hashedPassword), name).Scan(
		&account.ID, &account.Email, &account.PasswordHash, &account.Name,
		&account.CreatedAt, &account.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	// Create initial categories for the new account
	initialCategories := []string{"bills", "necessities", "discretionary"}
	for _, catName := range initialCategories {
		_, _ = s.CreateCategory(account.ID, catName, nil)
	}

	return &account, nil
}

// GetAccountByEmail retrieves an account by email
func (s *Storage) GetAccountByEmail(email string) (*Account, error) {
	var account Account
	query := `
		SELECT id, email, password_hash, name, created_at, updated_at
		FROM accounts
		WHERE email = $1
	`
	err := s.db.QueryRow(query, email).Scan(
		&account.ID, &account.Email, &account.PasswordHash, &account.Name,
		&account.CreatedAt, &account.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return &account, nil
}

// AuthenticateAccount verifies email and password
func (s *Storage) AuthenticateAccount(email, password string) (*Account, error) {
	account, err := s.GetAccountByEmail(email)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	return account, nil
}

// CreateSession creates a new session for an account
func (s *Storage) CreateSession(accountID int, duration time.Duration) (*Session, error) {
	// Generate a random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	// Calculate expiration time
	expiresAt := time.Now().Add(duration)

	// Insert the session
	var session Session
	query := `
		INSERT INTO sessions (account_id, token, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, account_id, token, expires_at, created_at, updated_at
	`
	err := s.db.QueryRow(query, accountID, token, expiresAt).Scan(
		&session.ID, &session.AccountID, &session.Token, &session.ExpiresAt,
		&session.CreatedAt, &session.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &session, nil
}

// GetSessionByToken retrieves a session by token
func (s *Storage) GetSessionByToken(token string) (*Session, error) {
	var session Session
	query := `
		SELECT id, account_id, token, expires_at, created_at, updated_at
		FROM sessions
		WHERE token = $1 AND expires_at > NOW()
	`
	err := s.db.QueryRow(query, token).Scan(
		&session.ID, &session.AccountID, &session.Token, &session.ExpiresAt,
		&session.CreatedAt, &session.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return &session, nil
}

// GetAccountBySession retrieves an account by session token
func (s *Storage) GetAccountBySession(token string) (*Account, error) {
	session, err := s.GetSessionByToken(token)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil
	}

	// Get the account
	var account Account
	query := `
		SELECT id, email, password_hash, name, created_at, updated_at
		FROM accounts
		WHERE id = $1
	`
	err = s.db.QueryRow(query, session.AccountID).Scan(
		&account.ID, &account.Email, &account.PasswordHash, &account.Name,
		&account.CreatedAt, &account.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get account by session: %w", err)
	}

	return &account, nil
}

// DeleteSession deletes a session by token
func (s *Storage) DeleteSession(token string) error {
	query := `DELETE FROM sessions WHERE token = $1`
	_, err := s.db.Exec(query, token)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// CleanExpiredSessions removes expired sessions
func (s *Storage) CleanExpiredSessions() error {
	query := `DELETE FROM sessions WHERE expires_at <= NOW()`
	_, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to clean expired sessions: %w", err)
	}
	return nil
}

// CreateCategory creates a new category for an account
func (s *Storage) CreateCategory(accountID int, name string, parentID *int) (*Category, error) {
	var category Category
	query := `
		INSERT INTO categories (account_id, name, parent_id)
		VALUES ($1, $2, $3)
		RETURNING id, account_id, name, parent_id, created_at, updated_at
	`
	err := s.db.QueryRow(query, accountID, name, parentID).Scan(
		&category.ID, &category.AccountID, &category.Name, &category.ParentID,
		&category.CreatedAt, &category.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}
	return &category, nil
}

// GetCategoriesByAccount retrieves all categories for an account
func (s *Storage) GetCategoriesByAccount(accountID int) ([]Category, error) {
	query := `
		SELECT id, account_id, name, parent_id, created_at, updated_at
		FROM categories
		WHERE account_id = $1
		ORDER BY parent_id NULLS FIRST, name
	`
	rows, err := s.db.Query(query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.AccountID, &c.Name, &c.ParentID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, nil
}

// UpdateCategory updates a category's name or parent
func (s *Storage) UpdateCategory(accountID, categoryID int, name string, parentID *int) error {
	query := `
		UPDATE categories
		SET name = $1, parent_id = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3 AND account_id = $4
	`
	_, err := s.db.Exec(query, name, parentID, categoryID, accountID)
	if err != nil {
		return fmt.Errorf("failed to update category: %w", err)
	}
	return nil
}

// DeleteCategory deletes a category and moves its children to its parent
func (s *Storage) DeleteCategory(accountID, categoryID int) error {
	// First, get the category to find its parent
	var parentID *int
	query := `SELECT parent_id FROM categories WHERE id = $1 AND account_id = $2`
	err := s.db.QueryRow(query, categoryID, accountID).Scan(&parentID)
	if err != nil {
		return fmt.Errorf("failed to get category: %w", err)
	}

	// Update all children to point to the deleted category's parent
	updateQuery := `UPDATE categories SET parent_id = $1 WHERE parent_id = $2 AND account_id = $3`
	_, err = s.db.Exec(updateQuery, parentID, categoryID, accountID)
	if err != nil {
		return fmt.Errorf("failed to update child categories: %w", err)
	}

	// Now delete the category
	deleteQuery := `DELETE FROM categories WHERE id = $1 AND account_id = $2`
	_, err = s.db.Exec(deleteQuery, categoryID, accountID)
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}
	return nil
}

// CreateTransaction creates a new transaction
func (s *Storage) CreateTransaction(accountID int, date time.Time, originalPayee, payee string, categoryID *int, amount int) (*Transaction, error) {
	var t Transaction
	query := `
		INSERT INTO transactions (account_id, date, original_payee, payee, category_id, amount)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, account_id, date, original_payee, payee, category_id, amount, created_at, updated_at
	`
	err := s.db.QueryRow(query, accountID, date, originalPayee, payee, categoryID, amount).Scan(
		&t.ID, &t.AccountID, &t.Date, &t.OriginalPayee, &t.Payee, &t.CategoryID, &t.Amount, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// GetTransactionsByAccount retrieves all unreviewed transactions for an account
func (s *Storage) GetTransactionsByAccount(accountID int) ([]Transaction, error) {
	query := `
		SELECT id, account_id, date, original_payee, payee, category_id, amount, reviewed, created_at, updated_at
		FROM transactions
		WHERE account_id = $1 AND reviewed = FALSE
		ORDER BY date DESC, id DESC
	`
	rows, err := s.db.Query(query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.AccountID, &t.Date, &t.OriginalPayee, &t.Payee, &t.CategoryID, &t.Amount, &t.Reviewed, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, t)
	}
	return txs, nil
}

// GetAllTransactionsByAccount retrieves all transactions (both reviewed and unreviewed) for an account
func (s *Storage) GetAllTransactionsByAccount(accountID int) ([]Transaction, error) {
	query := `
		SELECT id, account_id, date, original_payee, payee, category_id, amount, reviewed, created_at, updated_at
		FROM transactions
		WHERE account_id = $1
		ORDER BY date DESC, id DESC
	`
	rows, err := s.db.Query(query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.AccountID, &t.Date, &t.OriginalPayee, &t.Payee, &t.CategoryID, &t.Amount, &t.Reviewed, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, t)
	}
	return txs, nil
}

// GetTransactionsByMonth retrieves all transactions for a specific month
func (s *Storage) GetTransactionsByMonth(accountID int, year int, month int) ([]Transaction, error) {
	query := `
		SELECT id, account_id, date, original_payee, payee, category_id, amount, reviewed, created_at, updated_at
		FROM transactions
		WHERE account_id = $1
		AND EXTRACT(YEAR FROM date) = $2
		AND EXTRACT(MONTH FROM date) = $3
		ORDER BY date DESC, id DESC
	`
	rows, err := s.db.Query(query, accountID, year, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.AccountID, &t.Date, &t.OriginalPayee, &t.Payee, &t.CategoryID, &t.Amount, &t.Reviewed, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, t)
	}
	return txs, nil
}

// UpdateTransactionPayeeCategory updates a transaction's payee and category
func (s *Storage) UpdateTransactionPayeeCategory(accountID, transactionID int, payee string, categoryID *int) error {
	query := `
		UPDATE transactions
		SET payee = $1, category_id = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3 AND account_id = $4
	`
	_, err := s.db.Exec(query, payee, categoryID, transactionID, accountID)
	return err
}

// BulkInsertTransactions inserts multiple transactions (for CSV upload)
func (s *Storage) BulkInsertTransactions(accountID int, txs []Transaction) error {
	for _, t := range txs {
		// Apply rules to the transaction
		modifiedTx, err := s.ApplyRulesToTransaction(accountID, t)
		if err != nil {
			return fmt.Errorf("failed to apply rules to transaction: %w", err)
		}

		payee := modifiedTx.Payee
		if payee == "" {
			payee = modifiedTx.OriginalPayee
		}
		// Create the transaction
		// Bill matching is now handled dynamically by the view, so no explicit matching needed
		if _, err = s.CreateTransaction(accountID, modifiedTx.Date, modifiedTx.OriginalPayee, payee, modifiedTx.CategoryID, modifiedTx.Amount); err != nil {
			return err
		}
	}
	return nil
}

// Add UpdateTransactionPayee and UpdateTransactionCategory methods
func (s *Storage) UpdateTransactionPayee(accountID, transactionID int, payee string) error {
	query := `UPDATE transactions SET payee = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 AND account_id = $3`
	_, err := s.db.Exec(query, payee, transactionID, accountID)
	return err
}

func (s *Storage) UpdateTransactionCategory(accountID, transactionID int, categoryID *int) error {
	query := `UPDATE transactions SET category_id = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 AND account_id = $3`
	_, err := s.db.Exec(query, categoryID, transactionID, accountID)
	return err
}

// MarkTransactionReviewed sets the reviewed flag to true for a transaction
func (s *Storage) MarkTransactionReviewed(accountID, transactionID int) error {
	query := `UPDATE transactions SET reviewed = TRUE, updated_at = CURRENT_TIMESTAMP WHERE id = $1 AND account_id = $2`
	_, err := s.db.Exec(query, transactionID, accountID)
	return err
}

// UpdateTransactionReviewed sets the reviewed flag to the given value for a transaction
func (s *Storage) UpdateTransactionReviewed(accountID, transactionID int, reviewed bool) error {
	query := `UPDATE transactions SET reviewed = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 AND account_id = $3`
	_, err := s.db.Exec(query, reviewed, transactionID, accountID)
	return err
}

// CategoryOption represents a category option in a dropdown with hierarchy info
type CategoryOption struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Level    int    `json:"level"`
	Disabled bool   `json:"disabled"`
}

// GetCategoryHierarchyOptions returns categories formatted for dropdown with hierarchy and cyclic reference prevention
func (s *Storage) GetCategoryHierarchyOptions(accountID, excludeID int) ([]CategoryOption, error) {
	categories, err := s.GetCategoriesByAccount(accountID)
	if err != nil {
		return nil, err
	}

	// Build a map for quick lookups
	catMap := make(map[int]*Category)
	for i := range categories {
		catMap[categories[i].ID] = &categories[i]
	}

	// Get descendants of the excluded category (to prevent cyclic references)
	var descendants map[int]bool
	if excludeID > 0 {
		descendants = s.GetDescendants(catMap, excludeID)
	}

	var options []CategoryOption
	for _, cat := range categories {
		if cat.ID == excludeID {
			continue // Skip the category being edited
		}

		// Check if this category is a descendant (would create cyclic reference)
		isDescendant := descendants != nil && descendants[cat.ID]

		// Calculate hierarchy level
		level := 0
		current := &cat
		for current.ParentID != nil {
			level++
			if parent, exists := catMap[*current.ParentID]; exists {
				current = parent
			} else {
				break
			}
		}

		options = append(options, CategoryOption{
			ID:       cat.ID,
			Name:     cat.Name,
			Level:    level,
			Disabled: isDescendant,
		})
	}

	// Sort by hierarchy level and then by name
	// This will group top-level categories first, then their children, etc.
	return options, nil
}

// GetDescendants returns a set of all descendant category IDs (including the category itself)
func (s *Storage) GetDescendants(catMap map[int]*Category, categoryID int) map[int]bool {
	descendants := make(map[int]bool)
	descendants[categoryID] = true

	// Use a queue to traverse all descendants
	queue := []int{categoryID}
	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]

		// Find all children of the current category
		for _, cat := range catMap {
			if cat.ParentID != nil && *cat.ParentID == currentID {
				if !descendants[cat.ID] {
					descendants[cat.ID] = true
					queue = append(queue, cat.ID)
				}
			}
		}
	}

	return descendants
}

// Rule management methods

// CreateRule creates a new rule with its conditions
func (s *Storage) CreateRule(accountID int, name string, newPayee *string, categoryID *int, priority int, conditions []RuleCondition) (*Rule, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert the rule
	var rule Rule
	query := `
		INSERT INTO rules (account_id, name, new_payee, category_id, priority)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, account_id, name, new_payee, category_id, priority, active, created_at, updated_at
	`
	err = tx.QueryRow(query, accountID, name, newPayee, categoryID, priority).Scan(
		&rule.ID, &rule.AccountID, &rule.Name, &rule.NewPayee, &rule.CategoryID, &rule.Priority, &rule.Active,
		&rule.CreatedAt, &rule.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create rule: %w", err)
	}

	// Insert conditions
	for _, condition := range conditions {
		conditionQuery := `
			INSERT INTO rule_conditions (rule_id, field, operator, value)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`
		var conditionID int
		err = tx.QueryRow(conditionQuery, rule.ID, condition.Field, condition.Operator, condition.Value).Scan(&conditionID)
		if err != nil {
			return nil, fmt.Errorf("failed to create rule condition: %w", err)
		}
		condition.ID = conditionID
		condition.RuleID = rule.ID
		rule.Conditions = append(rule.Conditions, condition)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &rule, nil
}

// GetRulesByAccount retrieves all rules for an account
func (s *Storage) GetRulesByAccount(accountID int) ([]Rule, error) {
	query := `
		SELECT id, account_id, name, new_payee, category_id, priority, active, created_at, updated_at
		FROM rules
		WHERE account_id = $1
		ORDER BY priority DESC, name ASC
	`
	rows, err := s.db.Query(query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules: %w", err)
	}
	defer rows.Close()

	var rules []Rule
	for rows.Next() {
		var rule Rule
		err := rows.Scan(&rule.ID, &rule.AccountID, &rule.Name, &rule.NewPayee, &rule.CategoryID, &rule.Priority, &rule.Active, &rule.CreatedAt, &rule.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}
		rules = append(rules, rule)
	}

	// Load conditions for each rule
	for i := range rules {
		conditions, err := s.GetRuleConditions(rules[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get conditions for rule %d: %w", rules[i].ID, err)
		}
		rules[i].Conditions = conditions
	}

	return rules, nil
}

// GetRuleConditions retrieves all conditions for a rule
func (s *Storage) GetRuleConditions(ruleID int) ([]RuleCondition, error) {
	query := `
		SELECT id, rule_id, field, operator, value
		FROM rule_conditions
		WHERE rule_id = $1
		ORDER BY id ASC
	`
	rows, err := s.db.Query(query, ruleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule conditions: %w", err)
	}
	defer rows.Close()

	var conditions []RuleCondition
	for rows.Next() {
		var condition RuleCondition
		err := rows.Scan(&condition.ID, &condition.RuleID, &condition.Field, &condition.Operator, &condition.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rule condition: %w", err)
		}
		conditions = append(conditions, condition)
	}

	return conditions, nil
}

// UpdateRule updates a rule and its conditions
func (s *Storage) UpdateRule(accountID, ruleID int, name string, newPayee *string, categoryID *int, priority int, conditions []RuleCondition) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update the rule
	query := `
		UPDATE rules
		SET name = $1, new_payee = $2, category_id = $3, priority = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5 AND account_id = $6
	`
	result, err := tx.Exec(query, name, newPayee, categoryID, priority, ruleID, accountID)
	if err != nil {
		return fmt.Errorf("failed to update rule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("rule not found or not owned by account")
	}

	// Delete existing conditions
	_, err = tx.Exec("DELETE FROM rule_conditions WHERE rule_id = $1", ruleID)
	if err != nil {
		return fmt.Errorf("failed to delete existing conditions: %w", err)
	}

	// Insert new conditions
	for _, condition := range conditions {
		conditionQuery := `
			INSERT INTO rule_conditions (rule_id, field, operator, value)
			VALUES ($1, $2, $3, $4)
		`
		_, err = tx.Exec(conditionQuery, ruleID, condition.Field, condition.Operator, condition.Value)
		if err != nil {
			return fmt.Errorf("failed to create rule condition: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DeleteRule deletes a rule and its conditions
func (s *Storage) DeleteRule(accountID, ruleID int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete conditions first (due to foreign key constraint)
	_, err = tx.Exec("DELETE FROM rule_conditions WHERE rule_id = $1", ruleID)
	if err != nil {
		return fmt.Errorf("failed to delete rule conditions: %w", err)
	}

	// Delete the rule
	result, err := tx.Exec("DELETE FROM rules WHERE id = $1 AND account_id = $2", ruleID, accountID)
	if err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("rule not found or not owned by account")
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ToggleRuleActive toggles the active status of a rule
func (s *Storage) ToggleRuleActive(accountID, ruleID int) error {
	query := `
		UPDATE rules
		SET active = NOT active, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND account_id = $2
	`
	result, err := s.db.Exec(query, ruleID, accountID)
	if err != nil {
		return fmt.Errorf("failed to toggle rule active status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("rule not found or not owned by account")
	}

	return nil
}

// ApplyRulesToTransaction applies all active rules to a transaction and returns the modified transaction
func (s *Storage) ApplyRulesToTransaction(accountID int, tx Transaction) (*Transaction, error) {
	rules, err := s.GetRulesByAccount(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules: %w", err)
	}

	// Sort rules by priority (highest first)
	// Rules are already sorted by priority DESC from GetRulesByAccount

	// Apply rules in order
	for _, rule := range rules {
		if !rule.Active {
			continue
		}

		// Check if all conditions match
		allConditionsMatch := true
		for _, condition := range rule.Conditions {
			if !s.conditionMatches(condition, tx.OriginalPayee) {
				allConditionsMatch = false
				break
			}
		}

		if allConditionsMatch {
			// Apply the rule
			if rule.NewPayee != nil {
				tx.Payee = *rule.NewPayee
			}
			if rule.CategoryID != nil {
				tx.CategoryID = rule.CategoryID
			}
			// Only apply the first matching rule (highest priority)
			break
		}
	}

	return &tx, nil
}

// conditionMatches checks if a condition matches the given payee text
func (s *Storage) conditionMatches(condition RuleCondition, payeeText string) bool {
	switch condition.Operator {
	case "equals":
		return strings.EqualFold(payeeText, condition.Value)
	case "contains":
		return strings.Contains(strings.ToLower(payeeText), strings.ToLower(condition.Value))
	case "begins":
		return strings.HasPrefix(strings.ToLower(payeeText), strings.ToLower(condition.Value))
	case "ends":
		return strings.HasSuffix(strings.ToLower(payeeText), strings.ToLower(condition.Value))
	default:
		return false
	}
}

// ApplyRuleToAllTransactions applies a single rule to all existing transactions for an account and returns the number updated
func (s *Storage) ApplyRuleToAllTransactions(accountID int, rule Rule) (int, error) {
	txs, err := s.GetAllTransactionsByAccount(accountID)
	if err != nil {
		return 0, fmt.Errorf("failed to get transactions: %w", err)
	}
	updated := 0
	for _, tx := range txs {
		// Check if all conditions match
		allMatch := true
		for _, cond := range rule.Conditions {
			if !s.conditionMatches(cond, tx.OriginalPayee) {
				allMatch = false
				break
			}
		}
		if allMatch {
			// Only update if something would change
			newPayee := tx.Payee
			if rule.NewPayee != nil {
				newPayee = *rule.NewPayee
			}
			newCategoryID := tx.CategoryID
			if rule.CategoryID != nil {
				newCategoryID = rule.CategoryID
			}
			if newPayee != tx.Payee || (rule.CategoryID != nil && (tx.CategoryID == nil || *tx.CategoryID != *rule.CategoryID)) {
				err := s.UpdateTransactionPayeeCategory(accountID, tx.ID, newPayee, newCategoryID)
				if err == nil {
					updated++
				}
			}
		}
	}
	return updated, nil
}

// RecurringTransaction management methods

// CreateRecurringTransaction creates a new recurring transaction (expense or income)
func (s *Storage) CreateRecurringTransaction(accountID int, name, counterparty string, categoryID *int, expectedAmount, tolerance int, startDate time.Time, recurrenceUnit string, recurrenceValue int) (*RecurringTransaction, error) {
	var rt RecurringTransaction
	var endDate sql.NullTime
	query := `
		INSERT INTO recurring_transactions (account_id, name, counterparty, category_id, expected_amount, tolerance, start_date, recurrence_unit, recurrence_value, end_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, account_id, name, counterparty, category_id, expected_amount, tolerance, start_date, recurrence_unit, recurrence_value, end_date, type, archived, created_at, updated_at
	`
	err := s.db.QueryRow(query, accountID, name, counterparty, categoryID, expectedAmount, tolerance, startDate, recurrenceUnit, recurrenceValue, nil).Scan(
		&rt.ID, &rt.AccountID, &rt.Name, &rt.Counterparty, &rt.CategoryID,
		&rt.ExpectedAmount, &rt.Tolerance, &rt.StartDate, &rt.RecurrenceUnit, &rt.RecurrenceValue,
		&endDate, &rt.Type, &rt.Archived, &rt.CreatedAt, &rt.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create recurring transaction: %w", err)
	}
	if endDate.Valid {
		rt.EndDate = &endDate.Time
	}

	return &rt, nil
}

// GetRecurringTransactionsByAccount retrieves all recurring transactions for an account
func (s *Storage) GetRecurringTransactionsByAccount(accountID int) ([]RecurringTransaction, error) {
	query := `
		SELECT id, account_id, name, counterparty, category_id, expected_amount, tolerance, start_date, recurrence_unit, recurrence_value, end_date, type, archived, created_at, updated_at
		FROM recurring_transactions
		WHERE account_id = $1
		ORDER BY name ASC
	`
	rows, err := s.db.Query(query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recurring transactions: %w", err)
	}
	defer rows.Close()

	var rts []RecurringTransaction
	for rows.Next() {
		var rt RecurringTransaction
		var endDate sql.NullTime
		if err := rows.Scan(&rt.ID, &rt.AccountID, &rt.Name, &rt.Counterparty, &rt.CategoryID,
			&rt.ExpectedAmount, &rt.Tolerance, &rt.StartDate, &rt.RecurrenceUnit, &rt.RecurrenceValue,
			&endDate, &rt.Type, &rt.Archived, &rt.CreatedAt, &rt.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan recurring transaction: %w", err)
		}
		if endDate.Valid {
			rt.EndDate = &endDate.Time
		}
		rts = append(rts, rt)
	}
	return rts, nil
}

// GetRecurringExpensesByAccount retrieves all recurring expenses for an account
func (s *Storage) GetRecurringExpensesByAccount(accountID int) ([]RecurringTransaction, error) {
	query := `
		SELECT id, account_id, name, counterparty, category_id, expected_amount, tolerance, start_date, recurrence_unit, recurrence_value, end_date, type, archived, created_at, updated_at
		FROM recurring_transactions
		WHERE account_id = $1 AND type = 'expense'
		ORDER BY name ASC
	`
	rows, err := s.db.Query(query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recurring expenses: %w", err)
	}
	defer rows.Close()

	var rts []RecurringTransaction
	for rows.Next() {
		var rt RecurringTransaction
		var endDate sql.NullTime
		if err := rows.Scan(&rt.ID, &rt.AccountID, &rt.Name, &rt.Counterparty, &rt.CategoryID,
			&rt.ExpectedAmount, &rt.Tolerance, &rt.StartDate, &rt.RecurrenceUnit, &rt.RecurrenceValue,
			&endDate, &rt.Type, &rt.Archived, &rt.CreatedAt, &rt.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan recurring expense: %w", err)
		}
		if endDate.Valid {
			rt.EndDate = &endDate.Time
		}
		rts = append(rts, rt)
	}
	return rts, nil
}

// GetRecurringIncomeByAccount retrieves all recurring income for an account
func (s *Storage) GetRecurringIncomeByAccount(accountID int) ([]RecurringTransaction, error) {
	query := `
		SELECT id, account_id, name, counterparty, category_id, expected_amount, tolerance, start_date, recurrence_unit, recurrence_value, end_date, type, archived, created_at, updated_at
		FROM recurring_transactions
		WHERE account_id = $1 AND type = 'income'
		ORDER BY name ASC
	`
	rows, err := s.db.Query(query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recurring income: %w", err)
	}
	defer rows.Close()

	var rts []RecurringTransaction
	for rows.Next() {
		var rt RecurringTransaction
		var endDate sql.NullTime
		if err := rows.Scan(&rt.ID, &rt.AccountID, &rt.Name, &rt.Counterparty, &rt.CategoryID,
			&rt.ExpectedAmount, &rt.Tolerance, &rt.StartDate, &rt.RecurrenceUnit, &rt.RecurrenceValue,
			&endDate, &rt.Type, &rt.Archived, &rt.CreatedAt, &rt.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan recurring income: %w", err)
		}
		if endDate.Valid {
			rt.EndDate = &endDate.Time
		}
		rts = append(rts, rt)
	}
	return rts, nil
}

// GetRecurringTransaction retrieves a single recurring transaction by ID
func (s *Storage) GetRecurringTransaction(accountID, rtID int) (*RecurringTransaction, error) {
	var rt RecurringTransaction
	var endDate sql.NullTime
	query := `
		SELECT id, account_id, name, counterparty, category_id, expected_amount, tolerance, start_date, recurrence_unit, recurrence_value, end_date, type, archived, created_at, updated_at
		FROM recurring_transactions
		WHERE id = $1 AND account_id = $2
	`
	err := s.db.QueryRow(query, rtID, accountID).Scan(
		&rt.ID, &rt.AccountID, &rt.Name, &rt.Counterparty, &rt.CategoryID,
		&rt.ExpectedAmount, &rt.Tolerance, &rt.StartDate, &rt.RecurrenceUnit, &rt.RecurrenceValue,
		&endDate, &rt.Type, &rt.Archived, &rt.CreatedAt, &rt.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get recurring transaction: %w", err)
	}
	if endDate.Valid {
		rt.EndDate = &endDate.Time
	}
	return &rt, nil
}

// UpdateRecurringTransaction updates a recurring transaction
func (s *Storage) UpdateRecurringTransaction(accountID, rtID int, name, counterparty string, categoryID *int, expectedAmount, tolerance int, startDate time.Time, recurrenceUnit string, recurrenceValue int) error {
	query := `
		UPDATE recurring_transactions
		SET name = $1, counterparty = $2, category_id = $3, expected_amount = $4, tolerance = $5, start_date = $6, recurrence_unit = $7, recurrence_value = $8, updated_at = CURRENT_TIMESTAMP
		WHERE id = $9 AND account_id = $10
	`
	result, err := s.db.Exec(query, name, counterparty, categoryID, expectedAmount, tolerance, startDate, recurrenceUnit, recurrenceValue, rtID, accountID)
	if err != nil {
		return fmt.Errorf("failed to update recurring transaction: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("recurring transaction not found or not owned by account")
	}
	return nil
}

// ArchiveRecurringTransaction archives a recurring transaction by setting its end_date
func (s *Storage) ArchiveRecurringTransaction(accountID, rtID int, endDate time.Time) error {
	query := `
		UPDATE recurring_transactions
		SET end_date = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND account_id = $3
	`
	result, err := s.db.Exec(query, endDate, rtID, accountID)
	if err != nil {
		return fmt.Errorf("failed to archive recurring transaction: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("recurring transaction not found or not owned by account")
	}
	return nil
}

// DeleteRecurringTransaction deletes a recurring transaction
func (s *Storage) DeleteRecurringTransaction(accountID, rtID int) error {
	result, err := s.db.Exec("DELETE FROM recurring_transactions WHERE id = $1 AND account_id = $2", rtID, accountID)
	if err != nil {
		return fmt.Errorf("failed to delete recurring transaction: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("recurring transaction not found or not owned by account")
	}
	return nil
}

// GetMatchedTransactionsForRecurring returns all transaction dates matched to a recurring transaction in a given month
func (s *Storage) GetMatchedTransactionsForRecurring(rtID int, year, month int) ([]time.Time, error) {
	query := `
		SELECT transaction_date
		FROM recurring_transactions_view
		WHERE recurring_transaction_id = $1
		AND year = $2
		AND month = $3
		ORDER BY transaction_date
	`
	
	rows, err := s.db.Query(query, rtID, year, month)
	if err != nil {
		return nil, fmt.Errorf("failed to get matched transactions: %w", err)
	}
	defer rows.Close()
	
	var transactions []time.Time
	for rows.Next() {
		var txDate time.Time
		if err := rows.Scan(&txDate); err != nil {
			continue
		}
		transactions = append(transactions, txDate)
	}
	
	return transactions, nil
}

// IsRecurringTransactionMatchedForMonth checks if a recurring transaction has a matching transaction for a given month/year using the view
func (s *Storage) IsRecurringTransactionMatchedForMonth(rtID int, year, month int) (bool, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM recurring_transactions_view
		WHERE recurring_transaction_id = $1 AND year = $2 AND month = $3
	`
	err := s.db.QueryRow(query, rtID, year, month).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if recurring transaction is matched: %w", err)
	}
	return count > 0, nil
}

// GetRecurringTransactionDate gets the transaction date for a matched recurring transaction in a given month/year
func (s *Storage) GetRecurringTransactionDate(rtID int, year, month int) (*time.Time, error) {
	var transactionDate time.Time
	query := `
		SELECT transaction_date
		FROM recurring_transactions_view
		WHERE recurring_transaction_id = $1 AND year = $2 AND month = $3
		LIMIT 1
	`
	err := s.db.QueryRow(query, rtID, year, month).Scan(&transactionDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get recurring transaction date: %w", err)
	}
	return &transactionDate, nil
}

// MatchRecurringTransactionsToTransaction checks if a transaction matches any recurring transactions for an account
// Returns the matched recurring transaction if found, nil otherwise
// Note: This is now read-only - matches are determined dynamically by the view
func (s *Storage) MatchRecurringTransactionsToTransaction(accountID int, tx *Transaction) (*RecurringTransaction, error) {
	// Check the view to see if this transaction matches any recurring transaction
	var rtID int
	year := tx.Date.Year()
	month := int(tx.Date.Month())

	query := `
		SELECT recurring_transaction_id
		FROM recurring_transactions_view
		WHERE transaction_id = $1 AND year = $2 AND month = $3
		LIMIT 1
	`
	err := s.db.QueryRow(query, tx.ID, year, month).Scan(&rtID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No match found
		}
		return nil, fmt.Errorf("failed to check recurring transaction match: %w", err)
	}

	// Get the matched recurring transaction
	rt, err := s.GetRecurringTransaction(accountID, rtID)
	if err != nil {
		return nil, fmt.Errorf("failed to get matched recurring transaction: %w", err)
	}
	return rt, nil
}
