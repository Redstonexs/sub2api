package service

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// SortWindow 排序窗口标识。
type SortWindow string

const (
	SortWindow5h SortWindow = "5h"
	SortWindow7d SortWindow = "7d"
)

// WindowUsage 单窗口使用量摘要。
type WindowUsage struct {
	Utilization float64
	ResetsAt    *time.Time
	WindowStats *WindowStats
}

// GroupQuotaAccount 分组配额卡片中的单个账号条目。
type GroupQuotaAccount struct {
	AccountID   int64        // 匿名化时为 0
	DisplayName string       // 真实名称或 "账号N"
	FiveHour    *WindowUsage // 5h 窗口；nil 表示无数据
	SevenDay    *WindowUsage // 7d 窗口；nil 表示无数据
}

// GroupQuotaCardResult 分组配额卡片聚合结果。
type GroupQuotaCardResult struct {
	GroupID          int64
	TotalRemaining5h *float64            // 有 5h 数据的账号的平均 remaining；全无数据时为 nil
	TotalRemaining7d *float64            // 有 7d 数据的账号的平均 remaining；全无数据时为 nil
	Accounts         []GroupQuotaAccount // 按 active-window remaining desc + ID asc 排序
}

// AccountListReader 账号列表读取接口（GroupQuotaService 所需的最小契约）。
type AccountListReader interface {
	ListByGroup(ctx context.Context, groupID int64) ([]Account, error)
}

// PassiveUsageBatchReader 批量被动用量读取接口。
type PassiveUsageBatchReader interface {
	GetPassiveUsageBatch(ctx context.Context, accountIDs []int64) (map[int64]*UsageInfo, error)
}

// GroupQuotaService 分组配额聚合服务。
type GroupQuotaService struct {
	accountRepo      AccountListReader
	usageBatchReader PassiveUsageBatchReader
}

// NewGroupQuotaService 创建 GroupQuotaService 实例。
func NewGroupQuotaService(accountRepo AccountListReader, usageBatchReader PassiveUsageBatchReader) *GroupQuotaService {
	return &GroupQuotaService{
		accountRepo:      accountRepo,
		usageBatchReader: usageBatchReader,
	}
}

// GetGroupQuotaCard 获取分组配额卡片（聚合用量、匿名化、排序）。
func (s *GroupQuotaService) GetGroupQuotaCard(
	ctx context.Context,
	groupID int64,
	sortBy SortWindow,
	anonymize bool,
) (*GroupQuotaCardResult, error) {
	accounts, err := s.accountRepo.ListByGroup(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("list accounts by group: %w", err)
	}

	if len(accounts) == 0 {
		return &GroupQuotaCardResult{GroupID: groupID}, nil
	}

	// 收集账号 ID 用于批量查询用量
	accountIDs := make([]int64, len(accounts))
	for i, a := range accounts {
		accountIDs[i] = a.ID
	}

	usageMap, err := s.usageBatchReader.GetPassiveUsageBatch(ctx, accountIDs)
	if err != nil {
		return nil, fmt.Errorf("get passive usage batch: %w", err)
	}

	// 构建每账号条目
	entries := make([]groupQuotaEntry, 0, len(accounts))
	var sum5h, sum7d float64
	var count5h, count7d int

	for _, a := range accounts {
		usage, hasUsage := usageMap[a.ID]
		entry := groupQuotaEntry{
			account:   &a,
			accountID: a.ID,
			realName:  a.Name,
		}

		if hasUsage && usage.FiveHour != nil {
			rem := 100 - usage.FiveHour.Utilization
			entry.rem5h = &rem
			entry.hour = usageWindowToWindowUsage(usage.FiveHour)
			sum5h += rem
			count5h++
		}
		if hasUsage && usage.SevenDay != nil {
			rem := 100 - usage.SevenDay.Utilization
			entry.rem7d = &rem
			entry.day = usageWindowToWindowUsage(usage.SevenDay)
			sum7d += rem
			count7d++
		}

		entries = append(entries, entry)
	}

	// 匿名化
	if anonymize {
		applyAnonymization(entries)
	}

	// 排序
	sortEntries(entries, sortBy)

	result := &GroupQuotaCardResult{
		GroupID:  groupID,
		Accounts: make([]GroupQuotaAccount, 0, len(entries)),
	}
	if count5h > 0 {
		v := sum5h / float64(count5h)
		result.TotalRemaining5h = &v
	}
	if count7d > 0 {
		v := sum7d / float64(count7d)
		result.TotalRemaining7d = &v
	}
	for _, e := range entries {
		ga := GroupQuotaAccount{
			DisplayName: e.displayName,
			FiveHour:    e.hour,
			SevenDay:    e.day,
		}
		if !anonymize {
			ga.AccountID = e.accountID
			ga.DisplayName = e.realName
		}
		result.Accounts = append(result.Accounts, ga)
	}

	return result, nil
}

// groupQuotaEntry 内部聚合条目。
type groupQuotaEntry struct {
	account     *Account
	accountID   int64
	realName    string
	displayName string
	rem5h       *float64
	rem7d       *float64
	hour        *WindowUsage
	day         *WindowUsage
}

func usageWindowToWindowUsage(p *UsageProgress) *WindowUsage {
	if p == nil {
		return nil
	}
	return &WindowUsage{
		Utilization: p.Utilization,
		ResetsAt:    p.ResetsAt,
		WindowStats: p.WindowStats,
	}
}

func applyAnonymization(entries []groupQuotaEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].accountID < entries[j].accountID
	})
	for i := range entries {
		entries[i].displayName = fmt.Sprintf("账号%d", i+1)
		entries[i].accountID = 0
	}
}

func sortEntries(entries []groupQuotaEntry, sortBy SortWindow) {
	// 获取主排序字段：nil 视为最低优先级
	remaining := func(e *groupQuotaEntry) (hasData bool, rem float64) {
		switch sortBy {
		case SortWindow5h:
			if e.rem5h != nil {
				return true, *e.rem5h
			}
		case SortWindow7d:
			if e.rem7d != nil {
				return true, *e.rem7d
			}
		}
		return false, 0
	}

	// SliceStable：匿名化后 accountID 全为 0，稳定排序可保持匿名化阶段建立的 ID 升序相对次序
	sort.SliceStable(entries, func(i, j int) bool {
		ai, ri := remaining(&entries[i])
		aj, rj := remaining(&entries[j])

		// 有数据的条目优先于无数据的条目
		if ai != aj {
			return ai // true (有数据) < false (无数据)
		}
		// 都有数据：按 remaining 降序
		if ai && aj {
			if ri != rj {
				return ri > rj // 降序
			}
		}
		// tie-break：ID 升序
		return entries[i].accountID < entries[j].accountID
	})
}
