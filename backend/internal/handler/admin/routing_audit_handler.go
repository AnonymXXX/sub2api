package admin

import (
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type RoutingAuditHandler struct {
	service *service.RoutingAuditService
}

func NewRoutingAuditHandler(service *service.RoutingAuditService) *RoutingAuditHandler {
	return &RoutingAuditHandler{service: service}
}

// List handles routing audit log listing.
// GET /api/v1/admin/routing-audit/logs
func (h *RoutingAuditHandler) List(c *gin.Context) {
	filter, ok := parseRoutingAuditFilter(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	params := pagination.PaginationParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    c.DefaultQuery("sort_by", "created_at"),
		SortOrder: c.DefaultQuery("sort_order", "desc"),
	}
	items, result, err := h.service.List(c.Request.Context(), params, filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, result.Total, result.Page, result.PageSize)
}

// Summary returns aggregate routing audit stats.
// GET /api/v1/admin/routing-audit/summary
func (h *RoutingAuditHandler) Summary(c *gin.Context) {
	filter, ok := parseRoutingAuditFilter(c)
	if !ok {
		return
	}
	summary, err := h.service.Summary(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
}

func parseRoutingAuditFilter(c *gin.Context) (service.RoutingAuditFilter, bool) {
	var filter service.RoutingAuditFilter
	parseID := func(name string) (int64, bool) {
		raw := strings.TrimSpace(c.Query(name))
		if raw == "" {
			return 0, true
		}
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id < 0 {
			response.BadRequest(c, "Invalid "+name)
			return 0, false
		}
		return id, true
	}
	var ok bool
	if filter.UserID, ok = parseID("user_id"); !ok {
		return filter, false
	}
	if filter.APIKeyID, ok = parseID("api_key_id"); !ok {
		return filter, false
	}
	if filter.AccountID, ok = parseID("account_id"); !ok {
		return filter, false
	}
	if filter.GroupID, ok = parseID("group_id"); !ok {
		return filter, false
	}
	filter.Model = strings.TrimSpace(c.Query("model"))
	filter.SelectedPool = strings.TrimSpace(c.Query("selected_pool"))
	filter.FinalRoute = strings.TrimSpace(c.Query("final_route"))
	filter.DecisionReason = strings.TrimSpace(c.Query("decision_reason"))
	filter.RoutingPolicyVersion = strings.TrimSpace(c.Query("routing_policy_version"))

	userTZ := strings.TrimSpace(c.Query("timezone"))
	if start := strings.TrimSpace(c.Query("start_date")); start != "" {
		t, err := timezone.ParseInUserLocation("2006-01-02", start, userTZ)
		if err != nil {
			response.BadRequest(c, "Invalid start_date format, use YYYY-MM-DD")
			return filter, false
		}
		filter.StartTime = &t
	}
	if end := strings.TrimSpace(c.Query("end_date")); end != "" {
		t, err := timezone.ParseInUserLocation("2006-01-02", end, userTZ)
		if err != nil {
			response.BadRequest(c, "Invalid end_date format, use YYYY-MM-DD")
			return filter, false
		}
		t = t.AddDate(0, 0, 1)
		filter.EndTime = &t
	}
	if filter.StartTime == nil && filter.EndTime == nil {
		end := time.Now()
		start := end.Add(-48 * time.Hour)
		filter.StartTime = &start
		filter.EndTime = &end
	}
	return filter, true
}
