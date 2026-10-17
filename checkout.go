package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rob121/cannon/extension"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/client"
	"gorm.io/gorm"
)

func (app *stripeApp) checkoutEndpoint(item string, req extension.WireRequest) extension.WireResponse {
	if !strings.EqualFold(req.Method, http.MethodPost) {
		return extension.Error(http.StatusMethodNotAllowed, "POST required")
	}
	if !extension.UserAuthenticated(req) {
		return requireLogin(req)
	}
	uid, ok := extension.UserID(req)
	if !ok {
		return requireLogin(req)
	}
	values, err := formValues(req.Body)
	if err != nil {
		return extension.Error(http.StatusBadRequest, "invalid form")
	}
	productID, _ := strconv.ParseUint(strings.TrimSpace(values.Get("product_id")), 10, 64)
	if productID == 0 {
		return extension.Error(http.StatusBadRequest, "product_id is required")
	}
	prod, err := app.loadActiveProduct(uint(productID))
	if err == gorm.ErrRecordNotFound {
		return extension.Error(http.StatusNotFound, "product not found")
	}
	if err != nil {
		return extension.Error(http.StatusInternalServerError, err.Error())
	}
	priceID, err := app.resolvePriceID(prod)
	if err != nil {
		return extension.Error(http.StatusBadRequest, err.Error())
	}
	if _, found, err := app.findActiveForUserProduct(uint(uid), prod.ID); err != nil {
		return extension.Error(http.StatusInternalServerError, err.Error())
	} else if found {
		account := strings.TrimSpace(values.Get("account_path"))
		if account == "" {
			account = "/subscribe/account"
		}
		return extension.Redirect(http.StatusSeeOther, account)
	}

	sc, cfg, err := app.clientForProduct(prod)
	if err != nil {
		return extension.Error(http.StatusInternalServerError, err.Error())
	}

	successPath := strings.TrimSpace(values.Get("success_path"))
	if successPath == "" {
		successPath = strings.TrimSpace(cfg.SuccessPath)
	}
	if successPath == "" {
		successPath = "/subscribe/return"
	}
	cancelPath := strings.TrimSpace(values.Get("cancel_path"))
	if cancelPath == "" {
		cancelPath = strings.TrimSpace(cfg.CancelPath)
	}
	if cancelPath == "" {
		cancelPath = strings.TrimSpace(values.Get("return_path"))
	}
	if cancelPath == "" {
		cancelPath = "/"
	}
	if !strings.Contains(successPath, "session_id=") {
		if strings.Contains(successPath, "?") {
			successPath += "&session_id={CHECKOUT_SESSION_ID}"
		} else {
			successPath += "?session_id={CHECKOUT_SESSION_ID}"
		}
	}

	email := extension.UserString(req, "email")
	name := extension.UserDisplayName(req)
	meta := map[string]string{
		"user_id":    strconv.FormatUint(uid, 10),
		"product_id": strconv.FormatUint(uint64(prod.ID), 10),
		"site_id":    req.SiteID,
	}

	successURL := absoluteURL(req, successPath)
	cancelURL := absoluteURL(req, cancelPath)
	if !strings.HasPrefix(successURL, "http://") && !strings.HasPrefix(successURL, "https://") {
		return extension.Error(http.StatusInternalServerError, "cannot build absolute success_url for Stripe (missing request host)")
	}
	if !strings.HasPrefix(cancelURL, "http://") && !strings.HasPrefix(cancelURL, "https://") {
		return extension.Error(http.StatusInternalServerError, "cannot build absolute cancel_url for Stripe (missing request host)")
	}

	params := &stripe.CheckoutSessionParams{
		Mode:              stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL:        stripe.String(successURL),
		CancelURL:         stripe.String(cancelURL),
		ClientReferenceID: stripe.String(strconv.FormatUint(uid, 10)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		Metadata: meta,
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: meta,
		},
	}
	if email != "" {
		params.CustomerEmail = stripe.String(email)
	}
	_ = name

	// Reuse existing Stripe customer when we already have one for this user.
	if custID := app.customerIDForUser(uint(uid)); custID != "" {
		params.Customer = stripe.String(custID)
		params.CustomerEmail = nil
	}

	sess, err := sc.CheckoutSessions.New(params)
	if err != nil {
		return extension.Error(http.StatusBadGateway, fmt.Sprintf("stripe checkout: %v", err))
	}
	if sess.URL == "" {
		return extension.Error(http.StatusBadGateway, "stripe checkout returned no URL")
	}
	return extension.Redirect(http.StatusSeeOther, sess.URL)
}

func (app *stripeApp) portalEndpoint(item string, req extension.WireRequest) extension.WireResponse {
	if !strings.EqualFold(req.Method, http.MethodPost) {
		return redirectWithQuery("/subscribe/account", "error", "Use Manage payment from your subscriptions page.")
	}
	if !extension.UserAuthenticated(req) {
		return requireLogin(req)
	}
	uid, ok := extension.UserID(req)
	if !ok {
		return requireLogin(req)
	}
	values, _ := formValues(req.Body)
	returnPath := strings.TrimSpace(values.Get("return_path"))
	if returnPath == "" {
		returnPath = "/subscribe/account"
	}
	customerID := strings.TrimSpace(values.Get("customer_id"))
	if customerID == "" {
		customerID = app.customerIDForUser(uint(uid))
	}
	if customerID == "" {
		return redirectWithQuery(returnPath, "error", "No Stripe customer is on file for your account.")
	}

	sc, err := app.clientForCustomer(uint(uid), customerID)
	if err != nil {
		return redirectWithQuery(returnPath, "error", friendlyStripeError(err))
	}
	sess, err := sc.BillingPortalSessions.New(&stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(absoluteURL(req, returnPath)),
	})
	if err != nil && isStripeModeMismatch(err) {
		if sc2, err2 := app.alternateClientForCustomer(uint(uid), customerID); err2 == nil {
			sess, err = sc2.BillingPortalSessions.New(&stripe.BillingPortalSessionParams{
				Customer:  stripe.String(customerID),
				ReturnURL: stripe.String(absoluteURL(req, returnPath)),
			})
		}
	}
	if err != nil {
		return redirectWithQuery(returnPath, "error", friendlyStripeError(err))
	}
	return extension.Redirect(http.StatusSeeOther, sess.URL)
}

func (app *stripeApp) cancelEndpoint(item string, req extension.WireRequest) extension.WireResponse {
	if !strings.EqualFold(req.Method, http.MethodPost) {
		return redirectWithQuery("/subscribe/account", "error", "Use Cancel from your subscriptions page.")
	}
	if !extension.UserAuthenticated(req) {
		return requireLogin(req)
	}
	uid, ok := extension.UserID(req)
	if !ok {
		return requireLogin(req)
	}
	values, err := formValues(req.Body)
	if err != nil {
		return redirectWithQuery("/subscribe/account", "error", "Invalid cancel request.")
	}
	subRowID, _ := strconv.ParseUint(strings.TrimSpace(values.Get("subscriber_id")), 10, 64)
	stripeSubID := strings.TrimSpace(values.Get("subscription_id"))
	returnPath := strings.TrimSpace(values.Get("return_path"))
	if returnPath == "" {
		returnPath = "/subscribe/account"
	}

	db, err := app.db()
	if err != nil {
		return redirectWithQuery(returnPath, "error", "Could not load subscription data.")
	}
	var row subscriber
	q := db.Table(app.subscribersTable).Where("user_id = ?", uid)
	if subRowID > 0 {
		q = q.Where("subscriber_id = ?", subRowID)
	} else if stripeSubID != "" {
		q = q.Where("stripe_subscription_id = ?", stripeSubID)
	} else {
		return redirectWithQuery(returnPath, "error", "Missing subscription to cancel.")
	}
	if err := q.First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return redirectWithQuery(returnPath, "error", "Subscription not found.")
		}
		return redirectWithQuery(returnPath, "error", "Could not load that subscription.")
	}
	if row.StripeSubscriptionID == "" {
		return redirectWithQuery(returnPath, "error", "This subscription is missing a Stripe ID.")
	}

	prod, _ := app.loadProduct(row.ProductID)
	if err := app.cancelStripeSubscription(prod, row.StripeSubscriptionID); err != nil {
		return redirectWithQuery(returnPath, "error", friendlyStripeError(err))
	}
	now := time.Now().UTC()
	row.Status = "canceled"
	row.EndAt = &now
	row.UpdatedAt = now
	if err := db.Table(app.subscribersTable).Save(&row).Error; err != nil {
		return redirectWithQuery(returnPath, "error", "Canceled in Stripe, but saving the local record failed. Refresh shortly.")
	}
	return redirectWithQuery(returnPath, "notice", "Subscription canceled.")
}

func (app *stripeApp) cancelStripeSubscription(prod product, stripeSubID string) error {
	sc, _, err := app.clientForStoredProduct(prod)
	if err != nil {
		return err
	}
	_, err = sc.Subscriptions.Cancel(stripeSubID, nil)
	if err == nil || !isStripeModeMismatch(err) {
		return err
	}
	sc2, _, err2 := app.stripeClient(!prod.TestMode)
	if err2 != nil {
		return err
	}
	_, err2 = sc2.Subscriptions.Cancel(stripeSubID, nil)
	if err2 != nil {
		return err
	}
	return nil
}

func redirectWithQuery(path, key, value string) extension.WireResponse {
	path = strings.TrimSpace(path)
	if path == "" {
		path = "/subscribe/account"
	}
	u, err := url.Parse(path)
	if err != nil {
		u = &url.URL{Path: "/subscribe/account"}
	}
	q := u.Query()
	if key != "" && value != "" {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()
	return extension.Redirect(http.StatusSeeOther, u.String())
}

func (app *stripeApp) customerIDForUser(userID uint) string {
	db, err := app.db()
	if err != nil {
		return ""
	}
	var row subscriber
	err = db.Table(app.subscribersTable).
		Where("user_id = ? AND stripe_customer_id <> ''", userID).
		Order("subscriber_id desc").
		First(&row).Error
	if err != nil {
		return ""
	}
	return row.StripeCustomerID
}

func (app *stripeApp) clientForCustomer(userID uint, customerID string) (*client.API, error) {
	db, err := app.db()
	if err != nil {
		return nil, err
	}
	var row subscriber
	_ = db.Table(app.subscribersTable).
		Where("user_id = ? AND stripe_customer_id = ?", userID, customerID).
		Order("subscriber_id desc").
		First(&row).Error
	if row.ProductID > 0 {
		prod, err := app.loadProduct(row.ProductID)
		if err == nil {
			sc, _, err := app.clientForStoredProduct(prod)
			return sc, err
		}
	}
	sc, _, err := app.defaultClient()
	return sc, err
}

func (app *stripeApp) alternateClientForCustomer(userID uint, customerID string) (*client.API, error) {
	db, err := app.db()
	if err != nil {
		return nil, err
	}
	var row subscriber
	_ = db.Table(app.subscribersTable).
		Where("user_id = ? AND stripe_customer_id = ?", userID, customerID).
		Order("subscriber_id desc").
		First(&row).Error
	testMode := true
	if row.ProductID > 0 {
		if prod, err := app.loadProduct(row.ProductID); err == nil {
			testMode = !prod.TestMode
		}
	} else if cfg, err := app.loadSettings(); err == nil {
		testMode = !strings.EqualFold(cfg.Mode, "test")
	}
	sc, _, err := app.stripeClient(testMode)
	return sc, err
}
