# Stripe Subscriptions

Sell recurring plans with Stripe Checkout, the Customer Billing Portal, and signed webhooks.

## What It Provides

- Admin UI for products (Stripe Price IDs) and subscriber records
- Extension pages: product, my subscriptions (account), checkout return
- Extension endpoints: checkout, portal, cancel, webhook
- Configuration for live/test API keys and webhook signing secret

## Tables

On install the extension creates:

- `stripe_subscription_settings`
- `stripe_subscription_products`
- `stripe_subscription_subscribers`

## Non-goals (v1)

- No Cannon group grant/revoke on subscribe/cancel
- No Joomla data migration
- Prices are created in Stripe Dashboard and pasted as `price_…` IDs
