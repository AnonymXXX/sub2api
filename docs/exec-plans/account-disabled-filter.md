# Unified Account Disabled State Execution Plan

Related requirement: `docs/product-specs/account-disabled-filter.md`.

## Status

- [x] Confirm repository, production data shape, and deployment target.
- [x] Create and bootstrap an isolated task worktree.
- [x] Capture account-state compatibility and usage-date requirements.
- [x] Add failing backend repository, service, and handler tests.
- [x] Implement backend filtering and state normalization.
- [x] Add failing frontend filter, editor, switch, and usage-date tests.
- [x] Implement frontend status and usage-date changes.
- [x] Run focused and broad validation.
- [x] Integrate into `anonym/custom`, push, deploy, and observe production.

## Validation

Passed:

- Backend admin service unit tests covering manual status normalization,
  scheduling-switch coupling, and conflicting bulk requests.
- Backend admin handler validation for canonical `disabled` writes.
- Repository integration coverage for unified disabled filtering, including
  legacy `inactive`/`unschedulable` compatibility and unschedulable accounts.
- `go test ./... -count=1`
- `go build ./...`
- Frontend focused Vitest coverage for filters, single/bulk editors, row switch,
  filtered-list synchronization, shared date picker behavior, and user usage
  date defaults.
- `pnpm typecheck`
- ESLint on all changed frontend files.
- `pnpm build`
- `git diff --check`

Known baseline failures, reproduced outside this task diff:

- Two `CreateAccountModal` specs still expect the old `gpt-5.5`/`gpt-5.4`
  model list while the product data now contains `gpt-5.6-*` models.
- Two `GetModelPricing_OpenAICompactAliasesFallback` unit cases do not yet
  cover the new `gpt-5.6` compact aliases.
- Full frontend ESLint reports `vue/no-reserved-component-names` in the
  unchanged `CreateAccountModal.spec.ts` fixture.

## Deployment Boundary

The user authorized a push to `origin/anonym/custom` and deployment to
`racknerd-us`. Before replacing the application container, create a fresh
PostgreSQL backup and tag the current image for rollback. Recreate only the
application service, retain PostgreSQL and Redis, then verify health, logs,
account filtering/state controls, and the user usage page's Today default.

## Production Result

Deployed on 2026-07-23:

- Remote branch: `origin/anonym/custom`
- Application commit: `68042bae`
- Image: `sub2api-local:v0.1.153-h3`
- Reported version: `v0.1.153-h3`
- Application container: healthy with `RestartCount=0`
- Health response: `{"status":"ok"}`
- PostgreSQL and Redis remained healthy and were not recreated.

Production verification passed:

- The unified disabled query returned account IDs `19` and `26`; both were
  legacy-shaped `active` accounts with `schedulable=false`.
- Selecting Disabled in the admin UI displayed those two accounts.
- The status menu contains All, Active, Disabled, Error, Rate limited, and
  Temporarily unschedulable; the standalone Unschedulable option is gone.
- The user usage page opens with Today and hourly granularity, and Reset
  restores the same defaults.

Rollback assets:

- Database backup:
  `/opt/sub2api/backups/pre-disabled-filter-20260723T074720Z.sql.gz`
- Database backup SHA-256:
  `8d7014a10e20965fa45007510b3150ae1d10562c77db5f3205e396140a4e9d5c`
- Compose override backup:
  `/opt/sub2api/backups/pre-disabled-filter-20260723T074720Z.override.yml`
- Previous image tag:
  `sub2api-local:rollback-before-v0.1.153-h3-20260723T074720Z`
- Previous image ID:
  `sha256:1df3649ba5da25e7e410bb689ac2f0c7b5c92346bcfe8d455551e13d35f26ca9`
