# OpenAI Tool Output Guard Execution Plan

Related requirement: `docs/product-specs/openai-tool-output-guard.md`.

## Status

- [x] Confirm integration branch and create isolated task worktree.
- [x] Capture the product and rollout decisions.
- [x] Add validated gateway configuration with safe defaults.
- [x] Implement deterministic plain-text tool-output transformation.
- [x] Integrate after moderation and before model mapping/upstream forwarding.
- [x] Add focused unit, handler, configuration, and regression tests.
- [x] Validate the backend and prepare deployment settings.

## Validation

- `go test ./... -count=1`: passed.
- `CGO_ENABLED=0 go build -trimpath -o /private/tmp/sub2api-tool-output-guard-final ./cmd/server`: passed.
- Focused config, service, and handler tests: passed.
- All four Docker Compose files expand successfully with required placeholder
  deployment variables.
- `git diff --check`: passed.
- `golangci-lint run ./...`: not run because `golangci-lint` is not installed
  on the development machine.

## Remaining Operations

- Production deployment is not performed by this implementation task.
- First production configuration must use `mode=observe` and API key ID `1`.
- Review at least 24 hours of `openai.tool_output_guard_inspected` logs before
  changing to `mode=enforce`.

## Deployment Boundary

This task may implement and validate the change locally. Pushing, deploying,
changing production environment variables, or restarting production requires a
separate explicit confirmation.
