package admin

import (
	"strconv"
	"strings"

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

// 授权候选搜索的结果条数上限。
const (
	viewGrantCandidateDefaultLimit = 20
	viewGrantCandidateMaxLimit     = 50
)

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

// SearchViewGrantCandidates handles searching users to authorize on a group's quota card.
// GET /api/v1/admin/groups/:id/view-grant-candidates?q=&limit=
//
// q 匹配邮箱/用户名/备注（复用 UserListFilters.Search）；q 为空时返回最近注册的用户，
// 让下拉框在聚焦时就有内容可选。软删除用户不参与匹配，避免产生悬挂授权。
// 已授权用户以 granted=true 标记而非从结果中剔除——管理员需要看到"这个人已经有权限了"，
// 直接过滤掉会让人误以为搜索没找到。
func (h *GroupQuotaHandler) SearchViewGrantCandidates(c *gin.Context) {
	groupID, ok := parsePositiveIDParam(c, "id")
	if !ok {
		return
	}
	limit := parseViewGrantCandidateLimit(c.Query("limit"))
	keyword := strings.TrimSpace(c.Query("q"))

	// 空关键词按注册时间倒序给出"最近注册"，有关键词时按用户名升序便于扫读。
	sortBy, sortOrder := "created_at", "desc"
	if keyword != "" {
		sortBy, sortOrder = "username", "asc"
	}

	includeSubscriptions := false
	users, _, err := h.adminService.ListUsers(
		c.Request.Context(), 1, limit,
		service.UserListFilters{Search: keyword, IncludeSubscriptions: &includeSubscriptions},
		sortBy, sortOrder,
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	grants, err := h.grantService.ListGrantsByGroup(c.Request.Context(), groupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	granted := make(map[int64]struct{}, len(grants))
	for i := range grants {
		granted[grants[i].UserID] = struct{}{}
	}

	out := make([]dto.GroupViewGrantCandidate, 0, len(users))
	for i := range users {
		_, has := granted[users[i].ID]
		out = append(out, *dto.GroupViewGrantCandidateFromService(&users[i], has))
	}
	response.Success(c, out)
}

// parseViewGrantCandidateLimit 解析候选条数，非法值回落到默认值并夹紧到上限。
func parseViewGrantCandidateLimit(raw string) int {
	if raw == "" {
		return viewGrantCandidateDefaultLimit
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return viewGrantCandidateDefaultLimit
	}
	if n > viewGrantCandidateMaxLimit {
		return viewGrantCandidateMaxLimit
	}
	return n
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
