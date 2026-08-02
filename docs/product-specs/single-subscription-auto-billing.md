# Single Subscription And Automatic Billing

## Decision

Sub2API gives each user at most one active subscription across the site. Users
may choose another plan only after the current subscription expires or is
revoked. An administrator may switch the active subscription to another active
subscription group on the same platform without payment, refund, or price
adjustment.

Existing API keys normally keep their stored group. An administrator plan
switch is the exception: non-deleted API keys bound to the source subscription
group move to the target subscription group in the same transaction. Keys in
standard groups and keys bound to any other group remain unchanged. For every
API request, the gateway first attempts to use the user's active subscription.
It uses the subscription group only when the platform, model, capability,
group status, and daily,
weekly, and effective monthly quotas all permit the request. Otherwise it uses
the API key's original group and charges the user's balance. API key status,
expiry, IP rules, key quota, and rate limits always apply.

The final billing source is confirmed atomically during settlement. If the
subscription is no longer valid or cannot cover the whole request cost, the
whole request is charged to balance. A request must never be split between a
subscription and balance.

Settlement happens after the upstream request has incurred cost. If a request
was admitted against a subscription but must fall back at settlement time and
the user's balance is then insufficient, the full balance cost is still
recorded and the balance may become negative. This preserves the cost already
incurred and prevents an unbilled request; subsequent admission checks continue
to reject insufficient balance.

Usage logs record the effective group, subscription ID, and final billing type.
Routing, multipliers, account selection, usage caches, and balance caches use
that same final billing source. API key quota and amount-based rate limiting are
also charged using the final settled cost, including the balance fallback cost.

## Single Subscription Invariant

All subscription grants use one transaction boundary and lock the user before
checking for an active subscription. This includes purchases, administrator
assignments, redemption codes, default grants, renewals, and restoration.

- A different active subscription blocks a new grant or restoration.
- An administrator may adjust the dates of the same subscription without
  changing its plan.
- An administrator switch replaces the only active subscription atomically; it
  never creates a second active subscription.
- A subscription is active only when its status is active, `starts_at <= now`,
  and `expires_at > now`; a future subscription does not participate in
  routing, billing, renewal, active-list responses, or active counts.
- Each user may have at most one unfinished subscription payment order.
- Each subscription group may have at most one plan currently for sale. Past
  plans can remain in the database after `for_sale` becomes false.
- A pre-deployment audit must stop on overlapping subscriptions or unfinished
  subscription orders, or multiple sale plans for one group. It must not delete
  or merge production data.

`GET /subscriptions/active` retains its array response for compatibility, but
the server guarantees zero or one item.

## Renewal

A user with no active subscription may purchase any plan currently for sale.
While a subscription is active, other plans are unavailable. The current plan
can be renewed only after monthly usage reaches the base monthly quota plus the
current monthly bonus, and only while the plan remains for sale.

Renewal uses the current sale price. At successful payment time:

- the plan remains unchanged;
- `starts_at` becomes the payment completion time;
- `expires_at` becomes 30 days after the payment completion time;
- unused subscription time is discarded;
- daily, weekly, and monthly usage becomes zero;
- all three rolling windows restart at the payment completion time; and
- the monthly bonus becomes zero.

Subscription orders continue to use `order_type=subscription`. The server
derives and snapshots `subscription_action=purchase|renewal` and the target
`subscription_id`; clients cannot choose these values. Fulfilment and callback
retries remain idempotent. Users cannot request refunds. Exceptional refunds
remain an administrator-only manual process.

## Administrator Quota Operations

`POST /api/v1/admin/subscriptions/:id/reset-quota` accepts any non-empty
combination of daily, weekly, and monthly resets.

- Resetting daily usage does not change weekly or monthly usage.
- Resetting weekly usage does not change daily or monthly usage.
- Resetting monthly usage does not change daily or weekly usage and clears the
  monthly bonus.
- Each reset window restarts at the administrator action time.

`PUT /api/v1/admin/subscriptions/:id/monthly-bonus` sets the current monthly
window's total extra quota:

```json
{ "amount_usd": 100 }
```

The value must be finite and non-negative. Zero clears it. It is a replacement,
not an increment. The effective monthly quota is the plan group's base monthly
quota plus `monthly_bonus_usd`. The bonus does not change existing usage,
daily/weekly quota, or subscription dates, and it never bypasses daily or
weekly limits.

The bonus is cleared by an automatic monthly reset, administrator monthly
reset, renewal, expiry, or revocation.

These administrator writes use the existing idempotency wrapper and HTTP access
logging. This change does not introduce a general-purpose audit table.

## Administrator Subscription Switching

`POST /api/v1/admin/subscriptions/:id/switch` requires `Idempotency-Key` and
accepts:

```json
{ "target_group_id": 10 }
```

The source must still be the user's only effective subscription after the user
row is locked. The target group must be different, active, subscription-billed,
and on the same platform. Switching does not create a payment, refund, or price
adjustment.

The switch uses one database transaction. Any non-effective, non-deleted
subscription record for the target group is soft deleted first. The source is
then soft deleted and a new target subscription is created with the source's
`starts_at`, `expires_at`, daily/weekly/monthly window starts, corresponding
usage, and `monthly_bonus_usd`. Historical usage logs are not rewritten.

Only non-deleted API keys whose stored `group_id` equals the source subscription
group move to the target group. Standard-group keys and other keys remain
unchanged. After commit, both subscription-group caches and affected API-key
authentication caches are invalidated. A remote cache notification failure is
logged as a warning and does not turn a committed switch into an API failure.

The response contains the new subscription, `previous_subscription_id`,
`migrated_keys`, and `quota_warnings`. Warnings use `daily`, `weekly`, and
`monthly` when carried usage has already reached the corresponding target
limit. The monthly comparison includes the carried temporary quota. A warning
does not block the switch; that period immediately uses balance fallback.

Historical mismatches are repaired only through the standalone
`subscription-key-repair` command. It is dry-run by default. Writes require both
`--apply` and an exact `--expected-count`. Candidates must have exactly one
effective subscription, a key bound to a different same-platform subscription
group, and historical evidence that the user held that source group. The tool
reports and skips standard groups, cross-platform rows, ambiguous sources, and
soft-deleted keys. It never runs automatically as a startup migration.

## Data And API Contract

- `user_subscriptions.monthly_bonus_usd` is a non-null decimal defaulting to 0.
- `payment_orders.subscription_action` and `payment_orders.subscription_id` are
  nullable snapshots for subscription fulfilment.
- The database prevents more than one unfinished subscription order per user.
- The database prevents more than one sale plan per subscription group.
- User subscription responses expose `monthly_bonus_usd`,
  `effective_monthly_limit_usd`, `renewal_eligible`, and `renewal_price`.
- Subscription UI shows base quota, bonus quota, and effective monthly quota.
- Renewal confirmation states that remaining time is discarded, a new 30-day
  period begins, all usage and bonus quota reset, and current pricing applies.

## Assumptions

- Subscription periods are fixed at 30 days.
- Daily, weekly, and monthly quotas remain independent rolling windows.
- The API key's stored group remains the balance-billing fallback group. A
  successful administrator subscription switch updates that stored group only
  for keys bound to the source subscription group.
- Temporary monthly quota does not bypass daily or weekly quota.
- Existing API key restrictions continue to apply before either billing path.

## Status And Links

- Status: implemented and locally validated on
  `codex/single-subscription-auto-billing`; production data audit, backup,
  migration, deployment, and real payment verification have not been run.
- Related execution plan:
  `docs/exec-plans/single-subscription-auto-billing.md`.
- Supersedes the no-balance-fallback subscription behavior recorded in
  `docs/product-specs/zpay-subscription-payment.md`.
