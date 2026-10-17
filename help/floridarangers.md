# Florida Rangers Setup

First production target: **floridarangers**.

## 1. Install

1. Copy the `cannon-ext-stripe-subscription` binary into the site extensions directory.
2. Activate and install the extension in Admin → Extensions.
3. Grant roles the `stripe.view` / `stripe.manage` / `stripe.setup` permissions as needed.

## 2. Stripe Settings

Admin → Configuration → Extensions → Stripe Subscriptions:

| Field | Notes |
|-------|--------|
| Default Mode | `test` while integrating; switch to `live` for production |
| Secret / Publishable keys | From Stripe Dashboard → Developers → API keys |
| Webhook Signing Secret | `whsec_…` from the webhook endpoint below |
| Success Path | `/subscribe/return` |
| Cancel Path | product page path, e.g. `/subscribe` |

## 3. Products

Admin → Subscriptions → Products → Add Product:

- Paste the Stripe **Price** ID (`price_…`), not the Product ID
- Enable **test mode** on the product while validating with test keys

## 4. Routes

Create these routes (Access: Public unless noted):

| Path | Type | Extension page/endpoint | Notes |
|------|------|-------------------------|--------|
| `/subscribe` | Extension Page | `product` | Set `product_id`, `checkout_path=/subscribe/checkout`, `account_path=/subscribe/account` |
| `/subscribe/account` | Extension Page | `account` | Registered users; `portal_path=/subscribe/portal`, `cancel_path=/subscribe/cancel` |
| `/subscribe/return` | Extension Page | `return` | `account_path=/subscribe/account` |
| `/subscribe/checkout` | Extension Endpoint | `checkout` | POST from product page |
| `/subscribe/portal` | Extension Endpoint | `portal` | POST from account page |
| `/subscribe/cancel` | Extension Endpoint | `cancel` | POST; cancels immediately |
| `/webhooks/stripe` | Extension Endpoint | `webhook` | **Must** be under `/webhooks/` (CSRF-exempt in Cannon) |

## 5. Stripe Webhook

In Stripe Dashboard → Developers → Webhooks → Add endpoint:

- URL: `https://floridarangers.com/webhooks/stripe`
- Events:
  - `checkout.session.completed`
  - `customer.subscription.updated`
  - `customer.subscription.deleted`
- Copy the signing secret into extension settings

## 6. Menus

- Link “Subscribe” (or plan name) → `/subscribe`
- Link “My Subscriptions” → `/subscribe/account` (registered menu)

## Cancel Behavior

Cancel calls Stripe to end the subscription **immediately** and marks the local row `canceled`. Webhooks keep status in sync if the customer cancels from the Billing Portal.
