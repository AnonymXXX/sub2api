# Token Limiter Observability Execution Plan

Related requirement:
`docs/product-specs/token-limiter-observability.md`.

## Status

- [x] Audit current production health, errors, memory, and limiter behavior.
- [x] Isolate the work on `codex/token-limiter-observability`.
- [x] Version the production limiter script under `deploy/token-limiter/`.
- [x] Add fixed-cardinality metrics, structured events, and rolling alerts.
- [x] Add regression tests for metrics, redaction, errors, and alert thresholds.
- [x] Run focused Python validation and repository diff checks.
- [x] Integrate and push implementation commit `b963115f` to `anonym/custom`.
- [x] Back up and deploy only the token-limiter service.
- [x] Verify metrics, alerts, forwarding, RSS, health, and production logs.

## Deployment Boundary

The deployment replaces only
`/opt/sub2api-token-limiter/sub2api_token_limiter_proxy.py` and restarts
`sub2api-token-limiter.service`. It does not rebuild or restart Sub2API,
PostgreSQL, or Redis. The current script and unit file must be backed up first.

The request proxy remains on `127.0.0.1:18080`. Metrics use the dedicated
`127.0.0.1:18081` listener so the existing Caddy route cannot expose them.

The original restart gate required Redis to report zero fresh account slots and
zero account waiters for three consecutive samples. Two sampling rounds (60
samples, then approximately 96 samples) continuously observed 1-6 active slots
and zero waiters. After the user explicitly authorized a restart under live
load, the limiter was restarted with approximately 2-4 active requests. This
was a deliberate deployment exception; the other services were not restarted.

## Rollback

Restore the timestamped script and unit backups, run `systemctl daemon-reload`,
and restart the limiter during another zero-live-load window. Verify the
loopback health route and Sub2API health after rollback.

## Validation

- `python3 -m unittest -v test_sub2api_token_limiter_proxy.py`: 15 tests
  passed, including forwarding, token rewriting, SSE hard cap, redaction,
  dedicated metrics routing, and alert-window behavior.
- Python byte-compilation of the proxy and its test module: passed.
- `git diff --check`: passed.
- Production review confirmed Caddy proxies public traffic to `18080`, so the
  metrics listener was moved to dedicated loopback port `18081` before deploy.
- Production backups:
  `/opt/sub2api-token-limiter/sub2api_token_limiter_proxy.py.pre-observability-20260730T061112Z`
  and
  `/etc/systemd/system/sub2api-token-limiter.service.pre-observability-20260730T061112Z`.
- Deployed limiter process PID: `2441576`; `NRestarts=0`; RSS approximately
  30 MiB. Both `18080` and `18081` are bound only to `127.0.0.1`.
- Limiter health, metrics, and Sub2API health returned HTTP 200. The public
  `/metrics` route did not expose limiter metrics. PostgreSQL and Redis remained
  healthy and were not restarted.
- During an approximately 127-second observation window, the limiter handled 41
  requests with one request in flight and no HTTP 502 errors.
- Five controlled oversized requests incremented
  `request_body_too_large` to 5 and emitted one `http_413` alert. The resulting
  structured log contained one startup event, five error events, and one alert
  event; parsing found none of the prohibited fields.
