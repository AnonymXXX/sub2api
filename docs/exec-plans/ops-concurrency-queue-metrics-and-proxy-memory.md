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
- [x] Integrate and push the validated commit to `anonym/custom`.
- [x] Back up PostgreSQL and deploy only the Sub2API application container.
- [x] Back up the token-limiter systemd unit, add `MALLOC_ARENA_MAX=2`, and
  restart it only at zero live concurrency.
- [x] Verify health, deployed commit, queue samples, RSS, and logs.

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
- Commit `32f2f9b0` was pushed to `origin/anonym/custom` and deployed as
  `sub2api-local:v0.1.153-h5`.
- The application health endpoint returned HTTP `200`; PostgreSQL and Redis
  container IDs were unchanged.
- The first complete post-deploy minute stored `concurrency_queue_depth=0`;
  pre-deploy minute rows remained `NULL`.
- Post-deploy application logs contained no error, panic, fatal, OOM, HTTP 502,
  connection-reset, or connection-refused lines during the observation window.
- The token-limiter unit backup is
  `/etc/systemd/system/sub2api-token-limiter.service.pre-arena-20260729013547`.
  The service was restarted only after three consecutive samples reported zero
  account slots and zero waiters. The new process environment contains
  `MALLOC_ARENA_MAX=2`.
- The token limiter and application health endpoints both returned HTTP `200`
  after restart. New traffic reached two active account slots with zero waiters,
  and neither service logged errors, OOMs, HTTP 502s, or connection failures.
- Token-limiter RSS was approximately 18 MiB immediately after restart and
  approximately 33 MiB after one minute of resumed traffic.
- Docker build cache was removed. The root filesystem had 21 GiB available at
  41 percent use after cleanup.
