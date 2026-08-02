package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestActiveSubscriptionGroupIDsFiltersExpiredAndDedupes(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	subs := []UserSubscription{
		{GroupID: 1, ExpiresAt: now.Add(time.Hour)},     // active
		{GroupID: 2, ExpiresAt: now.Add(-time.Hour)},    // expired -> excluded
		{GroupID: 1, ExpiresAt: now.Add(2 * time.Hour)}, // duplicate group -> deduped
		{GroupID: 3, ExpiresAt: now.Add(time.Minute)},   // active
	}

	got := activeSubscriptionGroupIDs(subs, now)

	if len(got) != 2 {
		t.Fatalf("expected 2 active groups, got %d (%v)", len(got), got)
	}
	if _, ok := got[1]; !ok {
		t.Fatalf("expected group 1 present, got %v", got)
	}
	if _, ok := got[3]; !ok {
		t.Fatalf("expected group 3 present, got %v", got)
	}
	if _, ok := got[2]; ok {
		t.Fatalf("expected expired group 2 to be excluded, got %v", got)
	}
}

func TestActiveSubscriptionGroupIDsEmpty(t *testing.T) {
	if got := activeSubscriptionGroupIDs(nil, time.Now()); got != nil {
		t.Fatalf("expected nil for no subscriptions, got %v", got)
	}
}

func TestAnnouncementBroadcastSkipsUnsubscribedRecipients(t *testing.T) {
	ctx := context.Background()
	settingKey := notificationEmailPreferenceKey(NotificationEmailEventAnnouncementBroadcast, "user1@example.com")
	notificationEmailService := NewNotificationEmailService(settingRepoStub{values: map[string]string{
		settingKey: "unsubscribed",
	}}, nil)

	unsubscribed, err := notificationEmailService.IsUnsubscribed(ctx, "user1@example.com", NotificationEmailEventAnnouncementBroadcast)
	require.NoError(t, err)
	require.True(t, unsubscribed)

	svc := &AnnouncementBroadcastService{
		userRepo: &announcementUserRepoStub{users: []User{
			{ID: 1, Email: "user1@example.com", Username: "unsubscribed", Balance: 100, Subscriptions: []UserSubscription{{GroupID: 10, ExpiresAt: time.Now().Add(time.Hour)}}},
			{ID: 2, Email: "user2@example.com", Username: "subscribed", Balance: 100, Subscriptions: []UserSubscription{{GroupID: 10, ExpiresAt: time.Now().Add(time.Hour)}}},
		}},
		notificationEmailService: notificationEmailService,
		jobs:                     make(chan announcementBroadcastJob, announcementBroadcastBuffer),
		stopCh:                   make(chan struct{}),
	}

	targeting := AnnouncementTargeting{
		AnyOf: []domain.AnnouncementConditionGroup{{
			AllOf: []domain.AnnouncementCondition{
				{
					Type:     domain.AnnouncementConditionTypeBalance,
					Operator: domain.AnnouncementOperatorGTE,
					Value:    100,
				},
				{
					Type:     domain.AnnouncementConditionTypeSubscription,
					Operator: domain.AnnouncementOperatorIn,
					GroupIDs: []int64{10},
				},
			},
		}},
	}

	svc.resolveAndEnqueue(99, "公告", "<p>内容</p>", AnnouncementSeverityInfo, targeting)

	enqueued := make(map[int64]announcementBroadcastJob)
	for {
		select {
		case job := <-svc.jobs:
			enqueued[job.userID] = job
		default:
			goto drained
		}
	}

drained:
	require.Len(t, enqueued, 1)
	require.Contains(t, enqueued, int64(2))
	require.NotContains(t, enqueued, int64(1))
}

func TestAnnouncementBroadcastOnlyTargetsActiveUsers(t *testing.T) {
	userRepo := &announcementUserRepoStub{users: []User{
		{ID: 1, Email: "user1@example.com", Username: "user1", Balance: 100},
	}}
	svc := &AnnouncementBroadcastService{
		userRepo:                 userRepo,
		notificationEmailService: NewNotificationEmailService(settingRepoStub{}, nil),
		jobs:                     make(chan announcementBroadcastJob, announcementBroadcastBuffer),
		stopCh:                   make(chan struct{}),
	}

	svc.resolveAndEnqueue(99, "公告", "<p>内容</p>", AnnouncementSeverityInfo, AnnouncementTargeting{})

	require.Zero(t, userRepo.listCalls, "broadcast must not use the filter-less List, which ignores user status")
	require.Len(t, userRepo.recordedFilters, 1)
	require.Equal(t, StatusActive, userRepo.recordedFilters[0].Status,
		"disabled and banned users must be excluded from an announcement broadcast")
}
