package service

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	OpenAIToolOutputGuardModeOff     = "off"
	OpenAIToolOutputGuardModeObserve = "observe"
	OpenAIToolOutputGuardModeEnforce = "enforce"

	toolOutputGuardMarkerPrefix = "\n[Sub2API tool output guard: omitted "
)

// OpenAIToolOutputGuardConfig is the service-level policy used on the request
// hot path. It intentionally has no dependency on the config package.
type OpenAIToolOutputGuardConfig struct {
	Mode               string
	APIKeyIDs          []int64
	RequireCodexClient bool
	MinChars           int
	HeadChars          int
	TailChars          int
}

func (c OpenAIToolOutputGuardConfig) Allows(apiKeyID int64, isCodexClient bool) bool {
	mode := strings.ToLower(strings.TrimSpace(c.Mode))
	if mode != OpenAIToolOutputGuardModeObserve && mode != OpenAIToolOutputGuardModeEnforce {
		return false
	}
	if c.RequireCodexClient && !isCodexClient {
		return false
	}
	for _, allowedID := range c.APIKeyIDs {
		if apiKeyID == allowedID {
			return true
		}
	}
	return false
}

type OpenAIToolOutputGuardStats struct {
	Changed              bool
	CandidateCount       int
	TransformedCount     int
	OriginalChars        int
	ForwardedChars       int
	SavedChars           int
	BypassBelowThreshold int
	BypassNonString      int
	BypassBinaryLike     int
}

func (s OpenAIToolOutputGuardStats) InspectedCount() int {
	return s.CandidateCount + s.BypassBelowThreshold + s.BypassNonString + s.BypassBinaryLike
}

// ApplyOpenAIToolOutputGuard returns the original byte slice unless enforce
// mode changes at least one eligible output. Targeted sjson replacement avoids
// re-encoding unrelated request fields and preserves the prompt-cache prefix.
func ApplyOpenAIToolOutputGuard(body []byte, cfg OpenAIToolOutputGuardConfig) ([]byte, OpenAIToolOutputGuardStats, error) {
	stats := OpenAIToolOutputGuardStats{}
	mode := strings.ToLower(strings.TrimSpace(cfg.Mode))
	if mode != OpenAIToolOutputGuardModeObserve && mode != OpenAIToolOutputGuardModeEnforce {
		return body, stats, nil
	}

	input := gjson.GetBytes(body, "input")
	if !input.IsArray() {
		return body, stats, nil
	}

	result := body
	var transformErr error
	index := 0
	input.ForEach(func(_, item gjson.Result) bool {
		defer func() { index++ }()
		if !item.IsObject() {
			return true
		}
		typ := strings.TrimSpace(item.Get("type").String())
		if typ != "function_call_output" && typ != "custom_tool_call_output" {
			return true
		}
		output := item.Get("output")
		if output.Type != gjson.String {
			stats.BypassNonString++
			return true
		}
		text := output.String()
		charCount := utf8.RuneCountInString(text)
		if charCount <= cfg.MinChars || strings.Contains(text, toolOutputGuardMarkerPrefix) {
			stats.BypassBelowThreshold++
			return true
		}
		if looksLikeBinaryToolOutput(text) {
			stats.BypassBinaryLike++
			return true
		}

		filtered := filterToolOutput(text, cfg.HeadChars, cfg.TailChars)
		forwardedChars := utf8.RuneCountInString(filtered)
		if forwardedChars >= charCount {
			stats.BypassBelowThreshold++
			return true
		}
		stats.CandidateCount++
		stats.TransformedCount++
		stats.OriginalChars += charCount
		stats.ForwardedChars += forwardedChars
		stats.SavedChars += charCount - forwardedChars
		if mode != OpenAIToolOutputGuardModeEnforce {
			return true
		}

		result, transformErr = sjson.SetBytes(result, fmt.Sprintf("input.%d.output", index), filtered)
		if transformErr != nil {
			return false
		}
		stats.Changed = true
		return true
	})
	if transformErr != nil {
		return body, OpenAIToolOutputGuardStats{}, transformErr
	}
	return result, stats, nil
}

func filterToolOutput(text string, headChars, tailChars int) string {
	runes := []rune(text)
	omitted := len(runes) - headChars - tailChars
	digest := sha256.Sum256([]byte(text))
	marker := fmt.Sprintf("%s%d chars; sha256=%x]\n", toolOutputGuardMarkerPrefix, omitted, digest)
	return string(runes[:headChars]) + marker + string(runes[len(runes)-tailChars:])
}

func looksLikeBinaryToolOutput(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if strings.Contains(lower, "data:image/") ||
		strings.Contains(lower, "data:application/octet-stream") ||
		strings.Contains(lower, `"type":"input_image"`) ||
		strings.Contains(lower, `"type": "input_image"`) {
		return true
	}
	if strings.IndexByte(text, 0) >= 0 || !utf8.ValidString(text) {
		return true
	}
	return false
}
