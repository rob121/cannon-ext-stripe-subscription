package main

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rob121/cannon/extension"
	"gorm.io/gorm"
)

func (app *stripeApp) admin(req extension.WireRequest) extension.WireResponse {
	if !extension.UserAuthenticated(req) {
		return extension.Error(http.StatusForbidden, "sign in required")
	}
	canManage := adminCanManage(req)
	if !adminCanView(req) {
		return extension.Error(http.StatusForbidden, "permission denied")
	}

	db, err := app.db()
	if err != nil {
		return extension.Error(http.StatusInternalServerError, err.Error())
	}
	if err := app.migrateTables(db); err != nil {
		return extension.Error(http.StatusInternalServerError, err.Error())
	}

	adminURL, err := parseWireURL(req.URL)
	if err != nil {
		return extension.Error(http.StatusBadRequest, err.Error())
	}
	basePath, parts := adminRoute(adminURL.Path)
	if basePath == "" {
		basePath = adminHomeURL()
	}
	csrf := csrfField(req)

	switch {
	case len(parts) == 0:
		return app.adminHome(db, adminURL, basePath, csrf, canManage)
	case len(parts) == 1 && parts[0] == "products":
		return app.listProducts(db, adminURL, basePath, "", csrf, canManage)
	case len(parts) == 2 && parts[0] == "products" && parts[1] == "new":
		if !canManage {
			return extension.Error(http.StatusForbidden, "permission denied")
		}
		if strings.EqualFold(req.Method, http.MethodPost) {
			return app.saveProduct(db, req.Body, adminURL, basePath, 0, csrf)
		}
		return app.renderProductForm(basePath, product{Status: "active", Period: "monthly"}, true, "", csrf)
	case len(parts) == 2 && parts[0] == "products":
		id, ok := parseID(parts[1])
		if !ok {
			return extension.Error(http.StatusNotFound, "product not found")
		}
		if strings.EqualFold(req.Method, http.MethodPost) {
			if !canManage {
				return extension.Error(http.StatusForbidden, "permission denied")
			}
			return app.saveProduct(db, req.Body, adminURL, basePath, id, csrf)
		}
		return app.editProduct(db, basePath, id, "", csrf)
	case len(parts) == 3 && parts[0] == "products" && parts[2] == "delete":
		if !canManage {
			return extension.Error(http.StatusForbidden, "permission denied")
		}
		id, ok := parseID(parts[1])
		if !ok {
			return extension.Error(http.StatusNotFound, "product not found")
		}
		if !strings.EqualFold(req.Method, http.MethodPost) {
			return extension.Error(http.StatusMethodNotAllowed, "method not allowed")
		}
		if err := db.Table(app.productsTable).Where("product_id = ?", id).Delete(&product{}).Error; err != nil {
			return extension.Error(http.StatusInternalServerError, err.Error())
		}
		return app.listProducts(db, adminURL, basePath, "Product deleted.", csrf, canManage)
	case len(parts) == 1 && parts[0] == "subscribers":
		return app.listSubscribers(db, adminURL, basePath, "", csrf, canManage)
	case len(parts) == 2 && parts[0] == "subscribers" && parts[1] == "new":
		if !canManage {
			return extension.Error(http.StatusForbidden, "permission denied")
		}
		if strings.EqualFold(req.Method, http.MethodPost) {
			return app.saveSubscriber(db, req.Body, adminURL, basePath, 0, csrf)
		}
		return app.renderSubscriberForm(db, basePath, subscriber{Status: "active"}, true, "", csrf)
	case len(parts) == 2 && parts[0] == "subscribers":
		id, ok := parseID(parts[1])
		if !ok {
			return extension.Error(http.StatusNotFound, "subscriber not found")
		}
		if strings.EqualFold(req.Method, http.MethodPost) {
			if !canManage {
				return extension.Error(http.StatusForbidden, "permission denied")
			}
			return app.saveSubscriber(db, req.Body, adminURL, basePath, id, csrf)
		}
		return app.editSubscriber(db, basePath, id, "", csrf)
	case len(parts) == 3 && parts[0] == "subscribers" && parts[2] == "delete":
		if !canManage {
			return extension.Error(http.StatusForbidden, "permission denied")
		}
		id, ok := parseID(parts[1])
		if !ok {
			return extension.Error(http.StatusNotFound, "subscriber not found")
		}
		if !strings.EqualFold(req.Method, http.MethodPost) {
			return extension.Error(http.StatusMethodNotAllowed, "method not allowed")
		}
		if err := db.Table(app.subscribersTable).Where("subscriber_id = ?", id).Delete(&subscriber{}).Error; err != nil {
			return extension.Error(http.StatusInternalServerError, err.Error())
		}
		return app.listSubscribers(db, adminURL, basePath, "Subscriber deleted.", csrf, canManage)
	default:
		return extension.Error(http.StatusNotFound, "admin route not found")
	}
}

func (app *stripeApp) adminHome(db *gorm.DB, adminURL *url.URL, basePath string, csrf template.HTML, canManage bool) extension.WireResponse {
	var productTotal, activeProducts, subscriberTotal, activeSubs int64
	_ = db.Table(app.productsTable).Count(&productTotal).Error
	_ = db.Table(app.productsTable).Where("status = ?", "active").Count(&activeProducts).Error
	_ = db.Table(app.subscribersTable).Count(&subscriberTotal).Error
	_ = db.Table(app.subscribersTable).Where("status IN ?", []string{"active", "trialing", "past_due"}).Count(&activeSubs).Error
	return app.render("stripe/admin_home.html", map[string]any{
		"Title":           "Stripe Subscriptions",
		"Subtitle":        "Manage plans and subscriber records.",
		"BasePath":        basePath,
		"SettingsURL":     "/admin/configuration/extensions/" + url.PathEscape(extensionName),
		"ProductTotal":    productTotal,
		"ActiveProducts":  activeProducts,
		"SubscriberTotal": subscriberTotal,
		"ActiveSubs":      activeSubs,
		"CanManage":       canManage,
		"Notice":          adminURL.Query().Get("notice"),
		"Error":           adminURL.Query().Get("error"),
		"CSRFField":       csrf,
	})
}

func (app *stripeApp) listProducts(db *gorm.DB, adminURL *url.URL, basePath, notice string, csrf template.HTML, canManage bool) extension.WireResponse {
	page := parsePage(adminURL)
	var total int64
	_ = db.Table(app.productsTable).Count(&total).Error
	var rows []product
	err := db.Table(app.productsTable).Order("sort asc, name asc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error
	if err != nil {
		return extension.Error(http.StatusInternalServerError, err.Error())
	}
	return app.render("stripe/admin_products.html", map[string]any{
		"Title":     "Products",
		"Subtitle":  "Stripe Price IDs and display details for subscription plans.",
		"BasePath":  basePath,
		"Rows":      rows,
		"Page":      page,
		"PageSize":  pageSize,
		"Total":     total,
		"Notice":    notice,
		"CanManage": canManage,
		"CSRFField": csrf,
	})
}

func (app *stripeApp) editProduct(db *gorm.DB, basePath string, id uint, errMsg string, csrf template.HTML) extension.WireResponse {
	var row product
	err := db.Table(app.productsTable).Where("product_id = ?", id).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return extension.Error(http.StatusNotFound, "product not found")
	}
	if err != nil {
		return extension.Error(http.StatusInternalServerError, err.Error())
	}
	return app.renderProductForm(basePath, row, false, errMsg, csrf)
}

func (app *stripeApp) renderProductForm(basePath string, row product, isNew bool, errMsg string, csrf template.HTML) extension.WireResponse {
	title := "Edit Product"
	action := fmt.Sprintf("%s/products/%d", basePath, row.ID)
	if isNew {
		title = "Add Product"
		action = basePath + "/products/new"
	}
	return app.render("stripe/admin_product_form.html", map[string]any{
		"Title":     title,
		"Subtitle":  "Paste Stripe Price IDs from the Dashboard (price_…). Description accepts HTML or Markdown.",
		"BasePath":  basePath,
		"ActionURL": action,
		"CancelURL": basePath + "/products",
		"Row":       row,
		"IsNew":     isNew,
		"Error":     errMsg,
		"CSRFField": csrf,
		"Periods":   []string{"weekly", "monthly", "quarterly", "annually"},
	})
}

func (app *stripeApp) saveProduct(db *gorm.DB, body string, adminURL *url.URL, basePath string, id uint, csrf template.HTML) extension.WireResponse {
	row, errMsg := productFromBody(body, id)
	if errMsg != "" {
		return app.renderProductForm(basePath, row, id == 0, errMsg, csrf)
	}
	if id != 0 {
		var existing product
		err := db.Table(app.productsTable).Where("product_id = ?", id).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			return extension.Error(http.StatusNotFound, "product not found")
		}
		if err != nil {
			return extension.Error(http.StatusInternalServerError, err.Error())
		}
		row.ID = id
		row.CreatedAt = existing.CreatedAt
	}
	if err := db.Table(app.productsTable).Save(&row).Error; err != nil {
		return app.renderProductForm(basePath, row, id == 0, err.Error(), csrf)
	}
	return app.listProducts(db, adminURL, basePath, "Product saved.", csrf, true)
}

func productFromBody(body string, id uint) (product, string) {
	values, err := formValues(body)
	if err != nil {
		return product{}, "invalid form data"
	}
	name := strings.TrimSpace(values.Get("name"))
	if name == "" {
		return product{}, "name is required"
	}
	cost, _ := strconv.ParseFloat(strings.TrimSpace(values.Get("cost")), 64)
	sort, _ := strconv.Atoi(strings.TrimSpace(values.Get("sort")))
	period := strings.TrimSpace(values.Get("period"))
	if period == "" {
		period = "monthly"
	}
	status := strings.TrimSpace(values.Get("status"))
	if status != "active" && status != "inactive" {
		status = "active"
	}
	priceID := strings.TrimSpace(values.Get("stripe_price_id"))
	testPriceID := strings.TrimSpace(values.Get("stripe_test_price_id"))
	if priceID == "" && testPriceID == "" {
		return product{ID: id, Name: name, Cost: cost, Period: period, Status: status, Sort: sort}, "at least one Stripe Price ID is required"
	}
	return product{
		ID:                id,
		Name:              name,
		Description:       strings.TrimSpace(values.Get("description")),
		Cost:              cost,
		Period:            period,
		StripePriceID:     priceID,
		StripeTestPriceID: testPriceID,
		TestMode:          values.Get("test_mode") == "1" || values.Get("test_mode") == "on",
		Status:            status,
		Sort:              sort,
		UpdatedAt:         time.Now().UTC(),
		CreatedAt:         time.Now().UTC(),
	}, ""
}

func (app *stripeApp) listSubscribers(db *gorm.DB, adminURL *url.URL, basePath, notice string, csrf template.HTML, canManage bool) extension.WireResponse {
	page := parsePage(adminURL)
	var total int64
	_ = db.Table(app.subscribersTable).Count(&total).Error
	var rows []subscriber
	err := db.Table(app.subscribersTable).Order("subscriber_id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error
	if err != nil {
		return extension.Error(http.StatusInternalServerError, err.Error())
	}
	products := map[uint]string{}
	var prodRows []product
	_ = db.Table(app.productsTable).Find(&prodRows).Error
	for _, p := range prodRows {
		products[p.ID] = p.Name
	}
	userIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		userIDs = append(userIDs, row.UserID)
	}
	users := app.loadUserAccounts(db, userIDs)
	return app.render("stripe/admin_subscribers.html", map[string]any{
		"Title":     "Subscribers",
		"Subtitle":  "Local subscription records synced from Stripe.",
		"BasePath":  basePath,
		"Rows":      rows,
		"Products":  products,
		"Users":     users,
		"Page":      page,
		"PageSize":  pageSize,
		"Total":     total,
		"Notice":    notice,
		"CanManage": canManage,
		"CSRFField": csrf,
	})
}

func (app *stripeApp) editSubscriber(db *gorm.DB, basePath string, id uint, errMsg string, csrf template.HTML) extension.WireResponse {
	var row subscriber
	err := db.Table(app.subscribersTable).Where("subscriber_id = ?", id).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return extension.Error(http.StatusNotFound, "subscriber not found")
	}
	if err != nil {
		return extension.Error(http.StatusInternalServerError, err.Error())
	}
	return app.renderSubscriberForm(db, basePath, row, false, errMsg, csrf)
}

func (app *stripeApp) renderSubscriberForm(db *gorm.DB, basePath string, row subscriber, isNew bool, errMsg string, csrf template.HTML) extension.WireResponse {
	title := "Edit Subscriber"
	action := fmt.Sprintf("%s/subscribers/%d", basePath, row.ID)
	if isNew {
		title = "Add Subscriber"
		action = basePath + "/subscribers/new"
	}
	var products []product
	_ = db.Table(app.productsTable).Order("name asc").Find(&products).Error
	userLabel := ""
	if account, ok := app.loadUserAccount(db, row.UserID); ok {
		userLabel = account.Label()
	}
	return app.render("stripe/admin_subscriber_form.html", map[string]any{
		"Title":     title,
		"Subtitle":  "Manual corrections for Stripe IDs and status.",
		"BasePath":  basePath,
		"ActionURL": action,
		"CancelURL": basePath + "/subscribers",
		"Row":       row,
		"Products":  products,
		"UserLabel": userLabel,
		"IsNew":     isNew,
		"Error":     errMsg,
		"CSRFField": csrf,
		"Statuses":  []string{"active", "trialing", "past_due", "canceled", "incomplete", "incomplete_expired", "unpaid", "paused"},
	})
}

func (app *stripeApp) saveSubscriber(db *gorm.DB, body string, adminURL *url.URL, basePath string, id uint, csrf template.HTML) extension.WireResponse {
	row, errMsg := subscriberFromBody(body, id)
	if errMsg != "" {
		return app.renderSubscriberForm(db, basePath, row, id == 0, errMsg, csrf)
	}
	if id != 0 {
		var existing subscriber
		err := db.Table(app.subscribersTable).Where("subscriber_id = ?", id).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			return extension.Error(http.StatusNotFound, "subscriber not found")
		}
		if err != nil {
			return extension.Error(http.StatusInternalServerError, err.Error())
		}
		row.ID = id
		row.CreatedAt = existing.CreatedAt
	}
	if err := db.Table(app.subscribersTable).Save(&row).Error; err != nil {
		return app.renderSubscriberForm(db, basePath, row, id == 0, err.Error(), csrf)
	}
	return app.listSubscribers(db, adminURL, basePath, "Subscriber saved.", csrf, true)
}

func adminCanView(req extension.WireRequest) bool {
	if !userHasPermissions(req) {
		return true
	}
	return extension.UserCan(req, "stripe.view") || extension.UserCan(req, "stripe.manage")
}

func adminCanManage(req extension.WireRequest) bool {
	if !userHasPermissions(req) {
		return true
	}
	return extension.UserCan(req, "stripe.manage")
}

func userHasPermissions(req extension.WireRequest) bool {
	if req.User == nil {
		return false
	}
	perms, ok := req.User["permissions"].([]any)
	return ok && len(perms) > 0
}

func subscriberFromBody(body string, id uint) (subscriber, string) {
	values, err := formValues(body)
	if err != nil {
		return subscriber{}, "invalid form data"
	}
	userID, _ := strconv.ParseUint(strings.TrimSpace(values.Get("user_id")), 10, 64)
	productID, _ := strconv.ParseUint(strings.TrimSpace(values.Get("product_id")), 10, 64)
	if userID == 0 {
		return subscriber{}, "user_id is required"
	}
	if productID == 0 {
		return subscriber{}, "product_id is required"
	}
	status := strings.TrimSpace(values.Get("status"))
	if status == "" {
		status = "active"
	}
	row := subscriber{
		ID:                   id,
		UserID:               uint(userID),
		ProductID:            uint(productID),
		StripeCustomerID:     strings.TrimSpace(values.Get("stripe_customer_id")),
		StripeSubscriptionID: strings.TrimSpace(values.Get("stripe_subscription_id")),
		Status:               status,
		UpdatedAt:            time.Now().UTC(),
		CreatedAt:            time.Now().UTC(),
	}
	if s := strings.TrimSpace(values.Get("start_at")); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			row.StartAt = &t
		}
	}
	if s := strings.TrimSpace(values.Get("end_at")); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			row.EndAt = &t
		}
	}
	return row, ""
}
