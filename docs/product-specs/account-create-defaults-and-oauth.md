# Account Creation Defaults And OpenAI OAuth

## Decision

The add-account form defaults account concurrency to `3`. Load factor remains
unset so its existing placeholder and backend fallback follow concurrency;
therefore both controls initially display `3` without replacing the current
linked-default behavior.

When an administrator advances from step 1 for an OpenAI OAuth account, the
form immediately generates the authorization URL for step 2's default Manual
authorization method and copies it to the clipboard. Step 2 still exposes the
existing Generate authorization URL button so generation or copying can be
retried.

Changing to refresh-token, Codex JSON, or personal access token input after
step 2 opens does not generate another URL. Automatic generation does not run
for API-key or non-OpenAI account flows. A generation or clipboard failure does
not send credentials or create an account and remains visible through the
existing error handling.

## Success Criteria

- A newly opened add-account form displays `3` for Concurrency and Load factor.
- Resetting the form restores the same defaults.
- Advancing an OpenAI OAuth flow generates one authorization URL and copies
  the generated URL once.
- The form reaches step 2 after the automatic attempt, where the administrator
  can inspect the state and retry with the existing button.
- Other account and authorization flows keep their existing behavior.

## Assumptions And Open Questions

- Load factor remains unset until explicitly edited; its displayed default is
  derived from concurrency as before.
- No product questions remain for this change.

## Status And Links

- Status: deployed on 2026-07-27
- Implementation:
  `frontend/src/components/account/CreateAccountModal.vue`
- Regression coverage:
  `frontend/src/components/account/__tests__/CreateAccountModal.spec.ts`

## Deployment Record

- Production image: `sub2api-local:v0.1.153-h4`
- Built commit: `a6ed55aa`
- PostgreSQL backup:
  `/opt/sub2api/backups/pre-deploy-20260727063902.sql.gz`
- Compose override backup:
  `/opt/sub2api/backups/docker-compose.override.pre-deploy-20260727063902.yml`
- Rollback image: `sub2api-local:rollback-20260727063902`
- Verification: application healthy with zero restarts; PostgreSQL and Redis
  remained healthy and retained their pre-deployment container IDs.
