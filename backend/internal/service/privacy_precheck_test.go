package service

import (
	"context"
	"strings"
	"testing"
)

func TestEvaluatePrivacyPrecheckDetectsCredentialWithoutLeakingValue(t *testing.T) {
	body := []byte(`{"model":"gpt-5.5","input":[{"role":"user","content":[{"type":"input_text","text":"connect with password=SuperSecret123 and api_key=sample-token-value"}]}]}`)

	result := EvaluatePrivacyPrecheck(ContentModerationProtocolOpenAIResponses, body)
	if !result.Sensitive {
		t.Fatalf("expected sensitive precheck hit")
	}
	if result.Decision == "" || result.Decision == PrivacyPrecheckDecisionPass {
		t.Fatalf("decision = %q, want sensitive decision", result.Decision)
	}
	if strings.Contains(result.Decision, "SuperSecret") || strings.Contains(result.Decision, "sample-token") {
		t.Fatalf("decision leaked secret value: %q", result.Decision)
	}
	if len(result.AllowedPools) != 1 || result.AllowedPools[0] != RoutingAuditPoolTrustedPlus {
		t.Fatalf("allowed pools = %#v, want trusted-plus only", result.AllowedPools)
	}
}

func TestEvaluatePrivacyPrecheckScansSystemAndDeveloperText(t *testing.T) {
	body := []byte(`{"model":"gpt-5.5","instructions":"use Authorization: Bearer abcdefghijklmnopqrstuvwxyz123456","input":[{"role":"user","content":[{"type":"input_text","text":"hello"}]}]}`)

	result := EvaluatePrivacyPrecheck(ContentModerationProtocolOpenAIResponses, body)
	if !result.Sensitive {
		t.Fatalf("expected sensitive precheck hit from instructions")
	}
	if strings.Contains(result.Decision, "abcdefghijklmnopqrstuvwxyz") {
		t.Fatalf("decision leaked bearer token: %q", result.Decision)
	}
}

func TestEvaluatePrivacyPrecheckEnvVarReferencesDoNotRedirect(t *testing.T) {
	body := []byte(`{"model":"gpt-5.5","input":[{"role":"user","content":[{"type":"input_text","text":"const apiKey = process.env.OPENAI_API_KEY\npassword = os.Getenv(\"SSH_PASSWORD\")\ntoken = os.environ.get(\"API_TOKEN\")\nString secret = System.getenv(\"CLIENT_SECRET\");\nvar pwd = Environment.GetEnvironmentVariable(\"DB_PASSWORD\");\nlet token = std::env::var(\"API_TOKEN\")?;\napi_key=$OPENAI_API_KEY\nconst token = import.meta.env.VITE_TOKEN"}]}]}`)

	result := EvaluatePrivacyPrecheck(ContentModerationProtocolOpenAIResponses, body)
	if result.Sensitive {
		t.Fatalf("env var references should not be treated as leaked secrets: %+v", result)
	}
	if result.Decision != PrivacyPrecheckDecisionPass {
		t.Fatalf("decision = %q, want pass", result.Decision)
	}
}

func TestEvaluatePrivacyPrecheckEnvFileValuesRedirect(t *testing.T) {
	body := []byte(`{"model":"gpt-5.5","input":[{"role":"user","content":[{"type":"input_text","text":"OPENAI_API_KEY=sample-token-value\nDATABASE_URL=postgres://app:SecretPass123@db.local/app"}]}]}`)

	result := EvaluatePrivacyPrecheck(ContentModerationProtocolOpenAIResponses, body)
	if !result.Sensitive {
		t.Fatalf("actual env secret values should trigger privacy routing")
	}
	if strings.Contains(result.Decision, "SecretPass") || strings.Contains(result.Decision, "sample-token") {
		t.Fatalf("decision leaked env secret value: %q", result.Decision)
	}
}

func TestPrivacyRoutingContextAllowsTrustedPlusOnly(t *testing.T) {
	ctx := WithPrivacyRoutingAllowedPools(context.Background(), []string{RoutingAuditPoolTrustedPlus})

	plus := &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	if !IsAccountAllowedByPrivacyRoutingContext(ctx, plus) {
		t.Fatalf("trusted plus account should be allowed")
	}

	relay := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url": "https://relay.example.test/v1",
		},
	}
	if IsAccountAllowedByPrivacyRoutingContext(ctx, relay) {
		t.Fatalf("relay account should be rejected by privacy routing context")
	}
}
