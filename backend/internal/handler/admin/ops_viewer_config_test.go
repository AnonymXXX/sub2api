package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type opsViewerSettingRepo struct {
	values map[string]string
}

func (r *opsViewerSettingRepo) Get(context.Context, string) (*service.Setting, error) {
	return nil, service.ErrSettingNotFound
}
func (r *opsViewerSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	value, ok := r.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return value, nil
}
func (r *opsViewerSettingRepo) Set(_ context.Context, key, value string) error {
	r.values[key] = value
	return nil
}
func (r *opsViewerSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := r.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}
func (r *opsViewerSettingRepo) SetMultiple(_ context.Context, values map[string]string) error {
	for key, value := range values {
		r.values[key] = value
	}
	return nil
}
func (r *opsViewerSettingRepo) GetAll(context.Context) (map[string]string, error) {
	return r.values, nil
}
func (r *opsViewerSettingRepo) Delete(_ context.Context, key string) error {
	delete(r.values, key)
	return nil
}

func TestGetOpsViewerConfigReturnsMinimalReadOnlyPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminService := newStubAdminService()
	adminService.groups = []service.Group{
		{ID: 7, Name: "OpenAI", Platform: "openai", RateMultiplier: 2},
	}
	settingService := service.NewSettingService(&opsViewerSettingRepo{values: map[string]string{
		service.SettingKeyOpsMonitoringEnabled:         "true",
		service.SettingKeyOpsRealtimeMonitoringEnabled: "false",
		service.SettingKeyOpsQueryModeDefault:          "preagg",
		service.SettingKeyAdminAPIKey:                  "must-not-leak",
	}}, &config.Config{})
	handler := NewGroupHandler(adminService, nil, nil, settingService)

	router := gin.New()
	router.GET("/viewer-config", handler.GetOpsViewerConfig)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/viewer-config", nil))
	require.Equal(t, http.StatusOK, recorder.Code)

	var envelope struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	keys := make([]string, 0, len(envelope.Data))
	for key := range envelope.Data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	require.Equal(t, []string{
		"groups",
		"ops_monitoring_enabled",
		"ops_query_mode_default",
		"ops_realtime_monitoring_enabled",
	}, keys)
	require.NotContains(t, recorder.Body.String(), "must-not-leak")

	groups, ok := envelope.Data["groups"].([]any)
	require.True(t, ok)
	require.Len(t, groups, 1)
	group := groups[0].(map[string]any)
	require.Equal(t, map[string]any{"id": float64(7), "name": "OpenAI", "platform": "openai"}, group)
}
