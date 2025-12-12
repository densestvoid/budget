package templates

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/densestvoid/budget/data"

	g "github.com/maragudk/gomponents"
	"github.com/maragudk/gomponents/html"
)

func SummaryPage(moneyIn, moneyOut, netMoney float64, monthName string, allOccurrences []data.Occurrence, includeUpcoming bool, year, month, prevYear, prevMonth, nextYear, nextMonth int, budgetPlans []data.BudgetPlan, selectedBudgetPlanID int, budgetSummaries []data.BudgetSummary, yearlyIncome int) g.Node {
	return g.Group([]g.Node{
		// Pagination controls
		html.Div(
			html.Class("row mb-3"),
			html.Div(
				html.Class("col-6"),
				html.A(
					html.Class("btn btn-outline-primary"),
					html.Href(fmt.Sprintf("/?year=%d&month=%d&include_upcoming=%v&budget_plan_id=%d", prevYear, prevMonth, includeUpcoming, selectedBudgetPlanID)),
					g.Raw(`<i class="bi bi-chevron-left"></i> `),
					g.Text("Previous"),
				),
			),
			html.Div(
				html.Class("col-6 text-end"),
				html.A(
					html.Class("btn btn-outline-primary"),
					html.Href(fmt.Sprintf("/?year=%d&month=%d&include_upcoming=%v&budget_plan_id=%d", nextYear, nextMonth, includeUpcoming, selectedBudgetPlanID)),
					g.Text("Next "),
					g.Raw(`<i class="bi bi-chevron-right"></i>`),
				),
			),
		),
		html.Div(
			html.Class("row"),
			html.Div(
				html.Class("col-12"),
				html.H1(html.Class("display-4 mb-4 text-center"), g.Text("Monthly Summary")),
				html.P(html.Class("lead text-center"), g.Text(fmt.Sprintf("Financial Overview for %s", monthName))),
			),
		),
		html.Div(
			html.Class("row mt-5 justify-content-center"),
			html.Div(
				html.Class("col-12 col-lg-10"),
				html.Div(
					html.Class("card"),
					html.Div(
						html.Class("card-header d-flex justify-content-between align-items-center"),
						html.H5(html.Class("card-title mb-0"), g.Text("Money Flow")),
						html.Div(
							html.Class("form-check form-switch"),
							html.Input(
								html.Type("checkbox"),
								html.Class("form-check-input"),
								html.ID("include-upcoming-toggle"),
								func() g.Node {
									if includeUpcoming {
										return html.Checked()
									}
									return nil
								}(),
								g.Attr("onchange", "const checked = this.checked; htmx.ajax('GET', '/?include_upcoming=' + checked, {target: 'body', swap: 'outerHTML'})"),
							),
							html.Label(
								html.Class("form-check-label"),
								html.For("include-upcoming-toggle"),
								g.Text("Include Upcoming"),
							),
						),
					),
					html.Div(
						html.Class("card-body p-4"),
						// Desktop view: horizontal layout with large text
						html.Div(
							html.Class("d-none d-md-flex align-items-center justify-content-center flex-wrap gap-3"),
							// Money In Card
							html.Div(
								html.Class("card flex-fill"),
								g.Attr("style", "min-width: 200px; max-width: 300px;"),
								html.Div(
									html.Class("card-body text-center"),
									html.H6(html.Class("card-subtitle mb-2 text-muted"), g.Text("In")),
									html.Div(
										html.Class("display-6 fw-bold text-success"),
										g.Text(fmt.Sprintf("$%.2f", moneyIn)),
									),
								),
							),
							// Minus Sign
							html.Div(
								html.Class("d-flex align-items-center"),
								g.Attr("style", "font-size: 3rem; font-weight: bold; color: #6c757d;"),
								g.Text("−"),
							),
							// Money Out Card
							html.Div(
								html.Class("card flex-fill"),
								g.Attr("style", "min-width: 200px; max-width: 300px;"),
								html.Div(
									html.Class("card-body text-center"),
									html.H6(html.Class("card-subtitle mb-2 text-muted"), g.Text("Out")),
									html.Div(
										html.Class("display-6 fw-bold text-danger"),
										g.Text(fmt.Sprintf("$%.2f", moneyOut)),
									),
								),
							),
							// Equals Sign
							html.Div(
								html.Class("d-flex align-items-center"),
								g.Attr("style", "font-size: 3rem; font-weight: bold; color: #6c757d;"),
								g.Text("="),
							),
							// Net Money Card
							html.Div(
								html.Class("card flex-fill"),
								g.Attr("style", "min-width: 200px; max-width: 300px;"),
								html.Div(
									html.Class("card-body text-center"),
									html.H6(html.Class("card-subtitle mb-2 text-muted"), g.Text("Net")),
									html.Div(
										func() g.Node {
											if netMoney >= 0 {
												return html.Class("display-6 fw-bold text-success")
											}
											return html.Class("display-6 fw-bold text-danger")
										}(),
										g.Text(fmt.Sprintf("$%.2f", netMoney)),
									),
								),
							),
						),
						// Mobile Option 1: Inline with smaller text (all on one line, rounded to dollar)
						html.Div(
							html.Class("d-md-none"),
							html.Div(
								html.Class("d-flex align-items-center justify-content-center gap-1"),
								g.Attr("style", "flex-wrap: nowrap; overflow-x: auto;"),
								// Money In Card
								html.Div(
									html.Class("card"),
									g.Attr("style", "min-width: 80px; flex-shrink: 0;"),
									html.Div(
										html.Class("card-body text-center p-2"),
										html.H6(html.Class("card-subtitle mb-1 text-muted"), g.Attr("style", "font-size: 0.75rem;"), g.Text("In")),
										html.Div(
											html.Class("fw-bold text-success"),
											g.Attr("style", "font-size: 1.5rem; white-space: nowrap;"),
											g.Text(fmt.Sprintf("$%.0f", moneyIn)),
										),
									),
								),
								// Minus Sign
								html.Div(
									html.Class("d-flex align-items-center"),
									g.Attr("style", "font-size: 1.8rem; font-weight: bold; color: #6c757d; flex-shrink: 0;"),
									g.Text("−"),
								),
								// Money Out Card
								html.Div(
									html.Class("card"),
									g.Attr("style", "min-width: 80px; flex-shrink: 0;"),
									html.Div(
										html.Class("card-body text-center p-2"),
										html.H6(html.Class("card-subtitle mb-1 text-muted"), g.Attr("style", "font-size: 0.75rem;"), g.Text("Out")),
										html.Div(
											html.Class("fw-bold text-danger"),
											g.Attr("style", "font-size: 1.5rem; white-space: nowrap;"),
											g.Text(fmt.Sprintf("$%.0f", moneyOut)),
										),
									),
								),
								// Equals Sign
								html.Div(
									html.Class("d-flex align-items-center"),
									g.Attr("style", "font-size: 1.8rem; font-weight: bold; color: #6c757d; flex-shrink: 0;"),
									g.Text("="),
								),
								// Net Money Card
								html.Div(
									html.Class("card"),
									g.Attr("style", "min-width: 80px; flex-shrink: 0;"),
									html.Div(
										html.Class("card-body text-center p-2"),
										html.H6(html.Class("card-subtitle mb-1 text-muted"), g.Attr("style", "font-size: 0.75rem;"), g.Text("Net")),
										html.Div(
											func() g.Node {
												if netMoney >= 0 {
													return html.Class("fw-bold text-success")
												}
												return html.Class("fw-bold text-danger")
											}(),
											g.Attr("style", "font-size: 1.5rem; white-space: nowrap;"),
											g.Text(fmt.Sprintf("$%.0f", netMoney)),
										),
									),
								),
							),
						),
					),
				),
			),
		),
		// Combined recurring transactions table
		html.Div(
			html.Class("row mt-5 justify-content-center"),
			html.Div(
				html.Class("col-12 col-lg-10"),
				html.Div(
					html.Class("card"),
					html.Div(
						html.Class("card-header"),
						html.H5(html.Class("card-title mb-0"), g.Text(fmt.Sprintf("Recurring Transactions (%d)", len(allOccurrences)))),
					),
					html.Div(
						html.Class("card-body"),
						func() g.Node {
							if len(allOccurrences) == 0 {
								return html.P(html.Class("text-muted mb-0"), g.Text("No recurring transactions for this month."))
							}
							return html.Div(
								html.Class("table-responsive"),
								html.Table(
									html.Class("table table-hover"),
									g.El("thead",
										html.Tr(
											html.Th(g.Text("Name")),
											html.Th(g.Text("Status")),
											html.Th(g.Text("Date")),
											html.Th(html.Class("text-end"), g.Text("Amount")),
										),
									),
									g.El("tbody",
										g.Group(func() []g.Node {
											var rows []g.Node
											today := time.Now()
											for _, occ := range allOccurrences {
												rt := occ.RecurringTransaction
												var displayDate time.Time

												if occ.IsMatched && occ.TransactionDate != nil {
													displayDate = *occ.TransactionDate
												} else {
													displayDate = occ.ExpectedDate
												}

												dateText := displayDate.Format("Jan 2")

												// Determine status: matched, overdue, or upcoming
												var statusBadge g.Node
												if occ.IsMatched {
													if rt.IsExpense() {
														statusBadge = html.Span(
															html.Class("badge bg-success"),
															g.Text("Paid"),
														)
													} else {
														statusBadge = html.Span(
															html.Class("badge bg-success"),
															g.Text("Received"),
														)
													}
												} else {
													// Check if overdue (expected date has passed)
													if occ.ExpectedDate.Before(today) {
														statusBadge = html.Span(
															html.Class("badge bg-danger"),
															g.Text("Overdue"),
														)
													} else {
														if rt.IsExpense() {
															statusBadge = html.Span(
																html.Class("badge bg-warning text-dark"),
																g.Text("Upcoming"),
															)
														} else {
															statusBadge = html.Span(
																html.Class("badge bg-warning text-dark"),
																g.Text("Expected"),
															)
														}
													}
												}

												// Format amount: negative for expenses, positive for income
												var amountText string
												var amountClass string
												amount := float64(rt.ExpectedAmount) / 100.0
												if rt.IsExpense() {
													amountText = fmt.Sprintf("-$%.2f", -amount) // Show as negative
													amountClass = "text-danger"                 // Red text for expenses
												} else {
													amountText = fmt.Sprintf("$%.2f", amount)
													amountClass = "text-success" // Green text for income
												}

												rows = append(rows, html.Tr(
													html.Td(g.Text(rt.Name)),
													html.Td(statusBadge),
													html.Td(g.Text(dateText)),
													html.Td(
														html.Class(fmt.Sprintf("text-end fw-bold %s", amountClass)),
														g.Text(amountText),
													),
												))
											}
											return rows
										}()),
									),
								),
							)
						}(),
					),
				),
			),
		),
		// Budgets table
		func() g.Node {
			if len(budgetSummaries) > 0 {
				return html.Div(
					html.Class("row mt-5 justify-content-center"),
					html.Div(
						html.Class("col-12 col-lg-10"),
						html.Div(
							html.Class("card"),
							html.Div(
								html.Class("card-header"),
								html.H5(html.Class("card-title mb-0"), g.Text("Budgets")),
							),
							html.Div(
								html.Class("card-body"),
								BudgetSummaryTable(budgetSummaries),
							),
						),
					),
				)
			}
			return nil
		}(),
	})
}

func AboutPage() g.Node {
	return g.Group([]g.Node{
		html.Div(
			html.Class("row"),
			html.Div(
				html.Class("col-12"),
				html.H1(html.Class("display-4 mb-4 text-center"), g.Text("About Budget App")),
				html.P(html.Class("lead"), g.Text("Learn more about this modern web application architecture")),
			),
		),
		html.Div(
			html.Class("row mt-4"),
			html.Div(
				html.Class("col-md-8"),
				html.H3(g.Text("Architecture Overview")),
				html.P(g.Text("This application demonstrates a modern approach to web development using Go as the backend language with contemporary frontend technologies.")),

				html.H4(html.Class("mt-4"), g.Text("Backend (Go)")),
				html.Ul(
					html.Li(g.Text("Chi router for HTTP routing")),
					html.Li(g.Text("PostgreSQL for data persistence")),
					html.Li(g.Text("Goose for database migrations")),
					html.Li(g.Text("Gomponents for HTML generation")),
				),

				html.H4(html.Class("mt-4"), g.Text("Frontend")),
				html.Ul(
					html.Li(g.Text("HTMX for dynamic content loading")),
					html.Li(g.Text("Alpine.js for reactive UI components")),
					html.Li(g.Text("Bootstrap 5 for responsive design")),
				),

				html.H4(html.Class("mt-4"), g.Text("Key Benefits")),
				html.Ul(
					html.Li(g.Text("Type-safe HTML generation with Gomponents")),
					html.Li(g.Text("Minimal JavaScript with HTMX and Alpine.js")),
					html.Li(g.Text("Fast development with Go's simplicity")),
					html.Li(g.Text("Scalable database design with PostgreSQL")),
				),
			),
			html.Div(
				html.Class("col-md-4"),
				html.Div(
					html.Class("card"),
					html.Div(
						html.Class("card-body"),
						html.H5(html.Class("card-title"), g.Text("Quick Stats")),
						html.Ul(html.Class("list-unstyled"),
							html.Li(g.Text("Backend: Go 1.24+")),
							html.Li(g.Text("Database: PostgreSQL")),
							html.Li(g.Text("Frontend: HTMX + Alpine.js")),
							html.Li(g.Text("Styling: Bootstrap 5")),
						),
					),
				),
			),
		),
	})
}

func RegisterPage() g.Node {
	return html.Div(
		html.Class("row justify-content-center"),
		html.Div(
			html.Class("col-md-6 col-lg-4"),
			html.Div(
				html.Class("card"),
				html.Div(
					html.Class("card-body"),
					html.H3(
						html.Class("card-title text-center mb-4"),
						g.Text("Create Account"),
					),
					g.El("form",
						html.Action("/auth/register"),
						html.Method("POST"),
						html.Div(
							html.Class("mb-3"),
							html.Label(
								html.Class("form-label"),
								html.For("register-name"),
								g.Text("Name"),
							),
							html.Input(
								html.Class("form-control"),
								html.Type("text"),
								html.ID("register-name"),
								html.Name("name"),
								html.Required(),
							),
						),
						html.Div(
							html.Class("mb-3"),
							html.Label(
								html.Class("form-label"),
								html.For("register-email"),
								g.Text("Email"),
							),
							html.Input(
								html.Class("form-control"),
								html.Type("email"),
								html.ID("register-email"),
								html.Name("email"),
								html.Required(),
							),
						),
						html.Div(
							html.Class("mb-3"),
							html.Label(
								html.Class("form-label"),
								html.For("register-password"),
								g.Text("Password"),
							),
							html.Input(
								html.Class("form-control"),
								html.Type("password"),
								html.ID("register-password"),
								html.Name("password"),
								html.Required(),
							),
						),
						html.Button(
							html.Class("btn btn-primary w-100 mb-3"),
							html.Type("submit"),
							g.Text("Create Account"),
						),
						html.Div(
							html.Class("text-center"),
							html.A(
								html.Href("/"),
								g.Text("Already have an account? Go back"),
							),
						),
					),
				),
			),
		),
	)
}

// Helper to build the breadcrumb path for a category
func buildCategoryBreadcrumb(c data.Category, categories []data.Category) []string {
	var path []string
	current := &c
	catMap := map[int]*data.Category{}
	for i := range categories {
		catMap[categories[i].ID] = &categories[i]
	}
	for current != nil {
		path = append([]string{current.Name}, path...)
		if current.ParentID != nil {
			current = catMap[*current.ParentID]
		} else {
			current = nil
		}
	}
	return path
}

// Update CSS for a vertical tree with minimal indentation
var treeCSS = g.El("style", g.Raw(`
.categories-tree-list {
  position: relative;
  padding-left: 0;
  margin-bottom: 0;
}
.category-tree-row {
  display: flex;
  align-items: center;
  position: relative;
  min-height: 2.5rem;
  padding-left: 1.25rem;
}
.category-tree-row .tree-connector {
  position: absolute;
  left: 0.5rem;
  top: 0;
  bottom: 0;
  width: 1px;
  background: var(--bs-border-color, #444);
  z-index: 0;
}
.category-tree-row .tree-connector-horizontal {
  display: none;
}
`))

// flattenCategories returns a flat list of categories in parent-child order with their level, sorting children by name
func flattenCategories(categories []data.Category, parentID *int, level int) []struct {
	Cat   data.Category
	Level int
} {
	var result []struct {
		Cat   data.Category
		Level int
	}
	// Collect children
	var children []data.Category
	for _, cat := range categories {
		if (cat.ParentID == nil && parentID == nil) || (cat.ParentID != nil && parentID != nil && *cat.ParentID == *parentID) {
			children = append(children, cat)
		}
	}
	// Sort children by name
	sort.Slice(children, func(i, j int) bool {
		return children[i].Name < children[j].Name
	})
	for _, cat := range children {
		result = append(result, struct {
			Cat   data.Category
			Level int
		}{cat, level})
		result = append(result, flattenCategories(categories, &cat.ID, level+1)...)
	}
	return result
}

// CategoriesPage renders the category management page
func CategoriesPage(categories []data.Category) g.Node {
	// Build a map of parent -> children for nesting
	children := map[int][]data.Category{}
	var roots []data.Category
	for _, c := range categories {
		if c.ParentID == nil {
			roots = append(roots, c)
		} else {
			children[*c.ParentID] = append(children[*c.ParentID], c)
		}
	}

	var renderCategory func(c data.Category, isRoot bool, isLast bool) g.Node
	renderCategory = func(c data.Category, isRoot bool, isLast bool) g.Node {
		hasChildren := len(children[c.ID]) > 0
		modalID := "editCategoryModal-" + strconv.Itoa(c.ID)
		return g.Group([]g.Node{
			html.Li(
				html.Class("category-tree-row position-relative list-unstyled"),
				// Vertical connector (if not root and not last)
				func() g.Node {
					if !isRoot && !isLast {
						return html.Span(html.Class("tree-connector"))
					}
					return nil
				}(),
				// Category name
				html.Span(html.Class("fw-semibold me-2"), g.Text(c.Name)),
				// Action buttons
				html.Div(
					html.Class("btn-group btn-group-sm ms-auto"),
					html.Button(
						html.Class("btn btn-secondary"),
						html.Type("button"),
						html.DataAttr("bs-toggle", "modal"),
						html.DataAttr("bs-target", "#"+modalID),
						g.Text("Edit"),
					),
					g.El("form",
						html.Action("/categories/"+strconv.Itoa(c.ID)),
						html.Method("DELETE"),
						g.Attr("hx-delete", "/categories/"+strconv.Itoa(c.ID)),
						g.Attr("hx-target", "closest .category-tree-row"),
						g.Attr("hx-swap", "outerHTML"),
						html.Class("d-inline"),
						html.Button(
							html.Type("submit"),
							html.Class("btn btn-danger"),
							g.Text("Delete"),
						),
					),
				),
			),
			// Render children (always visible, no collapse)
			func() g.Node {
				if hasChildren {
					return html.Ul(
						html.Class("categories-tree-list ps-0 ms-0"),
						g.Group(func() []g.Node {
							var nodes []g.Node
							for i, child := range children[c.ID] {
								isLastChild := i == len(children[c.ID])-1
								nodes = append(nodes, renderCategory(child, false, isLastChild))
							}
							return nodes
						}()),
					)
				}
				return nil
			}(),
		})
	}

	return g.Group([]g.Node{
		treeCSS,
		html.Div(
			html.Class("row"),
			html.Div(
				html.Class("col-12 col-md-8 px-2"),
				html.H1(html.Class("display-6 my-2 text-center"), g.Text("Categories")),
				// Toolbar: left group for future actions, right group for add button
				html.Div(
					html.Class("card p-2 mb-3 bg-body-tertiary border"),
					html.Div(
						html.Class("d-flex align-items-center gap-2"),
						// Left-aligned group for actions (empty for now, ready for future actions)
						html.Div(
							html.Class("d-flex align-items-center gap-2"),
						),
						// Right-aligned add button
						html.Button(
							html.Class("btn btn-primary ms-auto"),
							html.Type("button"),
							html.DataAttr("bs-toggle", "modal"),
							html.DataAttr("bs-target", "#addCategoryModal"),
							html.I(html.Class("bi bi-plus-circle me-1")),
							g.Text("Add"),
						),
					),
				),
				html.Ul(
					html.Class("categories-tree-list ps-0 ms-0"),
					g.Group(func() []g.Node {
						var nodes []g.Node
						for i, root := range roots {
							isLastRoot := i == len(roots)-1
							nodes = append(nodes, renderCategory(root, true, isLastRoot))
						}
						return nodes
					}()),
				),
				// Add Category Modal
				html.Div(
					html.Class("modal fade"),
					html.ID("addCategoryModal"),
					html.DataAttr("tabindex", "-1"),
					html.DataAttr("aria-labelledby", "addCategoryModalLabel"),
					html.DataAttr("aria-hidden", "true"),
					html.Div(
						html.Class("modal-dialog"),
						html.Div(
							html.Class("modal-content"),
							html.Div(
								html.Class("modal-header"),
								html.H5(html.Class("modal-title"), html.ID("addCategoryModalLabel"), g.Text("Add Category")),
								html.Button(
									html.Class("btn-close"),
									html.Type("button"),
									html.DataAttr("bs-dismiss", "modal"),
									html.DataAttr("aria-label", "Close"),
								),
							),
							html.Div(
								html.Class("modal-body"),
								g.El("form",
									html.Action("/categories/"),
									html.Method("POST"),
									g.Attr("hx-post", "/categories/"),
									g.Attr("hx-target", "#categories-directory"),
									g.Attr("hx-swap", "outerHTML"),
									g.Attr("hx-on::after-request", "if (event.target === this) { bootstrap.Modal.getInstance(document.getElementById('addCategoryModal')).hide(); }"),
									html.Div(
										html.Class("mb-3"),
										html.Label(html.Class("form-label"), html.For("add-name"), g.Text("Name")),
										html.Input(html.Type("text"), html.Name("name"), html.ID("add-name"), html.Class("form-control"), html.Required()),
									),
									html.Div(
										html.Class("mb-3"),
										html.Label(html.Class("form-label"), html.For("add-parent"), g.Text("Parent Category")),
										CategoryBootstrapDropdown(categories, "parent_id", "add-parent", nil, nil, false, nil),
									),
									html.Div(
										html.Class("d-flex justify-content-end gap-2"),
										html.Button(html.Type("button"), html.Class("btn btn-secondary"), html.DataAttr("bs-dismiss", "modal"), g.Text("Cancel")),
										html.Button(html.Type("submit"), html.Class("btn btn-primary"), g.Text("Add Category")),
									),
								),
							),
						),
					),
				),
			),
		),
	})
}

// TransactionsPage renders the transactions management page
func TransactionsPage(transactions []data.Transaction, categories []data.Category, financialAccounts []data.FinancialAccount, errMsg string, successMsg string) g.Node {
	categoryMap := map[int]string{}
	for _, c := range categories {
		categoryMap[c.ID] = c.Name
	}
	return g.Group([]g.Node{
		html.Div(
			html.Class("row"),
			html.Div(
				html.Class("col-12 col-md-8 px-2"),
				html.H1(html.Class("display-6 my-2 text-center"), g.Text("Transactions")),
				// Toolbar: left group for future actions, right group for upload button
				html.Div(
					html.Class("card p-2 mb-3 bg-body-tertiary border"),
					html.Div(
						html.Class("d-flex align-items-center gap-2"),
						// Left-aligned group for actions
						html.Div(
							html.Class("d-flex align-items-center gap-2"),
							html.Button(
								html.Class("btn btn-success"),
								html.Type("button"),
								g.Attr("id", "mark-reviewed-btn"),
								g.Attr("disabled", "disabled"),
								g.Attr("hx-post", "/transactions/mark-reviewed"),
								g.Attr("hx-include", ".transaction-select:checked"),
								g.Attr("hx-target", "#transactions-list"),
								g.Attr("hx-swap", "outerHTML"),
								g.Raw(`<i class="bi bi-check"></i>`),
							),
						),
						// Right-aligned upload button
						html.Button(
							html.Class("btn btn-primary ms-auto"),
							html.Type("button"),
							html.DataAttr("bs-toggle", "modal"),
							html.DataAttr("bs-target", "#uploadCsvModal"),
							g.Text("Upload"),
						),
					),
				),
				// Success banner if present
				g.Group(func() []g.Node {
					if successMsg != "" {
						return []g.Node{
							html.Div(
								html.Class("alert alert-success alert-dismissible fade show"),
								html.Role("alert"),
								g.Text(successMsg),
								html.Button(
									html.Type("button"),
									html.Class("btn-close"),
									html.DataAttr("bs-dismiss", "alert"),
									html.DataAttr("aria-label", "Close"),
								),
							),
						}
					}
					return nil
				}()),
				// Error banner if present
				g.Group(func() []g.Node {
					if errMsg != "" {
						return []g.Node{html.Div(html.Class("alert alert-danger"), g.Text(errMsg))}
					}
					return nil
				}()),
				// Transaction list
				html.Div(
					html.ID("transactions-list"),
					html.Class("list-group"),
					g.Group(func() []g.Node {
						var rows []g.Node
						for _, t := range transactions {
							rows = append(rows, TransactionCardWithModalSelectable(t, categories, ""))
						}
						return rows
					}()),
				),
			),
		),
		// CSV Upload Modal
		html.Div(
			html.Class("modal fade"),
			html.ID("uploadCsvModal"),
			html.DataAttr("tabindex", "-1"),
			html.DataAttr("aria-labelledby", "uploadCsvModalLabel"),
			html.DataAttr("aria-hidden", "true"),
			html.Div(
				html.Class("modal-dialog"),
				html.Div(
					html.Class("modal-content"),
					html.Div(
						html.Class("modal-header"),
						html.H5(html.Class("modal-title"), html.ID("uploadCsvModalLabel"), g.Text("Upload Transactions CSV")),
						html.Button(
							html.Class("btn-close"),
							html.Type("button"),
							html.DataAttr("bs-dismiss", "modal"),
							html.DataAttr("aria-label", "Close"),
						),
					),
					html.Div(
						html.Class("modal-body"),
						g.El("form",
							html.Action("/transactions/upload"),
							html.Method("POST"),
							g.Attr("enctype", "multipart/form-data"),
							html.Div(
								html.Class("mb-3"),
								html.Label(html.Class("form-label"), html.For("financial_account_id"), g.Text("Financial Account")),
								html.Select(
									html.Name("financial_account_id"),
									html.ID("financial_account_id"),
									html.Class("form-select"),
									html.Required(),
									g.Group(func() []g.Node {
										var options []g.Node
										options = append(options, html.Option(html.Value(""), g.Text("Select an account...")))
										for _, fa := range financialAccounts {
											options = append(options, html.Option(html.Value(strconv.Itoa(fa.ID)), g.Text(fa.Name+" ("+fa.Type+")")))
										}
										return options
									}()),
								),
							),
							html.Div(
								html.Class("mb-3"),
								html.Label(html.Class("form-label"), html.For("csv"), g.Text("CSV File")),
								html.Input(html.Type("file"), html.Name("csv"), html.ID("csv"), html.Class("form-control"), html.Required()),
							),
							html.Div(
								html.Class("d-flex justify-content-end gap-2"),
								html.Button(html.Type("button"), html.Class("btn btn-secondary"), html.DataAttr("bs-dismiss", "modal"), g.Text("Cancel")),
								html.Button(html.Type("submit"), html.Class("btn btn-primary"), g.Text("Upload CSV")),
							),
						),
					),
				),
			),
		),
		// Shared Edit Transaction Modal
		html.Div(
			html.Class("modal fade"),
			html.ID("editTransactionModal"),
			html.DataAttr("tabindex", "-1"),
			html.DataAttr("aria-labelledby", "editTransactionModalLabel"),
			html.DataAttr("aria-hidden", "true"),
			html.Div(
				html.Class("modal-dialog"),
				html.Div(
					html.Class("modal-content"),
					html.Div(
						html.Class("modal-header"),
						html.H5(html.Class("modal-title"), html.ID("editTransactionModalLabel"), g.Text("Edit Transaction")),
						html.Button(
							html.Class("btn-close"),
							html.Type("button"),
							html.DataAttr("bs-dismiss", "modal"),
							html.DataAttr("aria-label", "Close"),
						),
					),
					html.Div(
						html.ID("editTransactionModalBody"),
						html.Class("modal-body"),
						// Content will be loaded via HTMX
					),
				),
			),
		),
		// Add a script to enable/disable the button based on selection
		html.Script(g.Raw(`
function updateMarkReviewedBtn() {
  var anyChecked = document.querySelectorAll('.transaction-select:checked').length > 0;
  var btn = document.getElementById('mark-reviewed-btn');
  if (btn) {
    btn.disabled = !anyChecked;
  }
}

function setupCheckboxListeners() {
  document.querySelectorAll('.transaction-select').forEach(function(cb) {
    cb.addEventListener('change', updateMarkReviewedBtn);
  });
  updateMarkReviewedBtn();
}

document.addEventListener('DOMContentLoaded', function() {
  setupCheckboxListeners();
});

// Listen for HTMX content updates to re-setup listeners
document.addEventListener('htmx:afterSwap', function() {
  setupCheckboxListeners();
});

// Also update button state after HTMX requests complete
document.addEventListener('htmx:afterRequest', function() {
  setTimeout(updateMarkReviewedBtn, 100); // Small delay to ensure DOM is updated
});
`)),
	})
}

// TransactionCardWithModal renders a single transaction card and its modal for editing
func TransactionCardWithModal(t data.Transaction, categories []data.Category, modalErrMsg string) g.Node {
	modalID := "editTransactionModal-" + strconv.Itoa(t.ID)
	cardID := "transaction-card-" + strconv.Itoa(t.ID)
	// Find category name
	var categoryName string
	if t.CategoryID == nil {
		categoryName = "Uncategorized"
	} else {
		for _, c := range categories {
			if c.ID == *t.CategoryID {
				categoryName = c.Name
				break
			}
		}
		if categoryName == "" {
			categoryName = "Uncategorized"
		}
	}
	return html.Div(
		g.Group([]g.Node{
			html.Div(
				html.Class("card mb-2"),
				html.ID(cardID),
				html.DataAttr("bs-toggle", "modal"),
				html.DataAttr("bs-target", "#"+modalID),
				g.Attr("style", "cursor: pointer;"),
				html.Div(
					html.Class("d-flex w-100 align-items-center py-3 gap-3 card-body"),
					// First column: Payee and Category stacked, text only
					html.Div(
						html.Class("d-flex flex-column flex-grow-1 gap-2"),
						html.Div(
							html.Class("fw-semibold"),
							g.Text(t.Payee),
						),
						html.Div(
							html.Class("text-muted small"),
							g.Text(categoryName),
						),
					),
					// Second column: Amount and Date stacked, right-aligned
					html.Div(
						html.Class("d-flex flex-column align-items-end gap-2"),
						html.Div(
							html.Class("fw-bold fs-5 mb-0"),
							g.Text(t.AmountDecimal()),
						),
						html.Div(
							html.Class("text-muted small mb-0"),
							g.Text(t.Date.Format("01/02/06")),
						),
					),
				),
			),
			// Modal for editing transaction
			html.Div(
				html.Class("modal fade"),
				html.ID(modalID),
				html.DataAttr("tabindex", "-1"),
				html.DataAttr("aria-labelledby", modalID+"Label"),
				html.DataAttr("aria-hidden", "true"),
				html.Div(
					html.Class("modal-dialog"),
					html.Div(
						html.Class("modal-content"),
						html.Div(
							html.Class("modal-header"),
							html.H5(html.Class("modal-title"), html.ID(modalID+"Label"), g.Text("Edit Transaction")),
							html.Button(
								html.Class("btn-close"),
								html.Type("button"),
								html.DataAttr("bs-dismiss", "modal"),
								html.DataAttr("aria-label", "Close"),
							),
						),
						html.Div(
							html.Class("modal-body"),
							// Error banner if present
							func() g.Node {
								if modalErrMsg != "" {
									return html.Div(html.Class("alert alert-danger"), g.Text(modalErrMsg))
								}
								return nil
							}(),
							g.El("form",
								html.Action("/transactions/"+strconv.Itoa(t.ID)),
								html.Method("PATCH"),
								g.Attr("hx-patch", "/transactions/"+strconv.Itoa(t.ID)),
								g.Attr("hx-target", "#"+cardID),
								g.Attr("hx-swap", "outerHTML"),
								g.Attr("hx-on::after-request", `if (event.target === this) { bootstrap.Modal.getInstance(this.closest('.modal')).hide(); }`),
								html.Div(
									html.Class("mb-3"),
									html.Label(html.Class("form-label"), html.For("edit-payee-"+strconv.Itoa(t.ID)), g.Text("Payee")),
									html.Input(
										html.Type("text"),
										html.Name("payee"),
										html.ID("edit-payee-"+strconv.Itoa(t.ID)),
										html.Value(t.Payee),
										html.Class("form-control"),
										html.Required(),
									),
								),
								html.Div(
									html.Class("mb-3"),
									html.Label(html.Class("form-label"), html.For("edit-category-"+strconv.Itoa(t.ID)), g.Text("Category")),
									CategoryBootstrapDropdown(categories, "category_id", "edit-category-"+strconv.Itoa(t.ID), t.CategoryID, nil, false, nil),
								),
								html.Div(
									html.Class("form-check mb-3"),
									html.Input(
										html.Class("form-check-input"),
										html.Type("checkbox"),
										html.ID("edit-reviewed-"+strconv.Itoa(t.ID)),
										html.Name("reviewed"),
										func() g.Node {
											if t.Reviewed {
												return g.Attr("checked", "checked")
											}
											return nil
										}(),
									),
									html.Label(
										html.Class("form-check-label"),
										html.For("edit-reviewed-"+strconv.Itoa(t.ID)),
										g.Text("Reviewed"),
									),
								),
								html.Div(
									html.Class("d-flex justify-content-end mt-3"),
									html.Button(
										html.Type("button"),
										html.Class("btn btn-outline-primary"),
										html.DataAttr("bs-dismiss", "modal"),
										g.Attr("onclick", "setTimeout(function(){openAddRuleModalWithPrefill('"+strings.ReplaceAll(t.OriginalPayee, "'", "\\'")+"')}, 400);"),
										g.Text("Create Rule from this Transaction"),
									),
								),
								html.Div(
									html.Class("d-flex justify-content-end gap-2"),
									html.Button(html.Type("button"), html.Class("btn btn-secondary"), html.DataAttr("bs-dismiss", "modal"), g.Text("Cancel")),
									html.Button(html.Type("submit"), html.Class("btn btn-primary"), g.Text("Save Changes")),
								),
							),
						),
					),
				),
			),
		}),
	)
}

// TransactionCardWithModalSelectable renders a single transaction card and its modal for editing
func TransactionCardWithModalSelectable(t data.Transaction, categories []data.Category, modalErrMsg string) g.Node {
	cardID := "transaction-card-" + strconv.Itoa(t.ID)
	// Find category name
	var categoryName string
	if t.CategoryID == nil {
		categoryName = "Uncategorized"
	} else {
		for _, c := range categories {
			if c.ID == *t.CategoryID {
				categoryName = c.Name
				break
			}
		}
		if categoryName == "" {
			categoryName = "Uncategorized"
		}
	}
	return html.Div(
		html.Class("card mb-2"),
		html.ID(cardID),
		html.Div(
			html.Class("d-flex w-100 align-items-center py-3 gap-3 card-body"),
			// Checkbox on the left side
			html.Div(
				html.Class("d-flex align-items-center"),
				html.Input(
					html.Class("form-check-input transaction-select me-3"),
					html.Type("checkbox"),
					html.Name("transaction_ids"),
					html.Value(strconv.Itoa(t.ID)),
					g.Attr("onclick", "event.stopPropagation(); updateMarkReviewedBtn();"),
				),
			),
			// Clickable content area (everything except checkbox)
			html.Div(
				html.Class("flex-grow-1"),
				g.Attr("style", "cursor: pointer;"),
				g.Attr("hx-get", "/transactions/"+strconv.Itoa(t.ID)+"/edit"),
				g.Attr("hx-target", "#editTransactionModalBody"),
				g.Attr("hx-trigger", "click"),
				html.DataAttr("bs-toggle", "modal"),
				html.DataAttr("bs-target", "#editTransactionModal"),
				html.Div(
					html.Class("d-flex align-items-center gap-3"),
					// First column: Payee and Category stacked, text only
					html.Div(
						html.Class("d-flex flex-column flex-grow-1 gap-2"),
						html.Div(
							html.Class("fw-semibold"),
							g.Text(t.Payee),
						),
						html.Div(
							html.Class("text-muted small"),
							g.Text(categoryName),
						),
					),
					// Second column: Amount and Date stacked, right-aligned
					html.Div(
						html.Class("d-flex flex-column align-items-end gap-2"),
						html.Div(
							html.Class("fw-bold fs-5 mb-0"),
							g.Text(t.AmountDecimal()),
						),
						html.Div(
							html.Class("text-muted small mb-0"),
							g.Text(t.Date.Format("01/02/06")),
						),
					),
				),
			),
		),
	)
}

// Helper function for max
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Add after TransactionCardWithModalSelectable
func RenderTransactionListWithSelectableCards(w io.Writer, txs []data.Transaction, cats []data.Category) error {
	return html.Div(
		html.ID("transactions-list"),
		html.Class("list-group"),
		g.Group(func() []g.Node {
			var rows []g.Node
			for _, t := range txs {
				rows = append(rows, TransactionCardWithModalSelectable(t, cats, ""))
			}
			return rows
		}()),
	).Render(w)
}

// CategoriesHeaderSection renders the header with title and action toolbar
func CategoriesHeaderSection() g.Node {
	return html.Div(
		html.Class("col-12 col-md-8 px-2 mx-auto"),
		html.H1(html.Class("display-6 my-2 text-center"), g.Text("Categories")),
		// Toolbar: left group for future actions, right group for add button
		html.Div(
			html.Class("card p-2 mb-3 bg-body-tertiary border"),
			html.Div(
				html.Class("d-flex align-items-center gap-2"),
				// Left-aligned group for actions (empty for now, ready for future actions)
				html.Div(
					html.Class("d-flex align-items-center gap-2"),
				),
				// Right-aligned add button
				html.Button(
					html.Class("btn btn-primary ms-auto"),
					html.Type("button"),
					html.DataAttr("bs-toggle", "modal"),
					html.DataAttr("bs-target", "#addCategoryModal"),
					html.I(html.Class("bi bi-plus-circle me-1")),
					g.Text("Add"),
				),
			),
		),
	)
}

// CategoriesDirectoryNavigation renders the breadcrumbs and category list
func CategoriesDirectoryNavigation(visibleCategories []data.Category, allCategories []data.Category, currentParent *data.Category, breadcrumb []data.Category) g.Node {
	return html.Div(
		html.ID("categories-directory"),
		html.Class("col-12 col-md-8 px-2 mx-auto"),
		// Breadcrumb
		g.El("nav",
			html.Class("mb-3"),
			html.Aria("label", "breadcrumb"),
			html.Ol(
				html.Class("breadcrumb flex-wrap small mb-1"),
				g.Group(func() []g.Node {
					var crumbs []g.Node
					if len(breadcrumb) == 0 {
						// Root
						crumbs = append(crumbs, html.Li(
							html.Class("breadcrumb-item active"),
							g.Text("Root"),
						))
					} else {
						// Root link
						crumbs = append(crumbs, html.Li(
							html.Class("breadcrumb-item"),
							html.A(
								g.Attr("hx-get", "/categories/"),
								g.Attr("hx-target", "#categories-directory"),
								g.Attr("hx-swap", "outerHTML"),
								g.Attr("hx-push-url", "true"),
								g.Text("Root"),
							),
						))
						for i, cat := range breadcrumb {
							active := i == len(breadcrumb)-1
							if active {
								crumbs = append(crumbs, html.Li(
									html.Class("breadcrumb-item active"),
									g.Text(cat.Name),
								))
							} else {
								crumbs = append(crumbs, html.Li(
									html.Class("breadcrumb-item"),
									html.A(
										g.Attr("hx-get", "/categories/?parent_id="+strconv.Itoa(cat.ID)),
										g.Attr("hx-target", "#categories-directory"),
										g.Attr("hx-swap", "outerHTML"),
										g.Attr("hx-push-url", "true"),
										g.Text(cat.Name),
									),
								))
							}
						}
					}
					return crumbs
				}()),
			),
		),
		// Categories grid
		html.Div(
			html.ID("categories-list"),
			html.Class("row row-cols-1 row-cols-md-2 row-cols-lg-3 g-3"),
			g.Group(func() []g.Node {
				var cards []g.Node
				for _, c := range visibleCategories {
					cards = append(cards, html.Div(
						html.Class("col"),
						CategoryCard(c, allCategories),
					))
				}
				return cards
			}()),
		),
	)
}

// CategoryCardOnly renders just the category card without the modal
func CategoryCardOnly(c data.Category, allCategories []data.Category) g.Node {
	return html.Div(
		html.ID("category-card-"+strconv.Itoa(c.ID)),
		html.Class("card"),
		html.Div(
			html.Class("card-body d-flex align-items-center justify-content-between py-3 px-3"),
			html.A(
				html.Class("text-decoration-none fw-semibold"),
				g.Attr("hx-get", "/categories/?parent_id="+strconv.Itoa(c.ID)),
				g.Attr("hx-target", "#categories-directory"),
				g.Attr("hx-swap", "outerHTML"),
				g.Attr("hx-push-url", "true"),
				g.Text(c.Name),
			),
			html.Div(
				html.Class("btn-group btn-group-sm"),
				html.Button(
					html.Class("btn btn-secondary"),
					html.Type("button"),
					html.DataAttr("bs-toggle", "modal"),
					html.DataAttr("bs-target", "#editCategoryModal-"+strconv.Itoa(c.ID)),
					g.Text("Edit"),
				),
				html.Button(
					html.Class("btn btn-danger"),
					html.Type("button"),
					g.Attr("hx-delete", "/categories/"+strconv.Itoa(c.ID)),
					g.Attr("hx-target", "#category-card-"+strconv.Itoa(c.ID)),
					g.Attr("hx-swap", "outerHTML"),
					g.Attr("hx-confirm", "Are you sure you want to delete this category?"),
					g.Text("Delete"),
				),
			),
		),
	)
}

// CategoryCard renders a single category card
func CategoryCard(c data.Category, allCategories []data.Category) g.Node {
	return g.Group([]g.Node{
		CategoryCardOnly(c, allCategories),
		// Edit Category Modal
		html.Div(
			html.Class("modal fade"),
			html.ID("editCategoryModal-"+strconv.Itoa(c.ID)),
			html.DataAttr("tabindex", "-1"),
			html.DataAttr("aria-labelledby", "editCategoryModalLabel-"+strconv.Itoa(c.ID)),
			html.DataAttr("aria-hidden", "true"),
			html.Div(
				html.Class("modal-dialog"),
				html.Div(
					html.Class("modal-content"),
					html.Div(
						html.Class("modal-header"),
						html.H5(html.Class("modal-title"), html.ID("editCategoryModalLabel-"+strconv.Itoa(c.ID)), g.Text("Edit Category")),
						html.Button(
							html.Class("btn-close"),
							html.Type("button"),
							html.DataAttr("bs-dismiss", "modal"),
							html.DataAttr("aria-label", "Close"),
						),
					),
					html.Div(
						html.Class("modal-body"),
						g.El("form",
							html.Action("/categories/"+strconv.Itoa(c.ID)),
							html.Method("PUT"),
							g.Attr("hx-put", "/categories/"+strconv.Itoa(c.ID)),
							g.Attr("hx-target", "#category-card-"+strconv.Itoa(c.ID)),
							g.Attr("hx-swap", "outerHTML"),
							g.Attr("hx-on::after-request", `if (event.target === this) { bootstrap.Modal.getInstance(document.getElementById('editCategoryModal-`+strconv.Itoa(c.ID)+`')).hide(); }`),
							html.Div(
								html.Class("mb-3"),
								html.Label(html.Class("form-label"), html.For("edit-name-"+strconv.Itoa(c.ID)), g.Text("Name")),
								html.Input(html.Type("text"), html.Name("name"), html.ID("edit-name-"+strconv.Itoa(c.ID)), html.Value(c.Name), html.Class("form-control"), html.Required()),
							),
							html.Div(
								html.Class("mb-3"),
								html.Label(html.Class("form-label"), html.For("edit-parent-"+strconv.Itoa(c.ID)), g.Text("Parent Category")),
								CategoryBootstrapDropdown(allCategories, "parent_id", "edit-parent-"+strconv.Itoa(c.ID), c.ParentID, &c.ID, true, &c.ID),
							),
							html.Div(
								html.Class("d-flex justify-content-end gap-2"),
								html.Button(html.Type("button"), html.Class("btn btn-secondary"), html.DataAttr("bs-dismiss", "modal"), g.Text("Cancel")),
								html.Button(html.Type("submit"), html.Class("btn btn-primary"), g.Text("Save Changes")),
							),
						),
					),
				),
			),
		),
	})
}

// AddCategoryModal renders the modal for adding a new category
func AddCategoryModal(allCategories []data.Category, currentParentID *int) g.Node {
	return html.Div(
		html.Class("modal fade"),
		html.ID("addCategoryModal"),
		html.DataAttr("tabindex", "-1"),
		html.DataAttr("aria-labelledby", "addCategoryModalLabel"),
		html.DataAttr("aria-hidden", "true"),
		html.Div(
			html.Class("modal-dialog"),
			html.Div(
				html.Class("modal-content"),
				html.Div(
					html.Class("modal-header"),
					html.H5(html.Class("modal-title"), html.ID("addCategoryModalLabel"), g.Text("Add Category")),
					html.Button(
						html.Class("btn-close"),
						html.Type("button"),
						html.DataAttr("bs-dismiss", "modal"),
						html.DataAttr("aria-label", "Close"),
					),
				),
				html.Div(
					html.Class("modal-body"),
					g.El("form",
						html.Action("/categories/"),
						html.Method("POST"),
						g.Attr("hx-post", "/categories/"),
						g.Attr("hx-target", "#categories-directory"),
						g.Attr("hx-swap", "outerHTML"),
						g.Attr("hx-on::after-request", "if (event.target === this) { bootstrap.Modal.getInstance(document.getElementById('addCategoryModal')).hide(); }"),
						html.Div(
							html.Class("mb-3"),
							html.Label(html.Class("form-label"), html.For("add-name"), g.Text("Name")),
							html.Input(html.Type("text"), html.Name("name"), html.ID("add-name"), html.Class("form-control"), html.Required()),
						),
						html.Div(
							html.Class("mb-3"),
							html.Label(html.Class("form-label"), html.For("add-parent"), g.Text("Parent Category")),
							CategoryBootstrapDropdown(allCategories, "parent_id", "add-parent", currentParentID, nil, false, nil),
						),
						html.Div(
							html.Class("d-flex justify-content-end gap-2"),
							html.Button(html.Type("button"), html.Class("btn btn-secondary"), html.DataAttr("bs-dismiss", "modal"), g.Text("Cancel")),
							html.Button(html.Type("submit"), html.Class("btn btn-primary"), g.Text("Add Category")),
						),
					),
				),
			),
		),
	)
}

func CategoriesDirectoryContent(visibleCategories []data.Category, allCategories []data.Category, currentParent *data.Category, breadcrumb []data.Category) g.Node {
	var currentParentID *int
	if currentParent != nil {
		currentParentID = &currentParent.ID
	}

	return g.Group([]g.Node{
		CategoriesHeaderSection(),
		CategoriesDirectoryNavigation(visibleCategories, allCategories, currentParent, breadcrumb),
		AddCategoryModal(allCategories, currentParentID),
	})
}

func CategoriesDirectoryPage(visibleCategories []data.Category, allCategories []data.Category, currentParent *data.Category, breadcrumb []data.Category) g.Node {
	var currentParentID *int
	if currentParent != nil {
		currentParentID = &currentParent.ID
	}

	return g.Group([]g.Node{
		html.Div(
			html.Class("row"),
			CategoriesHeaderSection(),
			CategoriesDirectoryNavigation(visibleCategories, allCategories, currentParent, breadcrumb),
			AddCategoryModal(allCategories, currentParentID),
		),
	})
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
				if !descendants[cat.ID] {
					descendants[cat.ID] = true
					queue = append(queue, cat.ID)
				}
			}
		}
	}

	return descendants
}

// Rules templates

// RulesPage renders the rules management page
func RulesPage(rules []data.Rule, categories []data.Category) g.Node {
	return html.Div(
		html.Class("row"),
		html.Div(
			html.Class("col-12 col-md-8 px-2"),
			html.H1(html.Class("display-6 my-2 text-center"), g.Text("Transaction Rules")),
			// Toolbar
			html.Div(
				html.Class("card p-2 mb-3 bg-body-tertiary border"),
				html.Div(
					html.Class("d-flex align-items-center gap-2"),
					html.Button(
						html.Class("btn btn-primary"),
						html.Type("button"),
						html.DataAttr("bs-toggle", "modal"),
						html.DataAttr("bs-target", "#addRuleModal"),
						g.Text("Add Rule"),
					),
				),
			),
			// Rules list
			html.Div(
				html.ID("rules-list"),
				html.Class("list-group"),
				g.Group(func() []g.Node {
					var rows []g.Node
					for _, rule := range rules {
						rows = append(rows, RuleCard(rule, categories))
					}
					return rows
				}()),
			),
		),
		// Add Rule Modal
		AddRuleModal(categories),
		// Edit Rule Modal
		EditRuleModal(categories),
		// JavaScript functions for rule actions
		html.Script(g.Raw(`
			function openEditRuleModal(ruleId) {
				const modal = document.getElementById('editRuleModal');
				if (!modal) return;
				
				// Load the edit form via HTMX
				fetch('/rules/' + ruleId + '/edit')
					.then(response => response.text())
					.then(html => {
						document.getElementById('editRuleModalBody').innerHTML = html;
						// Process HTMX attributes on the newly loaded content
						if (typeof htmx !== 'undefined') {
							htmx.process(document.getElementById('editRuleModalBody'));
						}
						const bsModal = bootstrap.Modal.getOrCreateInstance(modal);
						bsModal.show();
					})
					.catch(error => console.error('Error loading edit form:', error));
			}
			
			function toggleRule(ruleId) {
				if (confirm('Are you sure you want to toggle this rule?')) {
					fetch('/rules/' + ruleId + '/toggle', {
						method: 'PATCH',
						headers: {
							'Content-Type': 'application/json',
						},
					})
					.then(response => response.text())
					.then(html => {
						// Find the specific rule card and replace it
						const ruleCard = document.querySelector('[data-rule-id="' + ruleId + '"]');
						if (ruleCard) {
							// Parse the HTML and extract just the rule card
							const parser = new DOMParser();
							const doc = parser.parseFromString(html, 'text/html');
							const newRuleCard = doc.querySelector('[data-rule-id="' + ruleId + '"]');
							if (newRuleCard) {
								ruleCard.outerHTML = newRuleCard.outerHTML;
							}
						}
					})
					.catch(error => console.error('Error toggling rule:', error));
				}
			}
			
			function deleteRule(ruleId) {
				if (confirm('Are you sure you want to delete this rule? This action cannot be undone.')) {
					fetch('/rules/' + ruleId, {
						method: 'DELETE',
						headers: {
							'Content-Type': 'application/json',
						},
					})
					.then(response => response.text())
					.then(html => {
						document.getElementById('rules-list').outerHTML = html;
					})
					.catch(error => console.error('Error deleting rule:', error));
				}
			}
		`)),
	)
}

// RuleCard renders a single rule card
func RuleCard(rule data.Rule, categories []data.Category) g.Node {
	return html.Div(
		html.Class("card mb-3"),
		g.Attr("data-rule-id", strconv.Itoa(rule.ID)),
		html.Div(
			html.Class("card-body"),
			html.Div(
				html.Class("d-flex"),
				// Left side - Rule content (expanding)
				html.Div(
					html.Class("flex-grow-1 pe-3"),
					html.Div(
						html.Class("mb-3"),
						html.H5(html.Class("card-title mb-1"), g.Text(rule.Name)),
						html.Div(
							html.Class("d-flex align-items-center gap-2"),
							func() g.Node {
								if rule.Active {
									return html.Span(html.Class("badge bg-success"), g.Text("Active"))
								}
								return html.Span(html.Class("badge bg-secondary"), g.Text("Inactive"))
							}(),
							html.Span(html.Class("badge bg-primary"), g.Text(fmt.Sprintf("Priority: %d", rule.Priority))),
						),
					),
					html.Div(
						html.Class("mb-3"),
						html.H6(html.Class("card-subtitle mb-2"), g.Text("Conditions:")),
						html.Ul(
							html.Class("list-unstyled"),
							func() g.Node {
								var nodes []g.Node
								for _, condition := range rule.Conditions {
									nodes = append(nodes, html.Li(
										g.Text(fmt.Sprintf("%s %s '%s'", condition.Field, condition.Operator, condition.Value)),
									))
								}
								return g.Group(nodes)
							}(),
						),
					),
					html.Div(
						html.Class("mb-0"),
						html.H6(html.Class("card-subtitle mb-2"), g.Text("Actions:")),
						html.Ul(
							html.Class("list-unstyled"),
							func() g.Node {
								if rule.NewPayee != nil {
									return html.Li(g.Text(fmt.Sprintf("Set payee to '%s'", *rule.NewPayee)))
								}
								return html.Li(g.Text("No payee change"))
							}(),
							func() g.Node {
								if rule.CategoryID != nil {
									categoryName := "Unknown"
									for _, cat := range categories {
										if cat.ID == *rule.CategoryID {
											categoryName = cat.Name
											break
										}
									}
									return html.Li(g.Text(fmt.Sprintf("Set category to '%s'", categoryName)))
								}
								return html.Li(g.Text("No category change"))
							}(),
						),
					),
				),
				// Right side - Action buttons (compact, centrally aligned)
				html.Div(
					html.Class("d-flex flex-column justify-content-center align-items-center"),
					html.Div(
						html.Class("btn-group"),
						html.Button(
							html.Type("button"),
							html.Class("btn btn-primary"),
							g.Attr("onclick", fmt.Sprintf("openEditRuleModal(%d)", rule.ID)),
							g.Text("Edit"),
						),
						html.Button(
							html.Type("button"),
							html.Class("btn btn-primary dropdown-toggle dropdown-toggle-split"),
							g.Attr("data-bs-toggle", "dropdown"),
							g.Attr("aria-expanded", "false"),
							g.Attr("aria-label", "Toggle dropdown"),
						),
						html.Ul(
							html.Class("dropdown-menu"),
							html.Li(
								html.Class("dropdown-item"),
								html.Button(
									html.Type("button"),
									func() g.Node {
										if rule.Active {
											// Disable button - grey
											return html.Class("btn btn-secondary w-100")
										} else {
											// Enable button - green
											return html.Class("btn btn-success w-100")
										}
									}(),
									g.Attr("onclick", fmt.Sprintf("toggleRule(%d)", rule.ID)),
									g.Text(func() string {
										if rule.Active {
											return "Disable"
										}
										return "Enable"
									}()),
								),
							),
							html.Li(
								html.Class("dropdown-item"),
								html.Button(
									html.Type("button"),
									html.Class("btn btn-danger w-100"),
									g.Attr("onclick", fmt.Sprintf("deleteRule(%d)", rule.ID)),
									g.Text("Delete"),
								),
							),
						),
					),
				),
			),
		),
	)
}

// AddRuleModal renders the modal for adding a new rule
func AddRuleModal(categories []data.Category) g.Node {
	return html.Div(
		html.Class("modal fade"),
		html.ID("addRuleModal"),
		html.DataAttr("tabindex", "-1"),
		html.DataAttr("aria-labelledby", "addRuleModalLabel"),
		html.DataAttr("aria-hidden", "true"),
		html.Div(
			html.Class("modal-dialog modal-lg"),
			html.Div(
				html.Class("modal-content"),
				html.Div(
					html.Class("modal-header"),
					html.H5(html.Class("modal-title"), html.ID("addRuleModalLabel"), g.Text("Add Rule")),
					html.Button(
						html.Class("btn-close"),
						html.Type("button"),
						html.DataAttr("bs-dismiss", "modal"),
						html.DataAttr("aria-label", "Close"),
					),
				),
				html.Div(
					html.Class("modal-body"),
					g.El("form",
						html.Action("/rules/"),
						html.Method("POST"),
						g.Attr("hx-post", "/rules/"),
						g.Attr("hx-target", "#rules-list"),
						g.Attr("hx-swap", "outerHTML"),
						g.Attr("hx-on::after-request", "if (event.target === this) { bootstrap.Modal.getInstance(document.getElementById('addRuleModal')).hide(); }"),
						RuleFormFields(categories, nil, "add"),
						// Add checkbox for running rule immediately (only in AddRuleModal)
						html.Div(
							html.Class("form-check mb-3"),
							html.Input(
								html.Class("form-check-input"),
								html.Type("checkbox"),
								html.ID("run-immediately"),
								html.Name("run_immediately"),
							),
							html.Label(
								html.Class("form-check-label"),
								html.For("run-immediately"),
								g.Text("Run this rule immediately on all existing transactions"),
							),
						),
						html.Div(
							html.Class("d-flex justify-content-end gap-2"),
							html.Button(html.Type("button"), html.Class("btn btn-secondary"), html.DataAttr("bs-dismiss", "modal"), g.Text("Cancel")),
							html.Button(html.Type("submit"), html.Class("btn btn-primary"), g.Text("Add Rule")),
						),
					),
				),
			),
		),
		// Modal reset script
		html.Script(g.Raw(`
			document.getElementById('addRuleModal').addEventListener('show.bs.modal', function() {
				// Reset form when modal opens
				const form = this.querySelector('form');
				if (form) form.reset();
				
				// Reset conditions to just one empty condition
				const container = document.getElementById('add-conditions-container');
				if (container) {
					container.innerHTML = '';
					// Add one empty condition
					const template = document.getElementById('add-condition-card-template');
					if (template) {
						const clone = template.content.cloneNode(true);
						clone.querySelectorAll('[name]').forEach(el => {
							el.name = el.name.replace('__INDEX__', '0');
						});
						clone.querySelectorAll('[for]').forEach(el => {
							el.setAttribute('for', el.getAttribute('for').replace('__INDEX__', '0'));
						});
						container.appendChild(clone);
					}
				}
			});
		`)),
	)
}

// EditRuleModal renders the modal for editing a rule
func EditRuleModal(categories []data.Category) g.Node {
	return html.Div(
		html.Class("modal fade"),
		html.ID("editRuleModal"),
		html.DataAttr("tabindex", "-1"),
		html.DataAttr("aria-labelledby", "editRuleModalLabel"),
		html.DataAttr("aria-hidden", "true"),
		html.Div(
			html.Class("modal-dialog modal-lg"),
			html.Div(
				html.Class("modal-content"),
				html.Div(
					html.Class("modal-header"),
					html.H5(html.Class("modal-title"), html.ID("editRuleModalLabel"), g.Text("Edit Rule")),
					html.Button(
						html.Class("btn-close"),
						html.Type("button"),
						html.DataAttr("bs-dismiss", "modal"),
						html.DataAttr("aria-label", "Close"),
					),
				),
				html.Div(
					html.ID("editRuleModalBody"),
					html.Class("modal-body"),
					// Content will be loaded via HTMX
				),
			),
		),
	)
}

// RuleFormFields renders the form fields for a rule
func RuleFormFields(categories []data.Category, rule *data.Rule, modalType string) g.Node {
	return g.Group([]g.Node{
		html.Div(
			html.Class("mb-3"),
			html.Label(html.Class("form-label"), html.For("rule-name"), g.Text("Rule Name")),
			html.Input(
				html.Type("text"),
				html.Name("name"),
				html.ID("rule-name"),
				html.Class("form-control"),
				html.Required(),
				func() g.Node {
					if rule != nil {
						return html.Value(rule.Name)
					}
					return nil
				}(),
			),
		),
		html.Div(
			html.Class("mb-3"),
			html.Label(html.Class("form-label"), html.For("rule-priority"), g.Text("Priority")),
			html.Input(
				html.Type("number"),
				html.Name("priority"),
				html.ID("rule-priority"),
				html.Class("form-control"),
				html.Value("0"),
				func() g.Node {
					if rule != nil {
						return html.Value(strconv.Itoa(rule.Priority))
					}
					return nil
				}(),
			),
		),
		html.Div(
			html.Class("mb-3"),
			html.Div(
				html.Class("d-flex justify-content-between align-items-center mb-2"),
				html.Label(html.Class("form-label mb-0"), g.Text("Payee Conditions")),
				html.Button(
					html.Type("button"),
					html.Class("btn btn-sm btn-outline-secondary"),
					g.Attr("onclick", "addConditionRow('"+modalType+"')"),
					g.Text("Add Condition"),
				),
			),
			html.Div(
				html.ID("conditions-container-"+modalType),
				g.Group(func() []g.Node {
					if rule != nil && len(rule.Conditions) > 0 {
						var conditions []g.Node
						for i, condition := range rule.Conditions {
							conditions = append(conditions, ConditionCard(i, condition))
						}
						return conditions
					}
					return []g.Node{ConditionCard(0, data.RuleCondition{})}
				}()),
				// Add the hidden template for a condition card
				g.El("template",
					g.Attr("id", "condition-card-template-"+modalType),
					html.Div(
						html.Class("card mb-2"),
						html.Div(
							html.Class("card-body"),
							html.Div(
								html.Class("mb-3"),
								html.Label(html.Class("form-label"), html.For("condition-__INDEX__-operator"), g.Text("Operator")),
								html.Select(
									html.Name("conditions[__INDEX__][operator]"),
									html.Class("form-select category-dropdown"),
									html.Required(),
									html.Option(html.Value("equals"), g.Text("Equals")),
									html.Option(html.Value("contains"), g.Text("Contains")),
									html.Option(html.Value("begins"), g.Text("Begins with")),
									html.Option(html.Value("ends"), g.Text("Ends with")),
								),
							),
							html.Div(
								html.Class("mb-3"),
								html.Label(html.Class("form-label"), html.For("condition-__INDEX__-value"), g.Text("Value")),
								html.Input(
									html.Type("text"),
									html.Name("conditions[__INDEX__][value]"),
									html.Class("form-control"),
									html.Placeholder("Value"),
									html.Required(),
								),
							),
							html.Div(
								html.Class("d-flex justify-content-center"),
								html.Button(
									html.Type("button"),
									html.Class("btn btn-sm btn-outline-danger"),
									g.Attr("onclick", "this.closest('.card').remove()"),
									g.Text("Remove"),
								),
							),
						),
					),
				),
			),
		),
		html.Div(
			html.Class("mb-3"),
			html.Label(html.Class("form-label"), html.For("rule-new-payee"), g.Text("New Payee (optional)")),
			html.Input(
				html.Type("text"),
				html.Name("new_payee"),
				html.ID("rule-new-payee"),
				html.Class("form-control"),
				func() g.Node {
					if rule != nil && rule.NewPayee != nil {
						return html.Value(*rule.NewPayee)
					}
					return nil
				}(),
			),
		),
		html.Div(
			html.Class("mb-3"),
			html.Label(html.Class("form-label"), html.For("rule-category"), g.Text("Category (optional)")),
			CategoryBootstrapDropdown(categories, "category_id", "rule-category", func() *int {
				if rule != nil {
					return rule.CategoryID
				}
				return nil
			}(), nil, false, nil),
		),
		// Add checkbox for running rule on existing transactions (only in edit mode)
		func() g.Node {
			if modalType == "edit" {
				return html.Div(
					html.Class("form-check mb-3"),
					html.Input(
						html.Class("form-check-input"),
						html.Type("checkbox"),
						html.ID("run-on-existing-edit"),
						html.Name("run_on_existing"),
					),
					html.Label(
						html.Class("form-check-label"),
						html.For("run-on-existing-edit"),
						g.Text("Apply this rule to all existing transactions"),
					),
				)
			}
			return nil
		}(),
		html.Script(g.Raw(`
			function addConditionRow(modalType) {
				const container = document.getElementById('conditions-container-' + modalType);
				if (!container) return;
				const conditionCount = container.querySelectorAll('.card').length;
				const template = document.getElementById('condition-card-template-' + modalType);
				if (!template) return;
				const clone = template.content.cloneNode(true);
				clone.querySelectorAll('[name]').forEach(el => {
					el.name = el.name.replace('__INDEX__', conditionCount);
				});
				clone.querySelectorAll('[for]').forEach(el => {
					el.setAttribute('for', el.getAttribute('for').replace('__INDEX__', conditionCount));
				});
				container.insertBefore(clone, container.lastElementChild);
			}
		`)),
	})
}

// ConditionCard renders a single condition card
func ConditionCard(index int, condition data.RuleCondition) g.Node {
	return html.Div(
		html.Class("card mb-2"),
		html.Div(
			html.Class("card-body"),
			html.Div(
				html.Class("mb-3"),
				html.Label(html.Class("form-label"), html.For(fmt.Sprintf("condition-%d-operator", index)), g.Text("Operator")),
				html.Select(
					html.Name(fmt.Sprintf("conditions[%d][operator]", index)),
					html.Class("form-select category-dropdown"),
					html.Required(),
					html.Option(html.Value("equals"), g.Text("Equals"), func() g.Node {
						if condition.Operator == "equals" {
							return g.Attr("selected", "selected")
						}
						return nil
					}()),
					html.Option(html.Value("contains"), g.Text("Contains"), func() g.Node {
						if condition.Operator == "contains" {
							return g.Attr("selected", "selected")
						}
						return nil
					}()),
					html.Option(html.Value("begins"), g.Text("Begins with"), func() g.Node {
						if condition.Operator == "begins" {
							return g.Attr("selected", "selected")
						}
						return nil
					}()),
					html.Option(html.Value("ends"), g.Text("Ends with"), func() g.Node {
						if condition.Operator == "ends" {
							return g.Attr("selected", "selected")
						}
						return nil
					}()),
				),
			),
			html.Div(
				html.Class("mb-3"),
				html.Label(html.Class("form-label"), html.For(fmt.Sprintf("condition-%d-value", index)), g.Text("Value")),
				html.Input(
					html.Type("text"),
					html.Name(fmt.Sprintf("conditions[%d][value]", index)),
					html.Class("form-control"),
					html.Placeholder("Value"),
					html.Required(),
					func() g.Node {
						if condition.Value != "" {
							return html.Value(condition.Value)
						}
						return nil
					}(),
				),
			),
			html.Div(
				html.Class("d-flex justify-content-center"),
				html.Button(
					html.Type("button"),
					html.Class("btn btn-sm btn-outline-danger"),
					g.Attr("onclick", "this.closest('.card').remove()"),
					g.Text("Remove"),
				),
			),
		),
	)
}

// RenderRulesList renders the rules list for HTMX responses
func RenderRulesList(w io.Writer, rules []data.Rule, categories []data.Category) error {
	return html.Div(
		html.ID("rules-list"),
		html.Class("list-group"),
		g.Group(func() []g.Node {
			var rows []g.Node
			for _, rule := range rules {
				rows = append(rows, RuleCard(rule, categories))
			}
			return rows
		}()),
	).Render(w)
}

// CategoryBootstrapDropdown renders a hierarchical category dropdown using Bootstrap dropdowns
func CategoryBootstrapDropdown(categories []data.Category, selectName, selectID string, selectedID *int, currentNodeID *int, showCurrent bool, disableDescendantsOf *int) g.Node {
	flat := flattenCategories(categories, nil, 0)
	var descendants map[int]bool
	if disableDescendantsOf != nil {
		descendants = getDescendants(categories, *disableDescendantsOf)
	}

	// Find the selected category name without indentation for display
	var selectedDisplayName string
	var selectedValue string
	if selectedID != nil {
		selectedValue = strconv.Itoa(*selectedID)
		for _, item := range flat {
			if item.Cat.ID == *selectedID {
				selectedDisplayName = item.Cat.Name
				break
			}
		}
	} else {
		selectedValue = ""
		selectedDisplayName = "None (top-level)"
	}

	// Escape strings for JavaScript
	escapedSelectedValue := strings.ReplaceAll(selectedValue, "'", "\\'")
	escapedSelectedDisplay := strings.ReplaceAll(selectedDisplayName, "'", "\\'")

	return html.Div(
		html.Class("dropdown"),
		g.Attr("x-data", fmt.Sprintf(`{
			selectedValue: '%s',
			selectedDisplay: '%s',
			selectCategory(value, display) {
				this.selectedValue = value;
				this.selectedDisplay = display;
				// Immediately update the hidden input
				const hiddenInput = this.$el.querySelector('input[name="%s"]');
				if (hiddenInput) {
					hiddenInput.value = value || '';
				}
			},
			init() {
				// Set initial value on the hidden input
				const hiddenInput = this.$el.querySelector('input[name="%s"]');
				if (hiddenInput) {
					hiddenInput.value = this.selectedValue || '';
				}
				// Watch for changes and sync Alpine state
				this.$watch('selectedValue', (value) => {
					const hiddenInput = this.$el.querySelector('input[name="%s"]');
					if (hiddenInput) {
						hiddenInput.value = value || '';
					}
				});
			}
		}`, escapedSelectedValue, escapedSelectedDisplay, selectName, selectName, selectName)),
		// Dropdown button
		html.Button(
			html.Class("btn border dropdown-toggle w-100 d-flex justify-content-between align-items-center"),
			html.Type("button"),
			html.ID(selectID),
			html.DataAttr("bs-toggle", "dropdown"),
			html.DataAttr("bs-auto-close", "true"),
			g.Attr("x-text", "selectedDisplay"),
		),
		// Dropdown menu
		html.Ul(
			html.Class("dropdown-menu w-100"),
			g.Attr("style", "max-height: 300px; overflow-y: auto;"),
			// None option
			html.Li(
				html.A(
					html.Class("dropdown-item"),
					html.Href("#"),
					g.Attr("x-on:click", "selectCategory('', 'None (top-level)'); $event.preventDefault()"),
					g.Text("None (top-level)"),
				),
			),
			// Divider
			html.Li(html.Hr(html.Class("dropdown-divider"))),
			// Category options
			g.Group(func() []g.Node {
				var items []g.Node
				for _, item := range flat {
					cat := item.Cat

					// Create proper indentation
					var indentStyle string
					var indentText string
					if item.Level > 0 {
						// Use inline styles for reliable indentation
						indentStyle = fmt.Sprintf("padding-left: %dpx;", item.Level*20)
						// Add visual tree indicators
						indentText = strings.Repeat("  ", item.Level) + "└─ "
					}

					// Check if this category should be disabled
					isDisabled := false
					if showCurrent && currentNodeID != nil && cat.ID == *currentNodeID {
						isDisabled = true
					}
					if descendants != nil && descendants[cat.ID] {
						isDisabled = true
					}

					itemClass := "dropdown-item"
					if isDisabled {
						itemClass += " disabled"
					}

					// Add visual styling for different levels
					if item.Level == 0 {
						itemClass += " fw-semibold" // Top-level categories are bold
					} else {
						itemClass += " text-muted" // Sub-categories are muted
					}

					// Create the click handler for this category
					clickHandler := ""
					if !isDisabled {
						// Escape the category name for JavaScript
						escapedName := strings.ReplaceAll(cat.Name, "'", "\\'")
						clickHandler = fmt.Sprintf("selectCategory('%d', '%s'); $event.preventDefault()", cat.ID, escapedName)
					}

					items = append(items, html.Li(
						html.A(
							html.Class(itemClass),
							g.Attr("style", indentStyle),
							html.Href("#"),
							func() g.Node {
								if clickHandler != "" {
									return g.Attr("x-on:click", clickHandler)
								}
								return nil
							}(),
							g.Text(indentText+func() string {
								if showCurrent && currentNodeID != nil && cat.ID == *currentNodeID {
									return cat.Name + " ★"
								}
								return cat.Name
							}()),
						),
					))
				}
				return items
			}()),
		),
		// Hidden input to store the selected value
		html.Input(
			html.Type("hidden"),
			html.Name(selectName),
			html.Value(selectedValue),
			g.Attr("x-model", "selectedValue"),
			g.Attr("x-bind:value", "selectedValue"),
		),
	)
}

// RecurringTransactionsPage renders the recurring transactions management page
func RecurringTransactionsPage(rts []data.RecurringTransaction, categories []data.Category, financialAccounts []data.FinancialAccount, budgetPlans []data.BudgetPlan, selectedBudgetPlanID int) g.Node {
	return html.Div(
		html.Class("row"),
		html.Div(
			html.Class("col-12 col-md-8 px-2"),
			html.H1(html.Class("display-6 my-2 text-center"), g.Text("Recurring Transactions")),
			// Toolbar
			html.Div(
				html.Class("card p-2 mb-3 bg-body-tertiary border"),
				html.Div(
					html.Class("d-flex align-items-center gap-2"),
					html.Button(
						html.Class("btn btn-primary"),
						html.Type("button"),
						html.DataAttr("bs-toggle", "modal"),
						html.DataAttr("bs-target", "#addRecurringTransactionModal"),
						g.Text("Add Recurring Transaction"),
					),
				),
			),
			// Recurring transactions list
			html.Div(
				html.ID("recurring-transactions-list"),
				html.Class("list-group"),
				g.Group(func() []g.Node {
					var rows []g.Node
					for _, rt := range rts {
						rows = append(rows, RecurringTransactionCard(rt, categories))
					}
					return rows
				}()),
			),
		),
		// Add Recurring Transaction Modal
		AddRecurringTransactionModal(categories, financialAccounts, budgetPlans, selectedBudgetPlanID),
		// Edit Recurring Transaction Modal
		EditRecurringTransactionModal(categories),
		// JavaScript functions for recurring transaction actions
		html.Script(g.Raw(`
			function openEditRecurringTransactionModal(rtId) {
				const modal = document.getElementById('editRecurringTransactionModal');
				if (!modal) return;
				
				// Load the edit form via HTMX
				fetch('/bills/' + rtId + '/edit')
					.then(response => response.text())
					.then(html => {
						document.getElementById('editRecurringTransactionModalBody').innerHTML = html;
						// Process HTMX attributes on the newly loaded content
						if (typeof htmx !== 'undefined') {
							htmx.process(document.getElementById('editRecurringTransactionModalBody'));
						}
						const bsModal = bootstrap.Modal.getOrCreateInstance(modal);
						bsModal.show();
					})
					.catch(error => console.error('Error loading edit form:', error));
			}
			
			function archiveRecurringTransaction(rtId) {
				const endDate = prompt('Enter the last recurrence date for this recurring transaction (YYYY-MM-DD):');
				if (endDate && endDate.match(/^\d{4}-\d{2}-\d{2}$/)) {
					const formData = new FormData();
					formData.append('end_date', endDate);
					
					fetch('/bills/' + rtId + '/archive', {
						method: 'POST',
						headers: {
							'HX-Request': 'true',
						},
						body: formData,
					})
					.then(response => {
						if (response.ok) {
							return response.text();
						} else {
							throw new Error('Error archiving recurring transaction');
						}
					})
					.then(html => {
						// Replace the recurring transactions list
						document.getElementById('recurring-transactions-list').outerHTML = html;
					})
					.catch(error => {
						console.error('Error archiving recurring transaction:', error);
						alert('Failed to archive recurring transaction. Please try again.');
					});
				} else if (endDate) {
					alert('Invalid date format. Please use YYYY-MM-DD format.');
				}
			}
			
			function deleteRecurringTransaction(rtId) {
				if (confirm('Are you sure you want to delete this recurring transaction? This action cannot be undone.')) {
					fetch('/bills/' + rtId, {
						method: 'DELETE',
						headers: {
							'Content-Type': 'application/json',
							'HX-Request': 'true',
						},
					})
					.then(response => {
						if (response.ok) {
							// Find and remove the specific recurring transaction card
							const rtCard = document.querySelector('[data-recurring-transaction-id="' + rtId + '"]');
							if (rtCard) {
								rtCard.remove();
							}
						} else {
							console.error('Error deleting recurring transaction:', response.statusText);
						}
					})
					.catch(error => console.error('Error deleting recurring transaction:', error));
				}
			}
		`)),
	)
}

// RecurringTransactionCard renders a single recurring transaction card
func RecurringTransactionCard(rt data.RecurringTransaction, categories []data.Category) g.Node {
	typeBadgeClass := "badge bg-danger"
	typeBadgeText := "Expense"
	counterpartyLabel := "Payee"
	if rt.IsIncome() {
		typeBadgeClass = "badge bg-success"
		typeBadgeText = "Income"
		counterpartyLabel = "Payer"
	}

	return html.Div(
		html.Class("card mb-3"),
		g.Attr("data-recurring-transaction-id", strconv.Itoa(rt.ID)),
		html.Div(
			html.Class("card-body"),
			html.Div(
				html.Class("d-flex"),
				// Left side - Recurring transaction content
				html.Div(
					html.Class("flex-grow-1 pe-3"),
					html.Div(
						html.Class("mb-3"),
						html.Div(
							html.Class("d-flex align-items-center gap-2 mb-1"),
							html.H5(html.Class("card-title mb-0"), g.Text(rt.Name)),
							html.Span(
								html.Class(typeBadgeClass),
								g.Text(typeBadgeText),
							),
						),
						html.P(html.Class("mb-1 text-muted"), g.Text(fmt.Sprintf("%s: %s", counterpartyLabel, rt.Counterparty))),
						func() g.Node {
							if rt.CategoryID != nil {
								categoryName := "Unknown"
								for _, cat := range categories {
									if cat.ID == *rt.CategoryID {
										categoryName = cat.Name
										break
									}
								}
								return html.P(html.Class("mb-1 text-muted"), g.Text(fmt.Sprintf("Category: %s", categoryName)))
							}
							return html.P(html.Class("mb-1 text-muted"), g.Text("Category: None"))
						}(),
						html.P(html.Class("mb-1"), g.Text(fmt.Sprintf("Expected Amount: $%s", rt.ExpectedAmountDecimal()))),
						html.P(html.Class("mb-1 text-muted"), g.Text(fmt.Sprintf("Tolerance: $%s", rt.ToleranceDecimal()))),
						html.P(html.Class("mb-0"), g.Text(fmt.Sprintf("Recurrence: %s", rt.RecurrenceDisplay()))),
					),
				),
				// Right side - Action buttons
				html.Div(
					html.Class("d-flex flex-column justify-content-center align-items-center"),
					html.Div(
						html.Class("btn-group"),
						html.Button(
							html.Type("button"),
							html.Class("btn btn-primary"),
							g.Attr("onclick", fmt.Sprintf("openEditRecurringTransactionModal(%d)", rt.ID)),
							g.Text("Edit"),
						),
						html.Button(
							html.Type("button"),
							html.Class("btn btn-primary dropdown-toggle dropdown-toggle-split"),
							g.Attr("data-bs-toggle", "dropdown"),
							g.Attr("aria-expanded", "false"),
							g.Attr("aria-label", "Toggle dropdown"),
						),
						html.Ul(
							html.Class("dropdown-menu"),
							func() g.Node {
								if rt.Archived {
									return html.Li(
										html.Class("dropdown-item-text text-muted"),
										g.Text(fmt.Sprintf("Archived until %s", rt.EndDate.Format("Jan 2, 2006"))),
									)
								}
								return html.Li(
									html.Class("dropdown-item"),
									html.Button(
										html.Type("button"),
										html.Class("btn btn-warning w-100"),
										g.Attr("onclick", fmt.Sprintf("archiveRecurringTransaction(%d)", rt.ID)),
										g.Text("Archive"),
									),
								)
							}(),
							html.Li(html.Hr(html.Class("dropdown-divider"))),
							html.Li(
								html.Class("dropdown-item"),
								html.Button(
									html.Type("button"),
									html.Class("btn btn-danger w-100"),
									g.Attr("onclick", fmt.Sprintf("deleteRecurringTransaction(%d)", rt.ID)),
									g.Text("Delete"),
								),
							),
						),
					),
				),
			),
		),
	)
}

// AddRecurringTransactionModal renders the modal for adding a new recurring transaction
func AddRecurringTransactionModal(categories []data.Category, financialAccounts []data.FinancialAccount, budgetPlans []data.BudgetPlan, selectedBudgetPlanID int) g.Node {
	return html.Div(
		html.Class("modal fade"),
		html.ID("addRecurringTransactionModal"),
		html.DataAttr("tabindex", "-1"),
		html.DataAttr("aria-labelledby", "addRecurringTransactionModalLabel"),
		html.DataAttr("aria-hidden", "true"),
		html.Div(
			html.Class("modal-dialog"),
			html.Div(
				html.Class("modal-content"),
				html.Div(
					html.Class("modal-header"),
					html.H5(html.Class("modal-title"), html.ID("addRecurringTransactionModalLabel"), g.Text("Add Recurring Transaction")),
					html.Button(
						html.Class("btn-close"),
						html.Type("button"),
						html.DataAttr("bs-dismiss", "modal"),
						html.DataAttr("aria-label", "Close"),
					),
				),
				html.Div(
					html.Class("modal-body"),
					g.El("form",
						html.Action("/bills/"),
						html.Method("POST"),
						g.Attr("hx-post", "/bills/"),
						g.Attr("hx-target", "#recurring-transactions-list"),
						g.Attr("hx-swap", "outerHTML"),
						g.Attr("hx-on::after-request", "if (event.target === this) { bootstrap.Modal.getInstance(document.getElementById('addRecurringTransactionModal')).hide(); }"),
						RecurringTransactionFormFields(categories, financialAccounts, budgetPlans, selectedBudgetPlanID, nil),
						html.Div(
							html.Class("d-flex justify-content-end gap-2"),
							html.Button(html.Type("button"), html.Class("btn btn-secondary"), html.DataAttr("bs-dismiss", "modal"), g.Text("Cancel")),
							html.Button(html.Type("submit"), html.Class("btn btn-primary"), g.Text("Add Recurring Transaction")),
						),
					),
				),
			),
		),
		// Modal reset script
		html.Script(g.Raw(`
			document.getElementById('addRecurringTransactionModal').addEventListener('show.bs.modal', function() {
				const form = this.querySelector('form');
				if (form) form.reset();
			});
		`)),
	)
}

// EditRecurringTransactionModal renders the modal for editing a recurring transaction
func EditRecurringTransactionModal(categories []data.Category) g.Node {
	return html.Div(
		html.Class("modal fade"),
		html.ID("editRecurringTransactionModal"),
		html.DataAttr("tabindex", "-1"),
		html.DataAttr("aria-labelledby", "editRecurringTransactionModalLabel"),
		html.DataAttr("aria-hidden", "true"),
		html.Div(
			html.Class("modal-dialog"),
			html.Div(
				html.Class("modal-content"),
				html.Div(
					html.Class("modal-header"),
					html.H5(html.Class("modal-title"), html.ID("editRecurringTransactionModalLabel"), g.Text("Edit Recurring Transaction")),
					html.Button(
						html.Class("btn-close"),
						html.Type("button"),
						html.DataAttr("bs-dismiss", "modal"),
						html.DataAttr("aria-label", "Close"),
					),
				),
				html.Div(
					html.ID("editRecurringTransactionModalBody"),
					html.Class("modal-body"),
					// Content will be loaded via HTMX
				),
			),
		),
	)
}

// RecurringTransactionFormFields renders the form fields for a recurring transaction
func RecurringTransactionFormFields(categories []data.Category, financialAccounts []data.FinancialAccount, budgetPlans []data.BudgetPlan, selectedBudgetPlanID int, rt *data.RecurringTransaction) g.Node {
	return g.Group([]g.Node{
		html.Div(
			html.Class("mb-3"),
			html.Label(html.Class("form-label"), html.For("rt-budget-plan"), g.Text("Budget Plan")),
			html.Select(
				html.Name("budget_plan_id"),
				html.ID("rt-budget-plan"),
				html.Class("form-select"),
				html.Required(),
				g.Group(func() []g.Node {
					var options []g.Node
					for _, plan := range budgetPlans {
						selected := (rt != nil && rt.BudgetPlanID == plan.ID) || (rt == nil && plan.ID == selectedBudgetPlanID)
						activeText := ""
						if plan.IsActive {
							activeText = " (Active)"
						}
						options = append(options, html.Option(
							html.Value(strconv.Itoa(plan.ID)),
							func() g.Node {
								if selected {
									return html.Selected()
								}
								return nil
							}(),
							g.Text(plan.Name+activeText),
						))
					}
					return options
				}()),
			),
		),
		html.Div(
			html.Class("mb-3"),
			html.Label(html.Class("form-label"), html.For("rt-financial-account"), g.Text("Financial Account")),
			html.Select(
				html.Name("financial_account_id"),
				html.ID("rt-financial-account"),
				html.Class("form-select"),
				html.Required(),
				g.Group(func() []g.Node {
					var options []g.Node
					options = append(options, html.Option(html.Value(""), g.Text("Select an account...")))
					for _, fa := range financialAccounts {
						selected := rt != nil && rt.FinancialAccountID == fa.ID
						options = append(options, html.Option(
							html.Value(strconv.Itoa(fa.ID)),
							func() g.Node {
								if selected {
									return html.Selected()
								}
								return nil
							}(),
							g.Text(fa.Name+" ("+fa.Type+")"),
						))
					}
					return options
				}()),
			),
		),
		html.Div(
			html.Class("mb-3"),
			html.Label(html.Class("form-label"), html.For("rt-name"), g.Text("Name")),
			html.Input(
				html.Type("text"),
				html.Name("name"),
				html.ID("rt-name"),
				html.Class("form-control"),
				html.Required(),
				func() g.Node {
					if rt != nil {
						return html.Value(rt.Name)
					}
					return nil
				}(),
			),
		),
		html.Div(
			html.Class("mb-3"),
			html.Label(html.Class("form-label"), html.For("rt-counterparty"), g.Text("Counterparty")),
			html.Input(
				html.Type("text"),
				html.Name("counterparty"),
				html.ID("rt-counterparty"),
				html.Class("form-control"),
				html.Required(),
				func() g.Node {
					if rt != nil {
						return html.Value(rt.Counterparty)
					}
					return nil
				}(),
			),
		),
		html.Div(
			html.Class("mb-3"),
			html.Label(html.Class("form-label"), html.For("rt-category"), g.Text("Category (optional)")),
			CategoryBootstrapDropdown(categories, "category_id", "rt-category", func() *int {
				if rt != nil {
					return rt.CategoryID
				}
				return nil
			}(), nil, false, nil),
		),
		html.Div(
			html.Class("mb-3"),
			html.Label(html.Class("form-label"), html.For("rt-expected-amount"), g.Text("Expected Amount")),
			html.Input(
				html.Type("number"),
				html.Name("expected_amount"),
				html.ID("rt-expected-amount"),
				html.Class("form-control"),
				html.Step("0.01"),
				html.Required(),
				func() g.Node {
					if rt != nil {
						// Show actual signed value: negative for expenses, positive for income
						amount := float64(rt.ExpectedAmount) / 100.0
						return html.Value(fmt.Sprintf("%.2f", amount))
					}
					return nil
				}(),
			),
			html.Small(html.Class("form-text text-muted"), g.Text("Enter negative for expenses, positive for income")),
		),
		html.Div(
			html.Class("mb-3"),
			html.Label(html.Class("form-label"), html.For("rt-tolerance"), g.Text("Tolerance")),
			html.Input(
				html.Type("number"),
				html.Name("tolerance"),
				html.ID("rt-tolerance"),
				html.Class("form-control"),
				html.Step("0.01"),
				html.Required(),
				func() g.Node {
					if rt != nil {
						return html.Value(rt.ToleranceDecimal())
					}
					return html.Value("0")
				}(),
			),
		),
		html.Div(
			html.Class("mb-3"),
			html.Label(html.Class("form-label"), html.For("rt-start-date"), g.Text("Start Date")),
			html.Input(
				html.Type("date"),
				html.Name("start_date"),
				html.ID("rt-start-date"),
				html.Class("form-control"),
				html.Required(),
				func() g.Node {
					if rt != nil {
						return html.Value(rt.StartDate.Format("2006-01-02"))
					}
					return nil
				}(),
			),
		),
		html.Div(
			html.Class("mb-3"),
			html.Label(html.Class("form-label"), html.For("rt-recurrence-unit"), g.Text("Recurrence Unit")),
			html.Select(
				html.Name("recurrence_unit"),
				html.ID("rt-recurrence-unit"),
				html.Class("form-select"),
				html.Required(),
				g.Group([]g.Node{
					func() g.Node {
						if rt != nil && rt.RecurrenceUnit == "week" {
							return html.Option(html.Value("week"), g.Attr("selected", "selected"), g.Text("Week"))
						}
						return html.Option(html.Value("week"), g.Text("Week"))
					}(),
					func() g.Node {
						if rt != nil && rt.RecurrenceUnit == "month" {
							return html.Option(html.Value("month"), g.Attr("selected", "selected"), g.Text("Month"))
						}
						return html.Option(html.Value("month"), g.Text("Month"))
					}(),
					func() g.Node {
						if rt != nil && rt.RecurrenceUnit == "year" {
							return html.Option(html.Value("year"), g.Attr("selected", "selected"), g.Text("Year"))
						}
						return html.Option(html.Value("year"), g.Text("Year"))
					}(),
				}),
			),
		),
		html.Div(
			html.Class("mb-3"),
			html.Label(html.Class("form-label"), html.For("rt-recurrence-value"), g.Text("Recurrence Value")),
			html.Input(
				html.Type("number"),
				html.Name("recurrence_value"),
				html.ID("rt-recurrence-value"),
				html.Class("form-control"),
				html.Min("1"),
				html.Required(),
				func() g.Node {
					if rt != nil {
						return html.Value(strconv.Itoa(rt.RecurrenceValue))
					}
					return html.Value("1")
				}(),
			),
		),
	})
}

// RenderRecurringTransactionsList renders the recurring transactions list for HTMX responses
func RenderRecurringTransactionsList(w io.Writer, rts []data.RecurringTransaction, categories []data.Category) error {
	return html.Div(
		html.ID("recurring-transactions-list"),
		html.Class("list-group"),
		g.Group(func() []g.Node {
			var rows []g.Node
			for _, rt := range rts {
				rows = append(rows, RecurringTransactionCard(rt, categories))
			}
			return rows
		}()),
	).Render(w)
}

// PaycheckSummaryPage renders the paycheck summary page
func PaycheckSummaryPage(
	lastIncomeDate *time.Time,
	nextIncomeDate *time.Time,
	transactions []data.Transaction,
	expenseOccurrences []data.Occurrence,
	financialAccounts []data.FinancialAccount,
	selectedFinancialAccount *data.FinancialAccount,
	balanceCents *int,
	totalExpensesCents int64,
	expectedRemainingCents *int64,
	prevLastIncomeDate *time.Time,
	nextLastIncomeDate *time.Time,
	currentOffset int,
	budgetPlans []data.BudgetPlan,
	selectedBudgetPlanID int,
) g.Node {
	// Build pagination URLs using offset
	var prevURL, nextURL string
	baseURL := "/paycheck-summary"
	params := []string{}
	if selectedFinancialAccount != nil {
		params = append(params, "financial_account_id="+strconv.Itoa(selectedFinancialAccount.ID))
	}

	// Previous period: go back towards offset 0 (only if currentOffset > 0)
	// Next period: go forward to offset + 1
	nextOffset := currentOffset + 1

	// Previous button: only show if currentOffset > 0 (not at current period)
	// We don't need prevLastIncomeDate to be non-nil - we can always go back if offset > 0
	if currentOffset > 0 {
		prevOffset := currentOffset - 1
		prevParams := append([]string{}, params...)
		if prevOffset == 0 {
			// For offset 0, omit the offset parameter (it's the default)
			if len(prevParams) > 0 {
				prevURL = baseURL + "?" + strings.Join(prevParams, "&")
			} else {
				prevURL = baseURL
			}
		} else {
			prevParams = append(prevParams, fmt.Sprintf("offset=%d", prevOffset))
			prevURL = baseURL + "?" + strings.Join(prevParams, "&")
		}
	}

	if nextLastIncomeDate != nil {
		nextParams := append([]string{}, params...)
		nextParams = append(nextParams, fmt.Sprintf("offset=%d", nextOffset))
		if len(nextParams) > 0 {
			nextURL = baseURL + "?" + strings.Join(nextParams, "&")
		} else {
			nextURL = baseURL + fmt.Sprintf("?offset=%d", nextOffset)
		}
	}

	return g.Group([]g.Node{
		// Pagination controls
		html.Div(
			html.Class("row mb-3"),
			html.Div(
				html.Class("col-6"),
				func() g.Node {
					// Only show previous button if currentOffset > 0 (not at current period)
					if currentOffset > 0 && prevURL != "" {
						return html.A(
							html.Class("btn btn-outline-primary"),
							html.Href(prevURL),
							g.Raw(`<i class="bi bi-chevron-left"></i> `),
							g.Text("Previous"),
						)
					}
					// At offset 0 (current period), don't show previous button
					return nil
				}(),
			),
			html.Div(
				html.Class("col-6 text-end"),
				func() g.Node {
					if nextLastIncomeDate != nil && nextURL != "" {
						return html.A(
							html.Class("btn btn-outline-primary"),
							html.Href(nextURL),
							g.Text("Next "),
							g.Raw(`<i class="bi bi-chevron-right"></i>`),
						)
					}
					return html.Button(
						html.Class("btn btn-outline-primary"),
						html.Disabled(),
						g.Text("Next "),
						g.Raw(`<i class="bi bi-chevron-right"></i>`),
					)
				}(),
			),
		),
		html.Div(
			html.Class("row"),
			html.Div(
				html.Class("col-12"),
				html.H1(html.Class("display-4 mb-4 text-center"), g.Text("Paycheck Summary")),
				html.P(html.Class("lead text-center"), g.Text("Expenses between paychecks")),
			),
		),
		// Balance input and period info
		html.Div(
			html.Class("row mt-5 justify-content-center"),
			html.Div(
				html.Class("col-12 col-lg-10"),
				html.Div(
					html.Class("card"),
					html.Div(
						html.Class("card-body"),
						g.El("form",
							html.Method("GET"),
							html.Action("/paycheck-summary"),
							html.Div(
								html.Class("row align-items-end"),
								html.Div(
									html.Class("col-md-3 mb-3"),
									html.Label(
										html.Class("form-label"),
										html.For("financial_account_id"),
										g.Text("Financial Account"),
									),
									html.Select(
										html.Name("financial_account_id"),
										html.ID("financial_account_id"),
										html.Class("form-select"),
										g.Attr("onchange", "this.form.submit()"),
										g.Group(func() []g.Node {
											var options []g.Node
											if len(financialAccounts) == 0 {
												options = append(options, html.Option(html.Value(""), g.Text("No accounts available")))
											} else {
												for _, fa := range financialAccounts {
													selected := selectedFinancialAccount != nil && fa.ID == selectedFinancialAccount.ID
													options = append(options, html.Option(
														html.Value(strconv.Itoa(fa.ID)),
														func() g.Node {
															if selected {
																return html.Selected()
															}
															return nil
														}(),
														g.Text(fa.Name+" ("+fa.Type+")"),
													))
												}
											}
											return options
										}()),
									),
								),
								html.Div(
									html.Class("col-md-3 mb-3"),
									html.Label(
										html.Class("form-label"),
										html.For("balance-display"),
										g.Text("Account Balance"),
									),
									html.Input(
										html.Type("text"),
										html.Class("form-control"),
										html.ID("balance-display"),
										html.ReadOnly(),
										html.Disabled(),
										func() g.Node {
											if balanceCents != nil {
												balance := float64(*balanceCents) / 100.0
												return html.Value(fmt.Sprintf("$%.2f", balance))
											}
											return html.Value("$0.00")
										}(),
									),
								),
								html.Div(
									html.Class("col-md-4 mb-3"),
									html.Label(
										html.Class("form-label"),
										g.Text("Period"),
									),
									html.Div(
										html.Class("form-control-plaintext"),
										func() g.Node {
											if lastIncomeDate != nil && nextIncomeDate != nil {
												return g.Text(fmt.Sprintf("%s to %s",
													lastIncomeDate.Format("Jan 2, 2006"),
													nextIncomeDate.Format("Jan 2, 2006")))
											}
											return g.Text("No income data available")
										}(),
									),
								),
								html.Div(
									html.Class("col-md-2 mb-3"),
									html.Button(
										html.Type("submit"),
										html.Class("btn btn-primary w-100"),
										g.Text("Update"),
									),
								),
							),
						),
					),
				),
			),
		),
		// Money Flow card (similar to monthly summary)
		func() g.Node {
			if balanceCents != nil && expectedRemainingCents != nil {
				balanceFloat := float64(*balanceCents) / 100.0
				expensesFloat := float64(totalExpensesCents) / 100.0
				remainingFloat := float64(*expectedRemainingCents) / 100.0

				return html.Div(
					html.Class("row mt-5 justify-content-center"),
					html.Div(
						html.Class("col-12 col-lg-10"),
						html.Div(
							html.Class("card"),
							html.Div(
								html.Class("card-header"),
								html.H5(html.Class("card-title mb-0"), g.Text("Money Flow")),
							),
							html.Div(
								html.Class("card-body p-4"),
								// Desktop view: horizontal layout with large text
								html.Div(
									html.Class("d-none d-md-flex align-items-center justify-content-center flex-wrap gap-3"),
									// Starting Balance Card
									html.Div(
										html.Class("card flex-fill"),
										g.Attr("style", "min-width: 200px; max-width: 300px;"),
										html.Div(
											html.Class("card-body text-center"),
											html.H6(html.Class("card-subtitle mb-2 text-muted"), g.Text("Starting")),
											html.Div(
												html.Class("display-6 fw-bold"),
												g.Text(fmt.Sprintf("$%.2f", balanceFloat)),
											),
										),
									),
									// Minus Sign
									html.Div(
										html.Class("d-flex align-items-center"),
										g.Attr("style", "font-size: 3rem; font-weight: bold; color: #6c757d;"),
										g.Text("−"),
									),
									// Expenses Card
									html.Div(
										html.Class("card flex-fill"),
										g.Attr("style", "min-width: 200px; max-width: 300px;"),
										html.Div(
											html.Class("card-body text-center"),
											html.H6(html.Class("card-subtitle mb-2 text-muted"), g.Text("Expenses")),
											html.Div(
												html.Class("display-6 fw-bold text-danger"),
												g.Text(fmt.Sprintf("$%.2f", expensesFloat)),
											),
										),
									),
									// Equals Sign
									html.Div(
										html.Class("d-flex align-items-center"),
										g.Attr("style", "font-size: 3rem; font-weight: bold; color: #6c757d;"),
										g.Text("="),
									),
									// Remaining Balance Card
									html.Div(
										html.Class("card flex-fill"),
										g.Attr("style", "min-width: 200px; max-width: 300px;"),
										html.Div(
											html.Class("card-body text-center"),
											html.H6(html.Class("card-subtitle mb-2 text-muted"), g.Text("Remaining")),
											html.Div(
												func() g.Node {
													if remainingFloat >= 0 {
														return html.Class("display-6 fw-bold text-success")
													}
													return html.Class("display-6 fw-bold text-danger")
												}(),
												g.Text(fmt.Sprintf("$%.2f", remainingFloat)),
											),
										),
									),
								),
								// Mobile view: inline with smaller text
								html.Div(
									html.Class("d-md-none"),
									html.Div(
										html.Class("d-flex align-items-center justify-content-center gap-1"),
										g.Attr("style", "flex-wrap: nowrap; overflow-x: auto;"),
										// Starting Balance Card
										html.Div(
											html.Class("card"),
											g.Attr("style", "min-width: 80px; flex-shrink: 0;"),
											html.Div(
												html.Class("card-body text-center p-2"),
												html.H6(html.Class("card-subtitle mb-1 text-muted"), g.Attr("style", "font-size: 0.75rem;"), g.Text("Start")),
												html.Div(
													html.Class("fw-bold"),
													g.Attr("style", "font-size: 1.5rem; white-space: nowrap;"),
													g.Text(fmt.Sprintf("$%.0f", balanceFloat)),
												),
											),
										),
										// Minus Sign
										html.Div(
											html.Class("d-flex align-items-center"),
											g.Attr("style", "font-size: 1.8rem; font-weight: bold; color: #6c757d; flex-shrink: 0;"),
											g.Text("−"),
										),
										// Expenses Card
										html.Div(
											html.Class("card"),
											g.Attr("style", "min-width: 80px; flex-shrink: 0;"),
											html.Div(
												html.Class("card-body text-center p-2"),
												html.H6(html.Class("card-subtitle mb-1 text-muted"), g.Attr("style", "font-size: 0.75rem;"), g.Text("Exp")),
												html.Div(
													html.Class("fw-bold text-danger"),
													g.Attr("style", "font-size: 1.5rem; white-space: nowrap;"),
													g.Text(fmt.Sprintf("$%.0f", expensesFloat)),
												),
											),
										),
										// Equals Sign
										html.Div(
											html.Class("d-flex align-items-center"),
											g.Attr("style", "font-size: 1.8rem; font-weight: bold; color: #6c757d; flex-shrink: 0;"),
											g.Text("="),
										),
										// Remaining Balance Card
										html.Div(
											html.Class("card"),
											g.Attr("style", "min-width: 80px; flex-shrink: 0;"),
											html.Div(
												html.Class("card-body text-center p-2"),
												html.H6(html.Class("card-subtitle mb-1 text-muted"), g.Attr("style", "font-size: 0.75rem;"), g.Text("Remain")),
												html.Div(
													func() g.Node {
														if remainingFloat >= 0 {
															return html.Class("fw-bold text-success")
														}
														return html.Class("fw-bold text-danger")
													}(),
													g.Attr("style", "font-size: 1.5rem; white-space: nowrap;"),
													g.Text(fmt.Sprintf("$%.0f", remainingFloat)),
												),
											),
										),
									),
								),
							),
						),
					),
				)
			}
			return nil
		}(),
		// Expenses list
		func() g.Node {
			if lastIncomeDate != nil && nextIncomeDate != nil {
				return html.Div(
					html.Class("row mt-5 justify-content-center"),
					html.Div(
						html.Class("col-12 col-lg-10"),
						html.Div(
							html.Class("card"),
							html.Div(
								html.Class("card-header"),
								html.H5(html.Class("card-title mb-0"), g.Text("Expenses in Period")),
							),
							html.Div(
								html.Class("card-body"),
								func() g.Node {
									if len(transactions) == 0 && len(expenseOccurrences) == 0 {
										return html.P(html.Class("text-muted mb-0"), g.Text("No expenses in this period."))
									}
									return html.Div(
										html.Class("table-responsive"),
										html.Table(
											html.Class("table table-hover"),
											g.El("thead",
												html.Tr(
													html.Th(g.Text("Date")),
													html.Th(g.Text("Name")),
													html.Th(html.Class("text-end"), g.Text("Amount")),
												),
											),
											g.El("tbody",
												g.Group(func() []g.Node {
													var rows []g.Node
													// Add transactions
													for _, tx := range transactions {
														if tx.Amount < 0 {
															rows = append(rows, html.Tr(
																html.Td(g.Text(tx.Date.Format("Jan 2, 2006"))),
																html.Td(g.Text(tx.Payee)),
																html.Td(
																	html.Class("text-end fw-bold text-danger"),
																	g.Text(fmt.Sprintf("$%.2f", float64(-tx.Amount)/100.0)),
																),
															))
														}
													}
													// Add expense occurrences (only unmatched ones - matched ones are already shown as transactions)
													for _, occ := range expenseOccurrences {
														// Skip matched occurrences - they're already displayed as actual transactions
														if occ.IsMatched {
															continue
														}
														rt := occ.RecurringTransaction
														dateText := occ.ExpectedDate.Format("Jan 2, 2006")
														amount := float64(-rt.ExpectedAmount) / 100.0
														rows = append(rows, html.Tr(
															html.Td(g.Text(dateText)),
															html.Td(g.Text(rt.Name)),
															html.Td(
																html.Class("text-end fw-bold text-danger"),
																g.Text(fmt.Sprintf("$%.2f", amount)),
															),
														))
													}
													return rows
												}()),
											),
										),
									)
								}(),
							),
						),
					),
				)
			}
			return nil
		}(),
	})
}

// Budget plan templates

type PlanWithCount struct {
	data.BudgetPlan
	TransactionCount int
}

// BudgetPlansPage renders the budget plans management page
func BudgetPlansPage(plans []PlanWithCount) g.Node {
	return html.Div(
		html.Class("row"),
		html.Div(
			html.Class("col-12 col-md-8 px-2"),
			html.H1(html.Class("display-6 my-2 text-center"), g.Text("Budget Plans")),
			// Toolbar
			html.Div(
				html.Class("card p-2 mb-3 bg-body-tertiary border"),
				html.Button(
					html.Class("btn btn-primary"),
					html.DataAttr("bs-toggle", "modal"),
					html.DataAttr("bs-target", "#createBudgetPlanModal"),
					g.Text("Create Budget Plan"),
				),
			),
			// Hidden refresh trigger
			html.Div(
				html.ID("refresh-budget-plans-list"),
				g.Attr("hx-get", "/budget-plans/list"),
				g.Attr("hx-target", "#budget-plans-list"),
				g.Attr("hx-swap", "innerHTML"),
				g.Attr("hx-trigger", "refresh"),
				g.Attr("style", "display: none;"),
			),
			// Budget plans list
			html.Div(
				html.ID("budget-plans-list"),
				html.Class("row g-3"),
				func() g.Node {
					if len(plans) == 0 {
						return html.Div(
							html.Class("alert alert-info"),
							g.Text("No budget plans yet. Create your first budget plan to get started."),
						)
					}
					return g.Group(func() []g.Node {
						var nodes []g.Node
						for _, plan := range plans {
							nodes = append(nodes, BudgetPlanCard(plan))
						}
						return nodes
					}())
				}(),
			),
		),
		// Create modal
		html.Div(
			html.Class("modal fade"),
			html.ID("createBudgetPlanModal"),
			g.Attr("tabindex", "-1"),
			g.Attr("aria-labelledby", "createBudgetPlanModalLabel"),
			html.Div(
				html.Class("modal-dialog"),
				html.Div(
					html.Class("modal-content"),
					html.Div(
						html.Class("modal-header"),
						html.H5(html.Class("modal-title"), html.ID("createBudgetPlanModalLabel"), g.Text("Create Budget Plan")),
						html.Button(
							html.Type("button"),
							html.Class("btn-close"),
							html.DataAttr("bs-dismiss", "modal"),
							g.Attr("aria-label", "Close"),
						),
					),
					g.El("form",
						html.Action("/budget-plans/"),
						html.Method("POST"),
						g.Attr("hx-post", "/budget-plans/"),
						g.Attr("hx-target", "#budget-plans-list"),
						g.Attr("hx-swap", "innerHTML"),
						g.Attr("hx-on::after-request", `if (event.detail.successful) { const modal = bootstrap.Modal.getInstance(document.querySelector('#createBudgetPlanModal')); if (modal) modal.hide(); }`),
						html.Div(
							html.Class("modal-body"),
							html.Div(
								html.Class("mb-3"),
								html.Label(html.Class("form-label"), html.For("create-plan-name"), g.Text("Name")),
								html.Input(
									html.Type("text"),
									html.Class("form-control"),
									html.ID("create-plan-name"),
									html.Name("name"),
									html.Required(),
								),
							),
						),
						html.Div(
							html.Class("modal-footer"),
							html.Button(html.Type("button"), html.Class("btn btn-secondary"), html.DataAttr("bs-dismiss", "modal"), g.Text("Cancel")),
							html.Button(html.Type("submit"), html.Class("btn btn-primary"), g.Text("Create")),
						),
					),
				),
			),
		),
	)
}

// BudgetPlanCard renders a single budget plan card
func BudgetPlanCard(plan PlanWithCount) g.Node {
	var activeBadge g.Node
	if plan.IsActive {
		activeBadge = html.Span(
			html.Class("badge bg-success ms-2"),
			g.Text("Active"),
		)
	}
	return html.Div(
		html.Class("card mb-3"),
		html.Div(
			html.Class("card-body"),
			html.Div(
				html.Class("d-flex justify-content-between align-items-start"),
				html.Div(
					html.Class("flex-grow-1"),
					html.H5(
						html.Class("card-title mb-1"),
						g.Text(plan.Name),
						activeBadge,
					),
					html.P(html.Class("text-muted mb-0"), g.Text(fmt.Sprintf("%d recurring transaction(s)", plan.TransactionCount))),
				),
				html.Div(
					html.Class("btn-group"),
					func() g.Node {
						if !plan.IsActive {
							return html.Button(
								html.Class("btn btn-sm btn-outline-primary"),
								g.Attr("hx-post", fmt.Sprintf("/budget-plans/%d/activate", plan.ID)),
								g.Attr("hx-target", "#budget-plans-list"),
								g.Attr("hx-swap", "innerHTML"),
								g.Text("Activate"),
							)
						}
						return nil
					}(),
					html.Button(
						html.Class("btn btn-sm btn-outline-secondary"),
						html.DataAttr("bs-toggle", "modal"),
						html.DataAttr("bs-target", fmt.Sprintf("#copyBudgetPlanModal%d", plan.ID)),
						g.Text("Copy"),
					),
					html.Button(
						html.Class("btn btn-sm btn-outline-secondary"),
						html.DataAttr("bs-toggle", "modal"),
						html.DataAttr("bs-target", fmt.Sprintf("#editBudgetPlanModal%d", plan.ID)),
						g.Text("Edit"),
					),
					func() g.Node {
						if !plan.IsActive && plan.TransactionCount == 0 {
							return html.Button(
								html.Class("btn btn-sm btn-outline-danger"),
								g.Attr("hx-delete", fmt.Sprintf("/budget-plans/%d", plan.ID)),
								g.Attr("hx-target", "#budget-plans-list"),
								g.Attr("hx-swap", "innerHTML"),
								g.Attr("hx-confirm", "Are you sure you want to delete this budget plan?"),
								g.Text("Delete"),
							)
						}
						return nil
					}(),
				),
			),
		),
		// Edit modal
		html.Div(
			html.Class("modal fade"),
			html.ID(fmt.Sprintf("editBudgetPlanModal%d", plan.ID)),
			g.Attr("tabindex", "-1"),
			html.Div(
				html.Class("modal-dialog"),
				html.Div(
					html.Class("modal-content"),
					html.Div(
						html.Class("modal-header"),
						html.H5(html.Class("modal-title"), g.Text("Edit Budget Plan")),
						html.Button(html.Type("button"), html.Class("btn-close"), html.DataAttr("bs-dismiss", "modal")),
					),
					g.El("form",
						g.Attr("hx-put", fmt.Sprintf("/budget-plans/%d", plan.ID)),
						g.Attr("hx-swap", "none"),
						g.Attr("hx-on::after-request", fmt.Sprintf(`if (event.detail.successful) { const modal = bootstrap.Modal.getInstance(document.getElementById('editBudgetPlanModal%d')); if (modal) modal.hide(); htmx.trigger('#refresh-budget-plans-list', 'refresh'); }`, plan.ID)),
						html.Div(
							html.Class("modal-body"),
							html.Div(
								html.Class("mb-3"),
								html.Label(html.Class("form-label"), g.Text("Name")),
								html.Input(
									html.Type("text"),
									html.Class("form-control"),
									html.Name("name"),
									html.Value(plan.Name),
									html.Required(),
								),
							),
						),
						html.Div(
							html.Class("modal-footer"),
							html.Button(html.Type("button"), html.Class("btn btn-secondary"), html.DataAttr("bs-dismiss", "modal"), g.Text("Cancel")),
							html.Button(html.Type("submit"), html.Class("btn btn-primary"), g.Text("Save")),
						),
					),
				),
			),
		),
		// Copy modal
		html.Div(
			html.Class("modal fade"),
			html.ID(fmt.Sprintf("copyBudgetPlanModal%d", plan.ID)),
			g.Attr("tabindex", "-1"),
			html.Div(
				html.Class("modal-dialog"),
				html.Div(
					html.Class("modal-content"),
					html.Div(
						html.Class("modal-header"),
						html.H5(html.Class("modal-title"), g.Text("Copy Budget Plan")),
						html.Button(html.Type("button"), html.Class("btn-close"), html.DataAttr("bs-dismiss", "modal")),
					),
					g.El("form",
						g.Attr("hx-post", fmt.Sprintf("/budget-plans/%d/copy", plan.ID)),
						g.Attr("hx-swap", "none"),
						g.Attr("hx-on::after-request", fmt.Sprintf(`if (event.detail.successful) { const modal = bootstrap.Modal.getInstance(document.getElementById('copyBudgetPlanModal%d')); if (modal) modal.hide(); htmx.trigger('#refresh-budget-plans-list', 'refresh'); }`, plan.ID)),
						html.Div(
							html.Class("modal-body"),
							html.Div(
								html.Class("mb-3"),
								html.Label(html.Class("form-label"), g.Text("New Plan Name")),
								html.Input(
									html.Type("text"),
									html.Class("form-control"),
									html.Name("name"),
									html.Value(fmt.Sprintf("%s (Copy)", plan.Name)),
									html.Required(),
								),
							),
						),
						html.Div(
							html.Class("modal-footer"),
							html.Button(html.Type("button"), html.Class("btn btn-secondary"), html.DataAttr("bs-dismiss", "modal"), g.Text("Cancel")),
							html.Button(html.Type("submit"), html.Class("btn btn-primary"), g.Text("Copy")),
						),
					),
				),
			),
		),
	)
}

// RenderBudgetPlansList renders the budget plans list for HTMX updates
func RenderBudgetPlansList(w io.Writer, plans []PlanWithCount) error {
	if len(plans) == 0 {
		_, err := w.Write([]byte(`<div class="alert alert-info">No budget plans yet. Create your first budget plan to get started.</div>`))
		return err
	}
	var nodes []g.Node
	for _, plan := range plans {
		nodes = append(nodes, BudgetPlanCard(plan))
	}
	// Return just the cards wrapped in a div (not the row wrapper, since that's in the page template)
	return html.Div(
		g.Group(nodes),
	).Render(w)
}

// BudgetPlanSelector renders a dropdown to select budget plan
func BudgetPlanSelector(plans []data.BudgetPlan, selectedPlanID int, currentURL string) g.Node {
	return html.Div(
		html.Class("mb-3"),
		html.Label(html.Class("form-label"), g.Text("Budget Plan")),
		html.Select(
			html.Class("form-select"),
			g.Attr("onchange", fmt.Sprintf("window.location.href = '%s?budget_plan_id=' + this.value", currentURL)),
			func() g.Node {
				var options []g.Node
				for _, plan := range plans {
					selected := ""
					if plan.ID == selectedPlanID {
						selected = " selected"
					}
					activeText := ""
					if plan.IsActive {
						activeText = " (Active)"
					}
					options = append(options, html.Option(
						html.Value(strconv.Itoa(plan.ID)),
						g.Attr("selected", selected),
						g.Text(plan.Name+activeText),
					))
				}
				return g.Group(options)
			}(),
		),
	)
}

// Financial accounts templates

// FinancialAccountsPage renders the financial accounts management page
func FinancialAccountsPage(accounts []data.FinancialAccount) g.Node {
	return html.Div(
		html.Class("row"),
		html.Div(
			html.Class("col-12 col-md-8 px-2"),
			html.H1(html.Class("display-6 my-2 text-center"), g.Text("Financial Accounts")),
			// Toolbar
			html.Div(
				html.Class("card p-2 mb-3 bg-body-tertiary border"),
				html.Div(
					html.Class("d-flex align-items-center gap-2"),
					html.Button(
						html.Class("btn btn-primary"),
						html.Type("button"),
						html.DataAttr("bs-toggle", "modal"),
						html.DataAttr("bs-target", "#addFinancialAccountModal"),
						g.Text("Add Financial Account"),
					),
				),
			),
			// Financial accounts list
			html.Div(
				html.ID("financial-accounts-list"),
				html.Class("list-group"),
				g.Group(func() []g.Node {
					var rows []g.Node
					for _, fa := range accounts {
						rows = append(rows, FinancialAccountCard(fa))
					}
					return rows
				}()),
			),
		),
		// Add Financial Account Modal
		AddFinancialAccountModal(),
		// Edit Financial Account Modal
		EditFinancialAccountModal(),
		// JavaScript functions for financial account actions
		html.Script(g.Raw(`
			function openEditFinancialAccountModal(faId) {
				const modal = document.getElementById('editFinancialAccountModal');
				if (!modal) return;
				
				// Load the edit form via HTMX
				fetch('/financial-accounts/' + faId + '/edit')
					.then(response => response.text())
					.then(html => {
						document.getElementById('editFinancialAccountModalBody').innerHTML = html;
						// Process HTMX attributes on the newly loaded content
						if (typeof htmx !== 'undefined') {
							htmx.process(document.getElementById('editFinancialAccountModalBody'));
						}
						const bsModal = bootstrap.Modal.getOrCreateInstance(modal);
						bsModal.show();
					})
					.catch(error => console.error('Error loading edit form:', error));
			}
			
			function deleteFinancialAccount(faId) {
				if (confirm('Are you sure you want to delete this financial account? This will also delete all associated transactions. This action cannot be undone.')) {
					fetch('/financial-accounts/' + faId, {
						method: 'DELETE',
						headers: {
							'HX-Request': 'true',
						},
					})
					.then(response => response.text())
					.then(html => {
						document.getElementById('financial-accounts-list').outerHTML = html;
					})
					.catch(error => console.error('Error deleting financial account:', error));
				}
			}
		`)),
	)
}

// RenderFinancialAccountsList renders just the financial accounts list (for HTMX updates)
func RenderFinancialAccountsList(w io.Writer, accounts []data.FinancialAccount) error {
	list := html.Div(
		html.ID("financial-accounts-list"),
		html.Class("list-group"),
		g.Group(func() []g.Node {
			var rows []g.Node
			for _, fa := range accounts {
				rows = append(rows, FinancialAccountCard(fa))
			}
			return rows
		}()),
	)
	return list.Render(w)
}

// FinancialAccountCard renders a single financial account card
func FinancialAccountCard(fa data.FinancialAccount) g.Node {
	return html.Div(
		html.Class("list-group-item"),
		g.Attr("data-financial-account-id", strconv.Itoa(fa.ID)),
		html.Div(
			html.Class("d-flex justify-content-between align-items-start"),
			html.Div(
				html.Class("flex-grow-1"),
				html.H5(html.Class("mb-1"), g.Text(fa.Name)),
				html.P(
					html.Class("mb-1 text-muted"),
					g.Text(strings.Title(fa.Type)),
				),
				html.Small(
					html.Class("text-muted"),
					g.Text("Balance: $"+fa.BalanceDecimal()),
				),
			),
			html.Div(
				html.Class("btn-group"),
				html.Button(
					html.Class("btn btn-sm btn-outline-primary"),
					g.Attr("onclick", "openEditFinancialAccountModal("+strconv.Itoa(fa.ID)+")"),
					g.Text("Edit"),
				),
				html.Button(
					html.Class("btn btn-sm btn-outline-danger"),
					g.Attr("onclick", "deleteFinancialAccount("+strconv.Itoa(fa.ID)+")"),
					g.Text("Delete"),
				),
			),
		),
	)
}

// FinancialAccountFormFields renders form fields for creating/editing a financial account
func FinancialAccountFormFields(fa *data.FinancialAccount, mode string) g.Node {
	var action string
	var method string
	if mode == "edit" && fa != nil {
		action = "/financial-accounts/" + strconv.Itoa(fa.ID)
		method = "PUT"
	} else {
		action = "/financial-accounts"
		method = "POST"
	}

	var name, accountType, csvDateField, csvPayeeField, csvExpenseField, csvIncomeField string
	var csvCategoryField, csvBalanceField string
	var balanceValue string
	if fa != nil {
		name = fa.Name
		accountType = fa.Type
		balanceValue = fa.BalanceDecimal()
		csvDateField = fa.CSVDateField
		csvPayeeField = fa.CSVPayeeField
		csvExpenseField = fa.CSVExpenseField
		csvIncomeField = fa.CSVIncomeField
		if fa.CSVCategoryField != nil {
			csvCategoryField = *fa.CSVCategoryField
		}
		if fa.CSVBalanceField != nil {
			csvBalanceField = *fa.CSVBalanceField
		}
	}

	return g.El("form",
		g.Attr("hx-"+strings.ToLower(method), action),
		g.Attr("hx-target", "#financial-accounts-list"),
		g.Attr("hx-swap", "outerHTML"),
		g.Attr("hx-on::after-request", `if (event.target === this) { bootstrap.Modal.getInstance(this.closest('.modal')).hide(); }`),
		html.Div(
			html.Class("mb-3"),
			html.Label(html.Class("form-label"), html.For("name"), g.Text("Name")),
			html.Input(
				html.Type("text"),
				html.Name("name"),
				html.ID("name"),
				html.Class("form-control"),
				html.Value(name),
				html.Required(),
			),
		),
		html.Div(
			html.Class("mb-3"),
			html.Label(html.Class("form-label"), html.For("type"), g.Text("Type")),
			html.Select(
				html.Name("type"),
				html.ID("type"),
				html.Class("form-select"),
				html.Required(),
				html.Option(html.Value("checking"), func() g.Node {
					if accountType == "checking" {
						return html.Selected()
					}
					return nil
				}(), g.Text("Checking")),
				html.Option(html.Value("savings"), func() g.Node {
					if accountType == "savings" {
						return html.Selected()
					}
					return nil
				}(), g.Text("Savings")),
				html.Option(html.Value("credit_card"), func() g.Node {
					if accountType == "credit_card" {
						return html.Selected()
					}
					return nil
				}(), g.Text("Credit Card")),
			),
		),
		html.Div(
			html.Class("mb-3"),
			html.Label(html.Class("form-label"), html.For("balance"), g.Text("Balance")),
			html.Input(
				html.Type("number"),
				html.Name("balance"),
				html.ID("balance"),
				html.Class("form-control"),
				html.Step("0.01"),
				html.Value(balanceValue),
			),
			html.Small(html.Class("form-text text-muted"), g.Text("Current account balance. Can be manually updated.")),
		),
		html.Hr(),
		html.H6(g.Text("CSV Field Mappings")),
		html.Div(
			html.Class("mb-3"),
			html.Label(html.Class("form-label"), html.For("csv_date_field"), g.Text("Date Field")),
			html.Input(
				html.Type("text"),
				html.Name("csv_date_field"),
				html.ID("csv_date_field"),
				html.Class("form-control"),
				html.Value(csvDateField),
				html.Required(),
			),
		),
		html.Div(
			html.Class("mb-3"),
			html.Label(html.Class("form-label"), html.For("csv_payee_field"), g.Text("Payee Field")),
			html.Input(
				html.Type("text"),
				html.Name("csv_payee_field"),
				html.ID("csv_payee_field"),
				html.Class("form-control"),
				html.Value(csvPayeeField),
				html.Required(),
			),
		),
		html.Div(
			html.Class("mb-3"),
			html.Label(html.Class("form-label"), html.For("csv_expense_field"), g.Text("Expense Field")),
			html.Input(
				html.Type("text"),
				html.Name("csv_expense_field"),
				html.ID("csv_expense_field"),
				html.Class("form-control"),
				html.Value(csvExpenseField),
				html.Required(),
			),
		),
		html.Div(
			html.Class("mb-3"),
			html.Label(html.Class("form-label"), html.For("csv_income_field"), g.Text("Income Field")),
			html.Input(
				html.Type("text"),
				html.Name("csv_income_field"),
				html.ID("csv_income_field"),
				html.Class("form-control"),
				html.Value(csvIncomeField),
				html.Required(),
			),
		),
		html.Div(
			html.Class("mb-3"),
			html.Label(html.Class("form-label"), html.For("csv_category_field"), g.Text("Category Field (Optional)")),
			html.Input(
				html.Type("text"),
				html.Name("csv_category_field"),
				html.ID("csv_category_field"),
				html.Class("form-control"),
				html.Value(csvCategoryField),
			),
		),
		html.Div(
			html.Class("mb-3"),
			html.Label(html.Class("form-label"), html.For("csv_balance_field"), g.Text("Balance Field (Optional)")),
			html.Input(
				html.Type("text"),
				html.Name("csv_balance_field"),
				html.ID("csv_balance_field"),
				html.Class("form-control"),
				html.Value(csvBalanceField),
			),
		),
		html.Div(
			html.Class("d-flex justify-content-end gap-2"),
			html.Button(html.Type("button"), html.Class("btn btn-secondary"), html.DataAttr("bs-dismiss", "modal"), g.Text("Cancel")),
			html.Button(html.Type("submit"), html.Class("btn btn-primary"), g.Text("Save")),
		),
	)
}

// AddFinancialAccountModal renders the modal for adding a financial account
func AddFinancialAccountModal() g.Node {
	return html.Div(
		html.Class("modal fade"),
		html.ID("addFinancialAccountModal"),
		html.DataAttr("tabindex", "-1"),
		html.DataAttr("aria-labelledby", "addFinancialAccountModalLabel"),
		html.DataAttr("aria-hidden", "true"),
		html.Div(
			html.Class("modal-dialog"),
			html.Div(
				html.Class("modal-content"),
				html.Div(
					html.Class("modal-header"),
					html.H5(html.Class("modal-title"), html.ID("addFinancialAccountModalLabel"), g.Text("Add Financial Account")),
					html.Button(
						html.Class("btn-close"),
						html.Type("button"),
						html.DataAttr("bs-dismiss", "modal"),
						html.DataAttr("aria-label", "Close"),
					),
				),
				html.Div(
					html.Class("modal-body"),
					html.ID("addFinancialAccountModalBody"),
					FinancialAccountFormFields(nil, "create"),
				),
			),
		),
	)
}

// EditFinancialAccountModal renders the modal for editing a financial account
func EditFinancialAccountModal() g.Node {
	return html.Div(
		html.Class("modal fade"),
		html.ID("editFinancialAccountModal"),
		html.DataAttr("tabindex", "-1"),
		html.DataAttr("aria-labelledby", "editFinancialAccountModalLabel"),
		html.DataAttr("aria-hidden", "true"),
		html.Div(
			html.Class("modal-dialog"),
			html.Div(
				html.Class("modal-content"),
				html.Div(
					html.Class("modal-header"),
					html.H5(html.Class("modal-title"), html.ID("editFinancialAccountModalLabel"), g.Text("Edit Financial Account")),
					html.Button(
						html.Class("btn-close"),
						html.Type("button"),
						html.DataAttr("bs-dismiss", "modal"),
						html.DataAttr("aria-label", "Close"),
					),
				),
				html.Div(
					html.Class("modal-body"),
					html.ID("editFinancialAccountModalBody"),
				),
			),
		),
	)
}

// Budget templates

func BudgetsPage(budgetSummaries []data.BudgetSummary, categories []data.Category, yearlyIncome int, year, month int) g.Node {
	return g.Group([]g.Node{
		html.Div(
			html.Class("row mb-3"),
			html.Div(
				html.Class("col-12"),
				html.H1(html.Class("display-4 mb-4"), g.Text("Budgets")),
				html.P(html.Class("lead"), g.Text(fmt.Sprintf("Manage your budgets for %s", time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC).Format("January 2006")))),
			),
		),
		html.Div(
			html.Class("row mb-3"),
			html.Div(
				html.Class("col-12"),
				html.Button(
					html.Class("btn btn-primary"),
					html.DataAttr("bs-toggle", "modal"),
					html.DataAttr("bs-target", "#addBudgetModal"),
					g.Text("Add Budget"),
				),
			),
		),
		func() g.Node {
			if yearlyIncome > 0 {
				return html.Div(
					html.Class("row mb-3"),
					html.Div(
						html.Class("col-12"),
						html.Div(
							html.Class("alert alert-info"),
							g.Text(fmt.Sprintf("Expected Yearly Income: $%.2f", float64(yearlyIncome)/100.0)),
						),
					),
				)
			}
			return nil
		}(),
		html.Div(
			html.Class("row"),
			html.Div(
				html.Class("col-12"),
				html.Div(
					html.Class("card"),
					html.Div(
						html.Class("card-body"),
						html.ID("budgets-table"),
						BudgetTable(budgetSummaries, yearlyIncome),
					),
				),
			),
		),
		AddBudgetModal(categories, yearlyIncome),
		EditBudgetModal(),
	})
}

// BudgetSummaryTable renders a simplified read-only budget table for the home page
func BudgetSummaryTable(budgetSummaries []data.BudgetSummary) g.Node {
	if len(budgetSummaries) == 0 {
		return html.P(html.Class("text-muted mb-0"), g.Text("No budgets configured."))
	}

	return html.Div(
		html.Class("table-responsive"),
		html.Table(
			html.Class("table table-hover"),
			g.El("thead",
				html.Tr(
					html.Th(g.Text("Category")),
					html.Th(html.Class("text-end"), g.Text("Monthly Budget")),
					html.Th(html.Class("text-end"), g.Text("Spent")),
					html.Th(html.Class("text-end"), g.Text("Expected")),
					html.Th(html.Class("text-end"), g.Text("Remaining")),
				),
			),
			g.El("tbody",
				g.Group(func() []g.Node {
					var rows []g.Node
					for _, bs := range budgetSummaries {
						// Determine status class
						var statusClass string
						if bs.Remaining < 0 {
							statusClass = "text-danger"
						} else if bs.Remaining < bs.MonthlyAmount/10 {
							statusClass = "text-warning"
						} else {
							statusClass = "text-success"
						}

						rows = append(rows, html.Tr(
							html.Td(g.Text(bs.CategoryName)),
							html.Td(
								html.Class("text-end fw-bold"),
								g.Text(fmt.Sprintf("$%s", bs.MonthlyAmountDecimal())),
							),
							html.Td(
								html.Class("text-end"),
								g.Text(fmt.Sprintf("$%s", bs.SpentAmountDecimal())),
							),
							html.Td(
								html.Class("text-end"),
								g.Text(fmt.Sprintf("$%s", bs.ExpectedAmountDecimal())),
							),
							html.Td(
								html.Class(fmt.Sprintf("text-end fw-bold %s", statusClass)),
								g.Text(fmt.Sprintf("$%s", bs.RemainingDecimal())),
							),
						))
					}
					return rows
				}()),
			),
		),
	)
}

func BudgetTable(budgetSummaries []data.BudgetSummary, yearlyIncome int) g.Node {
	if len(budgetSummaries) == 0 {
		return html.P(html.Class("text-muted mb-0"), g.Text("No budgets configured. Click 'Add Budget' to create one."))
	}

	return html.Div(
		html.Class("table-responsive"),
		html.Table(
			html.Class("table table-hover"),
			g.El("thead",
				html.Tr(
					html.Th(g.Text("Category")),
					html.Th(html.Class("text-end"), g.Text("Monthly Budget")),
					html.Th(html.Class("text-center"), g.Text("Actions")),
				),
			),
			g.El("tbody",
				g.Group(func() []g.Node {
					var rows []g.Node
					for _, bs := range budgetSummaries {
						rows = append(rows, html.Tr(
							html.Td(g.Text(bs.CategoryName)),
							html.Td(
								html.Class("text-end fw-bold"),
								g.Text(fmt.Sprintf("$%s", bs.MonthlyAmountDecimal())),
							),
							html.Td(
								html.Class("text-center"),
								html.Div(
									html.Class("btn-group btn-group-sm"),
									html.Button(
										html.Class("btn btn-outline-primary"),
										html.DataAttr("hx-get", fmt.Sprintf("/budgets/%d/edit", bs.Budget.ID)),
										html.DataAttr("hx-target", "#editBudgetModalBody"),
										html.DataAttr("hx-swap", "innerHTML"),
										html.DataAttr("bs-toggle", "modal"),
										html.DataAttr("bs-target", "#editBudgetModal"),
										g.Text("Edit"),
									),
									html.Button(
										html.Class("btn btn-outline-danger"),
										html.DataAttr("hx-delete", fmt.Sprintf("/budgets/%d", bs.Budget.ID)),
										html.DataAttr("hx-target", "#budgets-table"),
										html.DataAttr("hx-swap", "innerHTML"),
										html.DataAttr("hx-confirm", "Are you sure you want to delete this budget?"),
										g.Text("Delete"),
									),
								),
							),
						))
					}
					return rows
				}()),
			),
		),
	)
}

func BudgetForm(budget *data.Budget, categories []data.Category, yearlyIncome int, isEdit bool) g.Node {
	var categoryID int
	var amountType string = "fixed"
	var amountValue string

	if budget != nil {
		if budget.CategoryID != nil {
			categoryID = *budget.CategoryID
		}
		amountType = budget.AmountType
		if budget.AmountType == "fixed" {
			amountValue = fmt.Sprintf("%.2f", float64(budget.Amount)/100.0)
		} else {
			amountValue = fmt.Sprintf("%.2f", float64(budget.Amount)/100.0)
		}
	}

	formAction := "/budgets"
	if isEdit && budget != nil {
		formAction = fmt.Sprintf("/budgets/%d", budget.ID)
	}

	return g.El("form",
		html.Action(formAction),
		html.Method("POST"),
		func() g.Node {
			if isEdit {
				return html.DataAttr("hx-put", formAction)
			}
			return html.DataAttr("hx-post", formAction)
		}(),
		html.DataAttr("hx-target", "#budgets-table"),
		html.DataAttr("hx-swap", "innerHTML"),
		html.DataAttr("hx-on::after-request", "if(event.detail.successful) { const modal = bootstrap.Modal.getInstance(document.getElementById('addBudgetModal') || document.getElementById('editBudgetModal')); if(modal) modal.hide(); }"),
		html.Div(
			html.Class("mb-3"),
			html.Label(
				html.Class("form-label"),
				html.For("budget-category"),
				g.Text("Category"),
			),
			html.Select(
				html.Class("form-select"),
				html.ID("budget-category"),
				html.Name("category_id"),
				html.Option(
					html.Value("0"),
					g.Text("Uncategorized"),
				),
				g.Group(func() []g.Node {
					var options []g.Node
					for _, cat := range categories {
						opt := html.Option(
							html.Value(strconv.Itoa(cat.ID)),
							g.Text(cat.Name),
						)
						if cat.ID == categoryID {
							opt = html.Option(
								html.Value(strconv.Itoa(cat.ID)),
								html.Selected(),
								g.Text(cat.Name),
							)
						}
						options = append(options, opt)
					}
					return options
				}()),
			),
		),
		html.Div(
			html.Class("mb-3"),
			html.Label(
				html.Class("form-label"),
				html.For("budget-amount-type"),
				g.Text("Amount Type"),
			),
			html.Select(
				html.Class("form-select"),
				html.ID("budget-amount-type"),
				html.Name("amount_type"),
				html.DataAttr("x-data", "{ amountType: '"+amountType+"' }"),
				html.DataAttr("x-model", "amountType"),
				html.Option(
					html.Value("fixed"),
					func() g.Node {
						if amountType == "fixed" {
							return html.Selected()
						}
						return nil
					}(),
					g.Text("Fixed Amount"),
				),
				html.Option(
					html.Value("percentage"),
					func() g.Node {
						if amountType == "percentage" {
							return html.Selected()
						}
						return nil
					}(),
					g.Text("Percentage of Income"),
				),
			),
		),
		html.Div(
			html.Class("mb-3"),
			html.Label(
				html.Class("form-label"),
				html.For("budget-amount"),
				html.DataAttr("x-show", "amountType === 'fixed'"),
				g.Text("Monthly Amount ($)"),
			),
			html.Label(
				html.Class("form-label"),
				html.For("budget-amount"),
				html.DataAttr("x-show", "amountType === 'percentage'"),
				g.Text("Percentage (%)"),
			),
			html.Input(
				html.Class("form-control"),
				html.Type("number"),
				html.ID("budget-amount"),
				html.Name("amount"),
				html.Step("0.01"),
				html.Value(amountValue),
				html.Required(),
				html.Placeholder(func() string {
					if amountType == "percentage" {
						return "e.g., 10.5 for 10.5%"
					}
					return "e.g., 100.50"
				}()),
			),
			func() g.Node {
				if amountType == "percentage" && yearlyIncome > 0 {
					percentage := 0.0
					if amountValue != "" {
						if p, err := strconv.ParseFloat(amountValue, 64); err == nil {
							percentage = p
						}
					}
					monthlyAmount := (float64(yearlyIncome) * (percentage / 100.0)) / 12.0
					return html.Small(
						html.Class("form-text text-muted"),
						g.Text(fmt.Sprintf("Monthly budget: $%.2f", monthlyAmount)),
					)
				}
				return nil
			}(),
		),
		html.Div(
			html.Class("mb-3"),
			html.Button(
				html.Class("btn btn-primary"),
				html.Type("submit"),
				func() g.Node {
					if isEdit {
						return g.Text("Update Budget")
					}
					return g.Text("Create Budget")
				}(),
			),
			html.Button(
				html.Class("btn btn-secondary ms-2"),
				html.Type("button"),
				html.DataAttr("bs-dismiss", "modal"),
				g.Text("Cancel"),
			),
		),
	)
}

func AddBudgetModal(categories []data.Category, yearlyIncome int) g.Node {
	return html.Div(
		html.Class("modal fade"),
		html.ID("addBudgetModal"),
		html.DataAttr("tabindex", "-1"),
		html.DataAttr("aria-labelledby", "addBudgetModalLabel"),
		html.DataAttr("aria-hidden", "true"),
		html.Div(
			html.Class("modal-dialog"),
			html.Div(
				html.Class("modal-content"),
				html.Div(
					html.Class("modal-header"),
					html.H5(html.Class("modal-title"), html.ID("addBudgetModalLabel"), g.Text("Add Budget")),
					html.Button(
						html.Class("btn-close"),
						html.Type("button"),
						html.DataAttr("bs-dismiss", "modal"),
						html.DataAttr("aria-label", "Close"),
					),
				),
				html.Div(
					html.Class("modal-body"),
					html.ID("addBudgetModalBody"),
					BudgetForm(nil, categories, yearlyIncome, false),
				),
			),
		),
	)
}

func EditBudgetModal() g.Node {
	return html.Div(
		html.Class("modal fade"),
		html.ID("editBudgetModal"),
		html.DataAttr("tabindex", "-1"),
		html.DataAttr("aria-labelledby", "editBudgetModalLabel"),
		html.DataAttr("aria-hidden", "true"),
		html.Div(
			html.Class("modal-dialog"),
			html.Div(
				html.Class("modal-content"),
				html.Div(
					html.Class("modal-header"),
					html.H5(html.Class("modal-title"), html.ID("editBudgetModalLabel"), g.Text("Edit Budget")),
					html.Button(
						html.Class("btn-close"),
						html.Type("button"),
						html.DataAttr("bs-dismiss", "modal"),
						html.DataAttr("aria-label", "Close"),
					),
				),
				html.Div(
					html.Class("modal-body"),
					html.ID("editBudgetModalBody"),
				),
			),
		),
	)
}
