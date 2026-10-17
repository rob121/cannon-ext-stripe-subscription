package main

import (
	"fmt"
	"strings"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/client"
)

// clientForStoredProduct picks API keys from the product's own test_mode flag.
// Unlike clientForProduct, it does not force the site Default Mode onto imported
// live (or test) subscriptions — that mismatch is what produces Stripe
// "similar object exists in live/test mode" errors on cancel/portal.
func (app *stripeApp) clientForStoredProduct(p product) (*client.API, stripeSettings, error) {
	return app.stripeClient(p.TestMode)
}

func isStripeModeMismatch(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "similar object exists in live mode") ||
		strings.Contains(msg, "similar object exists in test mode") ||
		(strings.Contains(msg, "resource_missing") && strings.Contains(msg, "mode"))
}

func friendlyStripeError(err error) string {
	if err == nil {
		return "Something went wrong talking to Stripe."
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "similar object exists in live mode"):
		return "This subscription lives in Stripe live mode, but the site is using a test API key. Switch Default Mode to live (or set the product to live) and try again."
	case strings.Contains(lower, "similar object exists in test mode"):
		return "This subscription lives in Stripe test mode, but the site is using a live API key. Switch Default Mode to test (or enable test mode on the product) and try again."
	case strings.Contains(lower, "no such subscription"):
		return "Stripe could not find this subscription. It may already be canceled, or the wrong Stripe mode (test vs live) is selected."
	case strings.Contains(lower, "no such customer"):
		return "Stripe could not find this customer. The wrong Stripe mode (test vs live) may be selected."
	default:
		// Avoid dumping raw Stripe JSON to visitors.
		if i := strings.Index(msg, "{"); i >= 0 {
			msg = strings.TrimSpace(msg[:i])
		}
		msg = strings.TrimSpace(msg)
		if msg == "" {
			return "Stripe could not complete that request. Please try again or contact support."
		}
		return msg
	}
}

func (app *stripeApp) usesTestKeys(p product) bool {
	if p.TestMode {
		return true
	}
	cfg, err := app.loadSettings()
	if err != nil {
		return false
	}
	return strings.EqualFold(cfg.Mode, "test")
}

func (app *stripeApp) secretKeyForProduct(p product) (string, stripeSettings, error) {
	cfg, err := app.loadSettings()
	if err != nil {
		return "", cfg, err
	}
	test := p.TestMode || strings.EqualFold(cfg.Mode, "test")
	if test {
		key := strings.TrimSpace(cfg.SecretKeyTest)
		if key == "" {
			return "", cfg, fmt.Errorf("stripe test secret key is not configured (expected sk_test_…)")
		}
		if err := validateSecretKey(key, true); err != nil {
			return "", cfg, err
		}
		return key, cfg, nil
	}
	key := strings.TrimSpace(cfg.SecretKeyLive)
	if key == "" {
		return "", cfg, fmt.Errorf("stripe live secret key is not configured (expected sk_live_…)")
	}
	if err := validateSecretKey(key, false); err != nil {
		return "", cfg, err
	}
	return key, cfg, nil
}

func validateSecretKey(key string, test bool) error {
	key = strings.TrimSpace(key)
	if test {
		if strings.HasPrefix(key, "sk_test_") {
			return nil
		}
		if strings.HasPrefix(key, "sk_live_") {
			return fmt.Errorf("test mode is selected, but Secret Key (Test) is a live key (sk_live_…). Paste your sk_test_… key from Stripe Dashboard → Developers → API keys (Test mode)")
		}
		return fmt.Errorf("Secret Key (Test) must start with sk_test_…")
	}
	if strings.HasPrefix(key, "sk_live_") {
		return nil
	}
	if strings.HasPrefix(key, "sk_test_") {
		return fmt.Errorf("live mode is selected, but Secret Key (Live) is a test key (sk_test_…). Paste your sk_live_… key, or switch Default Mode to test")
	}
	return fmt.Errorf("Secret Key (Live) must start with sk_live_…")
}

func (app *stripeApp) clientForProduct(p product) (*client.API, stripeSettings, error) {
	key, cfg, err := app.secretKeyForProduct(p)
	if err != nil {
		return nil, cfg, err
	}
	return client.New(key, nil), cfg, nil
}

func (app *stripeApp) defaultClient() (*client.API, stripeSettings, error) {
	cfg, err := app.loadSettings()
	if err != nil {
		return nil, cfg, err
	}
	p := product{TestMode: strings.EqualFold(cfg.Mode, "test")}
	return app.clientForProduct(p)
}

func (app *stripeApp) stripeClient(testMode bool) (*client.API, stripeSettings, error) {
	return app.clientForProduct(product{TestMode: testMode})
}

// resolvePriceID returns the Stripe Price ID for the current mode.
func (app *stripeApp) resolvePriceID(p product) (string, error) {
	if app.usesTestKeys(p) {
		if id := strings.TrimSpace(p.StripeTestPriceID); id != "" {
			return id, nil
		}
		return "", fmt.Errorf("product is in test mode but Stripe Price ID (Test) is empty")
	}
	if id := strings.TrimSpace(p.StripePriceID); id != "" {
		return id, nil
	}
	return "", fmt.Errorf("product has no Stripe Price ID (Live)")
}

func mapStripeStatus(status stripe.SubscriptionStatus) string {
	switch status {
	case stripe.SubscriptionStatusActive:
		return "active"
	case stripe.SubscriptionStatusTrialing:
		return "trialing"
	case stripe.SubscriptionStatusPastDue:
		return "past_due"
	case stripe.SubscriptionStatusCanceled:
		return "canceled"
	case stripe.SubscriptionStatusIncomplete:
		return "incomplete"
	case stripe.SubscriptionStatusIncompleteExpired:
		return "incomplete_expired"
	case stripe.SubscriptionStatusUnpaid:
		return "unpaid"
	case stripe.SubscriptionStatusPaused:
		return "paused"
	default:
		if status == "" {
			return "unknown"
		}
		return string(status)
	}
}
