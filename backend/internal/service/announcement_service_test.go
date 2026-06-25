package service

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type announcementRepoStub struct {
	item *Announcement
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
func (*announcementRepoStub) ListActive(context.Context, time.Time) ([]Announcement, error) { return nil, nil }

type userRepoStub struct {
	users []User
}

func (s *userRepoStub) Create(context.Context, *User) error { return nil }

func (s *userRepoStub) GetByID(_ context.Context, id int64) (*User, error) {
	for i := range s.users {
		if s.users[i].ID == id {
			user := s.users[i]
			return &user, nil
		}
	}
	return nil, ErrUserNotFound
}
func (s *userRepoStub) GetByIDIncludeDeleted(context.Context, int64) (*User, error) { return nil, ErrUserNotFound }
func (s *userRepoStub) GetByEmail(context.Context, string) (*User, error) { return nil, ErrUserNotFound }
func (s *userRepoStub) GetFirstAdmin(context.Context) (*User, error) { return nil, ErrUserNotFound }
func (s *userRepoStub) Update(context.Context, *User) error { return nil }
func (s *userRepoStub) Delete(context.Context, int64) error { return nil }
func (s *userRepoStub) GetUserAvatar(context.Context, int64) (*UserAvatar, error) { return nil, nil }
func (s *userRepoStub) UpsertUserAvatar(context.Context, int64, UpsertUserAvatarInput) (*UserAvatar, error) {
	return nil, nil
}
func (s *userRepoStub) DeleteUserAvatar(context.Context, int64) error { return nil }
func (s *userRepoStub) List(_ context.Context, _ pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	return append([]User(nil), s.users...), &pagination.PaginationResult{}, nil
}
func (s *userRepoStub) ListWithFilters(_ context.Context, _ pagination.PaginationParams, filters UserListFilters) ([]User, *pagination.PaginationResult, error) {
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
func (s *userRepoStub) GetLatestUsedAtByUserIDs(context.Context, []int64) (map[int64]*time.Time, error) { return nil, nil }
func (s *userRepoStub) GetLatestUsedAtByUserID(context.Context, int64) (*time.Time, error) { return nil, nil }
func (s *userRepoStub) UpdateUserLastActiveAt(context.Context, int64, time.Time) error { return nil }
func (s *userRepoStub) UpdateBalance(context.Context, int64, float64) error { return nil }
func (s *userRepoStub) DeductBalance(context.Context, int64, float64) error { return nil }
func (s *userRepoStub) UpdateConcurrency(context.Context, int64, int) error { return nil }
func (s *userRepoStub) BatchSetConcurrency(context.Context, []int64, int) (int, error) { return 0, nil }
func (s *userRepoStub) BatchAddConcurrency(context.Context, []int64, int) (int, error) { return 0, nil }
func (s *userRepoStub) ExistsByEmail(context.Context, string) (bool, error) { return false, nil }
func (s *userRepoStub) RemoveGroupFromAllowedGroups(context.Context, int64) (int64, error) { return 0, nil }
func (s *userRepoStub) AddGroupToAllowedGroups(context.Context, int64, int64) error { return nil }
func (s *userRepoStub) RemoveGroupFromUserAllowedGroups(context.Context, int64, int64) error { return nil }
func (s *userRepoStub) ListUserAuthIdentities(context.Context, int64) ([]UserAuthIdentityRecord, error) { return nil, nil }
func (s *userRepoStub) UnbindUserAuthProvider(context.Context, int64, string) error { return nil }
func (s *userRepoStub) UpdateTotpSecret(context.Context, int64, *string) error { return nil }
func (s *userRepoStub) EnableTotp(context.Context, int64) error { return nil }
func (s *userRepoStub) DisableTotp(context.Context, int64) error { return nil }

type userSubRepoStub struct{}

func (userSubRepoStub) Create(context.Context, *UserSubscription) error { return nil }
func (userSubRepoStub) GetByID(context.Context, int64) (*UserSubscription, error) { return nil, nil }
func (userSubRepoStub) GetByUserIDAndGroupID(context.Context, int64, int64) (*UserSubscription, error) { return nil, nil }
func (userSubRepoStub) GetActiveByUserIDAndGroupID(context.Context, int64, int64) (*UserSubscription, error) { return nil, nil }
func (userSubRepoStub) Update(context.Context, *UserSubscription) error { return nil }
func (userSubRepoStub) Delete(context.Context, int64) error { return nil }
func (userSubRepoStub) ListByUserID(context.Context, int64) ([]UserSubscription, error) { return nil, nil }
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
func (userSubRepoStub) ExistsByUserIDAndGroupID(context.Context, int64, int64) (bool, error) { return false, nil }
func (userSubRepoStub) ExtendExpiry(context.Context, int64, time.Time) error { return nil }
func (userSubRepoStub) UpdateStatus(context.Context, int64, string) error { return nil }
func (userSubRepoStub) UpdateNotes(context.Context, int64, string) error { return nil }
func (userSubRepoStub) ActivateWindows(context.Context, int64, time.Time) error { return nil }
func (userSubRepoStub) ResetDailyUsage(context.Context, int64, time.Time) error { return nil }
func (userSubRepoStub) ResetWeeklyUsage(context.Context, int64, time.Time) error { return nil }
func (userSubRepoStub) ResetMonthlyUsage(context.Context, int64, time.Time) error { return nil }
func (userSubRepoStub) IncrementUsage(context.Context, int64, float64) error { return nil }
func (userSubRepoStub) BatchUpdateExpiredStatus(context.Context) (int64, error) { return 0, nil }

type announcementReadRepoStub struct{}

func (announcementReadRepoStub) MarkRead(context.Context, int64, int64, time.Time) error { return nil }
func (announcementReadRepoStub) GetReadMapByUser(context.Context, int64, []int64) (map[int64]time.Time, error) {
	return map[int64]time.Time{}, nil
}
func (announcementReadRepoStub) GetReadMapByUsers(context.Context, int64, []int64) (map[int64]time.Time, error) {
	return map[int64]time.Time{}, nil
}
func (announcementReadRepoStub) CountByAnnouncementID(context.Context, int64) (int64, error) { return 0, nil }

type settingRepoStub struct {
	values map[string]string
	err    error
}

func (s settingRepoStub) Get(context.Context, string) (*Setting, error) { return nil, ErrSettingNotFound }
func (s settingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}
func (settingRepoStub) Set(context.Context, string, string) error { return nil }
func (settingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (settingRepoStub) SetMultiple(context.Context, map[string]string) error { return nil }
func (settingRepoStub) GetAll(context.Context) (map[string]string, error) { return map[string]string{}, nil }
func (settingRepoStub) Delete(context.Context, string) error { return nil }

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
		&userRepoStub{users: []User{
			{ID: 1, Email: "user1@example.com", Username: "unsubscribed", Balance: 100},
			{ID: 2, Email: "user2@example.com", Username: "subscribed", Balance: 100},
		}},
		userSubRepoStub{},
		nil,
		NewNotificationEmailService(settingRepoStub{values: map[string]string{
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
		&userRepoStub{users: []User{{ID: 1, Email: "user@example.com", Username: "user", Balance: 100}}},
		userSubRepoStub{},
		nil,
		&NotificationEmailService{settingRepo: settingRepoStub{err: context.Canceled}},
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
