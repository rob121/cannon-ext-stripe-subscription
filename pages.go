package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rob121/cannon/extension"
	"github.com/stripe/stripe-go/v82"
	"gorm.io/gorm"
)

type subscriberView struct {
	subscriber
	ProductName  string
	StartDisplay string
	CanManage    bool
}

func (app *stripeApp) productPage(item string, req extension.WireRequest) extension.WireResponse {
	id, ok := extension.PageDataInt(req, "product_id")
	if !ok || id <= 0 {
		return extension.Error(http.StatusBadRequest, "product_id is required on this route")
	}
	prod, err := app.loadActiveProduct(uint(id))
	if err == gorm.ErrRecordNotFound {
		return extension.Error(http.StatusNotFound, "product not found")
	}
	if err != nil {
		return extension.Error(http.StatusInternalServerError, err.Error())
	}

	checkoutPath := pageString(req, "checkout_path", "/subscribe/checkout")
	accountPath := pageString(req, "account_path", "/subscribe/account")
	authenticated := extension.UserAuthenticated(req)
	hasActive := false
	if authenticated {
		if uid, ok := extension.UserID(req); ok {
			if _, found, _ := app.findActiveForUserProduct(uint(uid), prod.ID); found {
				hasActive = true
			}
		}
	}

	selfPath := reqPath(req)
	return app.render("stripe/product.html", map[string]any{
		"Title":            pageString(req, "title", prod.Name),
		"IntroductionHTML": richTextHTML(extension.PageDataString(req, "introduction")),
		"Product":          prod,
		"DescriptionHTML":  richTextHTML(prod.Description),
		"CheckoutPath":     checkoutPath,
		"AccountPath":      accountPath,
		"SelfPath":         selfPath,
		"Authenticated":    authenticated,
		"HasActive":        hasActive,
		"CSRFField":        csrfField(req),
		"PageItem":         item,
		"LoginURL":         "/auth/login?return=" + selfPath,
	})
}

func (app *stripeApp) accountPage(item string, req extension.WireRequest) extension.WireResponse {
	if !extension.UserAuthenticated(req) {
		return requireLogin(req)
	}
	uid, ok := extension.UserID(req)
	if !ok {
		return requireLogin(req)
	}
	db, err := app.db()
	if err != nil {
		return extension.Error(http.StatusInternalServerError, err.Error())
	}
	var rows []subscriber
	if err := db.Table(app.subscribersTable).Where("user_id = ?", uid).Order("subscriber_id desc").Find(&rows).Error; err != nil {
		return extension.Error(http.StatusInternalServerError, err.Error())
	}
	products := map[uint]string{}
	var prodRows []product
	_ = db.Table(app.productsTable).Find(&prodRows).Error
	for _, p := range prodRows {
		products[p.ID] = p.Name
	}
	views := make([]subscriberView, 0, len(rows))
	for _, row := range rows {
		views = append(views, subscriberView{
			subscriber:   row,
			ProductName:  products[row.ProductID],
			StartDisplay: extension.FormatDatePtr(req, row.StartAt),
			CanManage:    row.Status == "active" || row.Status == "trialing" || row.Status == "past_due",
		})
	}
	notice := ""
	errMsg := ""
	if u, err := parseWireURL(req.URL); err == nil {
		notice = u.Query().Get("notice")
		errMsg = u.Query().Get("error")
	}
	return app.render("stripe/account.html", map[string]any{
		"Title":        pageString(req, "title", "My Subscriptions"),
		"Introduction": extension.PageDataString(req, "introduction"),
		"Rows":         views,
		"SelfPath":     reqPath(req),
		"PortalPath":   pageString(req, "portal_path", "/subscribe/portal"),
		"CancelPath":   pageString(req, "cancel_path", "/subscribe/cancel"),
		"Notice":       notice,
		"Error":        errMsg,
		"CSRFField":    csrfField(req),
		"PageItem":     item,
	})
}

func (app *stripeApp) returnPage(item string, req extension.WireRequest) extension.WireResponse {
	if !extension.UserAuthenticated(req) {
		return requireLogin(req)
	}
	sessionID := ""
	if u, err := parseWireURL(req.URL); err == nil {
		sessionID = strings.TrimSpace(u.Query().Get("session_id"))
	}
	notice := "Thanks — your subscription is being confirmed."
	errMsg := ""
	if sessionID != "" {
		if err := app.reconcileCheckoutSession(sessionID, req); err != nil {
			errMsg = err.Error()
		} else {
			notice = "Your subscription is active. You can manage it from your account."
		}
	}
	return app.render("stripe/return.html", map[string]any{
		"Title":        pageString(req, "title", "Subscription Confirmed"),
		"Introduction": extension.PageDataString(req, "introduction"),
		"Notice":       notice,
		"Error":        errMsg,
		"AccountPath":  pageString(req, "account_path", "/subscribe/account"),
		"PageItem":     item,
	})
}

func (app *stripeApp) reconcileCheckoutSession(sessionID string, req extension.WireRequest) error {
	sc, _, err := app.defaultClient()
	if err != nil {
		// Try both keys via settings mode already handled; also try product test.
		return err
	}
	sess, err := sc.CheckoutSessions.Get(sessionID, &stripe.CheckoutSessionParams{
		Expand: stripe.StringSlice([]string{"subscription"}),
	})
	if err != nil {
		// Retry with the other mode if default fails.
		cfg, _ := app.loadSettings()
		altTest := !strings.EqualFold(cfg.Mode, "test")
		sc2, _, err2 := app.stripeClient(altTest)
		if err2 != nil {
			return err
		}
		sess, err = sc2.CheckoutSessions.Get(sessionID, &stripe.CheckoutSessionParams{
			Expand: stripe.StringSlice([]string{"subscription"}),
		})
		if err != nil {
			return err
		}
		sc = sc2
	}
	_ = sc
	return app.applyCheckoutSession(sess)
}

func (app *stripeApp) applyCheckoutSession(sess *stripe.CheckoutSession) error {
	if sess == nil {
		return nil
	}
	userID := uint(0)
	productID := uint(0)
	if sess.ClientReferenceID != "" {
		if n, err := strconv.ParseUint(sess.ClientReferenceID, 10, 64); err == nil {
			userID = uint(n)
		}
	}
	if sess.Metadata != nil {
		if v := sess.Metadata["user_id"]; v != "" {
			if n, err := strconv.ParseUint(v, 10, 64); err == nil {
				userID = uint(n)
			}
		}
		if v := sess.Metadata["product_id"]; v != "" {
			if n, err := strconv.ParseUint(v, 10, 64); err == nil {
				productID = uint(n)
			}
		}
	}
	subID := ""
	customerID := ""
	status := "active"
	var startAt, endAt *time.Time
	if sess.Customer != nil {
		customerID = sess.Customer.ID
	}
	if sess.Subscription != nil {
		subID = sess.Subscription.ID
		status = mapStripeStatus(sess.Subscription.Status)
		if sess.Subscription.StartDate > 0 {
			t := time.Unix(sess.Subscription.StartDate, 0).UTC()
			startAt = &t
		}
		if sess.Subscription.CancelAt > 0 {
			t := time.Unix(sess.Subscription.CancelAt, 0).UTC()
			endAt = &t
		}
		if productID == 0 && sess.Subscription.Metadata != nil {
			if v := sess.Subscription.Metadata["product_id"]; v != "" {
				if n, err := strconv.ParseUint(v, 10, 64); err == nil {
					productID = uint(n)
				}
			}
		}
		if userID == 0 && sess.Subscription.Metadata != nil {
			if v := sess.Subscription.Metadata["user_id"]; v != "" {
				if n, err := strconv.ParseUint(v, 10, 64); err == nil {
					userID = uint(n)
				}
			}
		}
		if customerID == "" && sess.Subscription.Customer != nil {
			customerID = sess.Subscription.Customer.ID
		}
	}
	if userID == 0 || productID == 0 || subID == "" {
		return nil
	}
	now := time.Now().UTC()
	if startAt == nil {
		startAt = &now
	}
	return app.upsertSubscriber(subscriber{
		UserID:               userID,
		ProductID:            productID,
		StripeCustomerID:     customerID,
		StripeSubscriptionID: subID,
		Status:               status,
		StartAt:              startAt,
		EndAt:                endAt,
		CreatedAt:            now,
		UpdatedAt:            now,
	})
}

func reqPath(req extension.WireRequest) string {
	u, err := parseWireURL(req.URL)
	if err != nil {
		return "/"
	}
	if u.RequestURI() != "" {
		return u.RequestURI()
	}
	return u.Path
}
