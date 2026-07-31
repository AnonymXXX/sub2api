# ZPAY Subscription Payment

## Decision

Sub2API will use ZPAY through the built-in EasyPay provider for CNY payments.
The initial rollout enables Alipay only. WeChat remains disabled until the
Alipay flow has completed real-payment verification and the additional channel
cost is justified.

The 30-day subscription catalog is:

| Plan | Daily quota | Weekly quota | Monthly quota | Price |
| --- | ---: | ---: | ---: | ---: |
| Light | $75 | $225 | $500 | CNY 39 |
| Standard | $150 | $600 | $1,200 | CNY 69 |
| Advanced | $450 | $1,000 | $2,600 | CNY 169 |
| Flagship | $500 | $1,500 | $5,500 | CNY 329 |

User ID 1 remains a self-use account whose cost is excluded from subscription
profit calculations. Its original OpenAI group remains private to that user.
The original no-balance-fallback rule is superseded by
`docs/product-specs/single-subscription-auto-billing.md`: existing API keys now
prefer the user's active subscription and fall back to their original key group
and account balance whenever the subscription is not applicable.

## Payment Configuration

- Provider: EasyPay-compatible ZPAY.
- API base: `https://zpayz.cn`.
- Notify URL: `https://api.lovebirds.xin/api/v1/payment/webhook/easypay`.
- Return URL: `https://api.lovebirds.xin/payment/result`.
- Balance recharge conversion: CNY 1 credits USD 1 of site balance.
- Recharge range: CNY 10 to CNY 500 per order.
- Recharge daily limit: CNY 1,000 per user.
- Recharge surcharge: 0 percent.
- Order timeout: 30 minutes.
- Maximum pending orders: 3 per user.
- Refunds and user self-service refunds remain disabled. Exceptional refunds
  are handled manually by an administrator.
- Automatic renewal is not provided.

## Rollout Gates

Payment, visible payment methods, provider instances, and plan sales remain
disabled until all of the following are true:

1. ZPAY approves the merchant and the owner completes Alipay signing.
2. EasyPay credentials are stored through the encrypted provider configuration
   and are never copied into source control or project documentation.
3. A CNY 10 balance recharge verifies signature handling, callback idempotency,
   order completion, and credited balance.
4. A CNY 39 plan purchase verifies the 30-day subscription grant.
5. Refund requests remain unavailable to users.

Any failure closes the provider, visible method, plan sales, and global payment
switch before rollback or investigation.

## Assumptions And Risks

- ZPAY's actual combined fee is expected to be 1.6 percent; profitability must
  be recalculated if the approved channel uses a different rate.
- The merchant application truthfully describes the product as AI API usage
  and software technical services, not account sales.
- ZPAY's query-order endpoint is documented as GET. Recovery from delayed or
  missing callbacks depends on using that documented method.
- Existing subscriptions keep their purchased terms when future prices change.
- Subscription purchase, renewal, single-active-subscription, and automatic
  balance-fallback rules are defined by
  `docs/product-specs/single-subscription-auto-billing.md`.

## Status And Links

- Status: merchant review in progress; payment remains disabled.
- Related execution plan:
  `docs/exec-plans/zpay-subscription-payment-rollout.md`.
