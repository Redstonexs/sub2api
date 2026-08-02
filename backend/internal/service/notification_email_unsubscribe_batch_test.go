//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsUnsubscribedBatchMatchesSingleLookup(t *testing.T) {
	ctx := context.Background()
	event := NotificationEmailEventAnnouncementBroadcast

	svc := NewNotificationEmailService(settingRepoStub{values: map[string]string{
		notificationEmailPreferenceKey(event, "v2@example.com"):           "unsubscribed",
		legacyNotificationEmailPreferenceKey(event, "legacy@example.com"): "unsubscribed",
		notificationEmailPreferenceKey(event, "resubscribed@example.com"): "subscribed",
	}}, nil)

	emails := []string{
		"v2@example.com",
		"legacy@example.com",
		"resubscribed@example.com",
		"unknown@example.com",
	}

	got, err := svc.IsUnsubscribedBatch(ctx, emails, event)
	require.NoError(t, err)

	for _, email := range emails {
		want, err := svc.IsUnsubscribed(ctx, email, event)
		require.NoError(t, err)
		require.Equal(t, want, got[normalizeNotificationEmailKey(email)], "mismatch for %s", email)
	}

	require.True(t, got["v2@example.com"])
	require.True(t, got["legacy@example.com"], "the legacy preference key must still be honoured")
	require.False(t, got["resubscribed@example.com"])
	require.False(t, got["unknown@example.com"], "a recipient with no stored preference is subscribed")
}

func TestIsUnsubscribedBatchPrefersV2KeyOverLegacy(t *testing.T) {
	ctx := context.Background()
	event := NotificationEmailEventAnnouncementBroadcast

	// Both keys present and disagreeing: v2 wins, matching IsUnsubscribed's ordering.
	svc := NewNotificationEmailService(settingRepoStub{values: map[string]string{
		notificationEmailPreferenceKey(event, "both@example.com"):       "subscribed",
		legacyNotificationEmailPreferenceKey(event, "both@example.com"): "unsubscribed",
	}}, nil)

	got, err := svc.IsUnsubscribedBatch(ctx, []string{"both@example.com"}, event)
	require.NoError(t, err)
	require.False(t, got["both@example.com"])
}

func TestIsUnsubscribedBatchNormalizesAndDedupes(t *testing.T) {
	ctx := context.Background()
	event := NotificationEmailEventAnnouncementBroadcast

	svc := NewNotificationEmailService(settingRepoStub{values: map[string]string{
		notificationEmailPreferenceKey(event, "user@example.com"): "unsubscribed",
	}}, nil)

	got, err := svc.IsUnsubscribedBatch(ctx,
		[]string{"  User@Example.com  ", "user@example.com", "", "   "}, event)
	require.NoError(t, err)
	require.Len(t, got, 1, "blank inputs are skipped and case/whitespace variants collapse to one key")
	require.True(t, got["user@example.com"])
}

func TestIsUnsubscribedBatchShortCircuitsTransactionalEvents(t *testing.T) {
	ctx := context.Background()

	// A transactional (non-optional) event cannot be unsubscribed from, so the
	// batch must return all-false without touching the settings repository.
	svc := NewNotificationEmailService(settingRepoStub{err: context.Canceled}, nil)

	got, err := svc.IsUnsubscribedBatch(ctx, []string{"user@example.com"}, NotificationEmailEventAuthPasswordReset)
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestIsUnsubscribedBatchPropagatesErrors(t *testing.T) {
	svc := NewNotificationEmailService(settingRepoStub{err: context.Canceled}, nil)

	_, err := svc.IsUnsubscribedBatch(context.Background(),
		[]string{"user@example.com"}, NotificationEmailEventAnnouncementBroadcast)
	require.ErrorIs(t, err, context.Canceled)

	_, err = svc.IsUnsubscribedBatch(context.Background(), []string{"user@example.com"}, "not-a-real-event")
	require.Error(t, err)
}
