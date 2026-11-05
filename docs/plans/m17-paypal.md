# M17: PayPal as a second way to pay for Withdo Pro

Decided 2026-11-04. Waffo (M16) stays the main checkout (cards, Apple Pay, Google Pay, as merchant
of record). PayPal is a separate channel the user asked for: a PayPal button under the plan sheet's
main button. Both write the same per-account subscription, so Pro, the quota and the apps do not
care which one was used.

**Tax.** PayPal is not a merchant of record: for PayPal sales the seller (the owner) collects and
remits VAT and sales tax where they are due (EU VAT on digital services from the first sale).

## PayPal (sandbox, set up 2026-11-04)

- Developer app **Withdo** (Merchant) on the owner's PayPal account; its client id and secret are
  only on the server: `/opt/adhd-todo/sync/secrets/paypal-sandbox.json` (`{"client_id","secret","env"}`,
  mode 600, owner 65532).
- Created through the API: product, four plans and the webhook:

```json
{"product": "PROD-1NV88459X9209392Y", "monthly_trial": "P-3MA95878A4870393LNLVOKMA", "monthly": "P-6FL403268U768440TNLVOKMQ", "yearly_trial": "P-4UX93808L6969313JNLVOKMY", "yearly": "P-6MY38608R0206392FNLVOKNA", "webhook": "82659267CT025681J"}
```

  Plans: monthly $6.99 and yearly $58.72, each with a 7-day $0 trial (`*_trial`) and without one
  (for accounts that already had a trial, on either channel).
- Webhook `https://api.withdo.si/v1/billing/paypal` for BILLING.SUBSCRIPTION.ACTIVATED, UPDATED,
  CANCELLED, SUSPENDED, EXPIRED, RE-ACTIVATED, PAYMENT.FAILED, PAYMENT.SALE.COMPLETED, REFUNDED.

## PayPal (live since 2026-11-05)

- Live developer app **Withdo** on the business account (primary email `SORLIA7Z@GMAIL.COM`); its client
  id and secret are only on the server: `/opt/adhd-todo/sync/secrets/paypal-live.json` (mode 600, owner
  65532). `scripts/paypal-setup.py` made the same product, plans and webhook against `https://api-m.paypal.com`:

```json
{"product": "PROD-71454283MV4259044", "monthly_trial": "P-2VM317722C105582CNLWGE3Q", "monthly": "P-6XJ27566U3567571HNLWGE3Y", "yearly_trial": "P-1E6709973N259634XNLWGE4A", "yearly": "P-5FV90483Y88326410NLWGE4I", "webhook": "4YD71328AU0476456"}
```

- The server's `sync.env` points at them (`PAYPAL_ENV=live`); the sandbox values are in
  `sync.env.bak-paypal-live-20261105` next to it, and `paypal-sandbox.json` stays in `secrets/` for testing.
- Sandbox rows (mode `test`) count as no subscription on the live server (the other-mode rule).

## Server

- `POST /v1/billing/checkout` takes `"provider": "waffo"|"paypal"` (default waffo). For PayPal it
  creates `POST /v1/billing/subscriptions` with the plan (trial or not), `custom_id` = the account,
  `subscriber.email_address`, `application_context` (brand `Withdo`, `SUBSCRIBE_NOW`,
  `NO_SHIPPING`, return and cancel URLs `https://withdo.si/pro/`) and answers the `approve` link.
- `POST /v1/billing/paypal`: verify with `POST /v1/notifications/verify-webhook-signature` (the
  five PAYPAL-* headers, the webhook id, the raw event); then **read the subscription from PayPal**
  (`GET /v1/billing/subscriptions/{id}`) and apply its state, never the payload's: ACTIVE in the
  trial cycle → trialing, ACTIVE → active, SUSPENDED → past_due, CANCELLED → canceling until
  `billing_info.next_billing_time` (or the last payment's period end), then canceled; EXPIRED →
  canceled. The account is `custom_id`. Dedupe by the event id. 200 for every verified event.
- One row per account as in M16, with `provider` (`waffo`, `paypal`); `GET /v1/billing` says it.
  `manage_url` for PayPal is the payer's automatic-payments page
  (`https://www.paypal.com/myaccount/autopay/`, sandbox `https://www.sandbox.paypal.com/myaccount/autopay/`).
- Env: `PAYPAL_CREDENTIALS_FILE`, `PAYPAL_ENV` (`sandbox`|`live`), `PAYPAL_WEBHOOK_ID`,
  `PAYPAL_PLAN_MONTHLY`, `PAYPAL_PLAN_MONTHLY_TRIAL`, `PAYPAL_PLAN_YEARLY`, `PAYPAL_PLAN_YEARLY_TRIAL`.
  Without the credentials file PayPal is off and `/v1/config` says `"paypal": false`; the apps hide the button.

## Clients (design v46)

The plan sheet shows, under the main button, `或` and a PayPal-branded button `用 PayPal 订阅`
(`#FFC439`, the PayPal wordmark), which checks out the chosen plan with `provider: "paypal"`; the
waiting state says `在 PayPal 里完成订阅` / `重新打开 PayPal`. A PayPal subscription's Pro line adds
`通过 PayPal` / `via PayPal`.

## Server notes (as built)

- One row per account; `provider` (`waffo` default, `paypal`) is migration 007, and a PayPal row keeps
  the subscription id (`I-…`) in `order_id`. A PayPal `ACTIVE` subscription takes the row over like
  Waffo's `activated`; the later states of any other subscription are ignored. `trial_used` is the
  account's, whatever the channel; a plan with a TRIAL cycle sets it.
- Mode: the sandbox is stored as `test`, live as `prod`, and the other-mode rule applies per channel.
  With PayPal off, a PayPal row gives no Pro (the server cannot say which environment it belongs to).
- The event is only a pointer. The dedupe key is `paypal:<event id>`; a read of the subscription
  older than the row (by its `update_time`, `status_update_time`, last payment) is dropped.
- Trial: ACTIVE with a TRIAL cycle that has runs left, or no REGULAR cycle charged yet, and no failed
  payment. Period end is `next_billing_time`; after a cancellation, when PayPal drops it, the period
  the row already had, else the last real payment plus one interval. A trialing or active PayPal
  subscription stays Pro 24 hours past its period end (PayPal charges a renewal some time after it is
  due and the sale webhook follows); `canceling` does not. `GET /v1/billing` reports a lapsed
  `canceling` as `canceled`, since PayPal sends no event when the period ends.

## Polling fallback

In the first sandbox test (2026-11-04) PayPal created `BILLING.SUBSCRIPTION.ACTIVATED` and the
subscription was ACTIVE, but no webhook ever reached the server, so the account never became Pro.
Webhooks stay the main path; `GET /v1/billing` and the official-model quota now also read PayPal
themselves, only when PayPal is configured and never for Waffo rows:

- `POST /v1/billing/checkout` with `provider: "paypal"` remembers the new subscription id (`I-…`) in
  `billing_paypal_pending` (migration 008, one per account; a new checkout replaces it).
- `GET /v1/billing` (the apps call it every few seconds while waiting for PayPal) reads that
  subscription with `GET /v1/billing/subscriptions/{id}` at most once per 10 seconds per account and
  applies it through the same code as a webhook (`applyPayPalSubscription`: `custom_id` must be the
  account, else it is ignored). The dedupe id is `paypal-poll:<id>:<newest PayPal timestamp>`.
  The pending entry is deleted once the row holds the subscription, when PayPal does not know it or
  it names another account, and after 3 days; an unapproved one (`APPROVAL_PENDING`) stays.
- If the row is an existing PayPal subscription that is not `canceled`, `GET /v1/billing` refreshes
  it from PayPal at most once per 10 minutes per account, which sees a cancellation whose webhook was
  lost. The throttles are in memory, per server process.
- `IsProSubject` does the same refresh (same 10-minute throttle) for a PayPal row that is
  trialing, active or past_due but no longer Pro, i.e. a renewal whose sale webhook was lost.
- PayPal being down or slow (each poll is cut off after 8 seconds) never fails the request: it logs
  `billing: account=N paypal poll failed: …` and answers from the stored row.
