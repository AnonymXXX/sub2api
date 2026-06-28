package service

import (
	"context"
	"encoding/json"
	"regexp"
	"sort"
	"strings"
)

const (
	PrivacyPrecheckDecisionPass      = "rule_pass"
	PrivacyPrecheckDecisionSensitive = "rule_sensitive"
	PrivacyPrecheckTrustedPlusPool   = RoutingAuditPoolTrustedPlus
)

type privacyRoutingPoolFilterCtxKey struct{}

type PrivacyPrecheckResult struct {
	Sensitive    bool
	Decision     string
	Categories   []string
	AllowedPools []string
}

type privacyPrecheckPattern struct {
	category string
	pattern  *regexp.Regexp
	validate func(string) bool
}

var privacyPrecheckPatterns = []privacyPrecheckPattern{
	{"private_key", regexp.MustCompile(`(?i)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----`), nil},
	{"openai_api_key", regexp.MustCompile(`(?i)\bsk-(?:proj-|ant-)?[A-Za-z0-9._-]{16,}\b`), nil},
	{"bearer_token", regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]{24,}`), nil},
	{"jwt", regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\b`), nil},
	{"connection_uri_password", regexp.MustCompile(`(?i)\b(?:postgres(?:ql)?|mysql|redis|mongodb(?:\+srv)?|mongo|amqp|ssh)://[^/\s:@]+:[^@\s/]+@[^"'\s,;，。；、]+`), nil},
	{"env_secret_assignment", regexp.MustCompile(`(?m)\b(?:export\s+)?(?:[A-Z][A-Z0-9_]*(?:API_KEY|APIKEY|TOKEN|SECRET|PASSWORD|PASSWD|PWD|PRIVATE_KEY|COOKIE)[A-Z0-9_]*|DATABASE_URL|REDIS_URL|MONGO_URI|MONGODB_URI|POSTGRES_URL|POSTGRESQL_URL|MYSQL_URL|AWS_SECRET_ACCESS_KEY|AWS_SESSION_TOKEN)\s*[:=]\s*["']?[^"'\s,;，。；、]{6,}`), privacyPrecheckAssignmentHasActualSecretValue},
	{"credential_assignment", regexp.MustCompile(`(?i)\b(?:api[_-]?key|apikey|access[_-]?token|refresh[_-]?token|id[_-]?token|session[_-]?token|authorization|cookie|set[_-]?cookie|password|passwd|pwd|secret|client[_-]?secret|private[_-]?key|aws[_-]?secret[_-]?access[_-]?key|aws[_-]?session[_-]?token)\s*[:=]\s*["']?[^"'\s,;，。；、]{6,}`), privacyPrecheckAssignmentHasActualSecretValue},
	{"sshpass", regexp.MustCompile(`(?i)\bsshpass\s+-p\s+["']?[^"'\s]{3,}`), nil},
	{"ssh_password_uri", regexp.MustCompile(`(?i)\bssh://[^/\s:@]+:[^@\s/]{3,}@[^/\s]+`), nil},
	{"long_secret", regexp.MustCompile(`\b[A-Za-z0-9_-]{64,}\b`), nil},
}

func EvaluatePrivacyPrecheck(protocol string, body []byte) PrivacyPrecheckResult {
	text := extractPrivacyPrecheckText(protocol, body)
	if text == "" {
		return PrivacyPrecheckResult{Decision: PrivacyPrecheckDecisionPass}
	}
	categories := make(map[string]struct{})
	for _, item := range privacyPrecheckPatterns {
		matches := item.pattern.FindAllString(text, -1)
		for _, match := range matches {
			if item.validate != nil && !item.validate(match) {
				continue
			}
			categories[item.category] = struct{}{}
			break
		}
	}
	if len(categories) == 0 {
		return PrivacyPrecheckResult{Decision: PrivacyPrecheckDecisionPass}
	}
	out := make([]string, 0, len(categories))
	for category := range categories {
		out = append(out, category)
	}
	sort.Strings(out)
	return PrivacyPrecheckResult{
		Sensitive:    true,
		Decision:     PrivacyPrecheckDecisionSensitive + ":" + strings.Join(out, ","),
		Categories:   out,
		AllowedPools: []string{PrivacyPrecheckTrustedPlusPool},
	}
}

func extractPrivacyPrecheckText(protocol string, body []byte) string {
	content := ExtractContentModerationInput(protocol, body)
	parts := []string{content.Text}

	var decoded any
	if err := json.Unmarshal(body, &decoded); err == nil {
		collectPrivacyPrecheckStrings(decoded, &parts, 0)
	}

	text := strings.TrimSpace(strings.Join(parts, "\n"))
	if len(text) > 128*1024 {
		text = text[:128*1024]
	}
	return text
}

func collectPrivacyPrecheckStrings(value any, parts *[]string, depth int) {
	if depth > 16 {
		return
	}
	switch v := value.(type) {
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed != "" && !looksLikeInlineImageData(trimmed) {
			*parts = append(*parts, trimmed)
		}
	case []any:
		for _, item := range v {
			collectPrivacyPrecheckStrings(item, parts, depth+1)
		}
	case map[string]any:
		for key, item := range v {
			if shouldSkipPrivacyPrecheckJSONKey(key) {
				continue
			}
			collectPrivacyPrecheckStrings(item, parts, depth+1)
		}
	}
}

func shouldSkipPrivacyPrecheckJSONKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "image_url", "images", "data", "base64", "b64_json", "inline_data", "inlinedata":
		return true
	default:
		return false
	}
}

func looksLikeInlineImageData(value string) bool {
	lower := strings.ToLower(value)
	return strings.HasPrefix(lower, "data:image/") || strings.HasPrefix(lower, "data:application/octet-stream")
}

func privacyPrecheckAssignmentHasActualSecretValue(match string) bool {
	key, value, ok := privacyPrecheckAssignmentParts(match)
	if !ok {
		return false
	}
	value = strings.TrimSpace(value)
	if value == "" || privacyPrecheckValueIsReferenceOrPlaceholder(value) {
		return false
	}
	lowerKey := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "-", "_"), " ", ""))
	if strings.Contains(lowerKey, "database_url") || strings.Contains(lowerKey, "postgres") ||
		strings.Contains(lowerKey, "mysql") || strings.Contains(lowerKey, "redis") ||
		strings.Contains(lowerKey, "mongo") {
		return regexp.MustCompile(`(?i)^[a-z][a-z0-9+.-]*://[^/\s:@]+:[^@\s/]+@`).MatchString(value)
	}
	return true
}

func privacyPrecheckAssignmentParts(match string) (string, string, bool) {
	match = strings.TrimSpace(match)
	sep := strings.IndexAny(match, "=:")
	if sep <= 0 || sep >= len(match)-1 {
		return "", "", false
	}
	key := strings.TrimSpace(match[:sep])
	fields := strings.Fields(key)
	if len(fields) > 0 {
		key = fields[len(fields)-1]
	}
	value := strings.TrimSpace(match[sep+1:])
	value = strings.Trim(value, "\"'`")
	value = strings.TrimRight(value, ",;")
	return key, value, key != "" && value != ""
}

func privacyPrecheckValueIsReferenceOrPlaceholder(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return true
	}
	unquoted := strings.Trim(trimmed, "\"'`")
	lower := strings.ToLower(unquoted)
	switch lower {
	case "bearer", "basic", "none", "null", "nil", "undefined", "true", "false",
		"password", "passwd", "secret", "token", "api_key", "apikey", "key":
		return true
	}
	if strings.HasPrefix(unquoted, "$") ||
		regexp.MustCompile(`^\$\{[A-Z][A-Z0-9_]*\}$`).MatchString(unquoted) ||
		regexp.MustCompile(`^[A-Z][A-Z0-9_]{2,}$`).MatchString(unquoted) {
		return true
	}
	referenceMarkers := []string{
		"process.env", "import.meta.env", "deno.env", "bun.env",
		"os.getenv", "os.environ", "os.environ.get", "os.environ[",
		"system.getenv", "environment.getenvironmentvariable", "env::var", "std::env",
		"getenv(", "env.", "env[", "$env:", "config.", "configuration[", "settings.",
		"localstorage.", "sessionstorage.", "secretkeyref", "valuefrom", "configmapkeyref",
	}
	for _, marker := range referenceMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	if strings.Contains(unquoted, "(") || strings.Contains(unquoted, ")") {
		return true
	}
	placeholderMarkers := []string{
		"your_", "your-", "example", "placeholder", "changeme", "change_me",
		"dummy", "sample", "todo",
	}
	for _, marker := range placeholderMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	if regexp.MustCompile(`^[xX*._-]+$`).MatchString(unquoted) {
		return true
	}
	return false
}

func WithPrivacyRoutingAllowedPools(ctx context.Context, pools []string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	allowed := normalizePrivacyRoutingPools(pools)
	if len(allowed) == 0 {
		return ctx
	}
	return context.WithValue(ctx, privacyRoutingPoolFilterCtxKey{}, allowed)
}

func privacyRoutingAllowedPoolsFromContext(ctx context.Context) map[string]struct{} {
	if ctx == nil {
		return nil
	}
	pools, _ := ctx.Value(privacyRoutingPoolFilterCtxKey{}).(map[string]struct{})
	if len(pools) == 0 {
		return nil
	}
	return pools
}

func IsAccountAllowedByPrivacyRoutingContext(ctx context.Context, account *Account) bool {
	allowed := privacyRoutingAllowedPoolsFromContext(ctx)
	if len(allowed) == 0 {
		return true
	}
	if account == nil {
		return false
	}
	pool := classifyRoutingAuditAccountPool(account)
	if pool == "" {
		pool = RoutingAuditPoolUnknown
	}
	_, ok := allowed[pool]
	return ok
}

func normalizePrivacyRoutingPools(pools []string) map[string]struct{} {
	out := make(map[string]struct{}, len(pools))
	for _, pool := range pools {
		pool = strings.TrimSpace(pool)
		if pool == "" {
			continue
		}
		out[pool] = struct{}{}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
