package admin

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestPreviewAccountManagerDataClassifiesExistingByChatGPTAccountID(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{
		{
			ID:          41,
			Name:        "Existing",
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeOAuth,
			Credentials: map[string]any{"chatgpt_account_id": "acct-existing", "email": "old@example.com"},
		},
	}
	h := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	result, err := h.previewAccountManagerData(t.Context(), DataPayload{
		ExportedAt: "2026-07-06T00:00:00Z",
		Proxies:    []DataProxy{},
		Accounts: []DataAccount{
			{
				Name:        "Updated",
				Platform:    service.PlatformOpenAI,
				Type:        service.AccountTypeOAuth,
				Credentials: map[string]any{"chatgpt_account_id": "acct-existing", "email": "new@example.com"},
				Concurrency: 3,
				Priority:    50,
			},
			{
				Name:        "New",
				Platform:    service.PlatformOpenAI,
				Type:        service.AccountTypeOAuth,
				Credentials: map[string]any{"chatgpt_account_id": "acct-new", "email": "new2@example.com"},
				Concurrency: 3,
				Priority:    50,
			},
		},
	})

	require.NoError(t, err)
	require.Len(t, result.ExistingAccounts, 1)
	require.Equal(t, int64(41), result.ExistingAccounts[0].AccountID)
	require.Equal(t, "account:acct-existing", result.ExistingAccounts[0].AccountKey)
	require.Len(t, result.NewAccounts, 1)
	require.Equal(t, "account:acct-new", result.NewAccounts[0].AccountKey)
}

func TestPreviewAccountManagerDataUsesIDTokenIdentity(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{
		{
			ID:          41,
			Name:        "Existing",
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeOAuth,
			Credentials: map[string]any{"chatgpt_account_id": "acct-from-token"},
		},
	}
	h := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	result, err := h.previewAccountManagerData(t.Context(), DataPayload{
		ExportedAt: "2026-07-06T00:00:00Z",
		Proxies:    []DataProxy{},
		Accounts: []DataAccount{
			{
				Name:        "Token Only",
				Platform:    service.PlatformOpenAI,
				Type:        service.AccountTypeOAuth,
				Credentials: map[string]any{"id_token": fakeOpenAIIDToken("acct-from-token", "team", "team@example.com")},
				Concurrency: 3,
				Priority:    50,
			},
		},
	})

	require.NoError(t, err)
	require.Len(t, result.ExistingAccounts, 1)
	require.Equal(t, int64(41), result.ExistingAccounts[0].AccountID)
	require.Equal(t, "account:acct-from-token", result.ExistingAccounts[0].AccountKey)
}

func TestSyncAccountManagerDataUpdatesExistingAndCreatesSelectedNew(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{
		{
			ID:          41,
			Name:        "Existing",
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeOAuth,
			Credentials: map[string]any{"chatgpt_account_id": "acct-existing", "email": "old@example.com", "model_mapping": map[string]any{"keep": true}},
			Extra:       map[string]any{"quota_used": float64(12), "local": "keep"},
			Concurrency: 1,
			Priority:    10,
		},
	}
	h := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	result, err := h.syncAccountManagerData(t.Context(), DataPayload{
		ExportedAt: "2026-07-06T00:00:00Z",
		Proxies:    []DataProxy{},
		Accounts: []DataAccount{
			{
				Name:        "Updated",
				Platform:    service.PlatformOpenAI,
				Type:        service.AccountTypeOAuth,
				Credentials: map[string]any{"chatgpt_account_id": "acct-existing", "email": "new@example.com", "access_token": "new-access"},
				Extra:       map[string]any{"remote": "value"},
				Concurrency: 5,
				Priority:    40,
			},
			{
				Name:        "New",
				Platform:    service.PlatformOpenAI,
				Type:        service.AccountTypeOAuth,
				Credentials: map[string]any{"chatgpt_account_id": "acct-new", "email": "new2@example.com"},
				Concurrency: 3,
				Priority:    50,
			},
			{
				Name:        "Unselected",
				Platform:    service.PlatformOpenAI,
				Type:        service.AccountTypeOAuth,
				Credentials: map[string]any{"chatgpt_account_id": "acct-unselected", "email": "new3@example.com"},
				Concurrency: 3,
				Priority:    50,
			},
		},
	}, []string{"account:acct-new"})

	require.NoError(t, err)
	require.Equal(t, 1, result.Updated)
	require.Equal(t, 1, result.Created)
	require.Equal(t, 1, result.Skipped)
	require.Len(t, adminSvc.updatedAccounts, 1)
	require.Equal(t, int64(41), adminSvc.updatedAccountIDs[0])
	updated := adminSvc.updatedAccounts[0]
	require.Equal(t, "Updated", updated.Name)
	require.Equal(t, "new@example.com", updated.Credentials["email"])
	require.Equal(t, "new-access", updated.Credentials["access_token"])
	require.Contains(t, updated.Credentials, "model_mapping")
	require.Equal(t, "value", updated.Extra["remote"])
	require.Equal(t, float64(12), updated.Extra["quota_used"])
	require.Equal(t, 5, *updated.Concurrency)
	require.Equal(t, 40, *updated.Priority)
	require.Len(t, adminSvc.createdAccounts, 1)
	require.Equal(t, "New", adminSvc.createdAccounts[0].Name)
}

func TestFetchAccountManagerDataLogsInAndExportsPayload(t *testing.T) {
	var sawAuth string
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/login":
			var body map[string]string
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, "admin", body["username"])
			require.Equal(t, "password", body["password"])
			_ = json.NewEncoder(w).Encode(map[string]any{"accessToken": "remote-token"})
		case "/api/api-credentials/export-sub2api":
			sawAuth = r.Header.Get("Authorization")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"payload": map[string]any{
					"type":        "sub2api-data",
					"version":     1,
					"exported_at": "2026-07-06T00:00:00Z",
					"proxies":     []any{},
					"accounts": []any{
						map[string]any{
							"name":        "Remote",
							"platform":    "openai",
							"type":        "oauth",
							"credentials": map[string]any{"chatgpt_account_id": "acct-remote"},
							"concurrency": 3,
							"priority":    50,
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer remote.Close()

	payload, err := fetchAccountManagerData(t.Context(), AccountManagerSyncRequest{
		BaseURL:  remote.URL,
		Username: "admin",
		Password: "password",
	}, &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{
		Enabled:           false,
		AllowInsecureHTTP: true,
		AllowPrivateHosts: true,
	}}})

	require.NoError(t, err)
	require.Equal(t, "Bearer remote-token", sawAuth)
	require.Len(t, payload.Accounts, 1)
	require.Equal(t, "Remote", payload.Accounts[0].Name)
}

func fakeOpenAIIDToken(accountID, planType, email string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload, _ := json.Marshal(map[string]any{
		"email": email,
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_account_id": accountID,
			"chatgpt_plan_type":  planType,
			"chatgpt_user_id":    "user-" + accountID,
			"organizations":      []any{},
		},
	})
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + "."
}
