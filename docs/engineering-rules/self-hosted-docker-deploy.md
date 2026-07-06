# Self-Hosted Docker Deploy Rules

Use a dedicated long-lived custom branch for this self-hosted deployment. Do not deploy from one-off feature branches after validation; merge or cherry-pick the intended commits into the custom branch first.

Current custom branch:

- `anonym/custom`

## Safety rules

- Never print `.env`, config secrets, database rows, OAuth tokens, API keys, cookies, SSH passwords, or account credentials during deployment.
- Take a PostgreSQL backup before replacing the application container.
- Rebuild only the application image and recreate only the application container unless the change explicitly requires database or Redis maintenance.
- Keep PostgreSQL and Redis containers and their data volumes running during ordinary application updates.
- Verify the exact commit built into the image before considering deployment complete.
- After a successful deployment and health check, prune Docker build cache and dangling images to keep the RackNerd disk healthy.
- Do not run `docker system prune --volumes` during routine deployment cleanup.
- Do not delete `/opt/sub2api/backups`, PostgreSQL/Redis volumes, or tagged rollback images as part of routine deployment cleanup.

## Standard flow

1. Commit and push changes to the custom branch.
2. On the server, update the build checkout to the same custom branch.
3. Build the local application image with the same image tag used by the compose override.
4. Recreate the application container without recreating dependencies.
5. Verify health, commit, migrations, and key routes.
6. Prune Docker build cache and dangling images.
7. Re-check disk usage and container health.

## Server layout

Use placeholders in documentation and scripts. The current production shape is:

- runtime directory: `/opt/sub2api`
- build checkout: `/opt/sub2api-build`
- local app image: `sub2api-local:codex-default-models`
- compose override: runtime directory `docker-compose.override.yml`
- build swap: `/swapfile-sub2api-build` (`4G`, persistent via `/etc/fstab`)

The runtime directory owns `.env`, persistent app data, PostgreSQL data, Redis data, and backups. The build checkout is disposable source code for building the image.

## Commands

Update build checkout:

```bash
cd /opt/sub2api-build
git fetch origin anonym/custom
git checkout -B anonym/custom origin/anonym/custom
git rev-parse HEAD
git status --short
```

Backup database:

```bash
ts=$(date +%Y%m%d%H%M%S)
mkdir -p /opt/sub2api/backups
docker exec sub2api-postgres sh -lc 'PGPASSWORD="$POSTGRES_PASSWORD" pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB"' \
  | gzip > "/opt/sub2api/backups/pre-deploy-${ts}.sql.gz"
ls -lh "/opt/sub2api/backups/pre-deploy-${ts}.sql.gz"
```

Ensure build swap before image build:

```bash
if ! swapon --show=NAME | grep -qx /swapfile-sub2api-build; then
  if [ ! -f /swapfile-sub2api-build ]; then
    fallocate -l 4G /swapfile-sub2api-build || dd if=/dev/zero of=/swapfile-sub2api-build bs=1M count=4096
    chmod 600 /swapfile-sub2api-build
    mkswap /swapfile-sub2api-build
  fi
  swapon /swapfile-sub2api-build
fi
swapon --show=NAME,SIZE,USED
```

Build image:

```bash
cd /opt/sub2api-build
docker build \
  --build-arg COMMIT=$(git rev-parse --short HEAD) \
  --build-arg NODE_OPTIONS=--max-old-space-size=1536 \
  -t sub2api-local:codex-default-models .
```

Do not wait for Go compilation to be killed before adding swap. This RackNerd host is memory constrained for Docker builds, so ensure `/swapfile-sub2api-build` is enabled before every image build. Keep the file persistent unless disk pressure requires removal.

Recreate app container only:

```bash
cd /opt/sub2api
docker compose up -d --no-deps --force-recreate sub2api
```

## Verification

Health and containers:

```bash
curl -fsS http://127.0.0.1:8080/health
docker ps --format '{{.Names}}\t{{.Image}}\t{{.Status}}' | grep -E 'sub2api|postgres|redis'
docker inspect sub2api --format 'health={{.State.Health.Status}} image={{.Config.Image}} started={{.State.StartedAt}}'
```

Image commit:

```bash
docker exec sub2api /app/sub2api --version
```

This command prints the version and may then try to continue normal startup, which can log an expected `address already in use` error because the real server is already running. Treat the printed commit as the verification signal.

Route check example:

```bash
curl -sS -o /tmp/route-check.txt -w '%{http_code}\n' \
  http://127.0.0.1:8080/health
```

The health route should return `200`.

Post-deploy cleanup:

```bash
docker builder prune -af
docker image prune -f
docker system df
df -hT /
docker ps --format '{{.Names}}\t{{.Image}}\t{{.Status}}' | grep -E 'sub2api|postgres|redis'
curl -sS -o /tmp/sub2api-health.txt -w '%{http_code}\n' http://127.0.0.1:8080/health
```

This cleanup intentionally removes only reusable Docker build cache and dangling images. It must not remove volumes, backups, or tagged images that may be useful for rollback.

## Rollback

Use the previous local image or rebuild from the previous known-good commit, then recreate only the application container. Restore the database backup only if a migration or data write must be reverted; do not restore data as a routine code rollback step.
