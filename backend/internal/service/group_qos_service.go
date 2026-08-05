package service

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// groupQoSResolveTimeout bounds the hot-path usage lookup. QoS is an
// anti-abuse control, not a correctness gate — a slow cache must never hold up
// a request.
const groupQoSResolveTimeout = 300 * time.Millisecond

var (
	groupQoSResolveErrorTotal atomic.Int64
	groupQoSRecordErrorTotal  atomic.Int64
)

// GroupQoSService resolves the degradation tier in effect for a request and
// accumulates the per-(user, group) counter that drives it.
//
// Every failure path is fail-open: if the counter cannot be read, the request
// proceeds undegraded. Degrading or blocking a paying user because Redis
// hiccuped would be a worse outcome than briefly under-enforcing an abuser.
type GroupQoSService struct {
	repo  UserGroupQoSUsageRepository
	cache UserGroupQoSCache
}

func NewGroupQoSService(repo UserGroupQoSUsageRepository, cache UserGroupQoSCache) *GroupQoSService {
	return &GroupQoSService{repo: repo, cache: cache}
}

// GroupQoSEligible reports whether a group has an active ladder. It is a pure
// in-memory check on the auth snapshot, so the overwhelming majority of traffic
// (QoS disabled) costs nothing.
func GroupQoSEligible(group *Group) bool {
	return group != nil && group.QoSEnabled && len(group.QoSTiers) > 0
}

// ResolveDecision returns the tier in effect for this requester, or nil when no
// tier is reached, QoS is disabled, or the counter could not be read.
func (s *GroupQoSService) ResolveDecision(ctx context.Context, userID int64, group *Group) *GroupQoSDecision {
	if s == nil || !GroupQoSEligible(group) || userID <= 0 {
		return nil
	}

	lookupCtx, cancel := context.WithTimeout(ctx, groupQoSResolveTimeout)
	defer cancel()

	now := time.Now().UTC()
	record, err := s.loadUsage(lookupCtx, userID, group.ID, now)
	if err != nil {
		groupQoSResolveErrorTotal.Add(1)
		logger.LegacyPrintf("service.group_qos",
			"ALERT: qos usage lookup failed, request proceeds undegraded user=%d group=%d: %v",
			userID, group.ID, err)
		return nil
	}

	return ResolveGroupQoSTier(record.EffectiveUsageAt(now), group.QoSTiers)
}

// loadUsage reads the counter cache-aside. A cached record whose window has
// rolled over is discarded rather than trusted, so a stale entry can never keep
// a user degraded past their reset.
func (s *GroupQoSService) loadUsage(ctx context.Context, userID, groupID int64, now time.Time) (*UserGroupQoSUsageRecord, error) {
	if s.cache != nil {
		cached, err := s.cache.GetUserGroupQoSUsage(ctx, userID, groupID)
		if err == nil && cached != nil && !cached.HasExpiredWindowAt(now) {
			return cached, nil
		}
	}

	if s.repo == nil {
		return &UserGroupQoSUsageRecord{UserID: userID, GroupID: groupID}, nil
	}
	record, err := s.repo.GetByUserGroup(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		// No counter yet — this user has spent nothing in this group.
		record = &UserGroupQoSUsageRecord{UserID: userID, GroupID: groupID}
	}
	if s.cache != nil {
		if err := s.cache.SetUserGroupQoSUsage(ctx, record, UserGroupQoSCacheTTL); err != nil {
			logger.LegacyPrintf("service.group_qos",
				"Warning: qos usage cache fill failed user=%d group=%d: %v", userID, groupID, err)
		}
	}
	return record, nil
}

// RecordUsage accumulates one request's cost against the (user, group) counter.
//
// Redis is written synchronously so the next preflight sees the new total
// immediately — that keeps the over-spend window bounded by concurrent in-flight
// requests rather than growing with time. The durable row is written on a
// detached context in the background.
func (s *GroupQoSService) RecordUsage(ctx context.Context, userID, groupID int64, cost float64) {
	if s == nil || userID <= 0 || groupID <= 0 || cost <= 0 {
		return
	}

	if s.cache != nil {
		cacheCtx, cancel := context.WithTimeout(context.Background(), groupQoSResolveTimeout)
		if err := s.cache.IncrUserGroupQoSUsage(cacheCtx, userID, groupID, cost, UserGroupQoSCacheTTL); err != nil {
			logger.LegacyPrintf("service.group_qos",
				"ALERT: incr qos usage cache failed user=%d group=%d cost=%f: %v", userID, groupID, cost, err)
		}
		cancel()
	}

	if s.repo == nil {
		return
	}
	dbCtx, dbCancel := detachUpstreamContext(ctx)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.LegacyPrintf("service.group_qos",
					"ALERT: panic in qos usage incr goroutine user=%d group=%d: %v", userID, groupID, r)
			}
		}()
		defer dbCancel()
		if err := s.repo.IncrementUsageWithReset(dbCtx, userID, groupID, cost, time.Now().UTC()); err != nil {
			// ALERT: 持久化失败意味着 Redis 条目过期后这笔消耗永久丢失，
			// QoS 阶梯会低估该用户的真实用量。
			groupQoSRecordErrorTotal.Add(1)
			logger.LegacyPrintf("service.group_qos",
				"ALERT: incr qos usage DB failed user=%d group=%d cost=%f: %v", userID, groupID, cost, err)
		}
	}()
}

// ResetUsage zeroes a user's counter in one group (admin forgiveness action).
func (s *GroupQoSService) ResetUsage(ctx context.Context, userID, groupID int64) error {
	if s == nil || s.repo == nil {
		return nil
	}
	if err := s.repo.ResetWindows(ctx, userID, groupID, time.Now().UTC()); err != nil {
		return err
	}
	if s.cache != nil {
		if err := s.cache.InvalidateUserGroupQoSUsage(ctx, userID, groupID); err != nil {
			logger.LegacyPrintf("service.group_qos",
				"Warning: qos usage cache invalidate failed user=%d group=%d: %v", userID, groupID, err)
		}
	}
	return nil
}

// GroupQoSStats exposes failure counters for the ops dashboard.
func GroupQoSStats() (resolveErrors, recordErrors int64) {
	return groupQoSResolveErrorTotal.Load(), groupQoSRecordErrorTotal.Load()
}
