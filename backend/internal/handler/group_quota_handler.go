package handler

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/handler/quotaview"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// GroupQuotaHandler 普通用户的分组额度卡片接口。
// 需要管理员预先授予查看权限；返回的账号信息匿名化（账号N，无真实 ID/名称）。
type GroupQuotaHandler struct {
	groupService      *service.GroupService
	groupQuotaService *service.GroupQuotaService
	grantService      *service.GroupViewGrantService
}

// NewGroupQuotaHandler creates a new user-facing group quota handler
func NewGroupQuotaHandler(
	groupService *service.GroupService,
	groupQuotaService *service.GroupQuotaService,
	grantService *service.GroupViewGrantService,
) *GroupQuotaHandler {
	return &GroupQuotaHandler{
		groupService:      groupService,
		groupQuotaService: groupQuotaService,
		grantService:      grantService,
	}
}

// ListMyViewableGroups handles listing groups the current user may view quota cards for.
// GET /api/v1/user/my-viewable-groups
func (h *GroupQuotaHandler) ListMyViewableGroups(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	groups, err := h.grantService.ListViewableGroupsByUser(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.ViewableGroupEntry, 0, len(groups))
	for i := range groups {
		out = append(out, *dto.ViewableGroupFromService(&groups[i]))
	}
	response.Success(c, out)
}

// GetMyGroupQuotaCard handles getting the anonymized quota card for a granted group.
// GET /api/v1/user/groups/:id/quota-card?sort=5h|7d
func (h *GroupQuotaHandler) GetMyGroupQuotaCard(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || groupID <= 0 {
		response.BadRequest(c, "Invalid group ID")
		return
	}
	sortBy, ok := quotaview.ParseSortWindow(c.Query("sort"))
	if !ok {
		response.BadRequest(c, "Invalid sort, must be 5h or 7d")
		return
	}
	allowed, err := h.grantService.HasViewAccess(c.Request.Context(), subject.UserID, groupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if !allowed {
		response.Forbidden(c, "You do not have access to this group's quota card")
		return
	}
	group, err := h.groupService.GetByID(c.Request.Context(), groupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if !quotaview.SupportsQuotaCard(group.Platform) {
		response.BadRequest(c, "Quota card is not supported for this platform")
		return
	}
	result, err := h.groupQuotaService.GetGroupQuotaCard(c.Request.Context(), groupID, sortBy, true)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.GroupQuotaCardFromService(result, group))
}
