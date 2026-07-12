# OpenAI Tool Output Guard

## Decision

Sub2API provides a conservative OpenAI Responses API tool-output guard as a
second layer behind client-side RTK. Client-side RTK remains responsible for
command-aware compression. The gateway guard only limits abnormally large
plain-text tool outputs before they are forwarded upstream.

The guard is disabled by default and is rolled out by API key ID. The initial
production candidate is API key ID `1` (`My Codex`).

## Behavior

- Apply only to OpenAI HTTP Responses requests from an official Codex client.
- Inspect `input` array items whose type is `function_call_output` or
  `custom_tool_call_output` and whose `output` is a JSON string.
- Leave arrays, images, binary-looking data, unknown structures, and outputs at
  or below 32,000 Unicode characters unchanged.
- For a longer eligible output, retain the first 12,000 and last 12,000 Unicode
  characters. Replace the middle with a deterministic marker containing the
  omitted character count and the SHA-256 digest of the original output.
- The transformation must be deterministic and idempotent so retries and
  repeated context produce identical upstream bytes.
- Content moderation, request/session identity, and audit hashing use the
  original request. Model mapping and upstream retries use one transformed body.
- Do not store or log tool-output content. Metrics and logs may contain only
  counts, lengths, saved character counts, mode, and bypass reason.
- WebSocket Responses traffic is out of scope for the first release.

## Modes

- `off`: do not analyze or transform requests.
- `observe`: report what would be transformed without changing the request.
- `enforce`: transform eligible output before forwarding upstream.

## Configuration

Configuration is process-level and requires a restart:

```text
GATEWAY_OPENAI_TOOL_OUTPUT_GUARD_MODE=off|observe|enforce
GATEWAY_OPENAI_TOOL_OUTPUT_GUARD_API_KEY_IDS=1
GATEWAY_OPENAI_TOOL_OUTPUT_GUARD_REQUIRE_CODEX_CLIENT=true
GATEWAY_OPENAI_TOOL_OUTPUT_GUARD_MIN_CHARS=32000
GATEWAY_OPENAI_TOOL_OUTPUT_GUARD_HEAD_CHARS=12000
GATEWAY_OPENAI_TOOL_OUTPUT_GUARD_TAIL_CHARS=12000
```

Startup validation rejects unknown modes, non-positive IDs, non-positive
thresholds, and `head_chars + tail_chars >= min_chars`.

## Rollout And Acceptance

Run `observe` for at least 24 hours on API key ID `1`, then review candidate
counts and bypass reasons before enabling `enforce`. Keep the feature when the
combined local RTK and gateway rollout lowers monthly extrapolated cost by at
least 3%, eligible local command adoption reaches 70%, and information-loss
reruns remain below 2%.

Rollback is configuration-only: set the mode to `off` and restart the service.
No database migration or data rollback is required.
