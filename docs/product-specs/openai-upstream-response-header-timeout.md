# OpenAI Upstream Response Header Timeout

## Decision

Ordinary OpenAI HTTP upstream requests must not wait indefinitely for response
headers. Their default response-header timeout is 120 seconds and remains
configurable through `gateway.openai_response_header_timeout` or
`GATEWAY_OPENAI_RESPONSE_HEADER_TIMEOUT`.

OpenAI `/responses/compact` requests use a separate 300-second response-header
timeout through `gateway.openai_compact_response_header_timeout` or
`GATEWAY_OPENAI_COMPACT_RESPONSE_HEADER_TIMEOUT`. Compact requests also use a
separate cached HTTP client/transport profile so their longer timeout does not
replace or reuse the ordinary OpenAI transport entry.

An explicit value of `0` continues to disable the corresponding timeout for
operators who accept unbounded response-header waits. Streaming response-body
idle timeouts remain separate and are not changed by this decision.

When the HTTP transport returns an error before response headers arrive, the
affected account's idle upstream connections are closed before the existing
failover/error policy runs. This permits the next attempt to establish a fresh
connection without adding a new transparent retry.

## Success Criteria

- A default deployment stops waiting for ordinary OpenAI response headers
  after 120 seconds.
- `/responses/compact` stops waiting for response headers after 300 seconds.
- Ordinary and compact OpenAI requests use separate cached transports.
- Explicit positive and zero configuration values retain their documented
  behavior.
- A response-header transport error releases the request's connection-pool
  occupancy and closes idle connections for the affected account client.
- Existing account failover policy remains responsible for deciding whether a
  different account is attempted.
- SSE response-body timeout behavior and HTTP protocol selection are unchanged.

## Assumptions And Open Questions

- Assumption: 120 seconds is above the observed ordinary TTFT baseline while
  reducing the impact of repeated upstream incidents.
- Assumption: compact work may legitimately need longer than 120 seconds, so
  its default remains 300 seconds.
- Assumption: automatic HTTP/1.1 fallback is out of scope until protocol-level
  evidence shows it improves reliability.
- Open question: production may tune either timeout after observing timeout and
  TTFT metrics under the split limits.

## Status And Links

- Status: confirmed
- Related execution plan: `docs/exec-plans/openai-upstream-response-header-timeout.md`
- Supersedes: the single 300-second OpenAI timeout used for both ordinary and
  compact requests
