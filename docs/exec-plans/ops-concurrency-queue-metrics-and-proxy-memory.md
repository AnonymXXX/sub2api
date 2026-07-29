# Ops Queue Metrics And Proxy Memory Execution Plan

Related requirement:
`docs/product-specs/ops-concurrency-queue-metrics.md`.

## Status

- [x] Diagnose current concurrency, queueing, host pressure, and rejection data.
- [x] Identify zero-to-NULL persistence as the queue-history ambiguity.
- [x] Identify glibc arena retention in the token-limiter process.
- [x] Create an isolated task worktree from `anonym/custom`.
- [x] Add a failing repository regression test for queue depth zero.
- [x] Preserve zero only for `concurrency_queue_depth` and keep nil nullable.
- [x] Run focused and broad backend validation.
- [ ] Integrate and push the validated commit to `anonym/custom`.
- [ ] Back up PostgreSQL and deploy only the Sub2API application container.
- [ ] Back up the token-limiter systemd unit, add `MALLOC_ARENA_MAX=2`, and
  restart it only at zero live concurrency.
- [ ] Verify health, deployed commit, queue samples, RSS, and logs.

## Deployment Boundary

The deployment recreates only `sub2api`. PostgreSQL and Redis remain running.
The token-limiter restart is a separate operation and must be delayed until
Redis reports zero account slots and zero account waiters. The existing unit
file must be backed up before editing, and rollback consists of restoring the
backup and restarting the proxy.

Historical queue-depth rows remain unchanged. Validation checks newly collected
one-minute rows after the application deployment.

## Validation

- `go test ./internal/repository -run 'TestOpsRepositoryInsertSystemMetrics_' -count=1`: passed.
- `go test ./... -count=1`: passed.
- `go build ./cmd/server`: passed.
- `git diff --check`: passed.
