# Single Subscription And Automatic Billing Implementation

Related requirement:
`docs/product-specs/single-subscription-auto-billing.md`.

## Status

- [x] Capture the confirmed product behavior and supersede the old fallback rule.
- [x] Add database migrations, Ent schema fields, and generated code.
- [x] Enforce the single active subscription invariant at every grant path.
- [x] Add monthly bonus quota and independent administrator reset behavior.
- [x] Implement purchase versus renewal order snapshots and idempotent fulfilment.
- [x] Implement subscription-first request routing and atomic balance fallback.
- [x] Update user subscription/payment UI and administrator quota controls.
- [x] Add backend and frontend regression coverage.
- [x] Run backend tests and frontend tests, lint, typecheck, and production build.
- [x] Encode migration conflict checks and document production rollout prerequisites.
- [x] Commit, integrate locally when eligible, and clean up the task worktree.
- [ ] Back up and audit production data, run the migration, deploy, and verify a
  real payment after separate production authorization.

## Safety Boundary

All work is isolated in
`/Users/anonym/Documents/jobs/person/sub2api.worktrees/single-subscription-auto-billing`
on branch `codex/single-subscription-auto-billing`. Do not run migrations against
production, change payment provider state, push, or deploy without explicit
authorization. Migration preflight is read-only and must stop when existing
overlapping active subscriptions, duplicate unfinished subscription orders, or
multiple sale plans for one group are found.

## Implementation Notes

- Select the effective billing group only after platform, model, and capability
  parsing and before account routing.
- Pass one immutable effective billing decision through routing, multipliers,
  logging, and settlement.
- Settlement rechecks subscription eligibility in a database transaction. If
  it cannot charge the complete subscription cost, it atomically charges the
  complete balance cost instead. Because settlement follows an already-incurred
  upstream cost, this fallback can make the balance negative when funds became
  insufficient after admission.
- Keep API key restrictions and key quota independent from the selected billing
  source. Charge API key quota and amount-based rate limits using the final
  settled cost.
- Renewal snapshots are server-generated and callback processing is idempotent.
- Treat only subscriptions with `starts_at <= now < expires_at` and active
  status as active.
- Keep at most one `for_sale=true` plan per subscription group so renewal price
  and duration remain unambiguous.
- Administrator quota writes use existing idempotency and HTTP access logs; no
  generic audit table is added by this change.

## Validation

- Focused backend regression tests passed for future subscriptions, monthly
  bonus handling, and atomic final billing effects (`7 passed`).
- Frontend unit suite passed (`149` files, `974` tests).
- Frontend lint, TypeScript typecheck, and production build passed. The build
  reports existing dynamic/static import and large-chunk warnings.
- The backend full unit suite completed with `9113 passed`, `3 failed`, and `6
  skipped`. All three failures belong to the known OpenAI compact-pricing alias
  test and are reproducible on the task baseline; they are outside this feature.
- Production data checks, migration execution, deployment, and real payment
  verification remain intentionally unexecuted.

## Deployment Prerequisites

1. Back up the production database and verify the backup before migration.
2. Run read-only checks for overlapping active subscriptions, duplicate
   unfinished subscription orders, and multiple sale plans for one group.
3. Stop when any check returns conflicts and resolve them explicitly.
4. Deploy application and migrations together during a controlled window.
5. Verify subscription-first usage, balance fallback, and one real renewal with
   a test user before enabling sales broadly.
6. Observe effective group, billing type, subscription usage, and balance
   changes after deployment.
