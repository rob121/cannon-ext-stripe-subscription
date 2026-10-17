# Cannon Stripe Subscriptions

Stripe Checkout + Customer Portal + webhooks for Cannon sites.

## Features

- Admin products mapped to Stripe Price IDs (live/test)
- Admin subscriber list with manual correction
- Logged-in Checkout Session subscribe flow
- Billing Portal for payment method updates
- Immediate cancel
- Signed webhook sync (`checkout.session.completed`, `customer.subscription.updated|deleted`)

## Requirements

- Cannon with `/webhooks/` mounted without CSRF (core support)
- Stripe account with recurring Prices

## Quick Start

1. Install and activate `cannon-ext-stripe-subscription`
2. Configure API keys and webhook secret
3. Add a product with a `price_…` ID
4. Create routes (see `help/floridarangers.md`)
5. Point Stripe webhooks at `https://yoursite/webhooks/stripe`

## Capabilities

- `/page` — product, account, return
- `/endpoint` — checkout, portal, cancel, webhook
- `/admin` — products and subscribers
- `/configuration` — Stripe keys
- `/help`, `/templates`, `/install`

## Version

`0.1.0`
