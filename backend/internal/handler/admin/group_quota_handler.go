package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/handler/quotaview"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// GroupQuotaHandler 分组额度卡片与查看授权的管理端接口。
// 管理端视角不做匿名化：返回真实账号 ID 与名称。
type GroupQuotaHandler struct {
	adminService      service.AdminService
	groupQuotaService *service.GroupQuotaService
	grantService      *service.GroupViewGrantService
}

// NewGroupQuotaHandler creates a new admin group quota handler
func NewGroupQuotaHandler(
	adminService service.AdminService,
	groupQuotaService *service.GroupQuotaService,
	grantService *service.GroupViewGrantService,
) *GroupQuotaHandler {
	return &GroupQuotaHandler{
		adminService:      adminService,
		groupQuotaService: groupQuotaService,
		grantService:      grantService,
	}
}

// GrantViewAccessRequest represents the grant view access request body
type GrantViewAccessRequest struct {
	UserID int64 `json:"user_id" binding:"required,gt=0"`
}

// GetQuotaCard handles getting the aggregated quota card for a group.
// GET /api/v1/admin/groups/:id/quota-card?sort=5h|7d
func (h *GroupQuotaHandler) GetQuotaCard(c *gin.Context) {
	groupID, ok := parsePositiveIDParam(c, "id")
	if !ok {
		return
	}
	sortBy, ok := quotaview.ParseSortWindow(c.Query("sort"))
	if !ok {
		response.BadRequest(c, "Invalid sort, must be 5h or 7d")
		return
	}
	group, err := h.adminService.GetGroup(c.Request.Context(), groupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if !quotaview.SupportsQuotaCard(group.Platform) {
		response.BadRequest(c, "Quota card is not supported for this platform")
		return
	}
	result, err := h.groupQuotaService.GetGroupQuotaCard(c.Request.Context(), groupID, sortBy, false)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.GroupQuotaCardFromService(result, group))
}

// ListViewGrants handles listing users granted view access to a group's quota card.
// GET /api/v1/admin/groups/:id/view-grants
func (h *GroupQuotaHandler) ListViewGrants(c *gin.Context) {
	groupID, ok := parsePositiveIDParam(c, "id")
	if !ok {
		return
	}
	grants, err := h.grantService.ListGrantsByGroup(c.Request.Context(), groupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.GroupViewGrantEntry, 0, len(grants))
	for i := range grants {
		out = append(out, *dto.GroupViewGrantFromService(&grants[i]))
	}
	response.Success(c, out)
}

// GrantViewAccess handles granting a user view access to a group's quota card.
// POST /api/v1/admin/groups/:id/view-grants
func (h *GroupQuotaHandler) GrantViewAccess(c *gin.Context) {
	groupID, ok := parsePositiveIDParam(c, "id")
	if !ok {
		return
	}
	var req GrantViewAccessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	// 目标用户必须存在，避免悬挂授权
	if _, err := h.adminService.GetUser(c.Request.Context(), req.UserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if err := h.grantService.GrantViewAccess(c.Request.Context(), groupID, req.UserID, subject.UserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "View access granted"})
}

// RevokeViewAccess handles revoking a user's view access to a group's quota card.
// DELETE /api/v1/admin/groups/:id/view-grants/:userId
func (h *GroupQuotaHandler) RevokeViewAccess(c *gin.Context) {
	groupID, ok := parsePositiveIDParam(c, "id")
	if !ok {
		return
	}
	userID, ok := parsePositiveIDParam(c, "userId")
	if !ok {
		return
	}
	if err := h.grantService.RevokeViewAccess(c.Request.Context(), groupID, userID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "View access revoked"})
}
