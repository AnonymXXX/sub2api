# Sub2API Token Limiter

This directory versions the loopback token-limiter proxy used by the self-hosted
deployment. The proxy keeps the existing output-token clamp and Responses SSE
hard cap while exposing process-local Prometheus metrics on a separate
loopback-only listener.

## Validation

```bash
cd deploy/token-limiter
python3 -m unittest -v test_sub2api_token_limiter_proxy.py
python3 -m py_compile sub2api_token_limiter_proxy.py
```

## Metrics

The listener is bound to `127.0.0.1` by the systemd unit. Query metrics locally:

```bash
curl -fsS http://127.0.0.1:18081/metrics
```

Counters reset on restart. Event and alert labels are fixed in source; never add
request-derived labels. Caddy must continue to route public traffic only to the
request proxy on port `18080`; it must not route to the metrics port `18081`.

## Deployment

Back up the production script and unit before replacing them. Restart the
service only after Redis reports zero fresh account slots and zero account
waiters for three consecutive samples. Sub2API, PostgreSQL, and Redis do not
need to be restarted for this proxy-only change.
