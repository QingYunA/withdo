# M16: Withdo Pro through Waffo Pancake

Decided 2026-11-04. Pro buys more of the official model (M15); sync stays free for everyone.

| | Free | Pro |
|---|---|---|
| Official model | 20 turns a day | 100 turns a day |
| Price | $0 | $6.99 a month, or $58.72 a year (30 % off) |
| Trial | | 7 days, once per account, card required |

Waffo Pancake is the merchant of record: it charges the buyer, collects and remits sales tax and VAT,
and pays out. PayPal is not connected directly (a direct PayPal seller remits EU VAT itself); ask
Waffo whether its checkout can offer PayPal.

## Waffo (test mode, set up 2026-11-04)

- Merchant `MER_39eInAPdm3tF3TfUVkZyBx`; store **Withdo** `STO_5sXFc60AlpLP2ifCBlbxGm` (the
  account's other stores, Picpocket and pagepod, are not Withdo's).
- Products: **Withdo Pro — Monthly** `PROD_0MGY2QF0efduP3znHPthbD` ($6.99 / month) and **Withdo Pro
  — Yearly** `PROD_52fSnkiAZRUesjicT4ptwh` ($58.72 / year), both with a 7-day $0 trial, in the
  subscription group **Withdo Pro** with self-service plan changes and a shared trial.
- API key **withdo-sync (test)**: the RSA private key is only on the server,
  `/opt/adhd-todo/sync/secrets/waffo-test.pem` (PKCS#8 PEM, mode 600). It cannot be downloaded again.
- Webhook (test, Raw format, all events): `https://api.withdo.si/v1/billing/waffo`.
- Webhook public keys (not secret): `internal/syncserver/billing/waffo-{test,live}.pub.pem`.
- Test card: `4576 7500 0000 0110`.
- Live since 2026-11-04: both products were published to production (they keep their ids), the
  group **Withdo Pro** was made again there (self-service changes, shared trial), the API key
  **withdo-sync (prod)** is on the server as `/opt/adhd-todo/sync/secrets/waffo-prod.pem`, and the
  production webhook points at the same URL. The server runs `WAFFO_MODE=prod` with that key. A
  subscription stored in the other mode is no subscription (no Pro, no trial used).
- Store info (2026-11-06): logo (the app icon), website `https://withdo.si` verified by the DNS TXT
  record `_waffo-challenge.withdo.si` (keep it), and the support email `sorlia7z+withdo@gmail.com`
  (a Gmail plus alias: `sorlia7z@gmail.com` is already the support email of another store of this
  merchant, and Waffo allows one store per email).
- **Production is not open yet**: Waffo takes real payments only after the business verification (KYB,
  Settings > business details, 12 steps). Submitted 2026-11-06 as an individual (merchant name
  `Cyan-Cloud`, tax residence China), in review (1–3 business days, answer by email). For it the site
  gained `pricing.html` and full terms (Waffo's subscription template; refunds: 7-day first-subscriber
  refund under half the included use, duplicates, outages over 72 h, statutory rights, yearly not after
  30 days; PRC law) and privacy (adhd-todo-site#6). Until approved a Waffo checkout cannot take real
  money; PayPal can. The payout account is set up separately (Settings > payout accounts).

## Server: `internal/syncserver/billing`

- `POST /v1/billing/checkout` (session) `{"plan":"monthly"|"yearly"}` → `{"url": checkoutUrl}`.
  Calls `POST https://api.waffo.ai/v1/actions/checkout/create-session` signed with the API key:
  headers `X-Merchant-Id`, `X-Timestamp` (Unix seconds), `X-Signature` =
  Base64(RSA-SHA256 PKCS#1 v1.5 over `METHOD\nPATH\nTIMESTAMP\nBase64(SHA-256(body))`). Body:
  `productId`, `currency: "USD"`, `buyerEmail` (the account's), `metadata: {"account": <id>}`,
  `orderMerchantExternalId`, `withTrial` (false once the account has had a trial), `successUrl:
  "https://withdo.si/pro/"`, `language` from the request.
- `POST /v1/billing/waffo`: the webhook. Verify `X-Waffo-Signature: t=<ms>,v1=<b64>` with
  RSA-SHA256 over `<t>.<raw body>` using the public key of the payload's `mode`; accept `t` up to
  45 minutes old; drop duplicates by `id`; answer 200 within 10 s for every event, handled or not.
  The account is `data.orderMetadata.account`. Store per account: order id, product, status
  (`trialing`, `active`, `past_due`, `canceling`, `canceled`), `currentPeriodEnd`, trial used, mode.
- Entitlement: Pro while status is `trialing`, `active`, `canceling` or `past_due` (one retry day)
  and the period has not ended. Pro raises the account's daily turns to `LLM_DAILY_TURNS_PRO`
  (100); free is `LLM_DAILY_TURNS` (20).
- `GET /v1/billing` (session) → `{"plan":"free"|"pro","status":"","interval":"","period_end":"","trial_used":
  false,"monthly":"$6.99","yearly":"$58.72","manage_url":""}`; `manage_url` is Waffo's customer
  portal for the buyer.
- A change of plan sends the account's devices the sync event, so open apps refresh at once.
- Env: `WAFFO_MERCHANT_ID`, `WAFFO_KEY_FILE`, `WAFFO_STORE_ID`, `WAFFO_PRODUCT_MONTHLY`,
  `WAFFO_PRODUCT_YEARLY`, `WAFFO_MODE` (`test` or `prod`), `LLM_DAILY_TURNS_PRO`. Billing is off
  (routes answer 503 `billing_disabled`, `/v1/config` says `"billing": false`) without the key.

### Built (server)

- `POST /v1/billing/checkout` also takes `"lang"` (the app's language, mapped to the checkout page's);
  answers: 400 `bad_plan`, 409 `already_subscribed` (a Pro account manages its plan in the portal),
  429 `rate_limited` (10 a hour), 502 `billing_upstream`, 503 `billing_disabled`.
- A plan change sends the account's devices `event: billing` on `/v1/events` (old clients ignore it).
- Events are applied by their timestamp (a late retry never undoes a newer state) and only for the
  order the account is on (`activated` and `plan_changed` take a new order over); `trialing` is an
  `activated` with `periodNumber` 0, ended by the first real charge.
- `manage_url` is a fresh portal session link for the account's email, made on every
  `GET /v1/billing` of an account with a subscription (it lives 15 minutes and is never stored);
  when it cannot be made it is Waffo's portal sign-in page.
- `interval` is `monthly`, `yearly` or empty (migration 006, `billing_interval`): the webhooks' `billingPeriod`
  normalized (contains `year` / `annual` is yearly, `month` is monthly), else the product name's Monthly / Yearly.
  Events without it (`payment_succeeded`, a renewal that leaves it out) keep the stored value. Waffo's payload names
  the product by `productName` only, so the configured product ids are not used.

## Clients

Settings shows the plan and the way to Pro; the quota-used-up line offers Pro (design v45). The Pro footer (desktop)
and the Pro card's renewal line (phone) say the billing interval, `年付` / `Yearly` or `月付` / `Monthly`, from
`GET /v1/billing` `interval` (`syncclient.Billing.Interval`, the core's `billing` JSON `interval`); with it empty they
leave it out.
