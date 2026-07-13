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
- [ ] Integrate into `anonym/custom`, push, deploy, and observe production.

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
