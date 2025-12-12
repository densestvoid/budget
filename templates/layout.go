package templates

import (
	"strconv"

	"github.com/densestvoid/budget/data"

	g "github.com/maragudk/gomponents"
	"github.com/maragudk/gomponents/html"
)

// BaseLayoutWithAuth creates the base layout with authentication-aware content
func BaseLayoutWithAuth(title string, isAuthenticated bool, content ...g.Node) g.Node {
	return BaseLayoutWithAuthAndBudgetPlan(title, isAuthenticated, nil, 0, content...)
}

// BaseLayoutWithAuthAndBudgetPlan creates the base layout with authentication-aware content and budget plan selector
func BaseLayoutWithAuthAndBudgetPlan(title string, isAuthenticated bool, budgetPlans []data.BudgetPlan, selectedBudgetPlanID int, content ...g.Node) g.Node {
	return html.Doctype(
		html.HTML(
			html.Lang("en"),
			g.Attr("data-bs-theme", "dark"),
			html.Head(
				html.Meta(html.Charset("utf-8")),
				html.Meta(html.Name("viewport"), html.Content("width=device-width, initial-scale=1, user-scalable=no")),
				html.TitleEl(g.Text(title)),

				// Bootstrap 5 CSS
				html.Link(html.Rel("stylesheet"), html.Href("https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/css/bootstrap.min.css")),

				// Bootstrap Icons
				html.Link(html.Rel("stylesheet"), html.Href("https://cdn.jsdelivr.net/npm/bootstrap-icons@1.11.3/font/bootstrap-icons.css")),

				// HTMX
				html.Script(html.Src("https://unpkg.com/htmx.org@1.9.9")),

				// Alpine.js
				html.Script(html.Defer(), html.Src("https://unpkg.com/alpinejs@3.x.x/dist/cdn.min.js")),

				// Custom CSS for category dropdown
				g.El("style", g.Raw(`
					.category-dropdown option[selected] {
						padding-left: 0 !important;
					}
					@media (min-width: 992px) {
						.sidebar-layout-row {
							flex-wrap: nowrap !important;
						}
					}
				`)),
			),
			html.Body(
				html.Class("min-vh-100 d-flex flex-column"),
				// Top navigation bar
				html.Nav(
					html.Class("navbar navbar-expand-lg navbar-dark bg-primary sticky-top"),
					html.Div(
						html.Class("container-fluid"),
						html.Button(
							html.Class("btn btn-link text-light d-lg-none me-2"),
							html.Type("button"),
							html.DataAttr("bs-toggle", "offcanvas"),
							html.DataAttr("bs-target", "#sidePane"),
							html.DataAttr("aria-controls", "sidePane"),
							html.DataAttr("aria-label", "Toggle navigation"),
							html.I(html.Class("bi bi-list")),
						),
						html.A(
							html.Class("navbar-brand"),
							html.Href("/"),
							g.Text("Budget App"),
						),
					),
				),

				// Main content wrapper
				html.Div(
					html.Class("container-fluid px-2"),
					html.Div(
						html.Class("row sidebar-layout-row"),

						// Sidebar (visible on lg+ screens)
						html.Div(
							html.Class("d-none d-lg-block"),
							g.Attr("style", "width: 280px; flex-shrink: 0;"),
							html.Div(
								html.Class("position-fixed h-100 bg-body-secondary border-end"),
								g.Attr("style", "width: 280px; top: 56px; left: 0; z-index: 1000; overflow-y: auto;"),

								// Sidebar content
								html.Div(
									html.Class("p-3"),

									// Budget plan selector (if authenticated and plans available)
									func() g.Node {
										if isAuthenticated && len(budgetPlans) > 0 {
											return html.Div(
												html.Class("mb-4"),
												html.Label(
													html.Class("form-label text-muted"),
													g.Attr("style", "font-size: 0.875rem;"),
													g.Text("Budget Plan"),
												),
												html.Select(
													html.Class("form-select form-select-sm"),
													g.Attr("onchange", "const url = new URL(window.location.href); url.searchParams.set('budget_plan_id', this.value); window.location.href = url.toString();"),
													func() g.Node {
														var options []g.Node
														for _, plan := range budgetPlans {
															planName := plan.Name
															if plan.IsActive {
																planName = plan.Name + " (Active)"
															}
															opt := html.Option(
																html.Value(strconv.Itoa(plan.ID)),
																func() g.Node {
																	if plan.ID == selectedBudgetPlanID {
																		return html.Selected()
																	}
																	return nil
																}(),
																g.Text(planName),
															)
															options = append(options, opt)
														}
														return g.Group(options)
													}(),
												),
											)
										}
										return nil
									}(),
									// Navigation section
									html.Div(
										html.Class("mb-4"),
										html.H6(
											html.Class("text-muted mb-3"),
											g.Text("Navigation"),
										),
										html.Ul(
											html.Class("nav flex-column"),
											html.Li(
												html.Class("nav-item"),
												html.A(
													html.Class("nav-link"),
													html.Href("/"),
													g.Text("🏠 Home"),
												),
											),
											html.Li(
												html.Class("nav-item"),
												html.A(
													html.Class("nav-link"),
													html.Href("/about"),
													g.Text("ℹ️ About"),
												),
											),
											g.Group(func() []g.Node {
												if isAuthenticated {
													return []g.Node{
														html.Li(
															html.Class("nav-item"),
															html.A(
																html.Class("nav-link"),
																html.Href("/categories/"),
																g.Text("📂 Categories"),
															),
														),
														html.Li(
															html.Class("nav-item"),
															html.A(
																html.Class("nav-link"),
																html.Href("/transactions/"),
																g.Text("💸 Transactions"),
															),
														),
														html.Li(
															html.Class("nav-item"),
															html.A(
																html.Class("nav-link"),
																html.Href("/financial-accounts/"),
																g.Text("🏦 Financial Accounts"),
															),
														),
														html.Li(
															html.Class("nav-item"),
															html.A(
																html.Class("nav-link"),
																html.Href("/rules/"),
																g.Text("🧩 Rules"),
															),
														),
														html.Li(
															html.Class("nav-item"),
															html.A(
																html.Class("nav-link"),
																html.Href("/budget-plans"),
																g.Text("📊 Budget Plans"),
															),
														),
														html.Li(
															html.Class("nav-item"),
															html.A(
																html.Class("nav-link"),
																html.Href("/bills/"),
																g.Text("📄 Bills"),
															),
														),
														html.Li(
															html.Class("nav-item"),
															html.A(
																html.Class("nav-link"),
																html.Href("/budgets"),
																g.Text("💰 Budgets"),
															),
														),
														html.Li(
															html.Class("nav-item"),
															html.A(
																html.Class("nav-link"),
																html.Href("/paycheck-summary"),
																g.Text("💰 Paycheck Summary"),
															),
														),
													}
												}
												return nil
											}()),
										),
									),

									// Authentication section
									html.Div(
										html.Class("border-top pt-3"),
										html.H6(
											html.Class("text-muted mb-3"),
											g.Text("Account"),
										),

										// Conditional rendering for sidebar
										g.Group(func() []g.Node {
											if isAuthenticated {
												return []g.Node{
													g.El("form",
														html.Action("/auth/logout"),
														html.Method("POST"),
														html.Class(""),
														html.ID("logout-form"),
														html.Button(
															html.Class("btn btn-outline-danger btn-sm w-100"),
															html.Type("submit"),
															g.Text("Logout"),
														),
													),
												}
											} else {
												return []g.Node{
													html.Button(
														html.Class("btn btn-primary btn-sm w-100 mb-2"),
														html.Type("button"),
														html.DataAttr("bs-toggle", "modal"),
														html.DataAttr("bs-target", "#authModal"),
														g.Text("Login / Register"),
													),
												}
											}
										}()),
									),
								),
							),
						),

						// Main content area
						html.Div(
							html.Class("col-12"),
							g.Attr("style", "flex: 1; min-width: 0;"),
							html.Main(
								html.Div(
									html.Class("mx-2"),
									g.Group(content),
								),
							),
						),
					),
				),

				// Bootstrap Offcanvas for mobile
				html.Div(
					html.Class("offcanvas offcanvas-start"),
					html.ID("sidePane"),
					html.DataAttr("tabindex", "-1"),
					html.DataAttr("aria-labelledby", "sidePaneLabel"),

					// Offcanvas header
					html.Div(
						html.Class("offcanvas-header"),
						html.H5(
							html.Class("offcanvas-title"),
							html.ID("sidePaneLabel"),
							g.Text("Budget App"),
						),
						html.Button(
							html.Class("btn-close"),
							html.Type("button"),
							html.DataAttr("bs-dismiss", "offcanvas"),
							html.DataAttr("aria-label", "Close"),
						),
					),

					// Offcanvas body
					html.Div(
						html.Class("offcanvas-body"),

						// Budget plan selector (if authenticated and plans available)
						func() g.Node {
							if isAuthenticated && len(budgetPlans) > 0 {
								return html.Div(
									html.Class("mb-4"),
									html.Label(
										html.Class("form-label text-muted"),
										g.Attr("style", "font-size: 0.875rem;"),
										g.Text("Budget Plan"),
									),
									html.Select(
										html.Class("form-select form-select-sm"),
										g.Attr("onchange", "const url = new URL(window.location.href); url.searchParams.set('budget_plan_id', this.value); window.location.href = url.toString();"),
										func() g.Node {
											var options []g.Node
											for _, plan := range budgetPlans {
												planName := plan.Name
												if plan.IsActive {
													planName = plan.Name + " (Active)"
												}
												opt := html.Option(
													html.Value(strconv.Itoa(plan.ID)),
													func() g.Node {
														if plan.ID == selectedBudgetPlanID {
															return html.Selected()
														}
														return nil
													}(),
													g.Text(planName),
												)
												options = append(options, opt)
											}
											return g.Group(options)
										}(),
									),
								)
							}
							return nil
						}(),
						// Navigation section
						html.Div(
							html.Class("mb-4"),
							html.H6(
								html.Class("text-muted mb-3"),
								g.Text("Navigation"),
							),
							html.Ul(
								html.Class("nav flex-column"),
								html.Li(
									html.Class("nav-item"),
									html.A(
										html.Class("nav-link"),
										html.Href("/"),
										g.Text("🏠 Home"),
									),
								),
								html.Li(
									html.Class("nav-item"),
									html.A(
										html.Class("nav-link"),
										html.Href("/about"),
										g.Text("ℹ️ About"),
									),
								),
								g.Group(func() []g.Node {
									if isAuthenticated {
										return []g.Node{
											html.Li(
												html.Class("nav-item"),
												html.A(
													html.Class("nav-link"),
													html.Href("/categories/"),
													g.Text("📂 Categories"),
												),
											),
											html.Li(
												html.Class("nav-item"),
												html.A(
													html.Class("nav-link"),
													html.Href("/transactions/"),
													g.Text("💸 Transactions"),
												),
											),
											html.Li(
												html.Class("nav-item"),
												html.A(
													html.Class("nav-link"),
													html.Href("/financial-accounts/"),
													g.Text("🏦 Financial Accounts"),
												),
											),
											html.Li(
												html.Class("nav-item"),
												html.A(
													html.Class("nav-link"),
													html.Href("/rules/"),
													g.Text("🧩 Rules"),
												),
											),
											html.Li(
												html.Class("nav-item"),
												html.A(
													html.Class("nav-link"),
													html.Href("/budget-plans"),
													g.Text("📊 Budget Plans"),
												),
											),
											html.Li(
												html.Class("nav-item"),
												html.A(
													html.Class("nav-link"),
													html.Href("/bills/"),
													g.Text("📄 Bills"),
												),
											),
											html.Li(
												html.Class("nav-item"),
												html.A(
													html.Class("nav-link"),
													html.Href("/budgets"),
													g.Text("💰 Budgets"),
												),
											),
											html.Li(
												html.Class("nav-item"),
												html.A(
													html.Class("nav-link"),
													html.Href("/paycheck-summary"),
													g.Text("💰 Paycheck Summary"),
												),
											),
										}
									}
									return nil
								}()),
							),
						),

						// Authentication section
						html.Div(
							html.Class("border-top pt-3"),
							html.H6(
								html.Class("text-muted mb-3"),
								g.Text("Account"),
							),

							// Conditional rendering for offcanvas
							g.Group(func() []g.Node {
								if isAuthenticated {
									return []g.Node{
										g.El("form",
											html.Action("/auth/logout"),
											html.Method("POST"),
											html.Class(""),
											html.ID("logout-form-mobile"),
											html.Button(
												html.Class("btn btn-outline-danger btn-sm w-100"),
												html.Type("submit"),
												g.Text("Logout"),
											),
										),
									}
								} else {
									return []g.Node{
										html.Button(
											html.Class("btn btn-primary btn-sm w-100 mb-2"),
											html.Type("button"),
											html.DataAttr("bs-toggle", "modal"),
											html.DataAttr("bs-target", "#authModal"),
											g.Text("Login / Register"),
										),
									}
								}
							}()),
						),
					),
				),

				// Authentication Modal
				html.Div(
					html.Class("modal fade"),
					html.ID("authModal"),
					html.DataAttr("tabindex", "-1"),
					html.DataAttr("aria-labelledby", "authModalLabel"),
					html.DataAttr("aria-hidden", "true"),
					html.Div(
						html.Class("modal-dialog"),
						html.Div(
							html.Class("modal-content"),
							html.Div(
								html.Class("modal-header"),
								html.H5(
									html.Class("modal-title"),
									html.ID("authModalLabel"),
									g.Text("Authentication"),
								),
								html.Button(
									html.Class("btn-close"),
									html.Type("button"),
									html.DataAttr("bs-dismiss", "modal"),
									html.DataAttr("aria-label", "Close"),
								),
							),
							html.Div(
								html.Class("modal-body"),
								g.Attr("x-data", "{ showLogin: true }"),

								// Login form
								g.El("div",
									g.Attr("x-show", "showLogin"),
									g.El("form",
										html.Action("/auth/login"),
										html.Method("POST"),
										html.Div(
											html.Class("mb-3"),
											html.Label(
												html.Class("form-label"),
												html.For("login-email"),
												g.Text("Email"),
											),
											html.Input(
												html.Class("form-control"),
												html.Type("email"),
												html.ID("login-email"),
												html.Name("email"),
												html.Required(),
											),
										),
										html.Div(
											html.Class("mb-3"),
											html.Label(
												html.Class("form-label"),
												html.For("login-password"),
												g.Text("Password"),
											),
											html.Input(
												html.Class("form-control"),
												html.Type("password"),
												html.ID("login-password"),
												html.Name("password"),
												html.Required(),
											),
										),
										html.Div(
											html.Class("d-flex gap-2"),
											html.Button(
												html.Class("btn btn-primary flex-fill"),
												html.Type("submit"),
												g.Text("Login"),
											),
											html.Button(
												html.Class("btn btn-outline-secondary"),
												html.Type("button"),
												g.Attr("@click", "showLogin = false"),
												g.Text("Register"),
											),
										),
									),
								),

								// Register form
								g.El("div",
									g.Attr("x-show", "!showLogin"),
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
										html.Div(
											html.Class("d-flex gap-2"),
											html.Button(
												html.Class("btn btn-success flex-fill"),
												html.Type("submit"),
												g.Text("Register"),
											),
											html.Button(
												html.Class("btn btn-outline-secondary"),
												html.Type("button"),
												g.Attr("@click", "showLogin = true"),
												g.Text("Back to Login"),
											),
										),
									),
								),
							),
						),
					),
				),

				// Footer
				html.Footer(
					html.Class("mt-5 py-4"),
					html.Div(
						html.Class("container-fluid text-center"),
						g.Text("© 2024 Budget App. Built with Go, Chi, HTMX, and Alpine.js"),
					),
				),

				// Bootstrap 5 JS
				html.Script(html.Src("https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/js/bootstrap.bundle.min.js")),
			),
		),
	)
}

// BaseLayout for backward compatibility (defaults to not authenticated)
func BaseLayout(title string, content ...g.Node) g.Node {
	return BaseLayoutWithAuth(title, false, content...)
}
