//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Fake implementation of GroupViewGrantRepository
// ---------------------------------------------------------------------------

type fakeGroupViewGrantRepo struct {
	// storage keyed by "userID:groupID"
	grants map[string]*GroupViewGrant
	// sorted ordered lists for ListByGroup/ListByUser
	groupsByUser map[int64][]ViewableGroup

	createCalls []*GroupViewGrant
	deleteCalls []struct{ userID, groupID int64 }
	forceErr    error
}

func newFakeGroupViewGrantRepo() *fakeGroupViewGrantRepo {
	return &fakeGroupViewGrantRepo{
		grants:       make(map[string]*GroupViewGrant),
		groupsByUser: make(map[int64][]ViewableGroup),
	}
}

func (r *fakeGroupViewGrantRepo) key(userID, groupID int64) string {
	return itoa(userID) + ":" + itoa(groupID)
}

func (r *fakeGroupViewGrantRepo) Create(_ context.Context, grant *GroupViewGrant) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.createCalls = append(r.createCalls, grant)
	r.grants[r.key(grant.UserID, grant.GroupID)] = grant
	return nil
}

func (r *fakeGroupViewGrantRepo) DeleteByUserAndGroup(_ context.Context, userID, groupID int64) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.deleteCalls = append(r.deleteCalls, struct{ userID, groupID int64 }{userID, groupID})
	delete(r.grants, r.key(userID, groupID))
	return nil
}

func (r *fakeGroupViewGrantRepo) ListByGroup(_ context.Context, groupID int64) ([]GroupViewGrantWithUser, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	var result []GroupViewGrantWithUser
	for _, grant := range r.grants {
		if grant.GroupID == groupID {
			result = append(result, GroupViewGrantWithUser{GroupViewGrant: *grant})
		}
	}
	return result, nil
}

func (r *fakeGroupViewGrantRepo) ListByUser(_ context.Context, userID int64) ([]ViewableGroup, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	return r.groupsByUser[userID], nil
}

func (r *fakeGroupViewGrantRepo) ExistsByUserAndGroup(_ context.Context, userID, groupID int64) (bool, error) {
	if r.forceErr != nil {
		return false, r.forceErr
	}
	_, ok := r.grants[r.key(userID, groupID)]
	return ok, nil
}

// addViewableGroup sets up the groups returned by ListByUser for a user.
func (r *fakeGroupViewGrantRepo) addViewableGroup(userID int64, g ViewableGroup) {
	r.groupsByUser[userID] = append(r.groupsByUser[userID], g)
}

// itoa is a simple int-to-string for map keys since we're in tests.
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	s := make([]byte, 0, 20)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		s = append([]byte{byte('0' + n%10)}, s...)
		n /= 10
	}
	if neg {
		s = append([]byte{'-'}, s...)
	}
	return string(s)
}

// ---------------------------------------------------------------------------
// HasViewAccess tests
// ---------------------------------------------------------------------------

func TestHasViewAccess_Granted_ReturnsTrue(t *testing.T) {
	// Given: a grant exists for (user=10, group=100)
	repo := newFakeGroupViewGrantRepo()
	repo.grants[repo.key(10, 100)] = &GroupViewGrant{UserID: 10, GroupID: 100}
	svc := NewGroupViewGrantService(repo)

	// When
	hasAccess, err := svc.HasViewAccess(context.Background(), 10, 100)

	// Then: access is granted
	require.NoError(t, err)
	assert.True(t, hasAccess)
}

func TestHasViewAccess_NotGranted_ReturnsFalse(t *testing.T) {
	// Given: no grant exists for (user=10, group=100)
	repo := newFakeGroupViewGrantRepo()
	svc := NewGroupViewGrantService(repo)

	// When
	hasAccess, err := svc.HasViewAccess(context.Background(), 10, 100)

	// Then: access is not granted
	require.NoError(t, err)
	assert.False(t, hasAccess)
}

func TestHasViewAccess_RepoError_Propagates(t *testing.T) {
	// Given: the repository returns an error
	repo := newFakeGroupViewGrantRepo()
	repo.forceErr = errors.New("db down")
	svc := NewGroupViewGrantService(repo)

	// When
	hasAccess, err := svc.HasViewAccess(context.Background(), 10, 100)

	// Then: error is propagated, result is false
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db down")
	assert.False(t, hasAccess)
}

// ---------------------------------------------------------------------------
// GrantViewAccess tests
// ---------------------------------------------------------------------------

func TestGrantViewAccess_AlreadyGranted_NoCreate(t *testing.T) {
	// Given: a grant already exists for (user=10, group=100)
	repo := newFakeGroupViewGrantRepo()
	repo.grants[repo.key(10, 100)] = &GroupViewGrant{UserID: 10, GroupID: 100}
	svc := NewGroupViewGrantService(repo)

	// When: granting again (idempotent)
	err := svc.GrantViewAccess(context.Background(), 100, 10, 1)

	// Then: no error, and Create was NOT called
	require.NoError(t, err)
	assert.Empty(t, repo.createCalls)
}

func TestGrantViewAccess_NotGranted_CreatesOnce(t *testing.T) {
	// Given: no grant exists for (user=10, group=100)
	repo := newFakeGroupViewGrantRepo()
	svc := NewGroupViewGrantService(repo)

	// When: granting access
	err := svc.GrantViewAccess(context.Background(), 100, 10, 1)

	// Then: Create called once with correct fields
	require.NoError(t, err)
	require.Len(t, repo.createCalls, 1)
	call := repo.createCalls[0]
	assert.Equal(t, int64(10), call.UserID)
	assert.Equal(t, int64(100), call.GroupID)
	assert.Equal(t, int64(1), call.GrantedByUserID)
	assert.False(t, call.GrantedAt.IsZero(), "GrantedAt should be set to current time")
}

func TestGrantViewAccess_RepoError_Propagates(t *testing.T) {
	// Given: repository returns error on Create
	repo := newFakeGroupViewGrantRepo()
	repo.forceErr = errors.New("db down")
	svc := NewGroupViewGrantService(repo)

	// When: granting access
	err := svc.GrantViewAccess(context.Background(), 100, 10, 1)

	// Then: error propagated
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db down")
}

// ---------------------------------------------------------------------------
// RevokeViewAccess tests
// ---------------------------------------------------------------------------

func TestRevokeViewAccess_Granted_CallsDelete(t *testing.T) {
	// Given: a grant exists for (user=10, group=100)
	repo := newFakeGroupViewGrantRepo()
	repo.grants[repo.key(10, 100)] = &GroupViewGrant{UserID: 10, GroupID: 100}
	svc := NewGroupViewGrantService(repo)

	// When: revoking access
	err := svc.RevokeViewAccess(context.Background(), 100, 10)

	// Then: Delete called once
	require.NoError(t, err)
	require.Len(t, repo.deleteCalls, 1)
	assert.Equal(t, int64(10), repo.deleteCalls[0].userID)
	assert.Equal(t, int64(100), repo.deleteCalls[0].groupID)
}

func TestRevokeViewAccess_NotGranted_NoDelete(t *testing.T) {
	// Given: no grant exists for (user=10, group=100)
	repo := newFakeGroupViewGrantRepo()
	svc := NewGroupViewGrantService(repo)

	// When: revoking access (idempotent)
	err := svc.RevokeViewAccess(context.Background(), 100, 10)

	// Then: no error, Delete NOT called
	require.NoError(t, err)
	assert.Empty(t, repo.deleteCalls)
}

func TestRevokeViewAccess_RepoError_Propagates(t *testing.T) {
	// Given: grant exists, but repository returns error on Delete
	repo := newFakeGroupViewGrantRepo()
	repo.grants[repo.key(10, 100)] = &GroupViewGrant{UserID: 10, GroupID: 100}
	repo.forceErr = errors.New("db down")
	svc := NewGroupViewGrantService(repo)

	// When: revoking access
	err := svc.RevokeViewAccess(context.Background(), 100, 10)

	// Then: error propagated
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db down")
}

// ---------------------------------------------------------------------------
// ListGrantsByGroup tests
// ---------------------------------------------------------------------------

func TestListGrantsByGroup_Passthrough(t *testing.T) {
	// Given: grants exist for multiple users in group 100
	repo := newFakeGroupViewGrantRepo()
	repo.grants[repo.key(10, 100)] = &GroupViewGrant{UserID: 10, GroupID: 100}
	repo.grants[repo.key(20, 100)] = &GroupViewGrant{UserID: 20, GroupID: 100}
	// Also a grant in a different group (should not appear)
	repo.grants[repo.key(10, 200)] = &GroupViewGrant{UserID: 10, GroupID: 200}
	svc := NewGroupViewGrantService(repo)

	// When
	results, err := svc.ListGrantsByGroup(context.Background(), 100)

	// Then: only grants for group 100 returned
	require.NoError(t, err)
	require.Len(t, results, 2)
	userIDs := make(map[int64]bool)
	for _, r := range results {
		userIDs[r.UserID] = true
	}
	assert.True(t, userIDs[10])
	assert.True(t, userIDs[20])
}

// ---------------------------------------------------------------------------
// ListViewableGroupsByUser tests
// ---------------------------------------------------------------------------

func TestListViewableGroupsByUser_FiltersByPlatformAndStatus(t *testing.T) {
	// Given: fake returns 5 groups —
	//   anthropic+active → KEEP
	//   openai+active → KEEP
	//   gemini+active → FILTER OUT (wrong platform)
	//   anthropic+disabled → FILTER OUT (wrong status)
	//   composite+active → FILTER OUT (wrong platform)
	repo := newFakeGroupViewGrantRepo()
	userID := int64(1)
	repo.addViewableGroup(userID, ViewableGroup{GroupID: 1, GroupName: "Anthropic Active", Platform: PlatformAnthropic, Status: StatusActive})
	repo.addViewableGroup(userID, ViewableGroup{GroupID: 2, GroupName: "OpenAI Active", Platform: PlatformOpenAI, Status: StatusActive})
	repo.addViewableGroup(userID, ViewableGroup{GroupID: 3, GroupName: "Gemini Active", Platform: PlatformGemini, Status: StatusActive})
	repo.addViewableGroup(userID, ViewableGroup{GroupID: 4, GroupName: "Anthropic Disabled", Platform: PlatformAnthropic, Status: StatusDisabled})
	repo.addViewableGroup(userID, ViewableGroup{GroupID: 5, GroupName: "Composite Active", Platform: PlatformComposite, Status: StatusActive})
	svc := NewGroupViewGrantService(repo)

	// When
	groups, err := svc.ListViewableGroupsByUser(context.Background(), userID)

	// Then: only Anthropic+Active and OpenAI+Active returned, order preserved
	require.NoError(t, err)
	require.Len(t, groups, 2)
	assert.Equal(t, int64(1), groups[0].GroupID)
	assert.Equal(t, "Anthropic Active", groups[0].GroupName)
	assert.Equal(t, int64(2), groups[1].GroupID)
	assert.Equal(t, "OpenAI Active", groups[1].GroupName)
}

func TestListViewableGroupsByUser_AllFiltered_ReturnsEmpty(t *testing.T) {
	// Given: fake returns only gemini+active groups
	repo := newFakeGroupViewGrantRepo()
	userID := int64(1)
	repo.addViewableGroup(userID, ViewableGroup{GroupID: 3, GroupName: "Gemini Active", Platform: PlatformGemini, Status: StatusActive})
	svc := NewGroupViewGrantService(repo)

	// When
	groups, err := svc.ListViewableGroupsByUser(context.Background(), userID)

	// Then: empty slice returned
	require.NoError(t, err)
	assert.Empty(t, groups)
}

func TestListViewableGroupsByUser_RepoError_Propagates(t *testing.T) {
	// Given: repo returns error
	repo := newFakeGroupViewGrantRepo()
	repo.forceErr = errors.New("db down")
	svc := NewGroupViewGrantService(repo)

	// When
	groups, err := svc.ListViewableGroupsByUser(context.Background(), 1)

	// Then: error propagated
	require.Error(t, err)
	assert.Nil(t, groups)
	assert.Contains(t, err.Error(), "db down")
}
