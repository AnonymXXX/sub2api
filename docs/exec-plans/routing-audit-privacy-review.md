# Routing Audit And Privacy Review

## Goal

Use Sub2API as the single cc-switch provider while keeping enough evidence to review routing choices between local Plus pools and relay subscriptions.

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

## Privacy Boundary

- Do not store prompt text, request bodies, secret values, cookies, OAuth tokens, API keys, SSH passwords, or private keys in routing audit logs.
- Store only request/account/group IDs, model/endpoint labels, aggregate pool snapshots, token/cost counts, and privacy decision categories such as `rule_sensitive:env_secret_assignment`.
- If a future AI privacy reviewer is added, persist only its category/decision metadata unless a separate encrypted evidence store is explicitly designed and approved.

## Validation

- Backend: `go test ./internal/service ./internal/repository ./internal/handler/admin`
- Frontend: `npm run typecheck`
