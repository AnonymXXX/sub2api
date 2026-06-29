# Rollback Before APIPod

## Goal

Restore code behavior to `296969fb` (`build: 支持配置前端构建内存`) without rewriting Git history.

## Preserved Rules

- Keep `docs/engineering-rules/self-hosted-docker-deploy.md`.
- Keep `docs/engineering-rules/openai-oauth-codex.md` with local OpenAI default concurrency `2`.
- Keep the deployment rule link from `openai-oauth-codex.md` to `self-hosted-docker-deploy.md`.

## Removed Scope

- Remove APIPod relay integration UI and quota display changes.
- Remove routing audit APIs, UI, migration, repository, service, and privacy precheck code added after `296969fb`.
- Remove APIPod/routing-audit product and execution documents that no longer match the rollback target.

## Validation Plan

- `git diff --check`
- `pnpm --dir frontend install --frozen-lockfile`
- `pnpm --dir frontend run typecheck`
- `pnpm --dir frontend run build`
- `cd backend && go test ./internal/service ./internal/handler/dto ./internal/handler/admin ./internal/server/routes`

## Deployment Plan

- Merge the rollback commit into `anonym/custom`.
- Push `origin/anonym/custom`.
- Deploy using `docs/engineering-rules/self-hosted-docker-deploy.md`.
- Do not drop already-created database tables during routine rollback.
