# Ops Concurrency Queue Metrics

## Decision

The one-minute operations metric `concurrency_queue_depth` must preserve a
collected value of `0`. SQL `NULL` is reserved for collection failure or an
unavailable collector dependency.

The self-hosted token-limiter proxy must run with `MALLOC_ARENA_MAX=2` on the
two-vCPU production host. This limits glibc arena retention without changing
request rewriting, response streaming, or output-token enforcement behavior.

## Success Criteria

- A collected queue depth of `0` is stored as database value `0`.
- A missing queue-depth sample remains SQL `NULL`.
- Existing nullable latency and status metrics keep their current zero-to-NULL
  behavior.
- The token-limiter service restarts only after real-time account concurrency
  reaches `0`.
- After restart, the proxy is active, requests still reach Sub2API, and its RSS
  does not return to the previously observed high-water range of approximately
  900 MiB under comparable traffic.
- PostgreSQL and Redis are not restarted by this rollout.

## Assumptions And Boundaries

- Existing historical `NULL` queue samples are not backfilled because the
  database cannot distinguish an old zero value from a failed collection.
- `MALLOC_ARENA_MAX=2` is appropriate for the current two-vCPU host and current
  application concurrency limit. It must be re-evaluated if the host CPU count
  or proxy concurrency model changes materially.
- Request-body parsing and forwarding optimizations are intentionally deferred
  until the allocator mitigation has been observed in production.

## Status And Links

- Status: deployed and verified in production
- Related execution plan:
  `docs/exec-plans/ops-concurrency-queue-metrics-and-proxy-memory.md`
