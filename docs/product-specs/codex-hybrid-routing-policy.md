# Codex Hybrid Routing Policy

## Goal

Use one self-hosted Sub2API endpoint as the only cc-switch provider for team Codex usage. Sub2API then routes requests across an external relay subscription and trusted local OpenAI Plus accounts while keeping privacy-sensitive traffic away from third-party relays.

## Confirmed Package And Capacity Assumptions

- External relay: APIPod Code, connected as an upstream OpenAI-compatible API key account inside Sub2API.
- Purchase price discussed: `250 RMB/month`.
- Package quota discussed earlier: daily `$150`, weekly `$500`, monthly `$1500`, with relay billing approximated as `0.8x` official token price.
- Current weekly quota target: `$550/week`, overriding the earlier `$500/week` planning number.
- Current monthly target remains `$1500/month` unless the provider contract says otherwise.
- Local Plus account planning price: `40 RMB/Plus/month`.

Sizing background from the prior usage review:

- Recent 30-day usage was about `$2228.63` official-price equivalent, or `$1782.91` at `0.8x`.
- Recent 7-day usage was about `$1192.02` official-price equivalent, or `$953.61` at `0.8x`.
- The main bottleneck is weekly quota. A `$550/week` relay plan still needs Plus fallback for peak weeks.

## People And Pool Allocation

Target monthly purchase plan after current remaining quotas are exhausted:

| Person | Purchase / Pool | Monthly Cost |
| --- | --- | ---: |
| Anonym / shared ops | 1 APIPod relay package + 4 public backup Plus accounts | `250 + 160 = 410 RMB` |
| yangkai | 2 dedicated Plus accounts | `80 RMB` |
| liuxue | No dedicated Plus at first; uses relay and public backup | `0 RMB` |
| Total | 1 relay package + 6 Plus accounts | `490 RMB/month` |

Fair cost sharing can split the relay package across all three users:

| Person | Fair Monthly Share |
| --- | ---: |
| Anonym | about `243 RMB` (`4` public backup Plus + one third of relay) |
| yangkai | about `163 RMB` (`2` dedicated Plus + one third of relay) |
| liuxue | about `83 RMB` (one third of relay) |
| Total | about `489 RMB/month` |

Rationale from usage analysis:

- `yangkai` is a heavy Codex user, about `26%` of recent total cost. His 7-day usage was about `$312.31` official-price equivalent, or `$249.85` at `0.8x`.
- `liuxue` is lighter, about `11%` of recent total cost. His 7-day usage was about `$135.57` official-price equivalent, or `$108.46` at `0.8x`.
- Therefore `yangkai` should have a dedicated Plus pool; `liuxue` does not need dedicated Plus initially.

## Routing Model

All users should configure only one cc-switch provider:

- Provider name: Sub2API
- Local endpoint: `http://anonymzzz.local:10086`
- Public endpoint, when needed: `https://api.lovebirds.xin`
- Users must not directly configure APIPod Code in cc-switch.

Sub2API should own all routing:

- `relay-apipod`: APIPod Code external relay, default pool for ordinary non-sensitive traffic.
- `trusted-plus`: local trusted Plus/OAuth pool for privacy-sensitive traffic.
- `yk-plus`: yangkai dedicated Plus accounts.
- `plus-backup`: shared backup Plus accounts.
- `filter-model-pool`: trusted local `gpt-5.5` pool for privacy review only.

Target routing rules:

- Ordinary users default to `relay-apipod`.
- `yangkai` defaults to `yk-plus`, falls back to `relay-apipod`, then `plus-backup`.
- `liuxue` defaults to `relay-apipod`, then `plus-backup`.
- Sensitive requests must not be sent to APIPod; route them to `trusted-plus`.
- Public backup Plus should be used only when APIPod quota is tight, APIPod is unhealthy, or there is a temporary peak.

Strict "one cc-switch provider for everyone" with user-specific pools requires Sub2API to support API-key/user-level routing policy. A simple configuration-only setup can provide a unified endpoint, but cannot fully express `yangkai` dedicated-pool preference without either extra groups/API keys or backend policy support.

## Privacy Policy

Do not redact secrets that Codex needs to complete a task and then continue through APIPod. Redaction can break SSH login, API calls, config writes, and production troubleshooting.

Use routing instead:

- If the request has no sensitive data, send it to APIPod.
- If it contains example or placeholder secrets that are not needed for execution, it may be redacted and sent to APIPod.
- If it contains real SSH passwords, API keys, cookies, OAuth tokens, `.env` values, database URLs with credentials, or other task-required secrets, route the original request to trusted local Plus.
- If privacy review fails or is uncertain, do not send the request to APIPod.

Privacy filtering should be two-stage:

- Local high-recall precheck runs first and is cheap.
- Trusted local `gpt-5.5` review runs only when the request is APIPod-bound and precheck finds suspicious content.

Environment variable references are not sensitive by themselves:

- Safe reference examples: `process.env.OPENAI_API_KEY`, `os.Getenv("TOKEN")`, `System.getenv("TOKEN")`, `std::env::var("TOKEN")`.
- Actual values such as `OPENAI_API_KEY=...`, bearer tokens, cookies, passwords, and private keys are sensitive.

## Audit And Review Requirements

Routing and privacy logs must never store raw prompts, request bodies, API keys, OAuth tokens, cookies, SSH passwords, private keys, or complete upstream credentials.

They should store reviewable metadata:

- user/API key/model labels
- selected pool and selected account ID
- routing reason, fallback reason, privacy/quota/error redirect flags
- token and estimated cost totals
- routing policy version and precheck rules version
- Plus pool snapshot at decision time
- APIPod quota snapshot at decision time

The admin routing audit page should make chart-based review the primary workflow. It should visualize request success, redirect composition, pool share, decision-reason share, token/cost concentration, latency, and latest pool/quota pressure from aggregate metadata only. Tables remain available for exact values and request-level drilldown, but operators should be able to judge routing health from the chart panels first without exposing prompt text or secrets.

APIPod quota snapshots should include daily/weekly/monthly limit, used, remaining, used percent, quota source, and whether warning/degrade/disable thresholds were active.

Plus snapshots should include total/active/available/rate-limited/error account counts, reset timing summaries, and available concurrency summaries.

Relay accounts such as APIPod should show a compact quota/subscription summary in the account management list. The preferred display is upstream balance and active subscription status. If the upstream does not expose a stable subscription API, the UI should fall back to configured package limits plus locally recorded usage, and clearly mark the data source as local estimate or configured metadata rather than upstream-confirmed balance.

APIPod Code exposes two useful quota surfaces:

- API-key balance: `GET /v1/usage`, authorized with the OpenAI-compatible API key. The response should be parsed like cc-switch: `remaining`, `quota.remaining`, or `balance`, plus `unit`.
- Web subscription catalog: `GET /api/v1/subscriptions?timezone=Asia%2FShanghai`, authorized with an APIPod web session token. This is optional and must not be required for normal operation. When present, plan fields include `name`, `price`, `rate_multiplier`, `daily_limit_usd`, `weekly_limit_usd`, `monthly_limit_usd`, and `validity_days`.

The currently expected APIPod plan is `Codex Basic`: `249 RMB/month`, `0.8x`, daily `$150`, weekly `$550`, monthly `$1500`, valid for `30` days. If no active subscription can be queried, display this as configured/package metadata rather than active-subscription proof.

After APIPod is connected, run at least 48 hours of observation before changing thresholds. Review:

- APIPod usage share and remaining quota
- local Plus usage reasons
- privacy redirect rate
- fallback reasons
- `yangkai` dedicated pool pressure
- whether `liuxue` needs a dedicated Plus account

## Current Implementation Status

- Implemented: routing audit table/API/UI, privacy precheck metadata, local Plus default model whitelist, Plus default concurrency `2`, deployment branch/runbook, and account-list APIPod quota/subscription summary display.
- Implemented: high-confidence rule precheck redirects obvious secret-bearing OpenAI Responses requests to trusted Plus.
- Configured in live backend: APIPod Code upstream account is connected as account `24`, bound to the `openai` group, marked `routing_pool=relay-apipod`, and populated with current upstream model mapping.
- Partially supported by configuration: APIPod participates in ordinary OpenAI Responses scheduling and can be preferred by priority, but the advanced scheduler uses top-K weighted selection, so it does not provide strict "APIPod first, Plus only as fallback" behavior.
- Not yet implemented: strict pool-order routing policy, APIPod quota snapshot/threshold behavior, API-key/user-level routing policy, `yk-plus` dedicated pool policy, and full trusted `gpt-5.5` AI privacy reviewer.
