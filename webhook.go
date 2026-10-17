package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rob121/cannon/extension"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
	"gorm.io/gorm"
)

func (app *stripeApp) webhookEndpoint(item string, req extension.WireRequest) extension.WireResponse {
	if !strings.EqualFold(req.Method, http.MethodPost) {
		return extension.Error(http.StatusMethodNotAllowed, "POST required")
	}
	cfg, err := app.loadSettings()
	if err != nil {
		return extension.Error(http.StatusInternalServerError, err.Error())
	}
	secret := strings.TrimSpace(cfg.WebhookSecret)
	if secret == "" {
		return extension.Error(http.StatusInternalServerError, "webhook secret is not configured")
	}
	sig := headerGet(req, "Stripe-Signature")
	if sig == "" {
		return extension.Error(http.StatusBadRequest, "missing Stripe-Signature")
	}
	event, err := webhook.ConstructEventWithOptions([]byte(req.Body), sig, secret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		return extension.Error(http.StatusBadRequest, "invalid signature: "+err.Error())
	}

	switch event.Type {
	case "checkout.session.completed":
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			return extension.Error(http.StatusBadRequest, err.Error())
		}
		// Expand subscription if only an ID is present by fetching when needed.
		if sess.Subscription != nil && sess.Subscription.Status == "" && sess.Subscription.ID != "" {
			if sc, _, err := app.defaultClient(); err == nil {
				if full, err := sc.Subscriptions.Get(sess.Subscription.ID, nil); err == nil {
					sess.Subscription = full
				}
			}
		}
		if err := app.applyCheckoutSession(&sess); err != nil {
			return extension.Error(http.StatusInternalServerError, err.Error())
		}
	case "customer.subscription.updated", "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return extension.Error(http.StatusBadRequest, err.Error())
		}
		if err := app.applySubscription(&sub); err != nil {
			return extension.Error(http.StatusInternalServerError, err.Error())
		}
	default:
		// Acknowledge ignored events.
	}
	return extension.OK()
}

func (app *stripeApp) applySubscription(sub *stripe.Subscription) error {
	if sub == nil || sub.ID == "" {
		return nil
	}
	userID := uint(0)
	productID := uint(0)
	if sub.Metadata != nil {
		if v := sub.Metadata["user_id"]; v != "" {
			if n, err := strconv.ParseUint(v, 10, 64); err == nil {
				userID = uint(n)
			}
		}
		if v := sub.Metadata["product_id"]; v != "" {
			if n, err := strconv.ParseUint(v, 10, 64); err == nil {
				productID = uint(n)
			}
		}
	}
	customerID := ""
	if sub.Customer != nil {
		customerID = sub.Customer.ID
	}
	status := mapStripeStatus(sub.Status)
	if sub.Status == stripe.SubscriptionStatusCanceled || strings.EqualFold(string(sub.Status), "canceled") {
		status = "canceled"
	}
	var startAt, endAt *time.Time
	if sub.StartDate > 0 {
		t := time.Unix(sub.StartDate, 0).UTC()
		startAt = &t
	}
	if sub.EndedAt > 0 {
		t := time.Unix(sub.EndedAt, 0).UTC()
		endAt = &t
	} else if sub.CancelAt > 0 {
		t := time.Unix(sub.CancelAt, 0).UTC()
		endAt = &t
	} else if status == "canceled" {
		now := time.Now().UTC()
		endAt = &now
	}

	existing, err := app.findSubscriberByStripeID(sub.ID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if err == nil {
		if userID == 0 {
			userID = existing.UserID
		}
		if productID == 0 {
			productID = existing.ProductID
		}
		if customerID == "" {
			customerID = existing.StripeCustomerID
		}
		if startAt == nil {
			startAt = existing.StartAt
		}
	}
	if userID == 0 || productID == 0 {
		// Cannot create a row without ownership metadata.
		if err == gorm.ErrRecordNotFound {
			return nil
		}
	}
	now := time.Now().UTC()
	row := subscriber{
		UserID:               userID,
		ProductID:            productID,
		StripeCustomerID:     customerID,
		StripeSubscriptionID: sub.ID,
		Status:               status,
		StartAt:              startAt,
		EndAt:                endAt,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if err == nil {
		row.ID = existing.ID
		row.CreatedAt = existing.CreatedAt
	}
	return app.upsertSubscriber(row)
}
