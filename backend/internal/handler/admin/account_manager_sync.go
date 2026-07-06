package admin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
	"github.com/gin-gonic/gin"
)

type AccountManagerSyncRequest struct {
	BaseURL             string   `json:"base_url" binding:"required"`
	Username            string   `json:"username" binding:"required"`
	Password            string   `json:"password" binding:"required"`
	SelectedAccountKeys []string `json:"selected_account_keys"`
}

type AccountManagerPreviewResult struct {
	NewAccounts      []AccountManagerPreviewAccount `json:"new_accounts"`
	ExistingAccounts []AccountManagerPreviewAccount `json:"existing_accounts"`
	Errors           []DataImportError              `json:"errors,omitempty"`
}

type AccountManagerPreviewAccount struct {
	AccountKey string `json:"account_key"`
	AccountID  int64  `json:"account_id,omitempty"`
	Name       string `json:"name"`
	Platform   string `json:"platform"`
	Type       string `json:"type"`
}

type AccountManagerSyncResult struct {
	Created int                      `json:"created"`
	Updated int                      `json:"updated"`
	Skipped int                      `json:"skipped"`
	Failed  int                      `json:"failed"`
	Items   []AccountManagerSyncItem `json:"items"`
	Errors  []DataImportError        `json:"errors,omitempty"`
}

type AccountManagerSyncItem struct {
	AccountKey string `json:"account_key,omitempty"`
	AccountID  int64  `json:"account_id,omitempty"`
	Name       string `json:"name,omitempty"`
	Platform   string `json:"platform,omitempty"`
	Type       string `json:"type,omitempty"`
	Action     string `json:"action"`
	Message    string `json:"message,omitempty"`
}

type accountManagerExportResponse struct {
	Payload DataPayload `json:"payload"`
}

func (h *AccountHandler) PreviewFromAccountManager(c *gin.Context) {
	var req AccountManagerSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	payload, err := fetchAccountManagerData(c.Request.Context(), req, h.cfg)
	if err != nil {
		response.InternalError(c, "Account Manager preview failed: "+err.Error())
		return
	}
	result, err := h.previewAccountManagerData(c.Request.Context(), payload)
	if err != nil {
		response.InternalError(c, "Account Manager preview failed: "+err.Error())
		return
	}
	response.Success(c, result)
}

func (h *AccountHandler) SyncFromAccountManager(c *gin.Context) {
	var req AccountManagerSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	payload, err := fetchAccountManagerData(c.Request.Context(), req, h.cfg)
	if err != nil {
		response.InternalError(c, "Account Manager sync failed: "+err.Error())
		return
	}
	result, err := h.syncAccountManagerData(c.Request.Context(), payload, req.SelectedAccountKeys)
	if err != nil {
		response.InternalError(c, "Account Manager sync failed: "+err.Error())
		return
	}
	response.Success(c, result)
}

func fetchAccountManagerData(ctx context.Context, req AccountManagerSyncRequest, cfg *config.Config) (DataPayload, error) {
	var zero DataPayload
	baseURL, err := normalizeAccountManagerBaseURL(req.BaseURL, cfg)
	if err != nil {
		return zero, err
	}
	username := strings.TrimSpace(req.Username)
	if username == "" || strings.TrimSpace(req.Password) == "" {
		return zero, errors.New("username and password are required")
	}
	client, err := httpclient.GetClient(httpclient.Options{
		Timeout:            20 * time.Second,
		ValidateResolvedIP: cfg == nil || cfg.Security.URLAllowlist.Enabled,
		AllowPrivateHosts:  cfg != nil && cfg.Security.URLAllowlist.AllowPrivateHosts,
	})
	if err != nil {
		return zero, fmt.Errorf("create http client failed: %w", err)
	}
	token, err := accountManagerLogin(ctx, client, baseURL, username, req.Password)
	if err != nil {
		return zero, err
	}
	payload, err := accountManagerExportSub2API(ctx, client, baseURL, token)
	if err != nil {
		return zero, err
	}
	if err := validateDataHeader(payload); err != nil {
		return zero, err
	}
	return payload, nil
}

func normalizeAccountManagerBaseURL(raw string, cfg *config.Config) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("base_url is required")
	}
	if cfg != nil && cfg.Security.URLAllowlist.Enabled {
		return normalizeAccountManagerHTTPSBaseURL(raw, cfg.Security.URLAllowlist.AllowPrivateHosts)
	}
	allowHTTP := false
	if cfg != nil {
		allowHTTP = cfg.Security.URLAllowlist.AllowInsecureHTTP
	}
	normalized, err := urlvalidator.ValidateURLFormat(raw, allowHTTP)
	if err != nil {
		return "", fmt.Errorf("invalid base_url: %w", err)
	}
	return normalized, nil
}

func normalizeAccountManagerHTTPSBaseURL(raw string, allowPrivate bool) (string, error) {
	normalized, err := urlvalidator.ValidateHTTPSURL(raw, urlvalidator.ValidationOptions{
		AllowPrivate: allowPrivate,
	})
	if err != nil {
		return "", fmt.Errorf("invalid base_url: %w", err)
	}
	return normalized, nil
}

func accountManagerLogin(ctx context.Context, client *http.Client, baseURL, username, password string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/auth/login", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("account manager login failed: status=%d body=%s", resp.StatusCode, string(raw))
	}
	var parsed struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("account manager login parse failed: %w", err)
	}
	if strings.TrimSpace(parsed.AccessToken) == "" {
		return "", errors.New("account manager login failed: accessToken is missing")
	}
	return strings.TrimSpace(parsed.AccessToken), nil
}

func accountManagerExportSub2API(ctx context.Context, client *http.Client, baseURL, token string) (DataPayload, error) {
	var zero DataPayload
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/api-credentials/export-sub2api", nil)
	if err != nil {
		return zero, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return zero, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return zero, fmt.Errorf("account manager export failed: status=%d body=%s", resp.StatusCode, string(raw))
	}
	var parsed accountManagerExportResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return zero, fmt.Errorf("account manager export parse failed: %w", err)
	}
	return parsed.Payload, nil
}

func (h *AccountHandler) previewAccountManagerData(ctx context.Context, payload DataPayload) (AccountManagerPreviewResult, error) {
	result := AccountManagerPreviewResult{
		NewAccounts:      []AccountManagerPreviewAccount{},
		ExistingAccounts: []AccountManagerPreviewAccount{},
	}
	if err := validateDataHeader(payload); err != nil {
		return result, err
	}
	existingAccounts, err := h.listAccountsFiltered(ctx, "", "", "", "", 0, "", "created_at", "desc")
	if err != nil {
		return result, err
	}
	index := buildAccountManagerAccountIndex(existingAccounts)
	seen := map[string]struct{}{}
	for _, item := range payload.Accounts {
		if err := validateDataAccount(item); err != nil {
			result.Errors = append(result.Errors, DataImportError{Kind: "account", Name: item.Name, Message: err.Error()})
			continue
		}
		enrichCredentialsFromIDToken(&item)
		key := accountManagerIdentityKey(item.Platform, item.Type, item.Credentials, item.Extra)
		if key == "" {
			result.Errors = append(result.Errors, DataImportError{Kind: "account", Name: item.Name, Message: "account identity is missing"})
			continue
		}
		if _, ok := seen[key]; ok {
			result.Errors = append(result.Errors, DataImportError{Kind: "account", Name: item.Name, Message: "duplicate account identity in payload"})
			continue
		}
		seen[key] = struct{}{}
		preview := AccountManagerPreviewAccount{
			AccountKey: key,
			Name:       item.Name,
			Platform:   item.Platform,
			Type:       item.Type,
		}
		if existing := index[key]; existing != nil {
			preview.AccountID = existing.ID
			result.ExistingAccounts = append(result.ExistingAccounts, preview)
		} else {
			result.NewAccounts = append(result.NewAccounts, preview)
		}
	}
	return result, nil
}

func (h *AccountHandler) syncAccountManagerData(ctx context.Context, payload DataPayload, selectedKeys []string) (AccountManagerSyncResult, error) {
	result := AccountManagerSyncResult{Items: []AccountManagerSyncItem{}}
	if err := validateDataHeader(payload); err != nil {
		return result, err
	}
	existingAccounts, err := h.listAccountsFiltered(ctx, "", "", "", "", 0, "", "created_at", "desc")
	if err != nil {
		return result, err
	}
	index := buildAccountManagerAccountIndex(existingAccounts)
	selected := make(map[string]struct{}, len(selectedKeys))
	for _, key := range selectedKeys {
		selected[strings.TrimSpace(key)] = struct{}{}
	}
	seen := map[string]struct{}{}
	for _, item := range payload.Accounts {
		syncItem := AccountManagerSyncItem{Name: item.Name, Platform: item.Platform, Type: item.Type}
		if err := validateDataAccount(item); err != nil {
			syncItem.Action = "failed"
			syncItem.Message = err.Error()
			result.Failed++
			result.Items = append(result.Items, syncItem)
			result.Errors = append(result.Errors, DataImportError{Kind: "account", Name: item.Name, Message: err.Error()})
			continue
		}
		enrichCredentialsFromIDToken(&item)
		key := accountManagerIdentityKey(item.Platform, item.Type, item.Credentials, item.Extra)
		syncItem.AccountKey = key
		if key == "" {
			syncItem.Action = "failed"
			syncItem.Message = "account identity is missing"
			result.Failed++
			result.Items = append(result.Items, syncItem)
			result.Errors = append(result.Errors, DataImportError{Kind: "account", Name: item.Name, Message: syncItem.Message})
			continue
		}
		if _, ok := seen[key]; ok {
			syncItem.Action = "skipped"
			syncItem.Message = "duplicate account identity in payload"
			result.Skipped++
			result.Items = append(result.Items, syncItem)
			continue
		}
		seen[key] = struct{}{}
		if existing := index[key]; existing != nil {
			updateInput := buildAccountManagerUpdateInput(*existing, item)
			updated, updateErr := h.adminService.UpdateAccount(ctx, existing.ID, updateInput)
			if updateErr != nil {
				syncItem.Action = "failed"
				syncItem.Message = updateErr.Error()
				result.Failed++
				result.Items = append(result.Items, syncItem)
				result.Errors = append(result.Errors, DataImportError{Kind: "account", Name: item.Name, Message: updateErr.Error()})
				continue
			}
			syncItem.Action = "updated"
			syncItem.AccountID = existing.ID
			if updated != nil {
				syncItem.AccountID = updated.ID
				index[key] = updated
			}
			result.Updated++
			result.Items = append(result.Items, syncItem)
			continue
		}
		if _, ok := selected[key]; !ok {
			syncItem.Action = "skipped"
			syncItem.Message = "not selected"
			result.Skipped++
			result.Items = append(result.Items, syncItem)
			continue
		}
		created, createErr := h.adminService.CreateAccount(ctx, &service.CreateAccountInput{
			Name:                 item.Name,
			Notes:                item.Notes,
			Platform:             item.Platform,
			Type:                 item.Type,
			Credentials:          item.Credentials,
			Extra:                item.Extra,
			Concurrency:          item.Concurrency,
			Priority:             item.Priority,
			RateMultiplier:       item.RateMultiplier,
			ExpiresAt:            item.ExpiresAt,
			AutoPauseOnExpired:   item.AutoPauseOnExpired,
			SkipDefaultGroupBind: true,
		})
		if createErr != nil {
			syncItem.Action = "failed"
			syncItem.Message = createErr.Error()
			result.Failed++
			result.Items = append(result.Items, syncItem)
			result.Errors = append(result.Errors, DataImportError{Kind: "account", Name: item.Name, Message: createErr.Error()})
			continue
		}
		syncItem.Action = "created"
		if created != nil {
			syncItem.AccountID = created.ID
			index[key] = created
		}
		result.Created++
		result.Items = append(result.Items, syncItem)
	}
	return result, nil
}

func buildAccountManagerUpdateInput(existing service.Account, incoming DataAccount) *service.UpdateAccountInput {
	credentials := mergeCodexImportMap(existing.Credentials, incoming.Credentials)
	extra := mergeAccountManagerExtra(existing.Extra, incoming.Extra)
	update := &service.UpdateAccountInput{
		Name:               incoming.Name,
		Notes:              incoming.Notes,
		Type:               incoming.Type,
		Credentials:        credentials,
		Extra:              extra,
		Concurrency:        &incoming.Concurrency,
		Priority:           &incoming.Priority,
		RateMultiplier:     incoming.RateMultiplier,
		ExpiresAt:          incoming.ExpiresAt,
		AutoPauseOnExpired: incoming.AutoPauseOnExpired,
	}
	return update
}

func mergeAccountManagerExtra(existing, incoming map[string]any) map[string]any {
	out := mergeCodexImportMap(existing, incoming)
	for _, key := range []string{"quota_used", "quota_daily_used", "quota_daily_start", "quota_weekly_used", "quota_weekly_start"} {
		if v, ok := existing[key]; ok {
			out[key] = v
		}
	}
	return out
}

func buildAccountManagerAccountIndex(accounts []service.Account) map[string]*service.Account {
	index := make(map[string]*service.Account, len(accounts))
	for i := range accounts {
		account := accounts[i]
		key := accountManagerIdentityKey(account.Platform, account.Type, account.Credentials, account.Extra)
		if key != "" {
			index[key] = &account
		}
	}
	return index
}

func accountManagerIdentityKey(platform, accountType string, credentials, extra map[string]any) string {
	platform = strings.ToLower(strings.TrimSpace(platform))
	accountType = strings.ToLower(strings.TrimSpace(accountType))
	if platform == service.PlatformOpenAI && accountType == service.AccountTypeOAuth {
		if v := dataString(credentials, "chatgpt_account_id"); v != "" {
			return "account:" + v
		}
		if v := dataString(credentials, "chatgpt_user_id"); v != "" {
			return "user:" + v
		}
	}
	if accountType == service.AccountTypeAPIKey {
		if v := dataString(credentials, "api_key"); v != "" {
			return platform + ":" + accountType + ":api_key:" + sha256Hex(v)
		}
	}
	if v := dataString(credentials, "email"); v != "" {
		return platform + ":" + accountType + ":email:" + strings.ToLower(v)
	}
	if v := dataString(credentials, "user_id"); v != "" {
		return platform + ":" + accountType + ":user:" + v
	}
	if v := dataString(extra, "access_token_sha256"); v != "" {
		return platform + ":" + accountType + ":access:" + v
	}
	if v := dataString(credentials, "access_token"); v != "" {
		return platform + ":" + accountType + ":access:" + sha256Hex(v)
	}
	return ""
}

func dataString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	return strings.TrimSpace(codexStringValue(m[key]))
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])
}
