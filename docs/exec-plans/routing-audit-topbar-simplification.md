# Routing Audit Topbar Simplification

## Source Requirement

- Product spec: `docs/product-specs/codex-hybrid-routing-policy.md`
- User-visible decision: the admin routing audit page starts with summary metric cards, then a compact time-range selector. It does not show a duplicate in-page title, subtitle, manual refresh button, or broad filter form.

## Scope

- Simplify `frontend/src/views/admin/RoutingAuditView.vue`.
- Keep chart-first dashboard review and request detail pagination.
- Keep API requests limited to time range, sort order, and pagination from this page.

## Implementation Status

- [x] Remove duplicate page title, description, and refresh action.
- [x] Remove broad dimension filter form.
- [x] Add compact time-range selector below summary metric cards.
- [x] Update product spec with the durable page-top behavior.

## Validation Status

- [x] `pnpm --dir frontend install --frozen-lockfile`
- [x] `pnpm --dir frontend run typecheck`
- [x] `pnpm --dir frontend exec eslint src/views/admin/RoutingAuditView.vue`
- [x] `pnpm --dir frontend run build`
- [x] `git diff --check`

## Release Status

- [x] Commit feature branch `anonym/routing-audit-topbar` (`a6d79fa4`).
- [x] Merge into deployment branch `anonym/custom`.
- [ ] Push `anonym/custom`.
- [ ] Deploy from `anonym/custom` using `docs/engineering-rules/self-hosted-docker-deploy.md`.
- [ ] Verify health and deployed commit.
