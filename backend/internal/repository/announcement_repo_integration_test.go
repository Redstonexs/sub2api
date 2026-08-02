//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// TestMigration192AddsSeverityAndBannerDefaults proves migration 192 actually applied
// and that the defaults let a pre-existing row read back sanely.
func TestMigration192AddsSeverityAndBannerDefaults(t *testing.T) {
	tx := testTx(t)
	ctx := context.Background()

	var id int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO announcements (title, content, status, notify_mode)
VALUES ('migration-192', 'body', 'active', 'silent')
RETURNING id
`).Scan(&id))

	var severity string
	var showBanner bool
	require.NoError(t, tx.QueryRowContext(ctx,
		`SELECT severity, show_banner FROM announcements WHERE id = $1`, id,
	).Scan(&severity, &showBanner))

	require.Equal(t, "info", severity, "existing rows must default to info, not empty")
	require.False(t, showBanner, "banner must be opt-in for existing rows")
}

func TestAnnouncementRepositoryRoundTripsSeverityAndBanner(t *testing.T) {
	client := testEntClient(t)
	repo := NewAnnouncementRepository(client)
	ctx := context.Background()

	a := &service.Announcement{
		Title:      "严重通知",
		Content:    "内容",
		Status:     service.AnnouncementStatusActive,
		NotifyMode: service.AnnouncementNotifyModeSilent,
		Severity:   service.AnnouncementSeverityCritical,
		ShowBanner: true,
	}
	require.NoError(t, repo.Create(ctx, a))
	require.NotZero(t, a.ID)

	got, err := repo.GetByID(ctx, a.ID)
	require.NoError(t, err)
	require.Equal(t, service.AnnouncementSeverityCritical, got.Severity)
	require.True(t, got.ShowBanner)

	got.Severity = service.AnnouncementSeverityWarning
	got.ShowBanner = false
	require.NoError(t, repo.Update(ctx, got))

	got, err = repo.GetByID(ctx, a.ID)
	require.NoError(t, err)
	require.Equal(t, service.AnnouncementSeverityWarning, got.Severity)
	require.False(t, got.ShowBanner)
}

func TestAnnouncementRepositoryListPublished(t *testing.T) {
	client := testEntClient(t)
	repo := NewAnnouncementRepository(client)
	ctx := context.Background()

	now := time.Now()
	past := now.Add(-48 * time.Hour)
	expired := now.Add(-time.Hour)
	future := now.Add(48 * time.Hour)

	mustCreate := func(title, status string, startsAt, endsAt *time.Time) int64 {
		a := &service.Announcement{
			Title: title, Content: "内容", Status: status,
			NotifyMode: service.AnnouncementNotifyModeSilent,
			Severity:   service.AnnouncementSeverityInfo,
			StartsAt:   startsAt, EndsAt: endsAt,
		}
		require.NoError(t, repo.Create(ctx, a))
		return a.ID
	}

	activeID := mustCreate("published-active", service.AnnouncementStatusActive, nil, nil)
	archivedID := mustCreate("published-archived", service.AnnouncementStatusArchived, &past, nil)
	expiredID := mustCreate("published-expired", service.AnnouncementStatusActive, &past, &expired)
	draftID := mustCreate("published-draft", service.AnnouncementStatusDraft, nil, nil)
	futureID := mustCreate("published-future", service.AnnouncementStatusActive, &future, nil)

	items, err := repo.ListPublished(ctx, now, 500)
	require.NoError(t, err)

	found := map[int64]bool{}
	for i := range items {
		found[items[i].ID] = true
	}
	// Archived and expired notices stay readable in the archive; drafts and
	// not-yet-started announcements were never published.
	require.True(t, found[activeID])
	require.True(t, found[archivedID])
	require.True(t, found[expiredID])
	require.False(t, found[draftID])
	require.False(t, found[futureID])

	// ListActive is stricter: live delivery only.
	active, err := repo.ListActive(ctx, now)
	require.NoError(t, err)
	activeFound := map[int64]bool{}
	for i := range active {
		activeFound[active[i].ID] = true
	}
	require.True(t, activeFound[activeID])
	require.False(t, activeFound[archivedID])
	require.False(t, activeFound[expiredID])
}

func TestAnnouncementRepositorySortsBySeverity(t *testing.T) {
	client := testEntClient(t)
	repo := NewAnnouncementRepository(client)
	ctx := context.Background()

	_, _, err := repo.List(ctx,
		pagination.PaginationParams{Page: 1, PageSize: 10, SortBy: "severity", SortOrder: "asc"},
		service.AnnouncementListFilters{},
	)
	require.NoError(t, err, "sort_by=severity must map to a real column")
}
