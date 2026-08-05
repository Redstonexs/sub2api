//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

// pagedUserRepoStub serves users a page at a time so the scan's pagination and
// per-page batching can be exercised.
type pagedUserRepoStub struct {
	announcementUserRepoStub
	pageSize int
}

func (s *pagedUserRepoStub) ListWithFilters(
	_ context.Context, params pagination.PaginationParams, filters UserListFilters,
) ([]User, *pagination.PaginationResult, error) {
	s.recordedFilters = append(s.recordedFilters, filters)

	size := s.pageSize
	if size <= 0 {
		size = params.PageSize
	}
	start := (params.Page - 1) * size
	if start >= len(s.users) {
		return nil, &pagination.PaginationResult{Pages: (len(s.users) + size - 1) / size}, nil
	}
	end := min(start+size, len(s.users))
	return append([]User(nil), s.users[start:end]...),
		&pagination.PaginationResult{Pages: (len(s.users) + size - 1) / size}, nil
}

func subscribedTo(groupID int64) []UserSubscription {
	return []UserSubscription{{GroupID: groupID, ExpiresAt: time.Now().Add(time.Hour)}}
}

func groupTargeting(groupID int64) AnnouncementTargeting {
	return domain.AnnouncementTargeting{
		AnyOf: []domain.AnnouncementConditionGroup{{
			AllOf: []domain.AnnouncementCondition{{
				Type:     domain.AnnouncementConditionTypeSubscription,
				Operator: domain.AnnouncementOperatorIn,
				GroupIDs: []int64{groupID},
			}},
		}},
	}
}

func TestScanAnnouncementAudienceCountsEachStage(t *testing.T) {
	repo := &announcementUserRepoStub{users: []User{
		{ID: 1, Email: "match@example.com", Subscriptions: subscribedTo(10)},
		{ID: 2, Email: "unsubscribed@example.com", Subscriptions: subscribedTo(10)},
		{ID: 3, Email: "", Subscriptions: subscribedTo(10)},        // matches but unreachable
		{ID: 4, Email: "other@example.com"},                        // no subscription -> not matched
		{ID: 5, Email: "expired@example.com", Subscriptions: []UserSubscription{
			{GroupID: 10, ExpiresAt: time.Now().Add(-time.Hour)},
		}},
	}}
	emailSvc := NewNotificationEmailService(&settingRepoStub{values: map[string]string{
		notificationEmailPreferenceKey(
			NotificationEmailEventAnnouncementBroadcast, "unsubscribed@example.com"): "unsubscribed",
	}}, nil)

	var visited []AnnouncementRecipient
	stats, err := scanAnnouncementAudience(
		context.Background(), repo, emailSvc, groupTargeting(10), 0, nil,
		func(r AnnouncementRecipient) bool {
			visited = append(visited, r)
			return true
		},
	)
	require.NoError(t, err)

	require.Equal(t, int64(5), stats.Scanned)
	require.Equal(t, int64(3), stats.Matched, "expired and unsubscribed-group users must not match")
	require.Equal(t, int64(2), stats.WithEmail)
	require.Equal(t, int64(1), stats.Unsubscribed)
	require.Equal(t, int64(1), stats.Deliverable)
	require.False(t, stats.Truncated)

	require.Len(t, visited, 1)
	require.Equal(t, "match@example.com", visited[0].Email)
}

func TestScanAnnouncementAudienceOnlyScansActiveUsers(t *testing.T) {
	repo := &announcementUserRepoStub{users: []User{{ID: 1, Email: "a@example.com"}}}

	_, err := scanAnnouncementAudience(
		context.Background(), repo, nil, AnnouncementTargeting{}, 0, nil, nil,
	)
	require.NoError(t, err)
	require.Zero(t, repo.listCalls, "the filter-less List ignores user status")
	require.NotEmpty(t, repo.recordedFilters)
	require.Equal(t, StatusActive, repo.recordedFilters[0].Status)
}

func TestScanAnnouncementAudienceCountsWithoutVisiting(t *testing.T) {
	repo := &announcementUserRepoStub{users: []User{
		{ID: 1, Email: "a@example.com"},
		{ID: 2, Email: "b@example.com"},
	}}

	// visit == nil is the pure-counting mode the admin preview uses.
	stats, err := scanAnnouncementAudience(
		context.Background(), repo, nil, AnnouncementTargeting{}, 0, nil, nil,
	)
	require.NoError(t, err)
	require.Equal(t, int64(2), stats.Deliverable)
}

func TestScanAnnouncementAudienceAbortsWhenVisitReturnsFalse(t *testing.T) {
	repo := &announcementUserRepoStub{users: []User{
		{ID: 1, Email: "a@example.com"},
		{ID: 2, Email: "b@example.com"},
		{ID: 3, Email: "c@example.com"},
	}}

	count := 0
	stats, err := scanAnnouncementAudience(
		context.Background(), repo, nil, AnnouncementTargeting{}, 0, nil,
		func(AnnouncementRecipient) bool {
			count++
			return count < 2
		},
	)
	require.NoError(t, err)
	require.Equal(t, 2, count)
	require.Equal(t, int64(2), stats.Deliverable)
}

func TestScanAnnouncementAudienceMarksTruncatedAtMaxScan(t *testing.T) {
	users := make([]User, 0, 6)
	for i := 1; i <= 6; i++ {
		users = append(users, User{ID: int64(i), Email: "u@example.com"})
	}
	repo := &pagedUserRepoStub{announcementUserRepoStub: announcementUserRepoStub{users: users}, pageSize: 2}

	stats, err := scanAnnouncementAudience(
		context.Background(), repo, nil, AnnouncementTargeting{}, 2, nil, nil,
	)
	require.NoError(t, err)
	require.Equal(t, int64(2), stats.Scanned)
	require.True(t, stats.Truncated, "counts past the ceiling are lower bounds and must say so")
}

func TestScanAnnouncementAudienceNotTruncatedWhenEverythingFits(t *testing.T) {
	repo := &pagedUserRepoStub{
		announcementUserRepoStub: announcementUserRepoStub{users: []User{
			{ID: 1, Email: "a@example.com"},
			{ID: 2, Email: "b@example.com"},
		}},
		pageSize: 2,
	}

	stats, err := scanAnnouncementAudience(
		context.Background(), repo, nil, AnnouncementTargeting{}, 2, nil, nil,
	)
	require.NoError(t, err)
	require.Equal(t, int64(2), stats.Scanned)
	require.False(t, stats.Truncated)
}

func TestScanAnnouncementAudiencePagesThroughEveryUser(t *testing.T) {
	users := make([]User, 0, 5)
	for i := 1; i <= 5; i++ {
		users = append(users, User{ID: int64(i), Email: "u@example.com"})
	}
	repo := &pagedUserRepoStub{announcementUserRepoStub: announcementUserRepoStub{users: users}, pageSize: 2}

	stats, err := scanAnnouncementAudience(
		context.Background(), repo, nil, AnnouncementTargeting{}, 0, nil, nil,
	)
	require.NoError(t, err)
	require.Equal(t, int64(5), stats.Scanned)
	require.Len(t, repo.recordedFilters, 3)
}

func TestScanAnnouncementAudienceStopsOnClosedChannel(t *testing.T) {
	repo := &announcementUserRepoStub{users: []User{{ID: 1, Email: "a@example.com"}}}
	stop := make(chan struct{})
	close(stop)

	stats, err := scanAnnouncementAudience(
		context.Background(), repo, nil, AnnouncementTargeting{}, 0, stop, nil,
	)
	require.NoError(t, err)
	require.Zero(t, stats.Scanned)
}

func TestPreviewAudienceRejectsInvalidTargeting(t *testing.T) {
	svc := NewAnnouncementService(
		&announcementRepoStub{}, announcementReadRepoStub{},
		&announcementUserRepoStub{}, nil, nil, nil,
	)

	_, err := svc.PreviewAudience(context.Background(), domain.AnnouncementTargeting{
		AnyOf: []domain.AnnouncementConditionGroup{{AllOf: nil}},
	})
	require.ErrorIs(t, err, ErrAnnouncementInvalidTarget)
}

func TestPreviewAudienceMatchesWhatWouldBeSent(t *testing.T) {
	users := []User{
		{ID: 1, Email: "a@example.com", Subscriptions: subscribedTo(10)},
		{ID: 2, Email: "b@example.com", Subscriptions: subscribedTo(10)},
		{ID: 3, Email: "c@example.com"},
	}
	emailSvc := NewNotificationEmailService(&settingRepoStub{}, nil)

	svc := NewAnnouncementService(
		&announcementRepoStub{}, announcementReadRepoStub{},
		&announcementUserRepoStub{users: users}, nil, nil, emailSvc,
	)
	stats, err := svc.PreviewAudience(context.Background(), groupTargeting(10))
	require.NoError(t, err)
	require.Equal(t, int64(2), stats.Deliverable)

	// The broadcast walks the same scan, so the preview cannot drift from delivery.
	broadcaster := &AnnouncementBroadcastService{
		userRepo:                 &announcementUserRepoStub{users: users},
		notificationEmailService: emailSvc,
		jobs:                     make(chan announcementBroadcastJob, announcementBroadcastBuffer),
		stopCh:                   make(chan struct{}),
	}
	broadcaster.resolveAndEnqueue(1, "t", "<p>c</p>", AnnouncementSeverityInfo, groupTargeting(10))
	require.Len(t, broadcaster.jobs, int(stats.Deliverable))
}
