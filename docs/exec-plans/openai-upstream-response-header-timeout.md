# OpenAI Upstream Response Header Timeout Execution Plan

Related requirement:
`docs/product-specs/openai-upstream-response-header-timeout.md`.

## Status

- [x] Diagnose the production incident and confirm the wait occurs before
  response headers.
- [x] Confirm the integration branch and create an isolated task worktree.
- [x] Capture the timeout, connection recovery, and retry boundaries.
- [x] Add failing configuration and transport regression tests.
- [x] Implement the 300-second default and per-client connection recovery.
- [x] Update self-hosted deployment examples.
- [x] Run focused and broad backend validation.
- [x] Observe production behavior under the 300-second bound.
- [x] Add an independent 300-second compact transport profile.
- [x] Lower the ordinary OpenAI default to 120 seconds.
- [x] Merge upstream `7d239d62` (`v0.1.153`) while preserving custom behavior.
- [x] Run focused and release-level validation after the upstream merge.
- [x] Integrate into `anonym/custom`, push, deploy, and observe production.

## Validation

- `go test ./... -count=1`: passed.
- `go test -race ./internal/repository -run TestHTTPUpstreamSuite -count=1`:
  passed.
- `CGO_ENABLED=0 go build -trimpath -ldflags '-s -w' -o /tmp/sub2api-compact-timeout ./cmd/server`:
  passed.
- Focused configuration, repository, and OpenAI transport-error tests: passed.
- `pnpm exec vitest run src/__tests__/integration/data-import.spec.ts`: 7 tests
  passed after preserving custom import defaults across the upstream UI merge.
- `pnpm run build`: passed; the existing large-chunk warning remains non-blocking.
- All four Docker Compose files parse with `docker compose config
  --no-interpolate --quiet`.
- `git diff --check`: passed.

The upstream merge adopted the official differentiated GPT-5.6 Sol, Terra,
and Luna pricing introduced by `v0.1.153`; obsolete custom tests that expected
all three variants to share GPT-5.4 pricing were updated or removed.

## Deployment Boundary

The user authorized integrating, pushing, and deploying this change. Production
currently sets `GATEWAY_OPENAI_RESPONSE_HEADER_TIMEOUT=300`. The split-timeout
rollout changes it to `120`, adds
`GATEWAY_OPENAI_COMPACT_RESPONSE_HEADER_TIMEOUT=300`, and recreates only the
application container. PostgreSQL backup, rollback image tagging, commit/health
verification, and post-deploy log observation remain required before marking
the rollout complete.

## Deployment Result

- Pushed `anonym/custom` through commit `cc3abbe0` and deployed Sub2API
  `0.1.153` from image `sub2api-local:gpt56-cache-billing-rc1` (image ID
  `sha256:87ff2c47d56df213438ddc8f4040ce8e1e4649ebc063b2c16454543154d0d43c`).
- Created database backup
  `/opt/sub2api/backups/pre-deploy-20260713131814.sql.gz` and retained rollback
  image `sub2api-local:rollback-20260713131814` before replacement.
- Recreated only the `sub2api` service. PostgreSQL and Redis container IDs
  remained unchanged and all three services were healthy after deployment.
- Set and verified the running container environment values
  `GATEWAY_OPENAI_RESPONSE_HEADER_TIMEOUT=120` and
  `GATEWAY_OPENAI_COMPACT_RESPONSE_HEADER_TIMEOUT=300`. The production-owned
  `/opt/sub2api/docker-compose.yml` initially lacked the new compact variable,
  so its environment passthrough was added without replacing the rest of the
  server-specific Compose configuration.
- Verified `/health` returned HTTP 200 and the running binary reported commit
  `cc3abbe0`.
- Observed production from the final container start at
  `2026-07-13T13:28:29Z` through `2026-07-13T13:31:42Z`. During this short
  window, 15 requests completed, including 11 `/v1/responses` requests. The
  maximum completed latency was 34,296 ms; no 5xx response, response-header
  timeout, failover switch, panic, fatal error, or migration error appeared.
  No `/responses/compact` production request occurred in this window, so its
  production evidence is limited to configuration verification and the
  automated transport/configuration tests listed above.
- Pruned build cache after verification, reclaimed approximately 4.85 GB, and
  retained the rollback image and data volumes.
