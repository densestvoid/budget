package templates

import (
	"github.com/densestvoid/budget/data"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	g "github.com/maragudk/gomponents"
	"github.com/maragudk/gomponents/html"
)

func HomePage() g.Node {
	return g.Group([]g.Node{
		html.Div(
			html.Class("row"),
			html.Div(
				html.Class("col-12"),
				html.H1(html.Class("display-4 mb-4 text-center"), g.Text("Welcome to Budget App")),
				html.P(html.Class("lead"), g.Text("A modern budget management application built with Go, HTMX, and Alpine.js")),
			),
		),
		html.Div(
			html.Class("row mt-5"),
			html.Div(
				html.Class("col-md-6"),
				html.Div(
					html.Class("card h-100"),
					html.Div(
						html.Class("card-body"),
						html.H5(html.Class("card-title"), g.Text("HTMX Integration")),
						html.P(html.Class("card-text"), g.Text("Experience dynamic web interactions without writing JavaScript. HTMX allows you to access modern browser features directly from HTML.")),
						html.Button(
							html.Class("btn btn-primary"),
							html.DataAttr("hx-get", "/api/health"),
							html.DataAttr("hx-target", "#demo-output"),
							g.Text("Test HTMX"),
						),
						html.Div(
							html.ID("demo-output"),
							html.Class("mt-3 p-3 rounded"),
							g.Text("Click the button above to test HTMX..."),
						),
					),
				),
			),
			html.Div(
				html.Class("col-md-6"),
				html.Div(
					html.Class("card h-100"),
					html.Div(
						html.Class("card-body"),
						html.H5(html.Class("card-title"), g.Text("Alpine.js Demo")),
						html.P(html.Class("card-text"), g.Text("Alpine.js provides reactive and declarative behavior for your HTML using a lightweight JavaScript framework.")),
						html.Div(
							g.Attr("x-data", "{ count: 0 }"),
							html.P(
								g.Text("Count: "),
								html.Span(g.Attr("x-text", "count"), html.Class("fw-bold")),
							),
							html.Button(
								html.Class("btn btn-success me-2"),
								g.Attr("x-on:click", "count++"),
								g.Text("Increment"),
							),
							html.Button(
								html.Class("btn btn-danger"),
								g.Attr("x-on:click", "count = 0"),
								g.Text("Reset"),
							),
						),
					),
				),
			),
		),
		html.Div(
			html.Class("row mt-4"),
			html.Div(
				html.Class("col-12"),
				html.Div(
					html.Class("alert alert-info"),
					html.H6(html.Class("alert-heading"), g.Text("Technologies Used:")),
					html.Ul(
						html.Li(g.Text("Go with Chi router")),
						html.Li(g.Text("PostgreSQL database")),
						html.Li(g.Text("Goose for migrations")),
						html.Li(g.Text("HTMX for dynamic interactions")),
						html.Li(g.Text("Alpine.js for reactive UI")),
						html.Li(g.Text("Bootstrap 5 for styling")),
						html.Li(g.Text("Gomponents for HTML generation")),
					),
				),
			),
		),
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
func TransactionsPage(transactions []data.Transaction, categories []data.Category, errMsg ...string) g.Node {
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
				// Error banner if present
				g.Group(func() []g.Node {
					if len(errMsg) > 0 && errMsg[0] != "" {
						return []g.Node{html.Div(html.Class("alert alert-danger"), g.Text(errMsg[0]))}
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
								html.Class("input-group mb-3"),
								html.Input(html.Type("file"), html.Name("csv"), html.Class("form-control"), html.Required()),
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
			},
			init() {
				// Watch for form reset events and sync Alpine state
				this.$watch('selectedValue', (value) => {
					const hiddenInput = this.$el.querySelector('input[name="%s"]');
					if (hiddenInput) {
						hiddenInput.value = value;
					}
				});
			}
		}`, escapedSelectedValue, escapedSelectedDisplay, selectName)),
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
			g.Attr("x-model", "selectedValue"),
		),
	)
}
