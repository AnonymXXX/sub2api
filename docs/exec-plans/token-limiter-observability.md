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
- [ ] Integrate and push to `anonym/custom`.
- [ ] Back up and deploy only the token-limiter service at zero live load.
- [ ] Verify metrics, alerts, forwarding, RSS, health, and production logs.

## Deployment Boundary

The deployment replaces only
`/opt/sub2api-token-limiter/sub2api_token_limiter_proxy.py` and restarts
`sub2api-token-limiter.service`. It does not rebuild or restart Sub2API,
PostgreSQL, or Redis. The current script and unit file must be backed up first.

The request proxy remains on `127.0.0.1:18080`. Metrics use the dedicated
`127.0.0.1:18081` listener so the existing Caddy route cannot expose them.

The service restart is permitted only after Redis reports zero fresh account
slots and zero account waiters for three consecutive samples.

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
