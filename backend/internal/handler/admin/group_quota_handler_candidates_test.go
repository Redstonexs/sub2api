package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 捕获 ListUsers 入参并返回固定用户集的 admin service 桩。
type grantCandidateAdminStub struct {
	service.AdminService
	gotPageSize  int
	gotFilters   service.UserListFilters
	gotSortBy    string
	gotSortOrder string
}

func (s *grantCandidateAdminStub) ListUsers(
	_ context.Context, _, pageSize int, filters service.UserListFilters, sortBy, sortOrder string,
) ([]service.User, int64, error) {
	s.gotPageSize = pageSize
	s.gotFilters = filters
	s.gotSortBy = sortBy
	s.gotSortOrder = sortOrder
	return []service.User{
		{ID: 1, Username: "alice", Email: "alice@test.com", Role: service.RoleUser, Status: service.StatusActive},
		{ID: 2, Username: "bob", Email: "bob@test.com", Role: service.RoleAdmin, Status: service.StatusActive},
	}, 2, nil
}

// 只对已授权用户集合作出应答的授权仓储桩。
type grantCandidateGrantRepoStub struct {
	service.GroupViewGrantRepository
	grantedUserIDs []int64
}

func (s *grantCandidateGrantRepoStub) ListByGroup(
	_ context.Context, groupID int64,
) ([]service.GroupViewGrantWithUser, error) {
	out := make([]service.GroupViewGrantWithUser, 0, len(s.grantedUserIDs))
	for _, id := range s.grantedUserIDs {
		out = append(out, service.GroupViewGrantWithUser{
			GroupViewGrant: service.GroupViewGrant{UserID: id, GroupID: groupID},
		})
	}
	return out, nil
}

type candidateResponse struct {
	Data []struct {
		UserID   int64  `json:"user_id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Role     string `json:"role"`
		Status   string `json:"status"`
		Granted  bool   `json:"granted"`
	} `json:"data"`
}

func newCandidateRouter(admin *grantCandidateAdminStub, granted []int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	grantSvc := service.NewGroupViewGrantService(&grantCandidateGrantRepoStub{grantedUserIDs: granted})
	handler := NewGroupQuotaHandler(admin, nil, grantSvc)
	router := gin.New()
	router.GET("/admin/groups/:id/view-grant-candidates", handler.SearchViewGrantCandidates)
	return router
}

func getCandidates(t *testing.T, router *gin.Engine, query string) candidateResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/admin/groups/42/view-grant-candidates"+query, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp candidateResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	return resp
}

func TestSearchViewGrantCandidates_AnnotatesExistingGrants(t *testing.T) {
	stub := &grantCandidateAdminStub{}
	resp := getCandidates(t, newCandidateRouter(stub, []int64{2}), "?q=ali")

	require.Equal(t, "ali", stub.gotFilters.Search)
	require.False(t, stub.gotFilters.IncludeDeleted, "候选搜索不得返回软删除用户")
	require.NotNil(t, stub.gotFilters.IncludeSubscriptions)
	require.False(t, *stub.gotFilters.IncludeSubscriptions, "选人下拉不需要订阅信息")
	require.Equal(t, "username", stub.gotSortBy)
	require.Equal(t, "asc", stub.gotSortOrder)

	require.Len(t, resp.Data, 2)
	require.Equal(t, int64(1), resp.Data[0].UserID)
	require.Equal(t, "alice", resp.Data[0].Username)
	require.Equal(t, "alice@test.com", resp.Data[0].Email)
	require.Equal(t, service.RoleUser, resp.Data[0].Role)
	require.Equal(t, service.StatusActive, resp.Data[0].Status)
	require.False(t, resp.Data[0].Granted)
	// 已授权用户仍然返回，只是标记 granted=true（不能从结果里剔除）
	require.Equal(t, int64(2), resp.Data[1].UserID)
	require.True(t, resp.Data[1].Granted)
	require.Equal(t, service.RoleAdmin, resp.Data[1].Role)
}

func TestSearchViewGrantCandidates_EmptyQueryListsRecentUsers(t *testing.T) {
	stub := &grantCandidateAdminStub{}
	resp := getCandidates(t, newCandidateRouter(stub, nil), "")

	require.Empty(t, stub.gotFilters.Search)
	require.Equal(t, "created_at", stub.gotSortBy)
	require.Equal(t, "desc", stub.gotSortOrder)
	require.Len(t, resp.Data, 2, "空关键词应返回最近注册用户，而不是空列表")
}

func TestSearchViewGrantCandidates_LimitParsing(t *testing.T) {
	cases := []struct {
		name  string
		query string
		want  int
	}{
		{"default when absent", "", viewGrantCandidateDefaultLimit},
		{"default when unparsable", "?limit=abc", viewGrantCandidateDefaultLimit},
		{"default when non-positive", "?limit=0", viewGrantCandidateDefaultLimit},
		{"honors explicit value", "?limit=5", 5},
		{"clamped to max", "?limit=9999", viewGrantCandidateMaxLimit},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &grantCandidateAdminStub{}
			getCandidates(t, newCandidateRouter(stub, nil), tc.query)
			require.Equal(t, tc.want, stub.gotPageSize)
		})
	}
}

func TestSearchViewGrantCandidates_RejectsInvalidGroupID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	grantSvc := service.NewGroupViewGrantService(&grantCandidateGrantRepoStub{})
	handler := NewGroupQuotaHandler(&grantCandidateAdminStub{}, nil, grantSvc)
	router := gin.New()
	router.GET("/admin/groups/:id/view-grant-candidates", handler.SearchViewGrantCandidates)

	req := httptest.NewRequest(http.MethodGet, "/admin/groups/0/view-grant-candidates?q=a", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
