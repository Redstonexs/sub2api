//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func newAnnouncementTestEmailFixture(t *testing.T, ann *Announcement, admin User) (
	*AnnouncementService, *notificationEmailTestSMTPServer,
) {
	t.Helper()
	ctx := context.Background()

	repo := newNotificationEmailMemorySettingRepo()
	smtpServer := startNotificationEmailTestSMTPServer(t)
	require.NoError(t, repo.SetMultiple(ctx, smtpServer.settings()))

	emailSvc := NewNotificationEmailService(repo, NewEmailService(repo, nil))
	userRepo := &announcementUserRepoStub{users: []User{admin}}
	broadcaster := &AnnouncementBroadcastService{
		userRepo:                 userRepo,
		notificationEmailService: emailSvc,
		jobs:                     make(chan announcementBroadcastJob, announcementBroadcastBuffer),
		stopCh:                   make(chan struct{}),
	}

	svc := NewAnnouncementService(
		&announcementRepoStub{item: ann}, announcementReadRepoStub{},
		userRepo, nil, broadcaster, emailSvc,
	)
	return svc, smtpServer
}

func TestSendTestEmailDeliversTwiceToTheSameAdmin(t *testing.T) {
	ann := &Announcement{ID: 7, Title: "公告", Content: "内容", Severity: AnnouncementSeverityWarning}
	admin := User{ID: 42, Email: "admin@example.com", Username: "admin"}
	svc, smtpServer := newAnnouncementTestEmailFixture(t, ann, admin)

	recipient, err := svc.SendTestEmail(context.Background(), ann.ID, admin.ID)
	require.NoError(t, err)
	require.Equal(t, "admin@example.com", recipient)
	require.Equal(t, int64(1), smtpServer.messageCount())

	// A nonce ReminderKey is what stops the second test from deduping against the
	// first; without it the admin would silently receive nothing.
	_, err = svc.SendTestEmail(context.Background(), ann.ID, admin.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), smtpServer.messageCount())
}

func TestSendTestEmailDoesNotConsumeTheRealBroadcastDedupSlot(t *testing.T) {
	ctx := context.Background()
	ann := &Announcement{ID: 7, Title: "公告", Content: "内容"}
	admin := User{ID: 42, Email: "admin@example.com", Username: "admin"}
	svc, smtpServer := newAnnouncementTestEmailFixture(t, ann, admin)

	_, err := svc.SendTestEmail(ctx, ann.ID, admin.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), smtpServer.messageCount())

	// The real broadcast uses SourceType "announcement"; a test send must not have
	// burned that recipient's slot.
	svc.broadcaster.processJob(0, announcementBroadcastJob{
		announcementID: ann.ID,
		title:          ann.Title,
		contentHTML:    "<p>内容</p>",
		userID:         admin.ID,
		email:          admin.Email,
		name:           admin.Username,
	})
	require.Equal(t, int64(2), smtpServer.messageCount())
}

func TestSendTestEmailRefusesWhenTheAdminUnsubscribed(t *testing.T) {
	ctx := context.Background()
	ann := &Announcement{ID: 7, Title: "公告", Content: "内容"}
	admin := User{ID: 42, Email: "admin@example.com", Username: "admin"}

	repo := newNotificationEmailMemorySettingRepo()
	require.NoError(t, repo.Set(ctx,
		notificationEmailPreferenceKey(NotificationEmailEventAnnouncementBroadcast, admin.Email),
		"unsubscribed"))

	emailSvc := NewNotificationEmailService(repo, NewEmailService(repo, nil))
	userRepo := &announcementUserRepoStub{users: []User{admin}}
	svc := NewAnnouncementService(
		&announcementRepoStub{item: ann}, announcementReadRepoStub{}, userRepo, nil,
		&AnnouncementBroadcastService{
			userRepo:                 userRepo,
			notificationEmailService: emailSvc,
			jobs:                     make(chan announcementBroadcastJob, 1),
			stopCh:                   make(chan struct{}),
		},
		emailSvc,
	)

	// Send() silently no-ops for an unsubscribed recipient on an optional event, so
	// without the pre-check the admin would see a false success.
	_, err := svc.SendTestEmail(ctx, ann.ID, admin.ID)
	require.ErrorIs(t, err, ErrAnnouncementTestEmailUnsubscribed)
}

func TestSendTestEmailRequiresAnAdminAddress(t *testing.T) {
	ann := &Announcement{ID: 7, Title: "公告", Content: "内容"}
	admin := User{ID: 42, Email: "  ", Username: "admin"}
	svc, _ := newAnnouncementTestEmailFixture(t, ann, admin)

	_, err := svc.SendTestEmail(context.Background(), ann.ID, admin.ID)
	require.ErrorIs(t, err, ErrAnnouncementTestEmailUnavailable)
}

func TestSendTestEmailWithoutBroadcasterIsUnavailable(t *testing.T) {
	svc := NewAnnouncementService(
		&announcementRepoStub{item: &Announcement{ID: 7, Title: "公告", Content: "内容"}},
		announcementReadRepoStub{},
		&announcementUserRepoStub{users: []User{{ID: 42, Email: "admin@example.com"}}},
		nil, nil, nil,
	)

	_, err := svc.SendTestEmail(context.Background(), 7, 42)
	require.ErrorIs(t, err, ErrAnnouncementTestEmailUnavailable)
}
