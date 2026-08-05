package repository

import (
	"context"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/usergroupqosusage"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

var _ service.UserGroupQoSUsageRepository = (*userGroupQoSUsageRepository)(nil)

type userGroupQoSUsageRepository struct {
	client *dbent.Client
}

// NewUserGroupQoSUsageRepository 创建 QoS 计数器仓储实现。
func NewUserGroupQoSUsageRepository(client *dbent.Client) service.UserGroupQoSUsageRepository {
	return &userGroupQoSUsageRepository{client: client}
}

func (r *userGroupQoSUsageRepository) GetByUserGroup(ctx context.Context, userID, groupID int64) (*service.UserGroupQoSUsageRecord, error) {
	client := clientFromContext(ctx, r.client)
	e, err := client.UserGroupQoSUsage.Query().
		Where(
			usergroupqosusage.UserIDEQ(userID),
			usergroupqosusage.GroupIDEQ(groupID),
			usergroupqosusage.DeletedAtIsNil(),
		).
		Only(ctx)
	if dbent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return entQoSUsageToRecord(e), nil
}

// IncrementUsageWithReset 原子累加用量，窗口过期时先重置再累加。
//
// 并发建行策略与 user_platform_quotas.IncrementUsageWithReset 相同：未命中时用
// ON CONFLICT DO UPDATE 而非裸 INSERT——并发下另一请求可能在本事务
// SELECT FOR UPDATE 之后、INSERT 之前刚建行，裸 INSERT 会撞 partial unique index
// 导致事务回滚、本次 cost 丢失。
func (r *userGroupQoSUsageRepository) IncrementUsageWithReset(ctx context.Context, userID, groupID int64, cost float64, now time.Time) error {
	return r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		existing, err := txClient.UserGroupQoSUsage.Query().
			Where(
				usergroupqosusage.UserIDEQ(userID),
				usergroupqosusage.GroupIDEQ(groupID),
				usergroupqosusage.DeletedAtIsNil(),
			).
			ForUpdate().
			Only(txCtx)
		if dbent.IsNotFound(err) {
			const insertSQL = `INSERT INTO user_group_qos_usages
				(user_id, group_id, daily_usage_usd, weekly_usage_usd, monthly_usage_usd,
				 daily_window_start, weekly_window_start, monthly_window_start, created_at, updated_at)
				VALUES ($1, $2, $3, $3, $3, $4, $5, $6, $7, $7)
				ON CONFLICT (user_id, group_id) WHERE deleted_at IS NULL DO UPDATE SET
					daily_usage_usd   = user_group_qos_usages.daily_usage_usd   + EXCLUDED.daily_usage_usd,
					weekly_usage_usd  = user_group_qos_usages.weekly_usage_usd  + EXCLUDED.weekly_usage_usd,
					monthly_usage_usd = user_group_qos_usages.monthly_usage_usd + EXCLUDED.monthly_usage_usd,
					updated_at        = EXCLUDED.updated_at`
			// $6 = now：30 天滚动月度窗口以当前时刻为起始
			_, e := txClient.ExecContext(txCtx, insertSQL,
				userID, groupID, cost,
				timezone.StartOfDay(now), timezone.StartOfWeek(now), now, now)
			return e
		}
		if err != nil {
			return err
		}

		newDaily := maybeReset(existing.DailyUsageUsd, existing.DailyWindowStart, timezone.StartOfDay(now), cost)
		newWeekly := maybeReset(existing.WeeklyUsageUsd, existing.WeeklyWindowStart, timezone.StartOfWeek(now), cost)
		newMonthly, newMonthlyStart := monthlyMaybeReset(existing.MonthlyUsageUsd, existing.MonthlyWindowStart, cost, now)

		_, e := existing.Update().
			SetDailyUsageUsd(newDaily).
			SetWeeklyUsageUsd(newWeekly).
			SetMonthlyUsageUsd(newMonthly).
			SetDailyWindowStart(timezone.StartOfDay(now)).
			SetWeeklyWindowStart(timezone.StartOfWeek(now)).
			SetMonthlyWindowStart(newMonthlyStart). // 30 天滚动：仅过期时更新起始
			Save(txCtx)
		return e
	})
}

// ResetWindows 无条件把三个窗口归零并以 now 为新起点（管理员豁免操作）。
// 未命中活跃记录时不视为错误：没有计数器等价于用量为 0。
func (r *userGroupQoSUsageRepository) ResetWindows(ctx context.Context, userID, groupID int64, now time.Time) error {
	client := clientFromContext(ctx, r.client)
	_, err := client.UserGroupQoSUsage.Update().
		Where(
			usergroupqosusage.UserIDEQ(userID),
			usergroupqosusage.GroupIDEQ(groupID),
			usergroupqosusage.DeletedAtIsNil(),
		).
		SetDailyUsageUsd(0).
		SetWeeklyUsageUsd(0).
		SetMonthlyUsageUsd(0).
		SetDailyWindowStart(timezone.StartOfDay(now)).
		SetWeeklyWindowStart(timezone.StartOfWeek(now)).
		SetMonthlyWindowStart(now).
		Save(ctx)
	return err
}

func (r *userGroupQoSUsageRepository) withTx(ctx context.Context, fn func(txCtx context.Context, txClient *dbent.Client) error) error {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return fn(ctx, tx.Client())
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin user_group_qos_usage transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	if err := fn(txCtx, tx.Client()); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit user_group_qos_usage transaction: %w", err)
	}
	return nil
}

func entQoSUsageToRecord(e *dbent.UserGroupQoSUsage) *service.UserGroupQoSUsageRecord {
	if e == nil {
		return nil
	}
	return &service.UserGroupQoSUsageRecord{
		UserID:             e.UserID,
		GroupID:            e.GroupID,
		DailyUsageUSD:      e.DailyUsageUsd,
		WeeklyUsageUSD:     e.WeeklyUsageUsd,
		MonthlyUsageUSD:    e.MonthlyUsageUsd,
		DailyWindowStart:   e.DailyWindowStart,
		WeeklyWindowStart:  e.WeeklyWindowStart,
		MonthlyWindowStart: e.MonthlyWindowStart,
	}
}
