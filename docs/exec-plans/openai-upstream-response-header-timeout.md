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

## Validation

- `go test ./... -count=1`: passed.
- `go test -race ./internal/repository -run TestHTTPUpstreamSuite -count=1`:
  passed.
- `CGO_ENABLED=0 go build -trimpath -o /tmp/sub2api-openai-header-timeout ./cmd/server`:
  passed.
- Focused configuration, repository, and OpenAI transport-error tests: passed.
- All four Docker Compose files parse with `docker compose config
  --no-interpolate --quiet`.
- `git diff --check`: passed.

## Deployment Boundary

This task implements and validates the change locally. It does not push,
deploy, edit the production environment, or restart the production service.
Production currently sets `GATEWAY_OPENAI_RESPONSE_HEADER_TIMEOUT=0`, so rollout
requires changing that value to `300` and recreating the application container
after the code change is integrated.
