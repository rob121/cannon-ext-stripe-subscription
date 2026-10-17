package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/rob121/cannon/extension"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
)

func TestNewServerCapabilities(t *testing.T) {
	app, s := newServer()
	if app == nil || s == nil {
		t.Fatal("expected app and server")
	}
	if app.productsTable != "stripe_subscription_products" {
		t.Fatalf("products table = %q", app.productsTable)
	}
	if app.subscribersTable != "stripe_subscription_subscribers" {
		t.Fatalf("subscribers table = %q", app.subscribersTable)
	}
}

func TestRequireLoginRedirect(t *testing.T) {
	resp := requireLogin(extension.WireRequest{URL: "https://floridarangers.com/subscribe"})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	loc := ""
	if resp.Header != nil {
		loc = resp.Header["Location"][0]
	}
	if !strings.Contains(loc, "/auth/login?return=") {
		t.Fatalf("location = %q", loc)
	}
}

func TestCheckoutRejectsUnauthenticated(t *testing.T) {
	app, _ := newServer()
	resp := app.checkoutEndpoint("checkout", extension.WireRequest{
		Method: http.MethodPost,
		URL:    "https://example.com/subscribe/checkout",
		Body:   "product_id=1",
	})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d body=%s", resp.StatusCode, resp.Body)
	}
}

func TestPortalRejectsUnauthenticated(t *testing.T) {
	app, _ := newServer()
	resp := app.portalEndpoint("portal", extension.WireRequest{
		Method: http.MethodPost,
		URL:    "https://example.com/subscribe/portal",
	})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestCancelRejectsUnauthenticated(t *testing.T) {
	app, _ := newServer()
	resp := app.cancelEndpoint("cancel", extension.WireRequest{
		Method: http.MethodPost,
		URL:    "https://example.com/subscribe/cancel",
		Body:   "subscriber_id=1",
	})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestWebhookMissingSecret(t *testing.T) {
	app, _ := newServer()
	// Without OpenDB this will fail loading settings; still must not panic.
	resp := app.webhookEndpoint("webhook", extension.WireRequest{
		Method: http.MethodPost,
		URL:    "https://example.com/webhooks/stripe",
		Body:   `{}`,
		Header: map[string][]string{"Stripe-Signature": {"t=1,v1=abc"}},
	})
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("expected failure without configured secret, got OK")
	}
}

func TestMapStripeStatus(t *testing.T) {
	if got := mapStripeStatus(stripe.SubscriptionStatusActive); got != "active" {
		t.Fatalf("got %q", got)
	}
	if got := mapStripeStatus(stripe.SubscriptionStatusPastDue); got != "past_due" {
		t.Fatalf("got %q", got)
	}
	if got := mapStripeStatus(stripe.SubscriptionStatusCanceled); got != "canceled" {
		t.Fatalf("got %q", got)
	}
}

func TestProductFromBody(t *testing.T) {
	values := url.Values{}
	values.Set("name", "Ranger Dues")
	values.Set("cost", "25.00")
	values.Set("period", "monthly")
	values.Set("status", "active")
	values.Set("stripe_price_id", "price_live")
	values.Set("stripe_test_price_id", "price_test")
	values.Set("test_mode", "1")
	row, errMsg := productFromBody(values.Encode(), 0)
	if errMsg != "" {
		t.Fatalf("errMsg = %q", errMsg)
	}
	if row.Name != "Ranger Dues" || !row.TestMode || row.priceID() != "price_test" {
		t.Fatalf("row = %+v price=%s", row, row.priceID())
	}
}

func TestValidateSecretKey(t *testing.T) {
	if err := validateSecretKey("sk_test_abc", true); err != nil {
		t.Fatal(err)
	}
	err := validateSecretKey("sk_live_abc", true)
	if err == nil || !strings.Contains(err.Error(), "live key") {
		t.Fatalf("expected live-key-in-test error, got %v", err)
	}
	if err := validateSecretKey("sk_live_abc", false); err != nil {
		t.Fatal(err)
	}
}

func TestWebhookSignatureHelper(t *testing.T) {
	// Ensure stripe webhook package is wired for ConstructEvent usage.
	payload := []byte(`{"id":"evt_test","object":"event","type":"customer.subscription.updated","data":{"object":{}}}`)
	secret := "whsec_test_secret"
	ts := time.Now()
	sig := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{
		Payload:   payload,
		Secret:    secret,
		Timestamp: ts,
	})
	event, err := webhook.ConstructEventWithOptions(sig.Payload, sig.Header, secret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if event.Type != "customer.subscription.updated" {
		t.Fatalf("type = %q", event.Type)
	}
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatal(err)
	}
}

func TestAbsoluteURL(t *testing.T) {
	got := absoluteURL(extension.WireRequest{URL: "https://floridarangers.com/subscribe/checkout"}, "/subscribe/return")
	if got != "https://floridarangers.com/subscribe/return" {
		t.Fatalf("got %q", got)
	}
	// Wire URLs are often path-only; fall back to Host header (local Cannon).
	got = absoluteURL(extension.WireRequest{
		URL:    "/subscribe/checkout",
		Header: map[string][]string{"Host": {"127.0.0.1:8001"}},
	}, "/subscribe/return")
	if got != "http://127.0.0.1:8001/subscribe/return" {
		t.Fatalf("host fallback got %q", got)
	}
	got = absoluteURL(extension.WireRequest{
		URL:    "/subscribe/checkout",
		Header: map[string][]string{"Origin": {"http://127.0.0.1:8001"}},
	}, "/subscribe/return?session_id={CHECKOUT_SESSION_ID}")
	if got != "http://127.0.0.1:8001/subscribe/return?session_id={CHECKOUT_SESSION_ID}" {
		t.Fatalf("origin fallback got %q", got)
	}
}

func TestRichTextHTML(t *testing.T) {
	html := richTextHTML("<p>Hello <strong>world</strong></p>")
	if !strings.Contains(string(html), "<strong>world</strong>") {
		t.Fatalf("html passthrough got %q", html)
	}
	md := richTextHTML("Make a difference.\n\n- Item one\n- Item two")
	if !strings.Contains(string(md), "<li>") {
		t.Fatalf("markdown got %q", md)
	}
}

func TestFriendlyStripeError(t *testing.T) {
	err := fmt.Errorf(`stripe cancel: {"code":"resource_missing","message":"No such subscription: 'sub_x'; a similar object exists in live mode, but a test mode key was used to make this request.","type":"invalid_request_error"}`)
	if !isStripeModeMismatch(err) {
		t.Fatal("expected mode mismatch detection")
	}
	got := friendlyStripeError(err)
	if !strings.Contains(got, "live mode") || strings.Contains(got, "resource_missing") {
		t.Fatalf("friendly message = %q", got)
	}
}

func TestUserAccountLabel(t *testing.T) {
	u := userAccount{ID: 1, GivenName: "Rob", FamilyName: "Alfonso", Username: "admin", Email: "pr@floridarangers.com"}
	if got := u.DisplayName(); got != "Rob Alfonso" {
		t.Fatalf("DisplayName = %q", got)
	}
	if got := u.Label(); got != "Rob Alfonso (admin)" {
		t.Fatalf("Label = %q", got)
	}
	fallback := userAccount{ID: 9, Username: "Rampersad"}
	if got := fallback.DisplayName(); got != "Rampersad" {
		t.Fatalf("username fallback = %q", got)
	}
}
