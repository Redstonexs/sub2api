package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

var _ service.UserGroupQoSCache = (*userGroupQoSCache)(nil)

type userGroupQoSCache struct {
	rdb *redis.Client
}

// NewUserGroupQoSCache 创建 QoS 计数器的热路径缓存。
// 与 BillingCache 分开是刻意的：QoS 不必让既有 BillingCache 测试替身全部改签名。
func NewUserGroupQoSCache(rdb *redis.Client) service.UserGroupQoSCache {
	return &userGroupQoSCache{rdb: rdb}
}

const (
	qosFieldDailyUsage    = "daily_usage"
	qosFieldWeeklyUsage   = "weekly_usage"
	qosFieldMonthlyUsage  = "monthly_usage"
	qosFieldDailyStart    = "daily_start"
	qosFieldWeeklyStart   = "weekly_start"
	qosFieldMonthlyStart  = "monthly_start"
	qosFieldSchemaVersion = "schema_version"
)

func userGroupQoSCacheKey(userID, groupID int64) string {
	return fmt.Sprintf("billing:user_group_qos:%d:%d", userID, groupID)
}

// incrUserGroupQoSUsageScript 仅在条目已存在且 schema 版本匹配时累加。
//
// key 不存在时故意 no-op 而不是建条目：窗口起点只有从 DB 装载时才可信，
// 凭空建出的条目会缺少 *_start，读取端无法判断窗口是否已滚动。
// 未命中由下次读取的 cache-aside 装载补齐。
//
// 三个窗口一律累加；哪些仍然有效由读取端按 *_start 判定。
const incrUserGroupQoSUsageScript = `
if redis.call("EXISTS", KEYS[1]) == 0 then
    return 0
end
local ver = redis.call("HGET", KEYS[1], "schema_version")
if ver == false or tonumber(ver) ~= tonumber(ARGV[2]) then
    return 0
end
redis.call("HINCRBYFLOAT", KEYS[1], "daily_usage", ARGV[1])
redis.call("HINCRBYFLOAT", KEYS[1], "weekly_usage", ARGV[1])
redis.call("HINCRBYFLOAT", KEYS[1], "monthly_usage", ARGV[1])
redis.call("EXPIRE", KEYS[1], ARGV[3])
return 1
`

func (c *userGroupQoSCache) GetUserGroupQoSUsage(ctx context.Context, userID, groupID int64) (*service.UserGroupQoSUsageRecord, error) {
	data, err := c.rdb.HGetAll(ctx, userGroupQoSCacheKey(userID, groupID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	if v, _ := strconv.Atoi(data[qosFieldSchemaVersion]); v != service.UserGroupQoSCacheSchemaV1 {
		// 版本不匹配按未命中处理，由调用方回源 DB 重建。
		return nil, nil
	}

	record := &service.UserGroupQoSUsageRecord{
		UserID:             userID,
		GroupID:            groupID,
		DailyUsageUSD:      parseQoSFloat(data[qosFieldDailyUsage]),
		WeeklyUsageUSD:     parseQoSFloat(data[qosFieldWeeklyUsage]),
		MonthlyUsageUSD:    parseQoSFloat(data[qosFieldMonthlyUsage]),
		DailyWindowStart:   parseQoSTime(data[qosFieldDailyStart]),
		WeeklyWindowStart:  parseQoSTime(data[qosFieldWeeklyStart]),
		MonthlyWindowStart: parseQoSTime(data[qosFieldMonthlyStart]),
	}
	return record, nil
}

func (c *userGroupQoSCache) SetUserGroupQoSUsage(ctx context.Context, record *service.UserGroupQoSUsageRecord, ttl time.Duration) error {
	if record == nil {
		return nil
	}
	key := userGroupQoSCacheKey(record.UserID, record.GroupID)
	values := []any{
		qosFieldDailyUsage, strconv.FormatFloat(record.DailyUsageUSD, 'f', -1, 64),
		qosFieldWeeklyUsage, strconv.FormatFloat(record.WeeklyUsageUSD, 'f', -1, 64),
		qosFieldMonthlyUsage, strconv.FormatFloat(record.MonthlyUsageUSD, 'f', -1, 64),
		qosFieldDailyStart, formatQoSTime(record.DailyWindowStart),
		qosFieldWeeklyStart, formatQoSTime(record.WeeklyWindowStart),
		qosFieldMonthlyStart, formatQoSTime(record.MonthlyWindowStart),
		qosFieldSchemaVersion, service.UserGroupQoSCacheSchemaV1,
	}
	pipe := c.rdb.TxPipeline()
	pipe.HSet(ctx, key, values...)
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (c *userGroupQoSCache) IncrUserGroupQoSUsage(ctx context.Context, userID, groupID int64, cost float64, ttl time.Duration) error {
	_, err := c.rdb.Eval(ctx, incrUserGroupQoSUsageScript,
		[]string{userGroupQoSCacheKey(userID, groupID)},
		strconv.FormatFloat(cost, 'f', -1, 64),
		service.UserGroupQoSCacheSchemaV1,
		int(ttl.Seconds()),
	).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}
	return nil
}

func (c *userGroupQoSCache) InvalidateUserGroupQoSUsage(ctx context.Context, userID, groupID int64) error {
	return c.rdb.Del(ctx, userGroupQoSCacheKey(userID, groupID)).Err()
}

func parseQoSFloat(raw string) float64 {
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return v
}

// 窗口起点以 Unix 秒存储；空串/0/非法值表示"未初始化"。
func parseQoSTime(raw string) *time.Time {
	if raw == "" {
		return nil
	}
	sec, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || sec == 0 {
		return nil
	}
	t := time.Unix(sec, 0).UTC()
	return &t
}

func formatQoSTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return strconv.FormatInt(t.Unix(), 10)
}
