package service

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type announcementRepoStub struct {
	item *Announcement
	// published backs ListPublished; entries are expected newest-first (id DESC).
	published []Announcement
}

func (s *announcementRepoStub) Create(_ context.Context, a *Announcement) error {
	s.item = a
	return nil
}

func (s *announcementRepoStub) GetByID(_ context.Context, _ int64) (*Announcement, error) {
	if s.item == nil {
		return nil, ErrAnnouncementNotFound
	}
	return s.item, nil
}

func (s *announcementRepoStub) Update(_ context.Context, a *Announcement) error {
	s.item = a
	return nil
}

func (*announcementRepoStub) Delete(context.Context, int64) error { return nil }
func (*announcementRepoStub) List(context.Context, pagination.PaginationParams, AnnouncementListFilters) ([]Announcement, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *announcementRepoStub) ListPublished(_ context.Context, now time.Time, _ int) ([]Announcement, error) {
	out := make([]Announcement, 0, len(s.published))
	for i := range s.published {
		a := s.published[i]
		// Mirrors the repository predicate: active or archived, already started.
		if a.Status != AnnouncementStatusActive && a.Status != AnnouncementStatusArchived {
			continue
		}
		if a.StartsAt != nil && a.StartsAt.After(now) {
			continue
		}
		out = append(out, a)
	}
	return out, nil
}
func (*announcementRepoStub) ListActive(context.Context, time.Time) ([]Announcement, error) {
	return nil, nil
}

type announcementUserRepoStub struct {
	users []User
	// recordedFilters captures every UserListFilters passed to ListWithFilters so a
	// test can assert which filters a caller applies.
	recordedFilters []UserListFilters
	// listCalls counts the filter-less List calls, which apply no status filter.
	listCalls int
}

func (s *announcementUserRepoStub) Create(context.Context, *User) error { return nil }

func (s *announcementUserRepoStub) CreateWithEmailAliasGuard(context.Context, *User) error {
	return nil
}

func (s *announcementUserRepoStub) GetByID(_ context.Context, id int64) (*User, error) {
	for i := range s.users {
		if s.users[i].ID == id {
			user := s.users[i]
			return &user, nil
		}
	}
	return nil, ErrUserNotFound
}
func (s *announcementUserRepoStub) GetByIDIncludeDeleted(context.Context, int64) (*User, error) {
	return nil, ErrUserNotFound
}
func (s *announcementUserRepoStub) GetByEmail(context.Context, string) (*User, error) {
	return nil, ErrUserNotFound
}
func (s *announcementUserRepoStub) GetFirstAdmin(context.Context) (*User, error) {
	return nil, ErrUserNotFound
}
func (s *announcementUserRepoStub) Update(context.Context, *User, UserUpdateFields) error {
	return nil
}
func (s *announcementUserRepoStub) Delete(context.Context, int64) error { return nil }
func (s *announcementUserRepoStub) GetUserAvatar(context.Context, int64) (*UserAvatar, error) {
	return nil, nil
}
func (s *announcementUserRepoStub) UpsertUserAvatar(context.Context, int64, UpsertUserAvatarInput) (*UserAvatar, error) {
	return nil, nil
}
func (s *announcementUserRepoStub) DeleteUserAvatar(context.Context, int64) error { return nil }
func (s *announcementUserRepoStub) List(_ context.Context, _ pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	s.listCalls++
	return append([]User(nil), s.users...), &pagination.PaginationResult{}, nil
}
func (s *announcementUserRepoStub) ListWithFilters(_ context.Context, _ pagination.PaginationParams, filters UserListFilters) ([]User, *pagination.PaginationResult, error) {
	s.recordedFilters = append(s.recordedFilters, filters)
	if filters.Search == "" {
		return append([]User(nil), s.users...), &pagination.PaginationResult{}, nil
	}
	out := make([]User, 0, len(s.users))
	for i := range s.users {
		if s.users[i].Email == filters.Search || s.users[i].Username == filters.Search {
			out = append(out, s.users[i])
		}
	}
	return out, &pagination.PaginationResult{}, nil
}
func (s *announcementUserRepoStub) GetLatestUsedAtByUserIDs(context.Context, []int64) (map[int64]*time.Time, error) {
	return nil, nil
}
func (s *announcementUserRepoStub) GetLatestUsedAtByUserID(context.Context, int64) (*time.Time, error) {
	return nil, nil
}
func (s *announcementUserRepoStub) UpdateUserLastActiveAt(context.Context, int64, time.Time) error {
	return nil
}
func (s *announcementUserRepoStub) UpdateBalance(context.Context, int64, float64) error { return nil }
func (s *announcementUserRepoStub) DeductBalance(context.Context, int64, float64) error { return nil }
func (s *announcementUserRepoStub) AdjustBalance(context.Context, int64, float64) (BalanceChange, error) {
	panic("unexpected AdjustBalance call")
}
func (s *announcementUserRepoStub) SetBalance(context.Context, int64, float64) (BalanceChange, error) {
	panic("unexpected SetBalance call")
}
func (s *announcementUserRepoStub) UpdateConcurrency(context.Context, int64, int) error { return nil }
func (s *announcementUserRepoStub) BatchSetConcurrency(context.Context, []int64, int) (int, error) {
	return 0, nil
}
func (s *announcementUserRepoStub) BatchAddConcurrency(context.Context, []int64, int) (int, error) {
	return 0, nil
}
func (s *announcementUserRepoStub) ExistsByEmail(context.Context, string) (bool, error) {
	return false, nil
}

func (s *announcementUserRepoStub) ExistsByEmailAlias(context.Context, string) (bool, error) {
	return false, nil
}
func (s *announcementUserRepoStub) RemoveGroupFromAllowedGroups(context.Context, int64) (int64, error) {
	return 0, nil
}
func (s *announcementUserRepoStub) AddGroupToAllowedGroups(context.Context, int64, int64) error {
	return nil
}
func (s *announcementUserRepoStub) RemoveGroupFromUserAllowedGroups(context.Context, int64, int64) error {
	return nil
}
func (s *announcementUserRepoStub) ListUserAuthIdentities(context.Context, int64) ([]UserAuthIdentityRecord, error) {
	return nil, nil
}
func (s *announcementUserRepoStub) UnbindUserAuthProvider(context.Context, int64, string) error {
	return nil
}
func (s *announcementUserRepoStub) UpdateTotpSecret(context.Context, int64, *string) error {
	return nil
}
func (s *announcementUserRepoStub) EnableTotp(context.Context, int64) error  { return nil }
func (s *announcementUserRepoStub) DisableTotp(context.Context, int64) error { return nil }
func (s *announcementUserRepoStub) BatchUpdateLimits(context.Context, []int64, *int, *int) (int, error) {
	return 0, nil
}

type userSubRepoStub struct{}

func (userSubRepoStub) Create(context.Context, *UserSubscription) error           { return nil }
func (userSubRepoStub) GetByID(context.Context, int64) (*UserSubscription, error) { return nil, nil }
func (userSubRepoStub) GetByIDForUpdate(context.Context, int64) (*UserSubscription, error) {
	return nil, nil
}
func (userSubRepoStub) GetByIDIncludeDeleted(context.Context, int64) (*UserSubscription, error) {
	return nil, nil
}
func (userSubRepoStub) Restore(context.Context, int64, string) (*UserSubscription, error) {
	return nil, nil
}
func (userSubRepoStub) GetByUserIDAndGroupID(context.Context, int64, int64) (*UserSubscription, error) {
	return nil, nil
}
func (userSubRepoStub) GetActiveByUserIDAndGroupID(context.Context, int64, int64) (*UserSubscription, error) {
	return nil, nil
}
func (userSubRepoStub) Update(context.Context, *UserSubscription) error { return nil }
func (userSubRepoStub) Delete(context.Context, int64) error             { return nil }
func (userSubRepoStub) ListByUserID(context.Context, int64) ([]UserSubscription, error) {
	return nil, nil
}
func (userSubRepoStub) ListActiveByUserID(_ context.Context, userID int64) ([]UserSubscription, error) {
	if userID == 1 || userID == 2 {
		return []UserSubscription{{GroupID: 10}}, nil
	}
	return nil, nil
}
func (userSubRepoStub) ListByGroupID(context.Context, int64, pagination.PaginationParams) ([]UserSubscription, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (userSubRepoStub) List(context.Context, pagination.PaginationParams, *int64, *int64, string, string, string, string) ([]UserSubscription, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (userSubRepoStub) ExistsByUserIDAndGroupID(context.Context, int64, int64) (bool, error) {
	return false, nil
}
func (userSubRepoStub) ExistsActiveByUserIDAndGroupID(context.Context, int64, int64) (bool, error) {
	return false, nil
}
func (userSubRepoStub) ExtendExpiry(context.Context, int64, time.Time) error    { return nil }
func (userSubRepoStub) UpdateStatus(context.Context, int64, string) error       { return nil }
func (userSubRepoStub) UpdateNotes(context.Context, int64, string) error        { return nil }
func (userSubRepoStub) ActivateWindows(context.Context, int64, time.Time) error { return nil }
func (userSubRepoStub) ResetUsageWindows(context.Context, int64, bool, bool, bool, time.Time) error {
	return nil
}
func (userSubRepoStub) ResetDailyUsage(context.Context, int64, *time.Time, time.Time) error {
	return nil
}
func (userSubRepoStub) ResetWeeklyUsage(context.Context, int64, *time.Time, time.Time) error {
	return nil
}
func (userSubRepoStub) ResetMonthlyUsage(context.Context, int64, *time.Time, time.Time) error {
	return nil
}
func (userSubRepoStub) IncrementUsage(context.Context, int64, float64) error    { return nil }
func (userSubRepoStub) BatchUpdateExpiredStatus(context.Context) (int64, error) { return 0, nil }

type announcementReadRepoStub struct{}

func (announcementReadRepoStub) MarkRead(context.Context, int64, int64, time.Time) error { return nil }
func (announcementReadRepoStub) GetReadMapByUser(context.Context, int64, []int64) (map[int64]time.Time, error) {
	return map[int64]time.Time{}, nil
}
func (announcementReadRepoStub) GetReadMapByUsers(context.Context, int64, []int64) (map[int64]time.Time, error) {
	return map[int64]time.Time{}, nil
}
func (announcementReadRepoStub) CountByAnnouncementID(context.Context, int64) (int64, error) {
	return 0, nil
}

func TestAnnouncementServiceListUserReadStatusReflectsUnsubscribe(t *testing.T) {
	ctx := context.Background()
	ann := &Announcement{
		ID:         99,
		Title:      "公告",
		Content:    "内容",
		Status:     AnnouncementStatusActive,
		NotifyMode: AnnouncementNotifyModeEmail,
		Targeting: domain.AnnouncementTargeting{
			AnyOf: []domain.AnnouncementConditionGroup{{
				AllOf: []domain.AnnouncementCondition{{
					Type:     domain.AnnouncementConditionTypeBalance,
					Operator: domain.AnnouncementOperatorGTE,
					Value:    100,
				}, {
					Type:     domain.AnnouncementConditionTypeSubscription,
					Operator: domain.AnnouncementOperatorIn,
					GroupIDs: []int64{10},
				}},
			}},
		},
	}
	svc := NewAnnouncementService(
		&announcementRepoStub{item: ann},
		announcementReadRepoStub{},
		&announcementUserRepoStub{users: []User{
			{ID: 1, Email: "user1@example.com", Username: "unsubscribed", Balance: 100},
			{ID: 2, Email: "user2@example.com", Username: "subscribed", Balance: 100},
		}},
		userSubRepoStub{},
		nil,
		NewNotificationEmailService(&settingRepoStub{values: map[string]string{
			notificationEmailPreferenceKey(NotificationEmailEventAnnouncementBroadcast, "user1@example.com"): "unsubscribed",
		}}, nil),
	)

	statuses, _, err := svc.ListUserReadStatus(ctx, ann.ID, pagination.PaginationParams{Page: 1, PageSize: 10}, "")
	require.NoError(t, err)
	require.Len(t, statuses, 2)
	require.True(t, statuses[0].AnnouncementEmailUnsubscribed)
	require.False(t, statuses[1].AnnouncementEmailUnsubscribed)

	statusType := reflect.TypeOf(statuses[0])
	field, ok := statusType.FieldByName("AnnouncementEmailUnsubscribed")
	require.True(t, ok, "expected AnnouncementUserReadStatus to expose AnnouncementEmailUnsubscribed")
	require.Equal(t, reflect.Bool, field.Type.Kind())

	svcType := reflect.TypeOf(*svc)
	serviceField, ok := svcType.FieldByName("notificationEmailService")
	require.True(t, ok, "expected AnnouncementService to depend on NotificationEmailService")
	require.Equal(t, "*service.NotificationEmailService", serviceField.Type.String())
}

func TestAnnouncementServiceListUserReadStatusWrapsUnsubscribeErrors(t *testing.T) {
	ctx := context.Background()
	ann := &Announcement{ID: 99, Status: AnnouncementStatusActive, NotifyMode: AnnouncementNotifyModeEmail}
	svc := NewAnnouncementService(
		&announcementRepoStub{item: ann},
		announcementReadRepoStub{},
		&announcementUserRepoStub{users: []User{{ID: 1, Email: "user@example.com", Username: "user", Balance: 100}}},
		userSubRepoStub{},
		nil,
		&NotificationEmailService{settingRepo: &settingRepoStub{err: context.Canceled}},
	)

	_, _, err := svc.ListUserReadStatus(ctx, ann.ID, pagination.PaginationParams{Page: 1, PageSize: 10}, "")
	require.Error(t, err)
	require.ErrorContains(t, err, "check unsubscribe status")
}

func TestAnnouncementServiceCreateRejectsEqualStartEndTimes(t *testing.T) {
	repo := &announcementRepoStub{}
	svc := NewAnnouncementService(repo, nil, nil, nil, nil, nil)
	now := time.Unix(1776790020, 0)

	_, err := svc.Create(context.Background(), &CreateAnnouncementInput{
		Title:      "公告",
		Content:    "内容",
		Status:     AnnouncementStatusActive,
		NotifyMode: AnnouncementNotifyModePopup,
		StartsAt:   &now,
		EndsAt:     &now,
	})
	require.ErrorIs(t, err, ErrAnnouncementInvalidSchedule)
}

func TestAnnouncementServiceUpdateRejectsEqualStartEndTimes(t *testing.T) {
	repo := &announcementRepoStub{
		item: &Announcement{
			ID:         1,
			Title:      "公告",
			Content:    "内容",
			Status:     AnnouncementStatusActive,
			NotifyMode: AnnouncementNotifyModePopup,
		},
	}
	svc := NewAnnouncementService(repo, nil, nil, nil, nil, nil)
	now := time.Unix(1776790020, 0)
	startsAt := &now
	endsAt := &now

	_, err := svc.Update(context.Background(), 1, &UpdateAnnouncementInput{
		StartsAt: &startsAt,
		EndsAt:   &endsAt,
	})
	require.ErrorIs(t, err, ErrAnnouncementInvalidSchedule)
}

// countingUserSubRepoStub records ListActiveByUserID calls so a test can assert
// that a code path is not issuing one query per user.
type countingUserSubRepoStub struct {
	userSubRepoStub
	listActiveCalls int
}

func (s *countingUserSubRepoStub) ListActiveByUserID(ctx context.Context, userID int64) ([]UserSubscription, error) {
	s.listActiveCalls++
	return s.userSubRepoStub.ListActiveByUserID(ctx, userID)
}

func TestAnnouncementServiceTitleLimitCountsRunesNotBytes(t *testing.T) {
	ctx := context.Background()

	// 100 CJK characters occupy 300 bytes; a byte-based cap rejected this.
	cjkTitle := strings.Repeat("公", 100)
	require.Len(t, []byte(cjkTitle), 300)

	created, err := NewAnnouncementService(&announcementRepoStub{}, nil, nil, nil, nil, nil).
		Create(ctx, &CreateAnnouncementInput{Title: cjkTitle, Content: "内容"})
	require.NoError(t, err)
	require.Equal(t, cjkTitle, created.Title)

	_, err = NewAnnouncementService(&announcementRepoStub{}, nil, nil, nil, nil, nil).
		Create(ctx, &CreateAnnouncementInput{Title: strings.Repeat("公", announcementMaxTitleRunes), Content: "内容"})
	require.NoError(t, err)

	_, err = NewAnnouncementService(&announcementRepoStub{}, nil, nil, nil, nil, nil).
		Create(ctx, &CreateAnnouncementInput{Title: strings.Repeat("a", announcementMaxTitleRunes+1), Content: "内容"})
	require.ErrorIs(t, err, ErrAnnouncementInvalidTitle)
}

func TestAnnouncementServiceUpdateTitleLimitCountsRunesNotBytes(t *testing.T) {
	ctx := context.Background()
	cjkTitle := strings.Repeat("公", 100)

	repo := &announcementRepoStub{item: &Announcement{ID: 1, Title: "旧", Content: "内容"}}
	updated, err := NewAnnouncementService(repo, nil, nil, nil, nil, nil).
		Update(ctx, 1, &UpdateAnnouncementInput{Title: &cjkTitle})
	require.NoError(t, err)
	require.Equal(t, cjkTitle, updated.Title)

	tooLong := strings.Repeat("a", announcementMaxTitleRunes+1)
	_, err = NewAnnouncementService(
		&announcementRepoStub{item: &Announcement{ID: 1, Title: "旧", Content: "内容"}}, nil, nil, nil, nil, nil).
		Update(ctx, 1, &UpdateAnnouncementInput{Title: &tooLong})
	require.ErrorIs(t, err, ErrAnnouncementInvalidTitle)
}

func TestAnnouncementServiceContentLimit(t *testing.T) {
	ctx := context.Background()

	atLimit := strings.Repeat("公", announcementMaxContentRunes)
	_, err := NewAnnouncementService(&announcementRepoStub{}, nil, nil, nil, nil, nil).
		Create(ctx, &CreateAnnouncementInput{Title: "公告", Content: atLimit})
	require.NoError(t, err)

	overLimit := strings.Repeat("公", announcementMaxContentRunes+1)
	_, err = NewAnnouncementService(&announcementRepoStub{}, nil, nil, nil, nil, nil).
		Create(ctx, &CreateAnnouncementInput{Title: "公告", Content: overLimit})
	require.ErrorIs(t, err, ErrAnnouncementContentTooLong)

	_, err = NewAnnouncementService(
		&announcementRepoStub{item: &Announcement{ID: 1, Title: "公告", Content: "内容"}}, nil, nil, nil, nil, nil).
		Update(ctx, 1, &UpdateAnnouncementInput{Content: &overLimit})
	require.ErrorIs(t, err, ErrAnnouncementContentTooLong)

	blank := "   "
	_, err = NewAnnouncementService(
		&announcementRepoStub{item: &Announcement{ID: 1, Title: "公告", Content: "内容"}}, nil, nil, nil, nil, nil).
		Update(ctx, 1, &UpdateAnnouncementInput{Content: &blank})
	require.ErrorIs(t, err, ErrAnnouncementContentRequired)
}

func TestAnnouncementServiceListUserReadStatusIssuesNoPerUserQueries(t *testing.T) {
	ctx := context.Background()
	future := time.Now().Add(24 * time.Hour)
	ann := &Announcement{
		ID:     99,
		Status: AnnouncementStatusActive,
		Targeting: domain.AnnouncementTargeting{
			AnyOf: []domain.AnnouncementConditionGroup{{
				AllOf: []domain.AnnouncementCondition{{
					Type:     domain.AnnouncementConditionTypeSubscription,
					Operator: domain.AnnouncementOperatorIn,
					GroupIDs: []int64{10},
				}},
			}},
		},
	}

	subRepo := &countingUserSubRepoStub{}
	svc := NewAnnouncementService(
		&announcementRepoStub{item: ann},
		announcementReadRepoStub{},
		&announcementUserRepoStub{users: []User{
			// Eligibility must come from the eager-loaded subscriptions, not a per-user query.
			{ID: 1, Email: "in@example.com", Balance: 0, Subscriptions: []UserSubscription{
				{GroupID: 10, ExpiresAt: future, Status: SubscriptionStatusActive},
			}},
			{ID: 2, Email: "out@example.com", Balance: 0},
		}},
		subRepo,
		nil,
		NewNotificationEmailService(&settingRepoStub{}, nil),
	)

	statuses, _, err := svc.ListUserReadStatus(ctx, ann.ID, pagination.PaginationParams{Page: 1, PageSize: 10}, "")
	require.NoError(t, err)
	require.Len(t, statuses, 2)
	require.True(t, statuses[0].Eligible)
	require.False(t, statuses[1].Eligible)
	require.Zero(t, subRepo.listActiveCalls, "ListUserReadStatus must not query subscriptions per user")
}

func TestAnnouncementServiceListUserReadStatusIgnoresExpiredSubscriptions(t *testing.T) {
	ctx := context.Background()
	ann := &Announcement{
		ID:     99,
		Status: AnnouncementStatusActive,
		Targeting: domain.AnnouncementTargeting{
			AnyOf: []domain.AnnouncementConditionGroup{{
				AllOf: []domain.AnnouncementCondition{{
					Type:     domain.AnnouncementConditionTypeSubscription,
					Operator: domain.AnnouncementOperatorIn,
					GroupIDs: []int64{10},
				}},
			}},
		},
	}

	svc := NewAnnouncementService(
		&announcementRepoStub{item: ann},
		announcementReadRepoStub{},
		&announcementUserRepoStub{users: []User{
			{ID: 1, Email: "expired@example.com", Subscriptions: []UserSubscription{
				{GroupID: 10, ExpiresAt: time.Now().Add(-time.Hour), Status: SubscriptionStatusActive},
			}},
		}},
		&countingUserSubRepoStub{},
		nil,
		NewNotificationEmailService(&settingRepoStub{}, nil),
	)

	statuses, _, err := svc.ListUserReadStatus(ctx, ann.ID, pagination.PaginationParams{Page: 1, PageSize: 10}, "")
	require.NoError(t, err)
	require.Len(t, statuses, 1)
	require.False(t, statuses[0].Eligible, "an expired subscription must not satisfy targeting")
}

func TestAnnouncementServiceGetByIDWithStatsReturnsReadCount(t *testing.T) {
	svc := NewAnnouncementService(
		&announcementRepoStub{item: &Announcement{ID: 7, Title: "公告", Content: "内容"}},
		announcementReadCountRepoStub{count: 42},
		nil, nil, nil, nil,
	)

	ann, readCount, err := svc.GetByIDWithStats(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, int64(7), ann.ID)
	require.Equal(t, int64(42), readCount)
}

type announcementReadCountRepoStub struct {
	announcementReadRepoStub
	count int64
}

func (s announcementReadCountRepoStub) CountByAnnouncementID(context.Context, int64) (int64, error) {
	return s.count, nil
}

func TestAnnouncementServiceSeverityDefaultsAndValidates(t *testing.T) {
	ctx := context.Background()

	created, err := NewAnnouncementService(&announcementRepoStub{}, nil, nil, nil, nil, nil).
		Create(ctx, &CreateAnnouncementInput{Title: "公告", Content: "内容"})
	require.NoError(t, err)
	require.Equal(t, AnnouncementSeverityInfo, created.Severity)
	require.False(t, created.ShowBanner, "banner must be opt-in")

	banner := true
	created, err = NewAnnouncementService(&announcementRepoStub{}, nil, nil, nil, nil, nil).
		Create(ctx, &CreateAnnouncementInput{
			Title: "公告", Content: "内容",
			Severity: AnnouncementSeverityCritical, ShowBanner: &banner,
		})
	require.NoError(t, err)
	require.Equal(t, AnnouncementSeverityCritical, created.Severity)
	require.True(t, created.ShowBanner)

	_, err = NewAnnouncementService(&announcementRepoStub{}, nil, nil, nil, nil, nil).
		Create(ctx, &CreateAnnouncementInput{Title: "公告", Content: "内容", Severity: "urgent"})
	require.ErrorIs(t, err, ErrAnnouncementInvalidSeverity)
}

func TestAnnouncementServiceUpdateSeverityAndBanner(t *testing.T) {
	ctx := context.Background()
	base := func() *announcementRepoStub {
		return &announcementRepoStub{item: &Announcement{
			ID: 1, Title: "公告", Content: "内容",
			Severity: AnnouncementSeverityInfo, ShowBanner: true,
		}}
	}

	severity := AnnouncementSeverityWarning
	updated, err := NewAnnouncementService(base(), nil, nil, nil, nil, nil).
		Update(ctx, 1, &UpdateAnnouncementInput{Severity: &severity})
	require.NoError(t, err)
	require.Equal(t, AnnouncementSeverityWarning, updated.Severity)
	require.True(t, updated.ShowBanner, "an omitted show_banner must not be reset")

	off := false
	updated, err = NewAnnouncementService(base(), nil, nil, nil, nil, nil).
		Update(ctx, 1, &UpdateAnnouncementInput{ShowBanner: &off})
	require.NoError(t, err)
	require.False(t, updated.ShowBanner)

	invalid := "urgent"
	_, err = NewAnnouncementService(base(), nil, nil, nil, nil, nil).
		Update(ctx, 1, &UpdateAnnouncementInput{Severity: &invalid})
	require.ErrorIs(t, err, ErrAnnouncementInvalidSeverity)
}

func TestAnnouncementSeverityLabelIsLocalized(t *testing.T) {
	require.Equal(t, "Critical", announcementSeverityLabel(AnnouncementSeverityCritical, "en"))
	require.Equal(t, "紧急", announcementSeverityLabel(AnnouncementSeverityCritical, "zh"))
	require.Equal(t, "Important", announcementSeverityLabel(AnnouncementSeverityWarning, "en"))
	require.Equal(t, "Notice", announcementSeverityLabel(AnnouncementSeverityInfo, "en"))
	// An unknown severity must not produce an empty label.
	require.Equal(t, "Notice", announcementSeverityLabel("", "en"))
}

func announcementAt(id int64, status string, startsAt, endsAt *time.Time) Announcement {
	return Announcement{
		ID: id, Title: "公告", Content: "内容",
		Status: status, Severity: AnnouncementSeverityInfo,
		StartsAt: startsAt, EndsAt: endsAt,
	}
}

func TestListArchiveIncludesArchivedAndExpiredButNotDrafts(t *testing.T) {
	past := time.Now().Add(-48 * time.Hour)
	expired := time.Now().Add(-time.Hour)
	future := time.Now().Add(48 * time.Hour)

	repo := &announcementRepoStub{published: []Announcement{
		announcementAt(5, AnnouncementStatusActive, nil, nil),
		announcementAt(4, AnnouncementStatusArchived, &past, nil),
		announcementAt(3, AnnouncementStatusActive, &past, &expired), // window closed
		announcementAt(2, AnnouncementStatusDraft, nil, nil),         // never published
		announcementAt(1, AnnouncementStatusActive, &future, nil),    // not yet started
	}}
	svc := NewAnnouncementService(repo, announcementReadRepoStub{},
		&announcementUserRepoStub{users: []User{{ID: 1}}}, userSubRepoStub{}, nil, nil)

	items, page, err := svc.ListArchiveForUser(context.Background(), 1,
		pagination.PaginationParams{Page: 1, PageSize: 20}, AnnouncementArchiveFilters{})
	require.NoError(t, err)

	ids := make([]int64, 0, len(items))
	for i := range items {
		ids = append(ids, items[i].Announcement.ID)
	}
	require.Equal(t, []int64{5, 4, 3}, ids,
		"an archived or expired notice stays readable; a draft or unstarted one never was")
	require.Equal(t, int64(3), page.Total)
}

func TestListArchiveAppliesTargeting(t *testing.T) {
	repo := &announcementRepoStub{published: []Announcement{
		{ID: 2, Title: "all", Content: "内容", Status: AnnouncementStatusActive},
		{
			ID: 1, Title: "targeted", Content: "内容", Status: AnnouncementStatusActive,
			Targeting: domain.AnnouncementTargeting{
				AnyOf: []domain.AnnouncementConditionGroup{{
					AllOf: []domain.AnnouncementCondition{{
						Type:     domain.AnnouncementConditionTypeBalance,
						Operator: domain.AnnouncementOperatorGTE,
						Value:    100,
					}},
				}},
			},
		},
	}}
	svc := NewAnnouncementService(repo, announcementReadRepoStub{},
		&announcementUserRepoStub{users: []User{{ID: 1, Balance: 5}}}, userSubRepoStub{}, nil, nil)

	items, _, err := svc.ListArchiveForUser(context.Background(), 1,
		pagination.PaginationParams{Page: 1, PageSize: 20}, AnnouncementArchiveFilters{})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, int64(2), items[0].Announcement.ID)
}

func TestListArchiveSearchAndUnreadFilters(t *testing.T) {
	repo := &announcementRepoStub{published: []Announcement{
		{ID: 3, Title: "Maintenance window", Content: "内容", Status: AnnouncementStatusActive},
		{ID: 2, Title: "Pricing update", Content: "maintenance mentioned in body", Status: AnnouncementStatusActive},
		{ID: 1, Title: "Unrelated", Content: "内容", Status: AnnouncementStatusActive},
	}}
	userRepo := &announcementUserRepoStub{users: []User{{ID: 1}}}
	svc := NewAnnouncementService(repo, announcementReadRepoStub{}, userRepo, userSubRepoStub{}, nil, nil)

	items, page, err := svc.ListArchiveForUser(context.Background(), 1,
		pagination.PaginationParams{Page: 1, PageSize: 20},
		AnnouncementArchiveFilters{Search: "MAINTENANCE"})
	require.NoError(t, err)
	require.Len(t, items, 2, "search is case-insensitive across title and body")
	require.Equal(t, int64(2), page.Total)

	readSvc := NewAnnouncementService(repo, announcementReadMapRepoStub{read: map[int64]time.Time{3: time.Now()}},
		userRepo, userSubRepoStub{}, nil, nil)
	items, _, err = readSvc.ListArchiveForUser(context.Background(), 1,
		pagination.PaginationParams{Page: 1, PageSize: 20},
		AnnouncementArchiveFilters{UnreadOnly: true})
	require.NoError(t, err)
	require.Len(t, items, 2)
	for i := range items {
		require.NotEqual(t, int64(3), items[i].Announcement.ID)
	}
}

func TestListArchivePaginatesTheFilteredSet(t *testing.T) {
	published := make([]Announcement, 0, 5)
	for id := int64(5); id >= 1; id-- {
		published = append(published, announcementAt(id, AnnouncementStatusActive, nil, nil))
	}
	svc := NewAnnouncementService(&announcementRepoStub{published: published}, announcementReadRepoStub{},
		&announcementUserRepoStub{users: []User{{ID: 1}}}, userSubRepoStub{}, nil, nil)

	items, page, err := svc.ListArchiveForUser(context.Background(), 1,
		pagination.PaginationParams{Page: 2, PageSize: 2}, AnnouncementArchiveFilters{})
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, int64(3), items[0].Announcement.ID)
	// Total counts the filtered set, not the page.
	require.Equal(t, int64(5), page.Total)
	require.Equal(t, 3, page.Pages)

	// A page past the end is empty rather than a panic.
	items, _, err = svc.ListArchiveForUser(context.Background(), 1,
		pagination.PaginationParams{Page: 99, PageSize: 2}, AnnouncementArchiveFilters{})
	require.NoError(t, err)
	require.Empty(t, items)
}

type announcementReadMapRepoStub struct {
	announcementReadRepoStub
	read map[int64]time.Time
}

func (s announcementReadMapRepoStub) GetReadMapByUser(_ context.Context, _ int64, ids []int64) (map[int64]time.Time, error) {
	out := map[int64]time.Time{}
	for _, id := range ids {
		if at, ok := s.read[id]; ok {
			out[id] = at
		}
	}
	return out, nil
}
