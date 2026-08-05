-- 用户 × 分组维度 QoS 消耗表。驱动 groups.qos_tiers 降级阶梯。
--
-- 与 user_subscriptions 的窗口用量刻意分开：后者只对 subscription 类型分组存在，
-- 且累计的是「实际扣费」；QoS 需要同时覆盖余额（standard）分组，且默认按
-- 「未打折原价」计量——折扣越深越容易掩盖真实消耗。单独一张表让两种计费模式
-- 共用同一条代码路径，无需按 subscription_type 分叉。
--
-- 软删除：deleted_at IS NULL 的记录为活跃记录，部分唯一索引保证同用户同分组
-- 只有一条活跃计数器。

CREATE TABLE IF NOT EXISTS user_group_qos_usages (
    id                   BIGSERIAL PRIMARY KEY,
    user_id              BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id             BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,

    -- 当前窗口已用量（口径由 groups.qos_metric 决定）
    daily_usage_usd      DECIMAL(20,10) NOT NULL DEFAULT 0,
    weekly_usage_usd     DECIMAL(20,10) NOT NULL DEFAULT 0,
    monthly_usage_usd    DECIMAL(20,10) NOT NULL DEFAULT 0,

    -- 窗口起点（NULL = 首次尚未初始化，由首次累加填充）
    daily_window_start   TIMESTAMPTZ,
    weekly_window_start  TIMESTAMPTZ,
    monthly_window_start TIMESTAMPTZ,

    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMPTZ
);

-- 软删除友好唯一索引：同用户同分组只允许一条未删除记录
CREATE UNIQUE INDEX IF NOT EXISTS usergroupqosusage_user_id_group_id_uq
    ON user_group_qos_usages (user_id, group_id)
    WHERE deleted_at IS NULL;

-- 按 group_id 查询（分组停用/删除、管理员按分组查看消耗排行）
CREATE INDEX IF NOT EXISTS usergroupqosusage_group_id
    ON user_group_qos_usages (group_id);

COMMENT ON TABLE user_group_qos_usages IS '用户×分组 QoS 滚动窗口消耗计数器';
COMMENT ON COLUMN user_group_qos_usages.daily_usage_usd IS '当前日窗口已消耗（USD），口径见 groups.qos_metric';
COMMENT ON COLUMN user_group_qos_usages.daily_window_start IS '日窗口起点；NULL 表示尚未初始化';
