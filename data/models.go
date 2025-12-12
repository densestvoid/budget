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

// FinancialAccount represents a financial account (checking, savings, credit card)
type FinancialAccount struct {
	ID              int       `json:"id"`
	AccountID       int       `json:"account_id"` // user account
	Name            string    `json:"name"`
	Type            string    `json:"type"` // 'checking', 'savings', 'credit_card'
	Balance         int       `json:"balance"` // cents
	CSVDateField    string    `json:"csv_date_field"`
	CSVPayeeField   string    `json:"csv_payee_field"`
	CSVExpenseField string    `json:"csv_expense_field"`
	CSVIncomeField  string    `json:"csv_income_field"`
	CSVCategoryField *string  `json:"csv_category_field,omitempty"`
	CSVBalanceField  *string  `json:"csv_balance_field,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Transaction represents a financial transaction
type Transaction struct {
	ID                int       `json:"id"`
	AccountID         int       `json:"account_id"`
	FinancialAccountID int      `json:"financial_account_id"`
	Date              time.Time `json:"date"`
	OriginalPayee     string    `json:"original_payee"`
	Payee             string    `json:"payee"`
	CategoryID        *int      `json:"category_id"`
	Amount            int       `json:"amount"` // cents
	Reviewed          bool      `json:"reviewed"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
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

// BudgetPlan represents a budgeting plan
type BudgetPlan struct {
	ID        int       `json:"id"`
	AccountID int       `json:"account_id"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Budget represents a budget for a category within a budget plan
type Budget struct {
	ID           int       `json:"id"`
	AccountID    int       `json:"account_id"`
	BudgetPlanID int       `json:"budget_plan_id"`
	CategoryID   *int      `json:"category_id"`
	AmountType   string    `json:"amount_type"` // 'fixed' or 'percentage'
	Amount       int       `json:"amount"`      // cents if fixed, percentage (0-10000 for 0.00-100.00%) if percentage
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// BudgetSummary represents a budget with calculated spent and expected amounts
type BudgetSummary struct {
	Budget         Budget  `json:"budget"`
	CategoryName   string  `json:"category_name"`
	MonthlyAmount  int     `json:"monthly_amount"`  // calculated monthly budget amount in cents
	SpentAmount    int     `json:"spent_amount"`    // actual spent in cents
	ExpectedAmount int     `json:"expected_amount"` // expected (not yet matched) recurring transactions in cents
	Remaining      int     `json:"remaining"`       // monthly_amount - spent_amount - expected_amount
}

// RecurringTransaction represents a recurring expense or income
type RecurringTransaction struct {
	ID                int        `json:"id"`
	AccountID         int        `json:"account_id"`
	BudgetPlanID      int        `json:"budget_plan_id"`
	FinancialAccountID int        `json:"financial_account_id"`
	Name              string     `json:"name"`
	Counterparty      string     `json:"counterparty"` // payee for expenses, payer for income
	CategoryID        *int       `json:"category_id"`
	ExpectedAmount    int        `json:"expected_amount"` // cents, negative for expenses, positive for income
	Tolerance         int        `json:"tolerance"`       // cents
	StartDate         time.Time  `json:"start_date"`
	RecurrenceUnit    string     `json:"recurrence_unit"` // 'week', 'month', 'year'
	RecurrenceValue   int        `json:"recurrence_value"`
	EndDate           *time.Time `json:"end_date"` // nullable, NULL = active
	Type              string     `json:"type"`     // 'expense' or 'income' (generated column)
	Archived          bool       `json:"archived"` // computed: end_date IS NOT NULL
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// BalanceDecimal returns the balance as a decimal string (e.g., 1234 -> "12.34")
func (fa FinancialAccount) BalanceDecimal() string {
	return fmt.Sprintf("%.2f", float64(fa.Balance)/100.0)
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

// MonthlyAmount calculates the monthly budget amount based on yearly income
func (b Budget) MonthlyAmount(yearlyIncome int) int {
	if b.AmountType == "percentage" {
		// Amount is stored as percentage * 100 (e.g., 10.5% = 1050)
		// Calculate: (yearlyIncome * percentage) / 12
		percentage := float64(b.Amount) / 100.0 // Convert to actual percentage (10.5)
		yearlyBudget := float64(yearlyIncome) * (percentage / 100.0)
		return int(yearlyBudget / 12.0)
	}
	// Fixed amount
	return b.Amount
}

// AmountDecimal returns the amount as a decimal string for display
func (b Budget) AmountDecimal() string {
	if b.AmountType == "percentage" {
		// Amount is stored as percentage * 100 (e.g., 10.5% = 1050)
		return fmt.Sprintf("%.2f%%", float64(b.Amount)/100.0)
	}
	return fmt.Sprintf("%.2f", float64(b.Amount)/100.0)
}

// MonthlyAmountDecimal returns the monthly amount as a decimal string
func (bs BudgetSummary) MonthlyAmountDecimal() string {
	return fmt.Sprintf("%.2f", float64(bs.MonthlyAmount)/100.0)
}

// SpentAmountDecimal returns the spent amount as a decimal string
func (bs BudgetSummary) SpentAmountDecimal() string {
	return fmt.Sprintf("%.2f", float64(bs.SpentAmount)/100.0)
}

// ExpectedAmountDecimal returns the expected amount as a decimal string
func (bs BudgetSummary) ExpectedAmountDecimal() string {
	return fmt.Sprintf("%.2f", float64(bs.ExpectedAmount)/100.0)
}

// RemainingDecimal returns the remaining amount as a decimal string
func (bs BudgetSummary) RemainingDecimal() string {
	return fmt.Sprintf("%.2f", float64(bs.Remaining)/100.0)
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

// FinancialAccount management methods

// CreateFinancialAccount creates a new financial account
func (s *Storage) CreateFinancialAccount(accountID int, name, accountType string, csvDateField, csvPayeeField, csvExpenseField, csvIncomeField string, csvCategoryField, csvBalanceField *string) (*FinancialAccount, error) {
	var fa FinancialAccount
	query := `
		INSERT INTO financial_accounts (account_id, name, type, csv_date_field, csv_payee_field, csv_expense_field, csv_income_field, csv_category_field, csv_balance_field)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, account_id, name, type, balance, csv_date_field, csv_payee_field, csv_expense_field, csv_income_field, csv_category_field, csv_balance_field, created_at, updated_at
	`
	err := s.db.QueryRow(query, accountID, name, accountType, csvDateField, csvPayeeField, csvExpenseField, csvIncomeField, csvCategoryField, csvBalanceField).Scan(
		&fa.ID, &fa.AccountID, &fa.Name, &fa.Type, &fa.Balance,
		&fa.CSVDateField, &fa.CSVPayeeField, &fa.CSVExpenseField, &fa.CSVIncomeField,
		&fa.CSVCategoryField, &fa.CSVBalanceField, &fa.CreatedAt, &fa.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create financial account: %w", err)
	}
	return &fa, nil
}

// GetFinancialAccountsByAccount retrieves all financial accounts for a user account
func (s *Storage) GetFinancialAccountsByAccount(accountID int) ([]FinancialAccount, error) {
	query := `
		SELECT id, account_id, name, type, balance, csv_date_field, csv_payee_field, csv_expense_field, csv_income_field, csv_category_field, csv_balance_field, created_at, updated_at
		FROM financial_accounts
		WHERE account_id = $1
		ORDER BY name ASC
	`
	rows, err := s.db.Query(query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get financial accounts: %w", err)
	}
	defer rows.Close()

	var accounts []FinancialAccount
	for rows.Next() {
		var fa FinancialAccount
		if err := rows.Scan(&fa.ID, &fa.AccountID, &fa.Name, &fa.Type, &fa.Balance,
			&fa.CSVDateField, &fa.CSVPayeeField, &fa.CSVExpenseField, &fa.CSVIncomeField,
			&fa.CSVCategoryField, &fa.CSVBalanceField, &fa.CreatedAt, &fa.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan financial account: %w", err)
		}
		accounts = append(accounts, fa)
	}
	return accounts, nil
}

// GetFinancialAccount retrieves a single financial account by ID
func (s *Storage) GetFinancialAccount(accountID, financialAccountID int) (*FinancialAccount, error) {
	var fa FinancialAccount
	query := `
		SELECT id, account_id, name, type, balance, csv_date_field, csv_payee_field, csv_expense_field, csv_income_field, csv_category_field, csv_balance_field, created_at, updated_at
		FROM financial_accounts
		WHERE id = $1 AND account_id = $2
	`
	err := s.db.QueryRow(query, financialAccountID, accountID).Scan(
		&fa.ID, &fa.AccountID, &fa.Name, &fa.Type, &fa.Balance,
		&fa.CSVDateField, &fa.CSVPayeeField, &fa.CSVExpenseField, &fa.CSVIncomeField,
		&fa.CSVCategoryField, &fa.CSVBalanceField, &fa.CreatedAt, &fa.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get financial account: %w", err)
	}
	return &fa, nil
}

// UpdateFinancialAccount updates a financial account
func (s *Storage) UpdateFinancialAccount(accountID, financialAccountID int, name, accountType string, csvDateField, csvPayeeField, csvExpenseField, csvIncomeField string, csvCategoryField, csvBalanceField *string, balance *int) error {
	query := `
		UPDATE financial_accounts
		SET name = $1, type = $2, csv_date_field = $3, csv_payee_field = $4, csv_expense_field = $5, csv_income_field = $6, csv_category_field = $7, csv_balance_field = $8, balance = COALESCE($9, balance), updated_at = CURRENT_TIMESTAMP
		WHERE id = $10 AND account_id = $11
	`
	result, err := s.db.Exec(query, name, accountType, csvDateField, csvPayeeField, csvExpenseField, csvIncomeField, csvCategoryField, csvBalanceField, balance, financialAccountID, accountID)
	if err != nil {
		return fmt.Errorf("failed to update financial account: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("financial account not found or not owned by account")
	}
	return nil
}

// UpdateFinancialAccountBalance updates the balance of a financial account
func (s *Storage) UpdateFinancialAccountBalance(accountID, financialAccountID int, balance int) error {
	query := `
		UPDATE financial_accounts
		SET balance = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND account_id = $3
	`
	result, err := s.db.Exec(query, balance, financialAccountID, accountID)
	if err != nil {
		return fmt.Errorf("failed to update financial account balance: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("financial account not found or not owned by account")
	}
	return nil
}

// DeleteFinancialAccount deletes a financial account
func (s *Storage) DeleteFinancialAccount(accountID, financialAccountID int) error {
	result, err := s.db.Exec("DELETE FROM financial_accounts WHERE id = $1 AND account_id = $2", financialAccountID, accountID)
	if err != nil {
		return fmt.Errorf("failed to delete financial account: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("financial account not found or not owned by account")
	}
	return nil
}

// CreateTransaction creates a new transaction
func (s *Storage) CreateTransaction(accountID, financialAccountID int, date time.Time, originalPayee, payee string, categoryID *int, amount int) (*Transaction, error) {
	var t Transaction
	query := `
		INSERT INTO transactions (account_id, financial_account_id, date, original_payee, payee, category_id, amount)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, account_id, financial_account_id, date, original_payee, payee, category_id, amount, reviewed, created_at, updated_at
	`
	err := s.db.QueryRow(query, accountID, financialAccountID, date, originalPayee, payee, categoryID, amount).Scan(
		&t.ID, &t.AccountID, &t.FinancialAccountID, &t.Date, &t.OriginalPayee, &t.Payee, &t.CategoryID, &t.Amount, &t.Reviewed, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// GetTransactionsByAccount retrieves all unreviewed transactions for an account
func (s *Storage) GetTransactionsByAccount(accountID int) ([]Transaction, error) {
	query := `
		SELECT id, account_id, financial_account_id, date, original_payee, payee, category_id, amount, reviewed, created_at, updated_at
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
		if err := rows.Scan(&t.ID, &t.AccountID, &t.FinancialAccountID, &t.Date, &t.OriginalPayee, &t.Payee, &t.CategoryID, &t.Amount, &t.Reviewed, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, t)
	}
	return txs, nil
}

// GetAllTransactionsByAccount retrieves all transactions (both reviewed and unreviewed) for an account
func (s *Storage) GetAllTransactionsByAccount(accountID int) ([]Transaction, error) {
	query := `
		SELECT id, account_id, financial_account_id, date, original_payee, payee, category_id, amount, reviewed, created_at, updated_at
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
		if err := rows.Scan(&t.ID, &t.AccountID, &t.FinancialAccountID, &t.Date, &t.OriginalPayee, &t.Payee, &t.CategoryID, &t.Amount, &t.Reviewed, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, t)
	}
	return txs, nil
}

// GetTransactionsByMonth retrieves all transactions for a specific month
func (s *Storage) GetTransactionsByMonth(accountID int, year int, month int) ([]Transaction, error) {
	query := `
		SELECT id, account_id, financial_account_id, date, original_payee, payee, category_id, amount, reviewed, created_at, updated_at
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
		if err := rows.Scan(&t.ID, &t.AccountID, &t.FinancialAccountID, &t.Date, &t.OriginalPayee, &t.Payee, &t.CategoryID, &t.Amount, &t.Reviewed, &t.CreatedAt, &t.UpdatedAt); err != nil {
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
func (s *Storage) BulkInsertTransactions(accountID, financialAccountID int, txs []Transaction) error {
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
		if _, err = s.CreateTransaction(accountID, financialAccountID, modifiedTx.Date, modifiedTx.OriginalPayee, payee, modifiedTx.CategoryID, modifiedTx.Amount); err != nil {
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

// BudgetPlan management methods

// CreateBudgetPlan creates a new budget plan
func (s *Storage) CreateBudgetPlan(accountID int, name string) (*BudgetPlan, error) {
	var plan BudgetPlan
	query := `
		INSERT INTO budget_plans (account_id, name)
		VALUES ($1, $2)
		RETURNING id, account_id, name, is_active, created_at, updated_at
	`
	err := s.db.QueryRow(query, accountID, name).Scan(
		&plan.ID, &plan.AccountID, &plan.Name, &plan.IsActive, &plan.CreatedAt, &plan.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create budget plan: %w", err)
	}
	return &plan, nil
}

// GetBudgetPlansByAccount retrieves all budget plans for an account
func (s *Storage) GetBudgetPlansByAccount(accountID int) ([]BudgetPlan, error) {
	query := `
		SELECT id, account_id, name, is_active, created_at, updated_at
		FROM budget_plans
		WHERE account_id = $1
		ORDER BY is_active DESC, name ASC
	`
	rows, err := s.db.Query(query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get budget plans: %w", err)
	}
	defer rows.Close()

	var plans []BudgetPlan
	for rows.Next() {
		var plan BudgetPlan
		if err := rows.Scan(&plan.ID, &plan.AccountID, &plan.Name, &plan.IsActive, &plan.CreatedAt, &plan.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan budget plan: %w", err)
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

// GetBudgetPlan retrieves a single budget plan by ID
func (s *Storage) GetBudgetPlan(accountID, planID int) (*BudgetPlan, error) {
	var plan BudgetPlan
	query := `
		SELECT id, account_id, name, is_active, created_at, updated_at
		FROM budget_plans
		WHERE id = $1 AND account_id = $2
	`
	err := s.db.QueryRow(query, planID, accountID).Scan(
		&plan.ID, &plan.AccountID, &plan.Name, &plan.IsActive, &plan.CreatedAt, &plan.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get budget plan: %w", err)
	}
	return &plan, nil
}

// GetActiveBudgetPlan retrieves the active budget plan for an account
func (s *Storage) GetActiveBudgetPlan(accountID int) (*BudgetPlan, error) {
	var plan BudgetPlan
	query := `
		SELECT id, account_id, name, is_active, created_at, updated_at
		FROM budget_plans
		WHERE account_id = $1 AND is_active = TRUE
		LIMIT 1
	`
	err := s.db.QueryRow(query, accountID).Scan(
		&plan.ID, &plan.AccountID, &plan.Name, &plan.IsActive, &plan.CreatedAt, &plan.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get active budget plan: %w", err)
	}
	return &plan, nil
}

// UpdateBudgetPlan updates a budget plan's name
func (s *Storage) UpdateBudgetPlan(accountID, planID int, name string) error {
	query := `
		UPDATE budget_plans
		SET name = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND account_id = $3
	`
	result, err := s.db.Exec(query, name, planID, accountID)
	if err != nil {
		return fmt.Errorf("failed to update budget plan: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("budget plan not found or not owned by account")
	}
	return nil
}

// SetActiveBudgetPlan sets a budget plan as active and deactivates all others for the account
func (s *Storage) SetActiveBudgetPlan(accountID, planID int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// First, deactivate all plans for this account
	_, err = tx.Exec(`
		UPDATE budget_plans
		SET is_active = FALSE, updated_at = CURRENT_TIMESTAMP
		WHERE account_id = $1
	`, accountID)
	if err != nil {
		return fmt.Errorf("failed to deactivate budget plans: %w", err)
	}

	// Then, activate the specified plan
	result, err := tx.Exec(`
		UPDATE budget_plans
		SET is_active = TRUE, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND account_id = $2
	`, planID, accountID)
	if err != nil {
		return fmt.Errorf("failed to activate budget plan: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("budget plan not found or not owned by account")
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DeleteBudgetPlan deletes a budget plan
func (s *Storage) DeleteBudgetPlan(accountID, planID int) error {
	// Check if plan is active - prevent deletion of active plan
	plan, err := s.GetBudgetPlan(accountID, planID)
	if err != nil {
		return fmt.Errorf("failed to get budget plan: %w", err)
	}
	if plan == nil {
		return fmt.Errorf("budget plan not found")
	}
	if plan.IsActive {
		return fmt.Errorf("cannot delete active budget plan")
	}

	// Check if plan has recurring transactions
	var count int
	query := `SELECT COUNT(*) FROM recurring_transactions WHERE budget_plan_id = $1`
	err = s.db.QueryRow(query, planID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check recurring transactions: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete budget plan with recurring transactions")
	}

	// Delete the plan
	result, err := s.db.Exec("DELETE FROM budget_plans WHERE id = $1 AND account_id = $2", planID, accountID)
	if err != nil {
		return fmt.Errorf("failed to delete budget plan: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("budget plan not found or not owned by account")
	}
	return nil
}

// CopyBudgetPlan copies a budget plan and all its recurring transactions
func (s *Storage) CopyBudgetPlan(accountID, sourcePlanID int, newName string) (*BudgetPlan, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Verify source plan exists and belongs to account
	var sourcePlan BudgetPlan
	query := `
		SELECT id, account_id, name, is_active, created_at, updated_at
		FROM budget_plans
		WHERE id = $1 AND account_id = $2
	`
	err = tx.QueryRow(query, sourcePlanID, accountID).Scan(
		&sourcePlan.ID, &sourcePlan.AccountID, &sourcePlan.Name, &sourcePlan.IsActive,
		&sourcePlan.CreatedAt, &sourcePlan.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("source budget plan not found")
		}
		return nil, fmt.Errorf("failed to get source budget plan: %w", err)
	}

	// Create new plan
	var newPlan BudgetPlan
	insertQuery := `
		INSERT INTO budget_plans (account_id, name, is_active)
		VALUES ($1, $2, FALSE)
		RETURNING id, account_id, name, is_active, created_at, updated_at
	`
	err = tx.QueryRow(insertQuery, accountID, newName).Scan(
		&newPlan.ID, &newPlan.AccountID, &newPlan.Name, &newPlan.IsActive,
		&newPlan.CreatedAt, &newPlan.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create budget plan copy: %w", err)
	}

	// Copy all recurring transactions
	copyQuery := `
		INSERT INTO recurring_transactions (
			account_id, budget_plan_id, financial_account_id, name, counterparty,
			category_id, expected_amount, tolerance, start_date, recurrence_unit,
			recurrence_value, end_date
		)
		SELECT 
			account_id, $1, financial_account_id, name, counterparty,
			category_id, expected_amount, tolerance, start_date, recurrence_unit,
			recurrence_value, end_date
		FROM recurring_transactions
		WHERE budget_plan_id = $2
	`
	_, err = tx.Exec(copyQuery, newPlan.ID, sourcePlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to copy recurring transactions: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &newPlan, nil
}

// RecurringTransaction management methods

// CreateRecurringTransaction creates a new recurring transaction (expense or income)
func (s *Storage) CreateRecurringTransaction(accountID, budgetPlanID, financialAccountID int, name, counterparty string, categoryID *int, expectedAmount, tolerance int, startDate time.Time, recurrenceUnit string, recurrenceValue int) (*RecurringTransaction, error) {
	var rt RecurringTransaction
	var endDate sql.NullTime
	query := `
		INSERT INTO recurring_transactions (account_id, budget_plan_id, financial_account_id, name, counterparty, category_id, expected_amount, tolerance, start_date, recurrence_unit, recurrence_value, end_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, account_id, budget_plan_id, financial_account_id, name, counterparty, category_id, expected_amount, tolerance, start_date, recurrence_unit, recurrence_value, end_date, type, archived, created_at, updated_at
	`
	err := s.db.QueryRow(query, accountID, budgetPlanID, financialAccountID, name, counterparty, categoryID, expectedAmount, tolerance, startDate, recurrenceUnit, recurrenceValue, nil).Scan(
		&rt.ID, &rt.AccountID, &rt.BudgetPlanID, &rt.FinancialAccountID, &rt.Name, &rt.Counterparty, &rt.CategoryID,
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

// GetRecurringTransactionsByAccount retrieves all recurring transactions for an account and budget plan
func (s *Storage) GetRecurringTransactionsByAccount(accountID, budgetPlanID int) ([]RecurringTransaction, error) {
	query := `
		SELECT id, account_id, budget_plan_id, financial_account_id, name, counterparty, category_id, expected_amount, tolerance, start_date, recurrence_unit, recurrence_value, end_date, type, archived, created_at, updated_at
		FROM recurring_transactions
		WHERE account_id = $1 AND budget_plan_id = $2
		ORDER BY name ASC
	`
	rows, err := s.db.Query(query, accountID, budgetPlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recurring transactions: %w", err)
	}
	defer rows.Close()

	var rts []RecurringTransaction
	for rows.Next() {
		var rt RecurringTransaction
		var endDate sql.NullTime
		if err := rows.Scan(&rt.ID, &rt.AccountID, &rt.BudgetPlanID, &rt.FinancialAccountID, &rt.Name, &rt.Counterparty, &rt.CategoryID,
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

// GetRecurringExpensesByAccount retrieves all recurring expenses for an account and budget plan
func (s *Storage) GetRecurringExpensesByAccount(accountID, budgetPlanID int) ([]RecurringTransaction, error) {
	query := `
		SELECT id, account_id, budget_plan_id, financial_account_id, name, counterparty, category_id, expected_amount, tolerance, start_date, recurrence_unit, recurrence_value, end_date, type, archived, created_at, updated_at
		FROM recurring_transactions
		WHERE account_id = $1 AND budget_plan_id = $2 AND type = 'expense'
		ORDER BY name ASC
	`
	rows, err := s.db.Query(query, accountID, budgetPlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recurring expenses: %w", err)
	}
	defer rows.Close()

	var rts []RecurringTransaction
	for rows.Next() {
		var rt RecurringTransaction
		var endDate sql.NullTime
		if err := rows.Scan(&rt.ID, &rt.AccountID, &rt.BudgetPlanID, &rt.FinancialAccountID, &rt.Name, &rt.Counterparty, &rt.CategoryID,
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

// GetRecurringIncomeByAccount retrieves all recurring income for an account and budget plan
func (s *Storage) GetRecurringIncomeByAccount(accountID, budgetPlanID int) ([]RecurringTransaction, error) {
	query := `
		SELECT id, account_id, budget_plan_id, financial_account_id, name, counterparty, category_id, expected_amount, tolerance, start_date, recurrence_unit, recurrence_value, end_date, type, archived, created_at, updated_at
		FROM recurring_transactions
		WHERE account_id = $1 AND budget_plan_id = $2 AND type = 'income'
		ORDER BY name ASC
	`
	rows, err := s.db.Query(query, accountID, budgetPlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recurring income: %w", err)
	}
	defer rows.Close()

	var rts []RecurringTransaction
	for rows.Next() {
		var rt RecurringTransaction
		var endDate sql.NullTime
		if err := rows.Scan(&rt.ID, &rt.AccountID, &rt.BudgetPlanID, &rt.FinancialAccountID, &rt.Name, &rt.Counterparty, &rt.CategoryID,
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
		SELECT id, account_id, budget_plan_id, financial_account_id, name, counterparty, category_id, expected_amount, tolerance, start_date, recurrence_unit, recurrence_value, end_date, type, archived, created_at, updated_at
		FROM recurring_transactions
		WHERE id = $1 AND account_id = $2
	`
	err := s.db.QueryRow(query, rtID, accountID).Scan(
		&rt.ID, &rt.AccountID, &rt.BudgetPlanID, &rt.FinancialAccountID, &rt.Name, &rt.Counterparty, &rt.CategoryID,
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
func (s *Storage) UpdateRecurringTransaction(accountID, rtID, budgetPlanID, financialAccountID int, name, counterparty string, categoryID *int, expectedAmount, tolerance int, startDate time.Time, recurrenceUnit string, recurrenceValue int) error {
	query := `
		UPDATE recurring_transactions
		SET budget_plan_id = $1, financial_account_id = $2, name = $3, counterparty = $4, category_id = $5, expected_amount = $6, tolerance = $7, start_date = $8, recurrence_unit = $9, recurrence_value = $10, updated_at = CURRENT_TIMESTAMP
		WHERE id = $11 AND account_id = $12
	`
	result, err := s.db.Exec(query, budgetPlanID, financialAccountID, name, counterparty, categoryID, expectedAmount, tolerance, startDate, recurrenceUnit, recurrenceValue, rtID, accountID)
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
// Filters by budget_plan_id to ensure only matches from the selected budget plan are returned
func (s *Storage) GetMatchedTransactionsForRecurring(rtID, budgetPlanID int, year, month int) ([]time.Time, error) {
	query := `
		SELECT rtv.transaction_date
		FROM recurring_transactions_view rtv
		INNER JOIN recurring_transactions rt ON rtv.recurring_transaction_id = rt.id
		WHERE rtv.recurring_transaction_id = $1
		AND rt.budget_plan_id = $2
		AND rtv.year = $3
		AND rtv.month = $4
		ORDER BY rtv.transaction_date
	`
	
	rows, err := s.db.Query(query, rtID, budgetPlanID, year, month)
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
// Filters by budget_plan_id to ensure only matches from the selected budget plan are returned
func (s *Storage) IsRecurringTransactionMatchedForMonth(rtID, budgetPlanID int, year, month int) (bool, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM recurring_transactions_view rtv
		INNER JOIN recurring_transactions rt ON rtv.recurring_transaction_id = rt.id
		WHERE rtv.recurring_transaction_id = $1 
		AND rt.budget_plan_id = $2
		AND rtv.year = $3 
		AND rtv.month = $4
	`
	err := s.db.QueryRow(query, rtID, budgetPlanID, year, month).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if recurring transaction is matched: %w", err)
	}
	return count > 0, nil
}

// GetRecurringTransactionDate gets the transaction date for a matched recurring transaction in a given month/year
// Filters by budget_plan_id to ensure only matches from the selected budget plan are returned
func (s *Storage) GetRecurringTransactionDate(rtID, budgetPlanID int, year, month int) (*time.Time, error) {
	var transactionDate time.Time
	query := `
		SELECT rtv.transaction_date
		FROM recurring_transactions_view rtv
		INNER JOIN recurring_transactions rt ON rtv.recurring_transaction_id = rt.id
		WHERE rtv.recurring_transaction_id = $1 
		AND rt.budget_plan_id = $2
		AND rtv.year = $3 
		AND rtv.month = $4
		LIMIT 1
	`
	err := s.db.QueryRow(query, rtID, budgetPlanID, year, month).Scan(&transactionDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get recurring transaction date: %w", err)
	}
	return &transactionDate, nil
}

// MatchRecurringTransactionsToTransaction checks if a transaction matches any recurring transactions for an account and budget plan
// Returns the matched recurring transaction if found, nil otherwise
// Note: This is now read-only - matches are determined dynamically by the view
// Filters by budget_plan_id to ensure only matches from the selected budget plan are returned
func (s *Storage) MatchRecurringTransactionsToTransaction(accountID, budgetPlanID int, tx *Transaction) (*RecurringTransaction, error) {
	// Check the view to see if this transaction matches any recurring transaction
	var rtID int
	year := tx.Date.Year()
	month := int(tx.Date.Month())

	query := `
		SELECT rtv.recurring_transaction_id
		FROM recurring_transactions_view rtv
		INNER JOIN recurring_transactions rt ON rtv.recurring_transaction_id = rt.id
		WHERE rtv.transaction_id = $1 
		AND rt.budget_plan_id = $2
		AND rtv.year = $3 
		AND rtv.month = $4
		LIMIT 1
	`
	err := s.db.QueryRow(query, tx.ID, budgetPlanID, year, month).Scan(&rtID)
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

// GetLastReceivedIncomeOccurrence finds the most recent income transaction date from recurring_transactions_view
func (s *Storage) GetLastReceivedIncomeOccurrence(accountID, budgetPlanID int) (*time.Time, error) {
	query := `
		SELECT rtv.transaction_date
		FROM recurring_transactions_view rtv
		INNER JOIN recurring_transactions rt ON rtv.recurring_transaction_id = rt.id
		WHERE rt.account_id = $1 AND rt.budget_plan_id = $2 AND rt.type = 'income'
		ORDER BY rtv.transaction_date DESC
		LIMIT 1
	`
	var transactionDate time.Time
	err := s.db.QueryRow(query, accountID, budgetPlanID).Scan(&transactionDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No income received yet
		}
		return nil, fmt.Errorf("failed to get last received income occurrence: %w", err)
	}
	return &transactionDate, nil
}

// GetLastReceivedIncomeAmount gets the amount of the most recent income transaction
func (s *Storage) GetLastReceivedIncomeAmount(accountID, budgetPlanID int) (*int, error) {
	query := `
		SELECT t.amount
		FROM recurring_transactions_view rtv
		INNER JOIN recurring_transactions rt ON rtv.recurring_transaction_id = rt.id
		INNER JOIN transactions t ON rtv.transaction_id = t.id
		WHERE rt.account_id = $1 AND rt.budget_plan_id = $2 AND rt.type = 'income'
		ORDER BY rtv.transaction_date DESC
		LIMIT 1
	`
	var amount int
	err := s.db.QueryRow(query, accountID, budgetPlanID).Scan(&amount)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No income received yet
		}
		return nil, fmt.Errorf("failed to get last received income amount: %w", err)
	}
	return &amount, nil
}

// GetNextUpcomingIncomeOccurrence finds the next recurring income expected date that hasn't been received yet
func (s *Storage) GetNextUpcomingIncomeOccurrence(accountID, budgetPlanID int) (*time.Time, error) {
	// Get the last received income date to calculate from there
	lastIncomeDate, err := s.GetLastReceivedIncomeOccurrence(accountID, budgetPlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get last received income: %w", err)
	}
	
	// Get all recurring income transactions for this account and budget plan
	incomeRts, err := s.GetRecurringIncomeByAccount(accountID, budgetPlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recurring income: %w", err)
	}
	
	if len(incomeRts) == 0 {
		return nil, nil // No recurring income configured
	}
	
	var nextOccurrence *time.Time
	now := time.Now()
	
	for _, rt := range incomeRts {
		var startDate time.Time
		if lastIncomeDate != nil {
			// Start from the last received income date to find the next occurrence
			startDate = *lastIncomeDate
		} else {
			// No income received yet, start from now
			startDate = now
		}
		
		// Calculate next occurrence after the start date
		// Add 1 day to ensure we get the NEXT occurrence, not the same one
		next := rt.NextOccurrence(startDate.AddDate(0, 0, 1))
		if next.IsZero() {
			// If no next occurrence from last date, try from now
			if lastIncomeDate != nil {
				next = rt.NextOccurrence(now)
			}
			if next.IsZero() {
				continue // This recurring transaction has no more occurrences
			}
		}
		
		// If the calculated next occurrence is the same as the last income date, 
		// calculate the one after that
		if lastIncomeDate != nil && next.Equal(*lastIncomeDate) {
			next = rt.NextOccurrence(next.AddDate(0, 0, 1))
			if next.IsZero() {
				continue
			}
		}
		
		// Only include if it's in the future or today
		if !next.Before(now) {
			if nextOccurrence == nil || next.Before(*nextOccurrence) {
				nextOccurrence = &next
			}
		}
	}
	
	return nextOccurrence, nil
}

// GetPreviousIncomeOccurrenceBefore finds the most recent income occurrence before the given date
// Filters by budget_plan_id to ensure only matches from the selected budget plan are returned
func (s *Storage) GetPreviousIncomeOccurrenceBefore(accountID, budgetPlanID int, beforeDate time.Time) (*time.Time, error) {
	query := `
		SELECT rtv.transaction_date
		FROM recurring_transactions_view rtv
		INNER JOIN recurring_transactions rt ON rtv.recurring_transaction_id = rt.id
		WHERE rt.account_id = $1 AND rt.budget_plan_id = $2 AND rt.type = 'income' AND rtv.transaction_date < $3
		ORDER BY rtv.transaction_date DESC
		LIMIT 1
	`
	var transactionDate time.Time
	err := s.db.QueryRow(query, accountID, budgetPlanID, beforeDate).Scan(&transactionDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No previous income found
		}
		return nil, fmt.Errorf("failed to get previous income occurrence: %w", err)
	}
	return &transactionDate, nil
}

// GetNextIncomeOccurrenceAfter finds the next income occurrence after the given date
func (s *Storage) GetNextIncomeOccurrenceAfter(accountID, budgetPlanID int, afterDate time.Time) (*time.Time, error) {
	// Get all recurring income transactions for this account and budget plan
	incomeRts, err := s.GetRecurringIncomeByAccount(accountID, budgetPlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recurring income: %w", err)
	}
	
	if len(incomeRts) == 0 {
		return nil, nil // No recurring income configured
	}
	
	var nextOccurrence *time.Time
	
	for _, rt := range incomeRts {
		// Calculate next occurrence after the given date
		next := rt.NextOccurrence(afterDate.AddDate(0, 0, 1))
		if next.IsZero() {
			continue // This recurring transaction has no more occurrences
		}
		
		// Only include if it's after the given date
		if next.After(afterDate) {
			if nextOccurrence == nil || next.Before(*nextOccurrence) {
				nextOccurrence = &next
			}
		}
	}
	
	return nextOccurrence, nil
}

// GetTransactionsBetweenDates gets all transactions for an account between two dates (inclusive)
func (s *Storage) GetTransactionsBetweenDates(accountID int, startDate, endDate time.Time) ([]Transaction, error) {
	query := `
		SELECT id, account_id, financial_account_id, date, original_payee, payee, category_id, amount, reviewed, created_at, updated_at
		FROM transactions
		WHERE account_id = $1
		AND date >= $2
		AND date <= $3
		ORDER BY date ASC, id ASC
	`
	rows, err := s.db.Query(query, accountID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions between dates: %w", err)
	}
	defer rows.Close()

	var txs []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.AccountID, &t.FinancialAccountID, &t.Date, &t.OriginalPayee, &t.Payee, &t.CategoryID, &t.Amount, &t.Reviewed, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		txs = append(txs, t)
	}
	return txs, nil
}

// GetUpcomingExpenseOccurrencesBetweenDates gets upcoming recurring expense occurrences between two dates
func (s *Storage) GetUpcomingExpenseOccurrencesBetweenDates(accountID, budgetPlanID int, startDate, endDate time.Time) ([]Occurrence, error) {
	// Get all recurring expenses for this account and budget plan
	expenses, err := s.GetRecurringExpensesByAccount(accountID, budgetPlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recurring expenses: %w", err)
	}
	
	var occurrences []Occurrence
	
	for _, expense := range expenses {
		// Get all occurrences for the date range
		// We need to iterate through months/years that overlap with the date range
		current := startDate
		for !current.After(endDate) {
			year := current.Year()
			month := int(current.Month())
			
			// Get occurrences for this month
			monthOccurrences := expense.GetOccurrencesInMonth(year, time.Month(month))
			
			for _, occ := range monthOccurrences {
				// Only include occurrences that fall within our date range
				if !occ.ExpectedDate.Before(startDate) && !occ.ExpectedDate.After(endDate) {
					// Check if this occurrence has been matched
					matched, err := s.IsRecurringTransactionMatchedForMonth(expense.ID, budgetPlanID, year, month)
					if err == nil {
						if matched {
							// Get the transaction date if it was matched
							txDate, err := s.GetRecurringTransactionDate(expense.ID, budgetPlanID, year, month)
							if err == nil && txDate != nil {
								occ.IsMatched = true
								occ.TransactionDate = txDate
							}
						}
						// Include both matched and unmatched expenses in the period
						occurrences = append(occurrences, occ)
					}
				}
			}
			
			// Move to next month
			current = current.AddDate(0, 1, 0)
		}
	}
	
	return occurrences, nil
}

// Budget management methods

// CreateBudget creates a new budget
func (s *Storage) CreateBudget(accountID, budgetPlanID int, categoryID *int, amountType string, amount int) (*Budget, error) {
	var budget Budget
	query := `
		INSERT INTO budgets (account_id, budget_plan_id, category_id, amount_type, amount)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, account_id, budget_plan_id, category_id, amount_type, amount, created_at, updated_at
	`
	err := s.db.QueryRow(query, accountID, budgetPlanID, categoryID, amountType, amount).Scan(
		&budget.ID, &budget.AccountID, &budget.BudgetPlanID, &budget.CategoryID,
		&budget.AmountType, &budget.Amount, &budget.CreatedAt, &budget.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create budget: %w", err)
	}
	return &budget, nil
}

// GetBudgetsByBudgetPlan retrieves all budgets for a budget plan
func (s *Storage) GetBudgetsByBudgetPlan(accountID, budgetPlanID int) ([]Budget, error) {
	query := `
		SELECT id, account_id, budget_plan_id, category_id, amount_type, amount, created_at, updated_at
		FROM budgets
		WHERE account_id = $1 AND budget_plan_id = $2
		ORDER BY category_id NULLS LAST
	`
	rows, err := s.db.Query(query, accountID, budgetPlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get budgets: %w", err)
	}
	defer rows.Close()

	var budgets []Budget
	for rows.Next() {
		var b Budget
		if err := rows.Scan(&b.ID, &b.AccountID, &b.BudgetPlanID, &b.CategoryID,
			&b.AmountType, &b.Amount, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan budget: %w", err)
		}
		budgets = append(budgets, b)
	}
	return budgets, nil
}

// GetBudget retrieves a single budget by ID
func (s *Storage) GetBudget(accountID, budgetID int) (*Budget, error) {
	var budget Budget
	query := `
		SELECT id, account_id, budget_plan_id, category_id, amount_type, amount, created_at, updated_at
		FROM budgets
		WHERE id = $1 AND account_id = $2
	`
	err := s.db.QueryRow(query, budgetID, accountID).Scan(
		&budget.ID, &budget.AccountID, &budget.BudgetPlanID, &budget.CategoryID,
		&budget.AmountType, &budget.Amount, &budget.CreatedAt, &budget.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get budget: %w", err)
	}
	return &budget, nil
}

// UpdateBudget updates a budget
func (s *Storage) UpdateBudget(accountID, budgetID int, categoryID *int, amountType string, amount int) error {
	query := `
		UPDATE budgets
		SET category_id = $1, amount_type = $2, amount = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4 AND account_id = $5
	`
	result, err := s.db.Exec(query, categoryID, amountType, amount, budgetID, accountID)
	if err != nil {
		return fmt.Errorf("failed to update budget: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("budget not found or not owned by account")
	}
	return nil
}

// DeleteBudget deletes a budget
func (s *Storage) DeleteBudget(accountID, budgetID int) error {
	result, err := s.db.Exec("DELETE FROM budgets WHERE id = $1 AND account_id = $2", budgetID, accountID)
	if err != nil {
		return fmt.Errorf("failed to delete budget: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("budget not found or not owned by account")
	}
	return nil
}

// CalculateExpectedYearlyIncome calculates the expected yearly income from all recurring income transactions
func (s *Storage) CalculateExpectedYearlyIncome(accountID, budgetPlanID int) (int, error) {
	// Get all recurring income transactions for this budget plan
	incomeRts, err := s.GetRecurringIncomeByAccount(accountID, budgetPlanID)
	if err != nil {
		return 0, fmt.Errorf("failed to get recurring income: %w", err)
	}

	var totalYearlyIncome int64
	for _, rt := range incomeRts {
		// Calculate occurrences per year based on recurrence_unit and recurrence_value
		var occurrencesPerYear float64
		switch rt.RecurrenceUnit {
		case "week":
			occurrencesPerYear = 52.0 / float64(rt.RecurrenceValue)
		case "month":
			occurrencesPerYear = 12.0 / float64(rt.RecurrenceValue)
		case "year":
			occurrencesPerYear = 1.0 / float64(rt.RecurrenceValue)
		default:
			continue
		}

		// Multiply expected_amount by occurrences per year
		yearlyAmount := float64(rt.ExpectedAmount) * occurrencesPerYear
		totalYearlyIncome += int64(yearlyAmount)
	}

	return int(totalYearlyIncome), nil
}

// GetBudgetSpentAmount calculates the total spent amount for a category in a given month
func (s *Storage) GetBudgetSpentAmount(accountID, budgetPlanID int, categoryID *int, year, month int) (int, error) {
	var query string
	var args []interface{}
	
	if categoryID == nil {
		// Query for uncategorized transactions
		query = `
			SELECT COALESCE(SUM(amount), 0)
			FROM transactions
			WHERE account_id = $1
			AND EXTRACT(YEAR FROM date) = $2
			AND EXTRACT(MONTH FROM date) = $3
			AND category_id IS NULL
		`
		args = []interface{}{accountID, year, month}
	} else {
		// Query for specific category
		query = `
			SELECT COALESCE(SUM(amount), 0)
			FROM transactions
			WHERE account_id = $1
			AND EXTRACT(YEAR FROM date) = $2
			AND EXTRACT(MONTH FROM date) = $3
			AND category_id = $4
		`
		args = []interface{}{accountID, year, month, *categoryID}
	}
	
	var spent int
	err := s.db.QueryRow(query, args...).Scan(&spent)
	if err != nil {
		return 0, fmt.Errorf("failed to get budget spent amount: %w", err)
	}
	return spent, nil
}

// GetBudgetExpectedAmount calculates the expected amount from recurring transactions not yet matched
func (s *Storage) GetBudgetExpectedAmount(accountID, budgetPlanID int, categoryID *int, year, month int) (int, error) {
	// Get all recurring expenses for this budget plan and category
	expenses, err := s.GetRecurringExpensesByAccount(accountID, budgetPlanID)
	if err != nil {
		return 0, fmt.Errorf("failed to get recurring expenses: %w", err)
	}

	var totalExpected int64
	for _, expense := range expenses {
		// Check if this expense matches the category
		if categoryID != nil && expense.CategoryID != nil && *expense.CategoryID != *categoryID {
			continue
		}
		if categoryID == nil && expense.CategoryID != nil {
			continue
		}
		if categoryID != nil && expense.CategoryID == nil {
			continue
		}

		// Get occurrences for this month
		occurrences := expense.GetOccurrencesInMonth(year, time.Month(month))
		for _, occ := range occurrences {
			// Only count if not matched
			if !occ.IsMatched {
				// Expected amount is negative for expenses, make it positive for budget calculation
				amount := expense.ExpectedAmount
				if amount < 0 {
					amount = -amount
				}
				totalExpected += int64(amount)
			}
		}
	}

	return int(totalExpected), nil
}

// GetBudgetSummaries retrieves budgets with calculated spent and expected amounts
func (s *Storage) GetBudgetSummaries(accountID, budgetPlanID int, year, month int) ([]BudgetSummary, error) {
	// Get all budgets for this budget plan
	budgets, err := s.GetBudgetsByBudgetPlan(accountID, budgetPlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get budgets: %w", err)
	}

	// Calculate yearly income
	yearlyIncome, err := s.CalculateExpectedYearlyIncome(accountID, budgetPlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate yearly income: %w", err)
	}

	// Get all categories for category name lookup
	categories, err := s.GetCategoriesByAccount(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	categoryMap := make(map[int]string)
	for _, cat := range categories {
		categoryMap[cat.ID] = cat.Name
	}

	var summaries []BudgetSummary
	for _, budget := range budgets {
		summary := BudgetSummary{
			Budget: budget,
		}

		// Get category name
		if budget.CategoryID != nil {
			if name, ok := categoryMap[*budget.CategoryID]; ok {
				summary.CategoryName = name
			} else {
				summary.CategoryName = "Unknown"
			}
		} else {
			summary.CategoryName = "Uncategorized"
		}

		// Calculate monthly amount
		summary.MonthlyAmount = budget.MonthlyAmount(yearlyIncome)

		// Get spent amount
		spent, err := s.GetBudgetSpentAmount(accountID, budgetPlanID, budget.CategoryID, year, month)
		if err != nil {
			return nil, fmt.Errorf("failed to get spent amount: %w", err)
		}
		summary.SpentAmount = spent

		// Get expected amount
		expected, err := s.GetBudgetExpectedAmount(accountID, budgetPlanID, budget.CategoryID, year, month)
		if err != nil {
			return nil, fmt.Errorf("failed to get expected amount: %w", err)
		}
		summary.ExpectedAmount = expected

		// Calculate remaining
		summary.Remaining = summary.MonthlyAmount - summary.SpentAmount - summary.ExpectedAmount

		summaries = append(summaries, summary)
	}

	return summaries, nil
}
