package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type Announcement struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	Status     string `json:"status"`
	NotifyMode string `json:"notify_mode"`
	Severity   string `json:"severity"`
	ShowBanner bool   `json:"show_banner"`

	Targeting service.AnnouncementTargeting `json:"targeting"`

	StartsAt *time.Time `json:"starts_at,omitempty"`
	EndsAt   *time.Time `json:"ends_at,omitempty"`

	CreatedBy *int64 `json:"created_by,omitempty"`
	UpdatedBy *int64 `json:"updated_by,omitempty"`

	// ReadCount is populated only by the single-announcement endpoint; the list
	// endpoint leaves it nil so the list query stays a single flat select.
	ReadCount *int64 `json:"read_count,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserAnnouncement struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	NotifyMode string `json:"notify_mode"`
	Severity   string `json:"severity"`
	ShowBanner bool   `json:"show_banner"`

	StartsAt *time.Time `json:"starts_at,omitempty"`
	EndsAt   *time.Time `json:"ends_at,omitempty"`

	ReadAt *time.Time `json:"read_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func AnnouncementFromService(a *service.Announcement) *Announcement {
	if a == nil {
		return nil
	}
	return &Announcement{
		ID:         a.ID,
		Title:      a.Title,
		Content:    a.Content,
		Status:     a.Status,
		NotifyMode: a.NotifyMode,
		Severity:   a.Severity,
		ShowBanner: a.ShowBanner,
		Targeting:  a.Targeting,
		StartsAt:   a.StartsAt,
		EndsAt:     a.EndsAt,
		CreatedBy:  a.CreatedBy,
		UpdatedBy:  a.UpdatedBy,
		CreatedAt:  a.CreatedAt,
		UpdatedAt:  a.UpdatedAt,
	}
}

// AnnouncementWithStatsFromService maps an announcement together with its read
// count, for the detail endpoint.
func AnnouncementWithStatsFromService(a *service.Announcement, readCount int64) *Announcement {
	out := AnnouncementFromService(a)
	if out == nil {
		return nil
	}
	count := readCount
	out.ReadCount = &count
	return out
}

func UserAnnouncementFromService(a *service.UserAnnouncement) *UserAnnouncement {
	if a == nil {
		return nil
	}
	return &UserAnnouncement{
		ID:         a.Announcement.ID,
		Title:      a.Announcement.Title,
		Content:    a.Announcement.Content,
		NotifyMode: a.Announcement.NotifyMode,
		Severity:   a.Announcement.Severity,
		ShowBanner: a.Announcement.ShowBanner,
		StartsAt:   a.Announcement.StartsAt,
		EndsAt:     a.Announcement.EndsAt,
		ReadAt:     a.ReadAt,
		CreatedAt:  a.Announcement.CreatedAt,
		UpdatedAt:  a.Announcement.UpdatedAt,
	}
}
