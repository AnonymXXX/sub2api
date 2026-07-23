# Unified Account Disabled State

## Decision

The admin account list exposes one disabled filter. An account belongs to this
filter when its persisted status is `disabled` or legacy `inactive`, or when
its scheduling switch is off (`schedulable = false`). The former standalone
unschedulable filter is removed. Operational categories may overlap: a
rate-limited or errored account with scheduling disabled remains visible in
both its operational category and the disabled filter.

Manual account controls keep status and scheduling aligned. Disabling an
account or turning scheduling off writes `disabled` and `false`; enabling it or
turning scheduling on writes `active` and `true`; setting an account to error
writes `error` and `false`. Automatic rate-limit and expiry workflows may
continue to represent temporary conditions as `active` with scheduling off.

`disabled` is the canonical status for new writes. The API continues to accept
legacy `inactive` input and normalizes it to `disabled`; legacy `inactive` and
`unschedulable` list filters resolve to the unified disabled filter. Requests
that explicitly submit conflicting status and scheduling values are rejected
with HTTP 400.

The user usage page defaults to the current local calendar day, including
after resetting filters. The admin usage page already follows this behavior
and remains unchanged.

## Success Criteria

- Every account whose scheduling switch is off appears under Disabled.
- Disabled can overlap with rate-limited, expired, or error classifications.
- All manual status, scheduling, and bulk-edit entry points enforce the same
  paired values.
- Existing clients using `inactive` or `unschedulable` remain compatible.
- The user usage page initially requests and resets to today's date with hourly
  granularity.

## Assumptions And Open Questions

- Production currently contains only `active` accounts, so no data migration
  is required.
- Temporary system-driven scheduling suppression is not rewritten to a manual
  disabled status.
- No open product questions remain for this change.

## Status And Links

- Status: confirmed
- Related execution plan: `docs/exec-plans/account-disabled-filter.md`
- Supersedes: separate Disabled and Unschedulable admin list filters, and the
  user usage page's last-24-hours default
