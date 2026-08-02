package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	// announcementMaxTitleRunes caps the title in runes rather than bytes: the
	// column is VARCHAR(200), which PostgreSQL also counts in characters, so a
	// byte-based check rejected CJK titles at a third of the advertised limit.
	announcementMaxTitleRunes = 200
	// announcementMaxContentRunes caps the Markdown body. Content is rendered into
	// the in-app popup and into broadcast emails, so an unbounded body is a cost
	// borne by every recipient, not just the renderer.
	announcementMaxContentRunes = 20000
	// announcementActiveScanLimit mirrors the LIMIT applied by
	// AnnouncementRepository.ListActive. It is duplicated here rather than shared
	// because depguard forbids service -> repository imports; the constants must be
	// kept in sync (see repository/announcement_repo.go).
	announcementActiveScanLimit = 500
	// announcementArchiveScanLimit bounds the candidate set for the user-facing
	// archive. Targeting lives in jsonb and is evaluated in Go, so the archive must
	// over-fetch and filter in memory; this keeps that bounded.
	announcementArchiveScanLimit = 2000
)

type AnnouncementService struct {
	announcementRepo         AnnouncementRepository
	readRepo                 AnnouncementReadRepository
	userRepo                 UserRepository
	userSubRepo              UserSubscriptionRepository
	broadcaster              *AnnouncementBroadcastService
	notificationEmailService *NotificationEmailService
}

func NewAnnouncementService(
	announcementRepo AnnouncementRepository,
	readRepo AnnouncementReadRepository,
	userRepo UserRepository,
	userSubRepo UserSubscriptionRepository,
	broadcaster *AnnouncementBroadcastService,
	notificationEmailService *NotificationEmailService,
) *AnnouncementService {
	return &AnnouncementService{
		announcementRepo:         announcementRepo,
		readRepo:                 readRepo,
		userRepo:                 userRepo,
		userSubRepo:              userSubRepo,
		broadcaster:              broadcaster,
		notificationEmailService: notificationEmailService,
	}
}

// maybeBroadcastEmail dispatches an email broadcast when the saved announcement is
// active with the email notify mode and just transitioned into that state.
// prevActiveEmail reports whether it was already active+email before the save (always
// false for newly created announcements). Re-publishing is harmless regardless because
// delivery dedup prevents double-sends, but gating on the transition avoids re-scanning
// every user on unrelated edits.
func (s *AnnouncementService) maybeBroadcastEmail(a *Announcement, prevActiveEmail bool) {
	if s.broadcaster == nil || a == nil {
		return
	}
	nowActiveEmail := a.Status == AnnouncementStatusActive && a.NotifyMode == AnnouncementNotifyModeEmail
	if nowActiveEmail && !prevActiveEmail {
		s.broadcaster.Dispatch(a)
	}
}

type CreateAnnouncementInput struct {
	Title      string
	Content    string
	Status     string
	NotifyMode string
	Severity   string
	ShowBanner *bool
	Targeting  AnnouncementTargeting
	StartsAt   *time.Time
	EndsAt     *time.Time
	ActorID    *int64 // 管理员用户ID
}

type UpdateAnnouncementInput struct {
	Title      *string
	Content    *string
	Status     *string
	NotifyMode *string
	Severity   *string
	ShowBanner *bool
	Targeting  *AnnouncementTargeting
	StartsAt   **time.Time
	EndsAt     **time.Time
	ActorID    *int64 // 管理员用户ID
}

type UserAnnouncement struct {
	Announcement Announcement
	ReadAt       *time.Time
}

type AnnouncementUserReadStatus struct {
	UserID                        int64      `json:"user_id"`
	Email                         string     `json:"email"`
	Username                      string     `json:"username"`
	Balance                       float64    `json:"balance"`
	Eligible                      bool       `json:"eligible"`
	AnnouncementEmailUnsubscribed bool       `json:"announcement_email_unsubscribed"`
	ReadAt                        *time.Time `json:"read_at,omitempty"`
}

func (s *AnnouncementService) Create(ctx context.Context, input *CreateAnnouncementInput) (*Announcement, error) {
	if input == nil {
		return nil, ErrAnnouncementNilInput
	}

	title, err := normalizeAnnouncementTitle(input.Title)
	if err != nil {
		return nil, err
	}
	content, err := normalizeAnnouncementContent(input.Content)
	if err != nil {
		return nil, err
	}

	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = AnnouncementStatusDraft
	}
	if !isValidAnnouncementStatus(status) {
		return nil, ErrAnnouncementInvalidStatus
	}

	targeting, err := domain.AnnouncementTargeting(input.Targeting).NormalizeAndValidate()
	if err != nil {
		return nil, err
	}

	notifyMode := strings.TrimSpace(input.NotifyMode)
	if notifyMode == "" {
		notifyMode = AnnouncementNotifyModeSilent
	}
	if !isValidAnnouncementNotifyMode(notifyMode) {
		return nil, ErrAnnouncementInvalidNotifyMode
	}

	severity := strings.TrimSpace(input.Severity)
	if severity == "" {
		severity = AnnouncementSeverityInfo
	}
	if !isValidAnnouncementSeverity(severity) {
		return nil, ErrAnnouncementInvalidSeverity
	}

	if input.StartsAt != nil && input.EndsAt != nil {
		if !input.StartsAt.Before(*input.EndsAt) {
			return nil, ErrAnnouncementInvalidSchedule
		}
	}

	a := &Announcement{
		Title:      title,
		Content:    content,
		Status:     status,
		NotifyMode: notifyMode,
		Severity:   severity,
		ShowBanner: input.ShowBanner != nil && *input.ShowBanner,
		Targeting:  targeting,
		StartsAt:   input.StartsAt,
		EndsAt:     input.EndsAt,
	}
	if input.ActorID != nil && *input.ActorID > 0 {
		a.CreatedBy = input.ActorID
		a.UpdatedBy = input.ActorID
	}

	if err := s.announcementRepo.Create(ctx, a); err != nil {
		return nil, fmt.Errorf("create announcement: %w", err)
	}
	s.maybeBroadcastEmail(a, false)
	return a, nil
}

func (s *AnnouncementService) Update(ctx context.Context, id int64, input *UpdateAnnouncementInput) (*Announcement, error) {
	if input == nil {
		return nil, ErrAnnouncementNilInput
	}

	a, err := s.announcementRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Capture pre-update state so we only broadcast when the announcement transitions
	// into active+email (not on every subsequent edit while already published).
	prevActiveEmail := a.Status == AnnouncementStatusActive && a.NotifyMode == AnnouncementNotifyModeEmail

	if input.Title != nil {
		title, err := normalizeAnnouncementTitle(*input.Title)
		if err != nil {
			return nil, err
		}
		a.Title = title
	}
	if input.Content != nil {
		content, err := normalizeAnnouncementContent(*input.Content)
		if err != nil {
			return nil, err
		}
		a.Content = content
	}
	if input.Status != nil {
		status := strings.TrimSpace(*input.Status)
		if !isValidAnnouncementStatus(status) {
			return nil, ErrAnnouncementInvalidStatus
		}
		a.Status = status
	}

	if input.NotifyMode != nil {
		notifyMode := strings.TrimSpace(*input.NotifyMode)
		if !isValidAnnouncementNotifyMode(notifyMode) {
			return nil, ErrAnnouncementInvalidNotifyMode
		}
		a.NotifyMode = notifyMode
	}

	if input.Severity != nil {
		severity := strings.TrimSpace(*input.Severity)
		if !isValidAnnouncementSeverity(severity) {
			return nil, ErrAnnouncementInvalidSeverity
		}
		a.Severity = severity
	}

	if input.ShowBanner != nil {
		a.ShowBanner = *input.ShowBanner
	}

	if input.Targeting != nil {
		targeting, err := domain.AnnouncementTargeting(*input.Targeting).NormalizeAndValidate()
		if err != nil {
			return nil, err
		}
		a.Targeting = targeting
	}

	if input.StartsAt != nil {
		a.StartsAt = *input.StartsAt
	}
	if input.EndsAt != nil {
		a.EndsAt = *input.EndsAt
	}

	if a.StartsAt != nil && a.EndsAt != nil {
		if !a.StartsAt.Before(*a.EndsAt) {
			return nil, ErrAnnouncementInvalidSchedule
		}
	}

	if input.ActorID != nil && *input.ActorID > 0 {
		a.UpdatedBy = input.ActorID
	}

	if err := s.announcementRepo.Update(ctx, a); err != nil {
		return nil, fmt.Errorf("update announcement: %w", err)
	}
	s.maybeBroadcastEmail(a, prevActiveEmail)
	return a, nil
}

// PreviewAudience counts how many users an announcement with these targeting rules
// would actually be emailed.
//
// It runs synchronously with a hard scan ceiling rather than returning a sampled
// estimate: the number backs an irreversible decision ("am I about to email 8000
// people?"), so a lower bound flagged as truncated is more useful than a guess.
func (s *AnnouncementService) PreviewAudience(ctx context.Context, targeting AnnouncementTargeting) (AnnouncementAudienceStats, error) {
	normalized, err := domain.AnnouncementTargeting(targeting).NormalizeAndValidate()
	if err != nil {
		return AnnouncementAudienceStats{}, err
	}
	stats, err := scanAnnouncementAudience(
		ctx, s.userRepo, s.notificationEmailService,
		normalized, announcementAudiencePreviewMaxScan, nil, nil,
	)
	if err != nil {
		return AnnouncementAudienceStats{}, fmt.Errorf("scan announcement audience: %w", err)
	}
	return stats, nil
}

// SendTestEmail delivers the announcement to the acting admin's own address and
// returns it. The recipient is derived from actorID rather than taken from the
// request: an arbitrary-recipient endpoint on an admin session is an open relay.
func (s *AnnouncementService) SendTestEmail(ctx context.Context, id, actorID int64) (string, error) {
	if s.broadcaster == nil || s.userRepo == nil {
		return "", ErrAnnouncementTestEmailUnavailable
	}
	a, err := s.announcementRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return "", fmt.Errorf("get user: %w", err)
	}
	email := strings.TrimSpace(actor.Email)
	if email == "" {
		return "", ErrAnnouncementTestEmailUnavailable
	}
	if err := s.broadcaster.SendTest(ctx, a, email, actor.Username, actor.ID); err != nil {
		return "", err
	}
	return email, nil
}

func (s *AnnouncementService) Delete(ctx context.Context, id int64) error {
	if err := s.announcementRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete announcement: %w", err)
	}
	return nil
}

func (s *AnnouncementService) GetByID(ctx context.Context, id int64) (*Announcement, error) {
	return s.announcementRepo.GetByID(ctx, id)
}

// GetByIDWithStats returns an announcement along with the number of users who
// have marked it read.
func (s *AnnouncementService) GetByIDWithStats(ctx context.Context, id int64) (*Announcement, int64, error) {
	a, err := s.announcementRepo.GetByID(ctx, id)
	if err != nil {
		return nil, 0, err
	}
	readCount, err := s.readRepo.CountByAnnouncementID(ctx, id)
	if err != nil {
		return nil, 0, fmt.Errorf("count announcement reads: %w", err)
	}
	return a, readCount, nil
}

func (s *AnnouncementService) List(ctx context.Context, params pagination.PaginationParams, filters AnnouncementListFilters) ([]Announcement, *pagination.PaginationResult, error) {
	return s.announcementRepo.List(ctx, params, filters)
}

func (s *AnnouncementService) ListForUser(ctx context.Context, userID int64, unreadOnly bool) ([]UserAnnouncement, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	activeSubs, err := s.userSubRepo.ListActiveByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list active subscriptions: %w", err)
	}
	activeGroupIDs := make(map[int64]struct{}, len(activeSubs))
	for i := range activeSubs {
		activeGroupIDs[activeSubs[i].GroupID] = struct{}{}
	}

	now := time.Now()
	anns, err := s.announcementRepo.ListActive(ctx, now)
	if err != nil {
		return nil, fmt.Errorf("list active announcements: %w", err)
	}
	if len(anns) >= announcementActiveScanLimit {
		// The repository truncates silently, so an install with more concurrently
		// active announcements than the cap would hide the oldest ones from users
		// with no signal anywhere. Make that visible in the logs.
		logger.LegacyPrintf("service.announcement",
			"[Announcement] active announcement scan hit the %d-row cap for user %d; older active announcements are not being delivered",
			announcementActiveScanLimit, userID)
	}

	visible := make([]Announcement, 0, len(anns))
	ids := make([]int64, 0, len(anns))
	for i := range anns {
		a := anns[i]
		if !a.IsActiveAt(now) {
			continue
		}
		if !a.Targeting.Matches(user.Balance, activeGroupIDs) {
			continue
		}
		visible = append(visible, a)
		ids = append(ids, a.ID)
	}

	if len(visible) == 0 {
		return []UserAnnouncement{}, nil
	}

	readMap, err := s.readRepo.GetReadMapByUser(ctx, userID, ids)
	if err != nil {
		return nil, fmt.Errorf("get read map: %w", err)
	}

	out := make([]UserAnnouncement, 0, len(visible))
	for i := range visible {
		a := visible[i]
		readAt, ok := readMap[a.ID]
		if unreadOnly && ok {
			continue
		}
		var ptr *time.Time
		if ok {
			t := readAt
			ptr = &t
		}
		out = append(out, UserAnnouncement{
			Announcement: a,
			ReadAt:       ptr,
		})
	}

	// 未读优先、同状态按创建时间倒序
	sort.Slice(out, func(i, j int) bool {
		ai, aj := out[i], out[j]
		if (ai.ReadAt == nil) != (aj.ReadAt == nil) {
			return ai.ReadAt == nil
		}
		return ai.Announcement.ID > aj.Announcement.ID
	})

	return out, nil
}

// AnnouncementArchiveFilters narrows a user's announcement archive.
type AnnouncementArchiveFilters struct {
	UnreadOnly bool
	Search     string
}

// ListArchiveForUser returns the announcements this user was ever eligible to see,
// including archived and expired ones, newest first.
//
// Targeting lives in jsonb and is evaluated in Go, so SQL-level pagination cannot
// be exact. This fetches a bounded candidate set, filters in memory, then slices.
// Announcements are a small table; pushing targeting into jsonb predicates would be
// a large lift for no user-visible gain.
func (s *AnnouncementService) ListArchiveForUser(
	ctx context.Context,
	userID int64,
	params pagination.PaginationParams,
	filters AnnouncementArchiveFilters,
) ([]UserAnnouncement, *pagination.PaginationResult, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("get user: %w", err)
	}

	activeSubs, err := s.userSubRepo.ListActiveByUserID(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("list active subscriptions: %w", err)
	}
	activeGroupIDs := make(map[int64]struct{}, len(activeSubs))
	for i := range activeSubs {
		activeGroupIDs[activeSubs[i].GroupID] = struct{}{}
	}

	now := time.Now()
	candidates, err := s.announcementRepo.ListPublished(ctx, now, announcementArchiveScanLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("list published announcements: %w", err)
	}
	if len(candidates) >= announcementArchiveScanLimit {
		logger.LegacyPrintf("service.announcement",
			"[Announcement] archive scan hit the %d-row cap for user %d; older announcements are not listed",
			announcementArchiveScanLimit, userID)
	}

	search := strings.ToLower(strings.TrimSpace(filters.Search))
	visible := make([]Announcement, 0, len(candidates))
	ids := make([]int64, 0, len(candidates))
	for i := range candidates {
		a := candidates[i]
		if !a.Targeting.Matches(user.Balance, activeGroupIDs) {
			continue
		}
		if search != "" &&
			!strings.Contains(strings.ToLower(a.Title), search) &&
			!strings.Contains(strings.ToLower(a.Content), search) {
			continue
		}
		visible = append(visible, a)
		ids = append(ids, a.ID)
	}

	readMap := map[int64]time.Time{}
	if len(ids) > 0 {
		readMap, err = s.readRepo.GetReadMapByUser(ctx, userID, ids)
		if err != nil {
			return nil, nil, fmt.Errorf("get read map: %w", err)
		}
	}

	matched := make([]UserAnnouncement, 0, len(visible))
	for i := range visible {
		a := visible[i]
		readAt, ok := readMap[a.ID]
		if filters.UnreadOnly && ok {
			continue
		}
		var ptr *time.Time
		if ok {
			t := readAt
			ptr = &t
		}
		matched = append(matched, UserAnnouncement{Announcement: a, ReadAt: ptr})
	}

	// ListPublished already orders by id DESC, and the filters above preserve it.
	total := int64(len(matched))
	offset := params.Offset()
	if offset > len(matched) {
		offset = len(matched)
	}
	end := offset + params.Limit()
	if end > len(matched) {
		end = len(matched)
	}
	pages := 0
	if params.Limit() > 0 {
		pages = int((total + int64(params.Limit()) - 1) / int64(params.Limit()))
	}
	return matched[offset:end], &pagination.PaginationResult{
		Total:    total,
		Page:     params.Page,
		PageSize: params.Limit(),
		Pages:    pages,
	}, nil
}

func (s *AnnouncementService) MarkRead(ctx context.Context, userID, announcementID int64) error {
	// 安全：仅允许标记当前用户“可见”的公告
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	a, err := s.announcementRepo.GetByID(ctx, announcementID)
	if err != nil {
		return err
	}

	now := time.Now()
	if !a.IsActiveAt(now) {
		return ErrAnnouncementNotFound
	}

	activeSubs, err := s.userSubRepo.ListActiveByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("list active subscriptions: %w", err)
	}
	activeGroupIDs := make(map[int64]struct{}, len(activeSubs))
	for i := range activeSubs {
		activeGroupIDs[activeSubs[i].GroupID] = struct{}{}
	}

	if !a.Targeting.Matches(user.Balance, activeGroupIDs) {
		return ErrAnnouncementNotFound
	}

	if err := s.readRepo.MarkRead(ctx, announcementID, userID, now); err != nil {
		return fmt.Errorf("mark read: %w", err)
	}
	return nil
}

func (s *AnnouncementService) ListUserReadStatus(
	ctx context.Context,
	announcementID int64,
	params pagination.PaginationParams,
	search string,
) ([]AnnouncementUserReadStatus, *pagination.PaginationResult, error) {
	ann, err := s.announcementRepo.GetByID(ctx, announcementID)
	if err != nil {
		return nil, nil, err
	}

	filters := UserListFilters{
		Search: strings.TrimSpace(search),
	}

	users, page, err := s.userRepo.ListWithFilters(ctx, params, filters)
	if err != nil {
		return nil, nil, fmt.Errorf("list users: %w", err)
	}

	userIDs := make([]int64, 0, len(users))
	for i := range users {
		userIDs = append(userIDs, users[i].ID)
	}

	readMap, err := s.readRepo.GetReadMapByUsers(ctx, announcementID, userIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("get read map: %w", err)
	}

	// Unsubscribe state resolves in a single batched lookup rather than once per
	// row; ListWithFilters already eager-loads each user's active subscriptions,
	// so targeting is evaluated in memory. Both were N+1 queries per page.
	emails := make([]string, 0, len(users))
	for i := range users {
		if email := strings.TrimSpace(users[i].Email); email != "" {
			emails = append(emails, email)
		}
	}
	unsubscribed := map[string]bool{}
	if s.notificationEmailService != nil && len(emails) > 0 {
		unsubscribed, err = s.notificationEmailService.IsUnsubscribedBatch(ctx, emails, NotificationEmailEventAnnouncementBroadcast)
		if err != nil {
			return nil, nil, fmt.Errorf("check unsubscribe status: %w", err)
		}
	}

	now := time.Now()
	out := make([]AnnouncementUserReadStatus, 0, len(users))
	for i := range users {
		u := users[i]

		readAt, ok := readMap[u.ID]
		var ptr *time.Time
		if ok {
			t := readAt
			ptr = &t
		}

		out = append(out, AnnouncementUserReadStatus{
			UserID:                        u.ID,
			Email:                         u.Email,
			Username:                      u.Username,
			Balance:                       u.Balance,
			Eligible:                      domain.AnnouncementTargeting(ann.Targeting).Matches(u.Balance, activeSubscriptionGroupIDs(u.Subscriptions, now)),
			AnnouncementEmailUnsubscribed: unsubscribed[normalizeNotificationEmailKey(u.Email)],
			ReadAt:                        ptr,
		})
	}

	return out, page, nil
}

// normalizeAnnouncementTitle trims and validates a title, counting runes rather
// than bytes so a multi-byte title gets the full announcementMaxTitleRunes.
func normalizeAnnouncementTitle(raw string) (string, error) {
	title := strings.TrimSpace(raw)
	if title == "" || utf8.RuneCountInString(title) > announcementMaxTitleRunes {
		return "", ErrAnnouncementInvalidTitle
	}
	return title, nil
}

// normalizeAnnouncementContent trims and validates the Markdown body.
func normalizeAnnouncementContent(raw string) (string, error) {
	content := strings.TrimSpace(raw)
	if content == "" {
		return "", ErrAnnouncementContentRequired
	}
	if utf8.RuneCountInString(content) > announcementMaxContentRunes {
		return "", ErrAnnouncementContentTooLong
	}
	return content, nil
}

func isValidAnnouncementStatus(status string) bool {
	switch status {
	case AnnouncementStatusDraft, AnnouncementStatusActive, AnnouncementStatusArchived:
		return true
	default:
		return false
	}
}

func isValidAnnouncementSeverity(severity string) bool {
	switch severity {
	case AnnouncementSeverityInfo, AnnouncementSeverityWarning, AnnouncementSeverityCritical:
		return true
	default:
		return false
	}
}

func isValidAnnouncementNotifyMode(mode string) bool {
	switch mode {
	case AnnouncementNotifyModeSilent, AnnouncementNotifyModePopup, AnnouncementNotifyModeEmail:
		return true
	default:
		return false
	}
}
