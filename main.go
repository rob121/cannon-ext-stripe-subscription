package main

import (
	"embed"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/rob121/cannon/extension"
)

//go:embed help/*.md
var helpFiles embed.FS

//go:embed all:templates
var templateFiles embed.FS

const (
	extensionName        = "cannon-ext-stripe-subscription"
	extensionTitle       = "Stripe Subscriptions"
	extensionDescription = "Sell recurring plans with Stripe Checkout, Customer Portal, and webhooks. Admin manages products and subscribers."
	extensionVersion     = "0.1.7"
	updateURLBase        = "https://github.com/rob121/cannon-ext-stripe-subscription/releases/download"
	defaultAdminMenu     = "Subscriptions"
	pageSize             = 20
)

var stripeConfigSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "mode": {
      "type": "string",
      "title": "Default Mode",
      "description": "Used when a product is not forced into test mode.",
      "enum": ["test", "live"],
      "default": "test"
    },
    "secret_key_live": {
      "type": "string",
      "title": "Secret Key (Live)",
      "description": "Must start with sk_live_… from Stripe Dashboard (Live mode)."
    },
    "secret_key_test": {
      "type": "string",
      "title": "Secret Key (Test)",
      "description": "Must start with sk_test_… from Stripe Dashboard (Test mode toggle). Do not paste a live key here."
    },
    "publishable_key_live": {
      "type": "string",
      "title": "Publishable Key (Live)",
      "description": "pk_live_…"
    },
    "publishable_key_test": {
      "type": "string",
      "title": "Publishable Key (Test)",
      "description": "pk_test_…"
    },
    "webhook_secret": {
      "type": "string",
      "title": "Webhook Signing Secret",
      "description": "From the Stripe Dashboard endpoint (whsec_…). Point Stripe at https://yoursite/webhooks/stripe"
    },
    "success_path": {
      "type": "string",
      "title": "Checkout Success Path",
      "description": "Optional override, e.g. /subscribe/return. Defaults to the return page route if configured."
    },
    "cancel_path": {
      "type": "string",
      "title": "Checkout Cancel Path",
      "description": "Optional path when the customer abandons Checkout."
    }
  }
}`)

var stripeConfigUI = json.RawMessage(`{
  "type": "VerticalLayout",
  "elements": [
    {"type": "Control", "scope": "#/properties/mode"},
    {"type": "Control", "scope": "#/properties/secret_key_test"},
    {"type": "Control", "scope": "#/properties/publishable_key_test"},
    {"type": "Control", "scope": "#/properties/secret_key_live"},
    {"type": "Control", "scope": "#/properties/publishable_key_live"},
    {"type": "Control", "scope": "#/properties/webhook_secret"},
    {"type": "Control", "scope": "#/properties/success_path"},
    {"type": "Control", "scope": "#/properties/cancel_path"}
  ]
}`)

type stripeApp struct {
	info             extension.Info
	templates        *extension.Templates
	settingsTable    string
	productsTable    string
	subscribersTable string

	schemaOnce sync.Once
	schemaErr  error
}

func main() {
	_, s := newServer()
	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}

func newServer() (*stripeApp, *extension.Server) {
	info := extension.Info{
		Name:          extensionName,
		Title:         extensionTitle,
		Description:   extensionDescription,
		Version:       extensionVersion,
		UpdateURLBase: updateURLBase,
		AdminMenuName: defaultAdminMenu,
	}
	app := &stripeApp{
		info:             info,
		templates:        extension.NewTemplates(templateFiles, "templates"),
		settingsTable:    extension.TableName(info.TablePrefix(), "settings"),
		productsTable:    extension.TableName(info.TablePrefix(), "products"),
		subscribersTable: extension.TableName(info.TablePrefix(), "subscribers"),
	}

	s := extension.New(info)
	s.EmbedHelp(helpFiles, "help")
	s.EmbedTemplates(templateFiles, "templates")
	s.OnInstall(app.install)
	s.RegisterPermissions([]extension.PermissionDef{
		{ID: "stripe.view", DisplayName: "View subscriptions admin", Description: "View products and subscribers.", Category: "stripe"},
		{ID: "stripe.manage", DisplayName: "Manage subscriptions", Description: "Create and edit products and subscribers.", Category: "stripe"},
		{ID: "stripe.setup", DisplayName: "Configure Stripe keys", Description: "Edit Stripe API keys and webhook secret.", Category: "stripe", Dangerous: true},
	})
	s.OnConfiguration(extension.MapConfiguration([]extension.ConfigurationDefinition{
		{
			ID:       "general",
			Title:    "Stripe",
			Schema:   stripeConfigSchema,
			UISchema: stripeConfigUI,
		},
	}, configStore{table: app.settingsTable}))

	s.OnPageList(app.pageDefinitions)
	s.RegisterPage(extension.PageDefinition{ID: "product", Title: "Subscription Product"}, app.productPage)
	s.RegisterPage(extension.PageDefinition{ID: "account", Title: "My Subscriptions"}, app.accountPage)
	s.RegisterPage(extension.PageDefinition{ID: "return", Title: "Checkout Return"}, app.returnPage)

	s.RegisterEndpoint(extension.EndpointDefinition{ID: "checkout", Title: "Start Checkout"}, app.checkoutEndpoint)
	s.RegisterEndpoint(extension.EndpointDefinition{ID: "portal", Title: "Billing Portal"}, app.portalEndpoint)
	s.RegisterEndpoint(extension.EndpointDefinition{ID: "cancel", Title: "Cancel Subscription"}, app.cancelEndpoint)
	s.RegisterEndpoint(extension.EndpointDefinition{ID: "webhook", Title: "Stripe Webhook"}, app.webhookEndpoint)

	s.HandleAdmin("/admin/", app.admin)
	return app, s
}

func (app *stripeApp) install(req extension.WireRequest) error {
	db, _, err := extension.OpenDB()
	if err != nil {
		return err
	}
	return app.migrateTables(db)
}

func (app *stripeApp) pageDefinitions() ([]extension.PageDefinition, error) {
	productOpts := app.activeProductOptions()
	return []extension.PageDefinition{
		{
			ID:    "product",
			Title: "Subscription Product",
			Fields: []extension.PageField{
				{Name: "product_id", Label: "Product", Type: "select", Required: true, Options: productOpts},
				{Name: "title", Label: "Page Title", Type: "text"},
				{Name: "introduction", Label: "Introduction", Type: "textarea"},
				{Name: "checkout_path", Label: "Checkout Endpoint Path", Type: "text", Help: "Public Extension Endpoint path for checkout, e.g. /subscribe/checkout"},
				{Name: "account_path", Label: "Account Path", Type: "text", Help: "Path to My Subscriptions page, e.g. /subscribe/account"},
			},
		},
		{
			ID:    "account",
			Title: "My Subscriptions",
			Fields: []extension.PageField{
				{Name: "title", Label: "Page Title", Type: "text"},
				{Name: "introduction", Label: "Introduction", Type: "textarea"},
				{Name: "portal_path", Label: "Portal Endpoint Path", Type: "text", Help: "e.g. /subscribe/portal"},
				{Name: "cancel_path", Label: "Cancel Endpoint Path", Type: "text", Help: "e.g. /subscribe/cancel"},
			},
		},
		{
			ID:    "return",
			Title: "Checkout Return",
			Fields: []extension.PageField{
				{Name: "title", Label: "Page Title", Type: "text"},
				{Name: "introduction", Label: "Introduction", Type: "textarea"},
				{Name: "account_path", Label: "Account Path", Type: "text", Help: "e.g. /subscribe/account"},
			},
		},
	}, nil
}

func (app *stripeApp) render(name string, data map[string]any) extension.WireResponse {
	body, err := app.templates.WithFuncs(template.FuncMap{
		"money": func(v float64) string {
			return strconv.FormatFloat(v, 'f', 2, 64)
		},
		"nl2br": func(s string) template.HTML {
			escaped := template.HTMLEscapeString(s)
			return template.HTML(strings.ReplaceAll(escaped, "\n", "<br>\n"))
		},
	}).Execute(name, data)
	if err != nil {
		return extension.Error(http.StatusInternalServerError, err.Error())
	}
	return extension.HTML(http.StatusOK, body)
}

func csrfField(req extension.WireRequest) template.HTML {
	return template.HTML(extension.CSRFHiddenField(req))
}

func parseWireURL(raw string) (*url.URL, error) {
	return url.Parse(raw)
}

func adminHomeURL() string {
	return "/admin/extension-apps/" + url.PathEscape(extensionName)
}

func adminRoute(path string) (base string, parts []string) {
	path = strings.TrimSpace(path)
	marker := "/admin/extension-apps/" + extensionName
	idx := strings.Index(path, marker)
	if idx < 0 {
		return "", nil
	}
	base = path[:idx+len(marker)]
	rest := strings.Trim(path[idx+len(marker):], "/")
	if rest == "" {
		return base, nil
	}
	return base, strings.Split(rest, "/")
}

func parseID(s string) (uint, bool) {
	n, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	if err != nil || n == 0 {
		return 0, false
	}
	return uint(n), true
}

func parsePage(u *url.URL) int {
	page, _ := strconv.Atoi(u.Query().Get("page"))
	if page < 1 {
		return 1
	}
	return page
}

func pageString(req extension.WireRequest, key, fallback string) string {
	if v := strings.TrimSpace(extension.PageDataString(req, key)); v != "" {
		return v
	}
	return fallback
}

func requireLogin(req extension.WireRequest) extension.WireResponse {
	returnURL := "/"
	if u, err := url.Parse(req.URL); err == nil && u.RequestURI() != "" {
		returnURL = u.RequestURI()
	}
	return extension.Redirect(http.StatusSeeOther, "/auth/login?return="+url.QueryEscape(returnURL))
}

func absoluteURL(req extension.WireRequest, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		path = "/"
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if base := requestBaseURL(req); base != "" {
		return strings.TrimRight(base, "/") + path
	}
	if site, err := extension.SiteConfig(); err == nil {
		if host := strings.TrimSpace(site.Host); host != "" {
			return strings.TrimRight(host, "/") + path
		}
	}
	return path
}

// requestBaseURL derives scheme+host from the browser request. Wire URLs are often
// path-only (no host), which Stripe rejects for success_url / cancel_url.
func requestBaseURL(req extension.WireRequest) string {
	if origin := strings.TrimSpace(headerGet(req, "Origin")); origin != "" {
		return strings.TrimRight(origin, "/")
	}
	if ref := strings.TrimSpace(headerGet(req, "Referer")); ref != "" {
		if u, err := url.Parse(ref); err == nil && u.Scheme != "" && u.Host != "" {
			return u.Scheme + "://" + u.Host
		}
	}
	if u, err := url.Parse(req.URL); err == nil && u.Scheme != "" && u.Host != "" {
		return u.Scheme + "://" + u.Host
	}
	host := strings.TrimSpace(headerGet(req, "X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(headerGet(req, "Host"))
	}
	if host == "" {
		return ""
	}
	if i := strings.Index(host, ","); i >= 0 {
		host = strings.TrimSpace(host[:i])
	}
	proto := strings.ToLower(strings.TrimSpace(headerGet(req, "X-Forwarded-Proto")))
	if i := strings.Index(proto, ","); i >= 0 {
		proto = strings.TrimSpace(proto[:i])
	}
	if proto != "http" && proto != "https" {
		if strings.Contains(host, "localhost") || strings.HasPrefix(host, "127.") || strings.HasPrefix(host, "0.0.0.0") {
			proto = "http"
		} else {
			proto = "https"
		}
	}
	return proto + "://" + host
}

func formValues(body string) (url.Values, error) {
	return url.ParseQuery(body)
}

func headerGet(req extension.WireRequest, key string) string {
	if req.Header == nil {
		return ""
	}
	for k, vals := range req.Header {
		if strings.EqualFold(k, key) && len(vals) > 0 {
			return vals[0]
		}
	}
	return ""
}
