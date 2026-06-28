# OpenAI OAuth Codex Upstream Rules

## ChatGPT internal unsupported parameters

OpenAI OAuth accounts forward Codex requests to ChatGPT's internal Codex endpoint (`/backend-api/codex/responses`), not the public OpenAI Responses API. This upstream rejects public Responses API output limit fields such as `max_output_tokens` with `Unsupported parameter: max_output_tokens`.

Do not preserve or inject `max_output_tokens` for normal OpenAI OAuth Codex traffic. The field must stay in `openAICodexOAuthUnsupportedFields` and be stripped before forwarding. If output length control is needed for OAuth accounts, use a different mechanism such as prompt policy, a compatible upstream/account type, or a protocol-aware local stream limiter.

## Output length control

Prefer server-side Codex instruction policy before protocol-level truncation. `gateway.forced_codex_instructions_template_file` can prepend a reusable policy to OAuth Codex `instructions`; both Anthropic Messages compatibility and native `/v1/responses` OAuth forwarding must honor the same template.

Use the template for soft limits such as asking Codex to keep ordinary responses concise, split long work into phases, and avoid dumping large generated files or logs unless explicitly requested. This does not guarantee a hard token cap, but it avoids upstream incompatibility and preserves Responses SSE semantics, usage parsing, and tool-call continuity.

Only add a local stream limiter if a hard cap is truly required. A limiter must be Responses-protocol-aware: it must not cut JSON/SSE frames mid-event, must emit a valid terminal event, and must preserve billing/usage accounting behavior.

## Default OpenAI account whitelist

New OpenAI accounts should default to the local Codex/Plus model whitelist:

- `gpt-5.5`
- `codex-auto-review`
- `gpt-5.4`
- `gpt-5.4-mini`

Do not default new OpenAI accounts to every OpenAI model. In this LAN deployment, the default must avoid image and non-Codex models so ordinary traffic cannot silently spend Plus account quota outside the intended Codex pool. Users can still edit the whitelist manually when a specific account is intentionally allowed to serve other models.

## Local Codex pool defaults

For the local LAN deployment, new OpenAI OAuth accounts should use conservative Codex-first defaults:

- default account concurrency: `5`
- default group: the active OpenAI group named `openai`
- default client policy: Codex CLI only enabled
- image generation disabled at the group level

The UI must not hard-code the group ID. Resolve the default group by `platform=openai` and `name=openai`, because local databases can assign different IDs after migrations or restores.

Validation:

```bash
cd backend && go test ./internal/service -run 'TestApplyCodexOAuthTransform_StripsMaxOutputTokens|TestOpenAIGatewayService_OAuthResponsesStripsMaxOutputTokens|TestOpenAIGatewayService_OAuthResponsesAppliesForcedCodexInstructionsTemplate'
cd frontend && pnpm test:run src/composables/__tests__/useModelWhitelist.spec.ts src/components/account/__tests__/CreateAccountModal.spec.ts
```
