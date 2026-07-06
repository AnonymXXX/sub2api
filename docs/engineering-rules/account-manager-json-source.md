# Account Manager JSON Source

- `chatgpt-codex-account-manager` is only a JSON source for Sub2API API credentials.
- Sub2API synchronization from Account Manager must consume only the remote API Credential `sub2api-data` export. It must not import Account Manager password accounts, phone numbers, bindings, verification sources, or other manager-only records.
- The existing local data import remains create-only. Remote Account Manager synchronization owns the update-or-create behavior and must classify existing accounts by stable credential identity before writing.
- Remote sync endpoints must keep network access behind the same URL validation posture used by other admin-side remote fetch features, and must not log remote passwords or exported OAuth tokens.
