# Token Limiter Observability

## Decision

The self-hosted token-limiter proxy must expose low-cardinality Prometheus
metrics on a dedicated loopback-only listener and emit one-line structured JSON
for locally generated rejection and upstream-proxy failures.

Observability must not change the existing request-body limit, upstream timeout,
output-token clamp, Responses SSE hard cap, or forwarding behavior.

## Data Boundary

Logs and metrics may include:

- a fixed endpoint category;
- HTTP method and locally generated status code;
- request body size and configured limit;
- elapsed milliseconds and a fixed failure reason;
- a one-way, truncated hash of an existing request ID.

They must not include request or response bodies, authorization headers, API
keys, OAuth tokens, account or user IDs, query strings, arbitrary URL paths, or
raw exception messages.

## Metrics And Alerts

- `GET /metrics` returns Prometheus text from a dedicated listener bound to
  `127.0.0.1:18081`.
- The public Caddy route continues to use the request proxy on
  `127.0.0.1:18080` and must never route to the metrics listener.
- Counters use a fixed set of reason labels so untrusted input cannot create
  high-cardinality series.
- The proxy records total and in-flight forwarded requests, locally generated
  error reasons, and locally emitted alerts.
- Within a rolling ten-minute window, five request-body rejections or three
  locally generated HTTP 502 responses emit a structured warning.
- Repeated warnings for the same alert group are suppressed for ten minutes.
- Counters are process-local and reset when the service restarts.

## Success Criteria

- Existing token rewriting and SSE-limit tests continue to pass.
- An oversized request produces HTTP 413, increments its reason counter, and
  emits no sensitive request data.
- Connect and relay timeouts have distinct fixed reasons.
- `127.0.0.1:18081/metrics` is served without contacting Sub2API and contains
  no request data.
- Production deployment backs up the current script and systemd unit and leaves
  Sub2API, PostgreSQL, and Redis running. The normal restart gate requires zero
  account concurrency and zero waiters; a restart with active requests requires
  explicit user authorization and must be recorded as a deployment exception.

## Status And Links

- Status: deployed and production-validated on 2026-07-30
- Implementation commit: `b963115f`
- Related execution plan:
  `docs/exec-plans/token-limiter-observability.md`
