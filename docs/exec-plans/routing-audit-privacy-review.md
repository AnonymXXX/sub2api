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

### Pending

- Configure APIPod Code as an external relay account/package in the live backend.
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
- Last deployment status must be checked from the deployment runbook or live server before resuming operations.

### Resume Next

1. Check `git status --short` and confirm these docs are committed or intentionally pending.
2. Verify the live backend branch and deployed image/container.
3. Configure APIPod as a relay account without committing secrets.
4. Implement quota snapshot capture and user/group routing if missing in code.
5. Run validation commands, deploy, then observe routing logs for 48 hours.

## Privacy Boundary

- Do not store prompt text, request bodies, secret values, cookies, OAuth tokens, API keys, SSH passwords, or private keys in routing audit logs.
- Store only request/account/group IDs, model/endpoint labels, aggregate pool snapshots, token/cost counts, and privacy decision categories such as `rule_sensitive:env_secret_assignment`.
- If a future AI privacy reviewer is added, persist only its category/decision metadata unless a separate encrypted evidence store is explicitly designed and approved.

## Validation

- Backend: `go test ./internal/service ./internal/repository ./internal/handler/admin`
- Frontend: `npm run typecheck`
