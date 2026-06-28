//go:build unit

package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAccountConcurrencyCapsGrokOAuthUnlessUnsafe(t *testing.T) {
	t.Setenv(xai.EnvUnsafeAllowHighConcurrency, "")

	require.Equal(t, 1, normalizeAccountConcurrency(PlatformGrok, AccountTypeOAuth, 0))
	require.Equal(t, 1, normalizeAccountConcurrency(PlatformGrok, AccountTypeOAuth, -5))
	require.Equal(t, 1, normalizeAccountConcurrency(PlatformGrok, AccountTypeOAuth, 50))
	require.Equal(t, 2, normalizeAccountConcurrency(PlatformOpenAI, AccountTypeOAuth, 2))
	require.Equal(t, 2, normalizeAccountConcurrency(PlatformGrok, AccountTypeAPIKey, 2))
}

func TestNormalizeAccountConcurrencyDefaultsOpenAIPlusToTwo(t *testing.T) {
	require.Equal(t, openai.DefaultPlusAccountConcurrency, normalizeAccountConcurrency(PlatformOpenAI, AccountTypeOAuth, 0))
	require.Equal(t, openai.DefaultPlusAccountConcurrency, normalizeAccountConcurrency(PlatformOpenAI, AccountTypeSetupToken, -5))
	require.Equal(t, 5, normalizeAccountConcurrency(PlatformOpenAI, AccountTypeOAuth, 5))
	require.Equal(t, 0, normalizeAccountConcurrency(PlatformOpenAI, AccountTypeAPIKey, 0))
}

func TestNormalizeAccountConcurrencyAllowsGrokOAuthUnsafeOverride(t *testing.T) {
	t.Setenv(xai.EnvUnsafeAllowHighConcurrency, "true")

	require.Equal(t, 50, normalizeAccountConcurrency(PlatformGrok, AccountTypeOAuth, 50))
	require.Equal(t, 1, normalizeAccountConcurrency(PlatformGrok, AccountTypeOAuth, 0))
}
