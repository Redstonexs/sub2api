package service

import (
	"context"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// This file lives apart from announcement_broadcast_service.go on purpose: that
// file is on the fork's upstream-sync preservation list (see AGENTS.md), so
// keeping the shared scan here means upstream edits to the broadcast file still
// auto-resolve without dragging this logic onto the whitelist too.

const (
	// announcementAudiencePageSize is the user pagination size for an audience scan.
	announcementAudiencePageSize = 500
	// announcementAudienceListTimeout bounds a single page of user listing.
	announcementAudienceListTimeout = 30 * time.Second
	// announcementAudiencePreviewMaxScan caps a synchronous preview so an install
	// with a very large user table cannot hang an admin request. Past this the
	// result is reported as truncated rather than silently wrong.
	announcementAudiencePreviewMaxScan = 200000
)

// AnnouncementRecipient is one deliverable target of an announcement broadcast.
type AnnouncementRecipient struct {
	UserID int64
	Email  string
	Name   string
}

// AnnouncementAudienceStats summarises an audience scan.
type AnnouncementAudienceStats struct {
	// Scanned is the number of active users examined.
	Scanned int64 `json:"scanned"`
	// Matched is how many satisfied the targeting rules.
	Matched int64 `json:"matched"`
	// WithEmail is how many of those have a usable email address.
	WithEmail int64 `json:"with_email"`
	// Unsubscribed is how many of those opted out of announcement emails.
	Unsubscribed int64 `json:"unsubscribed"`
	// Deliverable is WithEmail minus Unsubscribed: the number that would be emailed.
	Deliverable int64 `json:"deliverable"`
	// Truncated reports that the scan stopped at maxScan, so the counts are lower bounds.
	Truncated bool `json:"truncated"`
}

// scanAnnouncementAudience pages through active users, applies the announcement's
// targeting rules, and reports who would actually receive an email.
//
// The broadcast and the admin's "estimate audience" both go through here, so the
// number an admin is shown before publishing is the number that gets mailed.
//
// visit, when non-nil, is called once per deliverable recipient; returning false
// aborts the scan. Passing nil makes this a pure counting pass. maxScan <= 0 means
// unbounded. stop, when non-nil, aborts the scan once closed.
func scanAnnouncementAudience(
	ctx context.Context,
	userRepo UserRepository,
	emailSvc *NotificationEmailService,
	targeting AnnouncementTargeting,
	maxScan int64,
	stop <-chan struct{},
	visit func(AnnouncementRecipient) bool,
) (AnnouncementAudienceStats, error) {
	var stats AnnouncementAudienceStats
	if userRepo == nil {
		return stats, nil
	}
	now := time.Now()

	for page := 1; ; page++ {
		select {
		case <-stop:
			return stats, nil
		case <-ctx.Done():
			return stats, ctx.Err()
		default:
		}

		// Status filter matters: UserRepository.List applies none, so a plain List
		// would count (and email) disabled and banned accounts.
		listCtx, cancel := context.WithTimeout(ctx, announcementAudienceListTimeout)
		users, result, err := userRepo.ListWithFilters(listCtx, pagination.PaginationParams{
			Page:     page,
			PageSize: announcementAudiencePageSize,
		}, UserListFilters{Status: StatusActive})
		cancel()
		if err != nil {
			return stats, err
		}

		matched := make([]AnnouncementRecipient, 0, len(users))
		emails := make([]string, 0, len(users))
		for i := range users {
			u := users[i]
			stats.Scanned++
			if !targeting.Matches(u.Balance, activeSubscriptionGroupIDs(u.Subscriptions, now)) {
				continue
			}
			stats.Matched++

			email := strings.TrimSpace(u.Email)
			if email == "" {
				continue
			}
			stats.WithEmail++

			name := strings.TrimSpace(u.Username)
			if name == "" {
				name = emailRecipientName(email)
			}
			matched = append(matched, AnnouncementRecipient{UserID: u.ID, Email: email, Name: name})
			emails = append(emails, email)
		}

		// One batched lookup per page rather than one query per recipient.
		unsubscribed := map[string]bool{}
		if emailSvc != nil && len(emails) > 0 {
			unsubscribeCtx, cancel := context.WithTimeout(ctx, announcementAudienceListTimeout)
			unsubscribed, err = emailSvc.IsUnsubscribedBatch(unsubscribeCtx, emails, NotificationEmailEventAnnouncementBroadcast)
			cancel()
			if err != nil {
				return stats, err
			}
		}

		for _, recipient := range matched {
			if unsubscribed[normalizeNotificationEmailKey(recipient.Email)] {
				stats.Unsubscribed++
				continue
			}
			stats.Deliverable++
			if visit != nil && !visit(recipient) {
				return stats, nil
			}
		}

		if maxScan > 0 && stats.Scanned >= maxScan {
			// More pages remain, so every count is a lower bound.
			stats.Truncated = result != nil && page < result.Pages
			return stats, nil
		}
		if result == nil || page >= result.Pages || len(users) == 0 {
			return stats, nil
		}
	}
}
