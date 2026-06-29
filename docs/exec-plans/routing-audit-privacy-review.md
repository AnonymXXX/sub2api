# Routing Audit And Privacy Review

## Goal

Use Sub2API as the single cc-switch provider while keeping enough evidence to review routing choices between local Plus pools and relay subscriptions.

Product policy source: `docs/product-specs/codex-hybrid-routing-policy.md`.

## Scope

- Record OpenAI Responses routing decisions without storing prompts, secrets, cookies, OAuth tokens, API keys, or SSH credentials.
- Store aggregate resource snapshots for Plus pools and relay pools at request time.
- Provide admin APIs and a compact UI for long-term review of pool distribution, privacy redirects, quota redirects, and error redirects.

## Current Phase

- `routing_audit_logs` stores the first reviewable event stream.
- Privacy filtering records rule/AI decision fields but does not persist request content.
- Rule precheck redirects obvious secret-bearing OpenAI Responses requests to the local `trusted-plus` pool before scheduling.
- Environment variable references in code such as `process.env.OPENAI_API_KEY`, `os.Getenv("TOKEN")`, `System.getenv("TOKEN")`, or `std::env::var("TOKEN")` do not trigger privacy redirect by themselves. Actual values in `.env`/config content, such as `OPENAI_API_KEY=sk-...` or password-bearing database URLs, do trigger redirect.
- GPT-based privacy review can be connected later by filling `privacy_decision`, `privacy_redirect`, and `precheck_rules_version`.

## Progress Ledger

Source requirement: `docs/product-specs/codex-hybrid-routing-policy.md`.

### Done

- Added routing audit persistence/API/UI for reviewable routing events.
- Added privacy precheck metadata without storing request bodies or prompt text.
- Added rule precheck for high-confidence secret-bearing OpenAI Responses requests.
- Added local Plus default model whitelist and Plus default concurrency `2`.
- Added self-hosted deployment branch/runbook notes.
- Captured the current Codex hybrid routing product policy and cost/quota assumptions.
- Committed and pushed product/exec-plan documentation to `origin/anonym/custom` at `54914743`.
- Synced `/opt/sub2api-build` on `racknerd-us` to `54914743`; live containers remained healthy and were not rebuilt or restarted for the docs-only update.
- Configured APIPod Code in the live backend as OpenAI API key account `24`, bound to group `openai` (`6`), with `routing_pool=relay-apipod`, priority `0`, concurrency `8`, and package quota metadata only.
- Synced APIPod model mapping from its upstream `/v1/models`; current mapped models include `gpt-5.5`, `codex-auto-review`, `gpt-5.4`, `gpt-5.4-mini`, `gpt-5.2`, `gpt-5.3-codex`, `gpt-5.3-codex-spark`, and `gpt-image-2`.
- Verified Redis scheduler snapshots for `6:openai:single` and `6:openai:forced` include APIPod account `24`.
- Verified high-confidence secret-bearing Responses requests are redirected to `trusted-plus` by rule precheck before scheduling.

### Pending

- Implement a strict pool-order routing policy: ordinary traffic should prefer `relay-apipod` first, then fall back to Plus only for quota, health, capacity, or explicit policy reasons.
- Add APIPod daily/weekly/monthly quota snapshots and threshold behavior.
- Add API-key/user-level routing policy so cc-switch can expose one provider while Sub2API schedules internally.
- Add `yk-plus` dedicated pool policy for `yangkai`.
- Add full trusted `gpt-5.5` AI privacy reviewer if rule precheck evidence shows it is needed.
- Run at least 48 hours of routing/privacy observation after APIPod is connected.

### Blockers And Open Decisions

- APIPod credentials and package quota values must be configured outside documentation and must not be stored in this repository.
- Final routing thresholds should be adjusted only after observation data exists.
- `liuxue` does not get a dedicated Plus account initially; revisit after usage data.

### Validation And Deployment Status

- Current documented validation commands: `go test ./internal/service ./internal/repository ./internal/handler/admin` and `npm run typecheck`.
- Latest source checkout status, reviewed on 2026-06-29: local `anonym/custom`, `origin/anonym/custom`, and server `/opt/sub2api-build` all point to `54914743`.
- Latest live service status, reviewed on 2026-06-29: `sub2api`, `sub2api-postgres`, and `sub2api-redis` were healthy; `/health` returned `{"status":"ok"}`.
- The currently running application image was not rebuilt from `54914743` because that commit only updated documentation.
- APIPod validation on 2026-06-29:
  - A direct upstream model-list check succeeded with `8` models.
  - A single ordinary short Responses request selected `trusted-plus` account `21`, showing strict APIPod default routing is not guaranteed by current advanced scheduling.
  - Six ordinary short Responses probes selected APIPod account `24` for three successful requests, selected Plus account `19` for one successful request, and selected a Plus account with `codex_cli_only` enabled for two local curl requests that returned `403`.
  - A fake secret-bearing test request triggered `privacy_redirect` with `rule_sensitive:env_secret_assignment,openai_api_key`; it did not select APIPod.

### Configuration Findings

- Current configuration can connect APIPod and make it eligible for normal OpenAI Responses traffic.
- Current configuration can keep obvious secret-bearing traffic away from APIPod through the existing privacy precheck pool filter.
- With `openai_advanced_scheduler_enabled=true`, current configuration cannot express "always use `relay-apipod` first, then Plus as fallback". The advanced scheduler uses top-K scoring plus weighted selection, so higher-priority APIPod is preferred probabilistically rather than strictly.
- A temporary no-code option is to set `openai_advanced_scheduler_enabled=false`, which returns selection to priority plus least-recently-used behavior. That should make account `24` win ordinary routing because its priority is `0`, while privacy precheck still constrains sensitive requests to `trusted-plus`. The tradeoff is losing the advanced scheduler's load/queue/error/TTFT scoring for OpenAI traffic.

### Resume Next

1. Decide whether to temporarily disable `openai_advanced_scheduler_enabled` for strict APIPod-first priority routing, or leave it enabled until a proper pool-order routing policy is implemented.
2. Implement API-key/user-level routing policy and strict pool-order fallback (`relay-apipod` -> `yk-plus` or `plus-backup` according to user policy).
3. Implement APIPod quota snapshot capture and threshold behavior.
4. Add `yk-plus` dedicated pool policy for `yangkai`.
5. Add full trusted `gpt-5.5` AI privacy reviewer only if rule precheck evidence shows it is needed.
6. Run validation commands, deploy code changes if any, then observe routing logs for 48 hours.

## Privacy Boundary

- Do not store prompt text, request bodies, secret values, cookies, OAuth tokens, API keys, SSH passwords, or private keys in routing audit logs.
- Store only request/account/group IDs, model/endpoint labels, aggregate pool snapshots, token/cost counts, and privacy decision categories such as `rule_sensitive:env_secret_assignment`.
- If a future AI privacy reviewer is added, persist only its category/decision metadata unless a separate encrypted evidence store is explicitly designed and approved.

## Validation

- Backend: `go test ./internal/service ./internal/repository ./internal/handler/admin`
- Frontend: `npm run typecheck`
