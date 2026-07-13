# OpenAI Upstream Response Header Timeout

## Decision

OpenAI HTTP upstream requests must not wait indefinitely for response headers.
The default response-header timeout is 300 seconds and remains configurable
through `gateway.openai_response_header_timeout` or
`GATEWAY_OPENAI_RESPONSE_HEADER_TIMEOUT`.

An explicit value of `0` continues to disable the timeout for operators who
accept unbounded response-header waits. Streaming response-body idle timeouts
remain separate and are not changed by this decision.

When the HTTP transport returns an error before response headers arrive, the
affected account's idle upstream connections are closed before the existing
failover/error policy runs. This permits the next attempt to establish a fresh
connection without adding a new transparent retry.

## Success Criteria

- A default deployment stops waiting for OpenAI response headers after 300
  seconds.
- Explicit positive and zero configuration values retain their documented
  behavior.
- A response-header transport error releases the request's connection-pool
  occupancy and closes idle connections for the affected account client.
- Existing account failover policy remains responsible for deciding whether a
  different account is attempted.
- SSE response-body timeout behavior and HTTP protocol selection are unchanged.

## Assumptions And Open Questions

- Assumption: 300 seconds is above the observed normal TTFT baseline while
  bounding the impact of an upstream incident.
- Assumption: automatic HTTP/1.1 fallback is out of scope until protocol-level
  evidence shows it improves reliability.
- Open question: production may lower the timeout after observing timeout and
  TTFT metrics under the 300-second limit.

## Status And Links

- Status: confirmed
- Related execution plan: `docs/exec-plans/openai-upstream-response-header-timeout.md`
- Supersedes: the default unlimited OpenAI response-header wait
