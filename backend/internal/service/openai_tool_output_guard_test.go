package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyOpenAIToolOutputGuardEnforce(t *testing.T) {
	original := strings.Repeat("甲", 20) + strings.Repeat("middle", 200) + strings.Repeat("乙", 20)
	body := mustToolOutputGuardBody(t, []any{
		map[string]any{"type": "function_call_output", "call_id": "call_1", "output": original},
	})
	cfg := OpenAIToolOutputGuardConfig{
		Mode:      OpenAIToolOutputGuardModeEnforce,
		MinChars:  60,
		HeadChars: 10,
		TailChars: 10,
	}

	got, stats, err := ApplyOpenAIToolOutputGuard(body, cfg)
	require.NoError(t, err)
	require.True(t, stats.Changed)
	require.Equal(t, 1, stats.CandidateCount)
	require.Equal(t, 1, stats.TransformedCount)
	require.Equal(t, len([]rune(original)), stats.OriginalChars)
	require.Less(t, stats.ForwardedChars, stats.OriginalChars)

	var payload struct {
		Input []struct {
			Output string `json:"output"`
		} `json:"input"`
	}
	require.NoError(t, json.Unmarshal(got, &payload))
	digest := sha256.Sum256([]byte(original))
	require.Equal(t,
		strings.Repeat("甲", 10)+"\n[Sub2API tool output guard: omitted 1220 chars; sha256="+hex.EncodeToString(digest[:])+"]\n"+strings.Repeat("乙", 10),
		payload.Input[0].Output,
	)

	second, secondStats, err := ApplyOpenAIToolOutputGuard(got, cfg)
	require.NoError(t, err)
	require.Equal(t, got, second)
	require.False(t, secondStats.Changed)
}

func TestApplyOpenAIToolOutputGuardObserveDoesNotChangeBody(t *testing.T) {
	body := mustToolOutputGuardBody(t, []any{
		map[string]any{"type": "custom_tool_call_output", "call_id": "call_1", "output": strings.Repeat("x", 1000)},
	})
	cfg := OpenAIToolOutputGuardConfig{Mode: OpenAIToolOutputGuardModeObserve, MinChars: 20, HeadChars: 5, TailChars: 5}

	got, stats, err := ApplyOpenAIToolOutputGuard(body, cfg)
	require.NoError(t, err)
	require.Equal(t, body, got)
	require.False(t, stats.Changed)
	require.Equal(t, 1, stats.CandidateCount)
	require.Equal(t, 1, stats.TransformedCount)
	require.Greater(t, stats.SavedChars, 0)
}

func TestApplyOpenAIToolOutputGuardBypassesIneligibleOutputs(t *testing.T) {
	body := mustToolOutputGuardBody(t, []any{
		map[string]any{"type": "function_call_output", "output": strings.Repeat("x", 19)},
		map[string]any{"type": "function_call_output", "output": []any{map[string]any{"type": "input_image", "image_url": "data:image/png;base64,AAAA"}}},
		map[string]any{"type": "message", "output": strings.Repeat("y", 100)},
		map[string]any{"type": "function_call_output", "output": "data:image/png;base64," + strings.Repeat("A", 100)},
		map[string]any{"type": "custom_tool_call_output", "output": `[{"type":"input_text","text":"ok"},{"type":"input_image","image_url":"data:image/png;base64,` + strings.Repeat("A", 100) + `"}]`},
	})
	cfg := OpenAIToolOutputGuardConfig{Mode: OpenAIToolOutputGuardModeEnforce, MinChars: 20, HeadChars: 5, TailChars: 5}

	got, stats, err := ApplyOpenAIToolOutputGuard(body, cfg)
	require.NoError(t, err)
	require.Equal(t, body, got)
	require.False(t, stats.Changed)
	require.Equal(t, 1, stats.BypassBelowThreshold)
	require.Equal(t, 1, stats.BypassNonString)
	require.Equal(t, 2, stats.BypassBinaryLike)
	require.Equal(t, 4, stats.InspectedCount())
}

func TestOpenAIToolOutputGuardConfigAllows(t *testing.T) {
	cfg := OpenAIToolOutputGuardConfig{
		Mode:               OpenAIToolOutputGuardModeObserve,
		APIKeyIDs:          []int64{1, 7},
		RequireCodexClient: true,
	}

	require.True(t, cfg.Allows(1, true))
	require.False(t, cfg.Allows(2, true))
	require.False(t, cfg.Allows(1, false))
	cfg.Mode = OpenAIToolOutputGuardModeOff
	require.False(t, cfg.Allows(1, true))
}

func mustToolOutputGuardBody(t *testing.T, input []any) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]any{"model": "gpt-test", "input": input})
	require.NoError(t, err)
	return body
}
