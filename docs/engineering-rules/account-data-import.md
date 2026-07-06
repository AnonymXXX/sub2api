# Account Data Import Rules

## OpenAI Account Identity

For account data import, an existing OpenAI account must be matched by normalized email within the same `platform` and `type`.

- Normalize email by trimming spaces and lowercasing.
- Read email from `credentials.email` first, then `extra.email`.
- Do not use `chatgpt_account_id` or `account_id` as the import upsert identity. Some K12 accounts can share the same account ID while representing different mailbox accounts.
- If multiple existing rows share the same `platform:type:email`, treat the import as ambiguous and fail that account instead of updating an arbitrary row.
