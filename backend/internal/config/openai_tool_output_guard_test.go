package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadDefaultOpenAIToolOutputGuard(t *testing.T) {
	resetViperWithJWTSecret(t)

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "off", cfg.Gateway.OpenAIToolOutputGuard.Mode)
	require.Empty(t, cfg.Gateway.OpenAIToolOutputGuard.APIKeyIDs)
	require.True(t, cfg.Gateway.OpenAIToolOutputGuard.RequireCodexClient)
	require.Equal(t, 32000, cfg.Gateway.OpenAIToolOutputGuard.MinChars)
	require.Equal(t, 12000, cfg.Gateway.OpenAIToolOutputGuard.HeadChars)
	require.Equal(t, 12000, cfg.Gateway.OpenAIToolOutputGuard.TailChars)
}

func TestLoadOpenAIToolOutputGuardFromEnv(t *testing.T) {
	resetViperWithJWTSecret(t)
	t.Setenv("GATEWAY_OPENAI_TOOL_OUTPUT_GUARD_MODE", "observe")
	t.Setenv("GATEWAY_OPENAI_TOOL_OUTPUT_GUARD_API_KEY_IDS", "1, 7")
	t.Setenv("GATEWAY_OPENAI_TOOL_OUTPUT_GUARD_REQUIRE_CODEX_CLIENT", "false")
	t.Setenv("GATEWAY_OPENAI_TOOL_OUTPUT_GUARD_MIN_CHARS", "100")
	t.Setenv("GATEWAY_OPENAI_TOOL_OUTPUT_GUARD_HEAD_CHARS", "30")
	t.Setenv("GATEWAY_OPENAI_TOOL_OUTPUT_GUARD_TAIL_CHARS", "40")

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "observe", cfg.Gateway.OpenAIToolOutputGuard.Mode)
	require.Equal(t, []int64{1, 7}, cfg.Gateway.OpenAIToolOutputGuard.APIKeyIDs)
	require.False(t, cfg.Gateway.OpenAIToolOutputGuard.RequireCodexClient)
	require.Equal(t, 100, cfg.Gateway.OpenAIToolOutputGuard.MinChars)
	require.Equal(t, 30, cfg.Gateway.OpenAIToolOutputGuard.HeadChars)
	require.Equal(t, 40, cfg.Gateway.OpenAIToolOutputGuard.TailChars)
}

func TestLoadOpenAIToolOutputGuardEmptyAPIKeyIDsFromEnv(t *testing.T) {
	resetViperWithJWTSecret(t)
	t.Setenv("GATEWAY_OPENAI_TOOL_OUTPUT_GUARD_API_KEY_IDS", "")

	cfg, err := Load()
	require.NoError(t, err)
	require.Empty(t, cfg.Gateway.OpenAIToolOutputGuard.APIKeyIDs)
}

func TestValidateOpenAIToolOutputGuard(t *testing.T) {
	resetViperWithJWTSecret(t)
	base, err := Load()
	require.NoError(t, err)

	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{name: "mode", mutate: func(c *Config) { c.Gateway.OpenAIToolOutputGuard.Mode = "bad" }, wantErr: "gateway.openai_tool_output_guard.mode"},
		{name: "api key", mutate: func(c *Config) { c.Gateway.OpenAIToolOutputGuard.APIKeyIDs = []int64{1, 0} }, wantErr: "gateway.openai_tool_output_guard.api_key_ids"},
		{name: "minimum", mutate: func(c *Config) { c.Gateway.OpenAIToolOutputGuard.MinChars = 0 }, wantErr: "gateway.openai_tool_output_guard.min_chars"},
		{name: "retained", mutate: func(c *Config) {
			c.Gateway.OpenAIToolOutputGuard.MinChars = 100
			c.Gateway.OpenAIToolOutputGuard.HeadChars = 50
			c.Gateway.OpenAIToolOutputGuard.TailChars = 50
		}, wantErr: "head_chars + tail_chars"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copy := *base
			tt.mutate(&copy)
			err := copy.Validate()
			require.Error(t, err)
			require.True(t, strings.Contains(err.Error(), tt.wantErr), err.Error())
		})
	}
}
