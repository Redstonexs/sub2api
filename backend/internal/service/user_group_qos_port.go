package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

// UserGroupQoSCacheSchemaV1 versions the cached counter layout. Bump it when the
// stored fields change so old entries are treated as misses instead of being
// misread.
const UserGroupQoSCacheSchemaV1 = 1

// UserGroupQoSCacheTTL bounds how long a cached counter may drift from the
// database. Increments are written straight to the shared Redis entry, so all
// replicas observe them immediately; the TTL only governs re-syncing from the
// durable row.
const UserGroupQoSCacheTTL = 5 * time.Minute

// UserGroupQoSUsageRecord is one user's accumulated consumption in one group.
// Window starts are nillable: NULL means the window has never been initialized.
type UserGroupQoSUsageRecord struct {
	UserID  int64
	GroupID int64

	DailyUsageUSD   float64
	WeeklyUsageUSD  float64
	MonthlyUsageUSD float64

	DailyWindowStart   *time.Time
	WeeklyWindowStart  *time.Time
	MonthlyWindowStart *time.Time
}

// EffectiveUsageAt collapses a stored record into the usage that actually counts
// right now, zeroing any window whose start has rolled over.
//
// Applying expiry at read time rather than trusting the stored numbers is what
// makes a cached snapshot safe: a stale entry can never keep a user degraded
// past the point where their window should have reset.
func (r *UserGroupQoSUsageRecord) EffectiveUsageAt(now time.Time) GroupQoSUsage {
	if r == nil {
		return GroupQoSUsage{}
	}
	usage := GroupQoSUsage{}
	if r.DailyWindowStart != nil && r.DailyWindowStart.Equal(timezone.StartOfDay(now)) {
		usage.DailyUSD = r.DailyUsageUSD
	}
	if r.WeeklyWindowStart != nil && r.WeeklyWindowStart.Equal(timezone.StartOfWeek(now)) {
		usage.WeeklyUSD = r.WeeklyUsageUSD
	}
	// 月度为 30 天滚动窗口，与 user_platform_quotas / 订阅模式语义一致。
	if r.MonthlyWindowStart != nil && now.Sub(*r.MonthlyWindowStart) < 30*24*time.Hour {
		usage.MonthlyUSD = r.MonthlyUsageUSD
	}
	return usage
}

// HasExpiredWindowAt reports whether any window has rolled over since the record
// was stored. A cached record that has expired is treated as a miss so the
// authoritative post-reset numbers are reloaded from the database.
func (r *UserGroupQoSUsageRecord) HasExpiredWindowAt(now time.Time) bool {
	if r == nil {
		return false
	}
	if r.DailyWindowStart != nil && !r.DailyWindowStart.Equal(timezone.StartOfDay(now)) {
		return true
	}
	if r.WeeklyWindowStart != nil && !r.WeeklyWindowStart.Equal(timezone.StartOfWeek(now)) {
		return true
	}
	if r.MonthlyWindowStart != nil && now.Sub(*r.MonthlyWindowStart) >= 30*24*time.Hour {
		return true
	}
	return false
}

// UserGroupQoSUsageRepository is the durable store behind the QoS counter.
type UserGroupQoSUsageRepository interface {
	// GetByUserGroup returns the stored counter, or (nil, nil) when absent.
	GetByUserGroup(ctx context.Context, userID, groupID int64) (*UserGroupQoSUsageRecord, error)
	// IncrementUsageWithReset atomically adds cost, resetting any window that has
	// rolled over before adding.
	IncrementUsageWithReset(ctx context.Context, userID, groupID int64, cost float64, now time.Time) error
	// ResetWindows zeroes all three windows for one (user, group) — the admin
	// "forgive this user" action. Absent rows are not an error.
	ResetWindows(ctx context.Context, userID, groupID int64, now time.Time) error
}

// UserGroupQoSCache is the hot-path cache in front of the counter. It is kept
// separate from BillingCache so that adding QoS does not force an update of
// every existing BillingCache test double.
type UserGroupQoSCache interface {
	// GetUserGroupQoSUsage returns the cached record, or (nil, nil) on a miss.
	GetUserGroupQoSUsage(ctx context.Context, userID, groupID int64) (*UserGroupQoSUsageRecord, error)
	// SetUserGroupQoSUsage stores a freshly loaded record.
	SetUserGroupQoSUsage(ctx context.Context, record *UserGroupQoSUsageRecord, ttl time.Duration) error
	// IncrUserGroupQoSUsage adds cost to an existing cached record. A miss is not
	// an error: the next read repopulates from the database.
	IncrUserGroupQoSUsage(ctx context.Context, userID, groupID int64, cost float64, ttl time.Duration) error
	// InvalidateUserGroupQoSUsage drops the cached record.
	InvalidateUserGroupQoSUsage(ctx context.Context, userID, groupID int64) error
}
