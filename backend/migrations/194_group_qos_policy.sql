-- Per-group QoS degradation ladder.
--
-- Today a group's daily/weekly/monthly USD limits are binary: full service until
-- the limit, then a hard rejection. QoS replaces that cliff with a ladder — as a
-- user's consumption in the group climbs, progressively cheaper service is
-- delivered (model reroute / reasoning-effort ceiling / RPM squeeze), with a
-- hard block only as the final rung.
--
-- qos_tiers is an ordered array, ascending in severity. The highest-index tier
-- whose window usage has reached its threshold is the one in effect:
--   [{"window":"daily","threshold_usd":50,
--     "model_mappings":[{"from":"gpt-5.6-sol*","to":"gpt-5.6-terra"}],
--     "max_reasoning_effort":"medium"},
--    {"window":"daily","threshold_usd":200,"block":true}]
ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS qos_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS qos_metric VARCHAR(16) NOT NULL DEFAULT 'list',
    ADD COLUMN IF NOT EXISTS qos_tiers JSONB NOT NULL DEFAULT '[]'::jsonb;

COMMENT ON COLUMN groups.qos_enabled IS '是否启用分组 QoS 降级阶梯。';
COMMENT ON COLUMN groups.qos_metric IS 'QoS 阈值计量口径：list=未打折原价（默认），charged=实际扣费。';
COMMENT ON COLUMN groups.qos_tiers IS 'QoS 阶梯数组，按严重程度升序；命中的最高档位整档生效。';

-- QoS fields are part of the API-key auth snapshot and are read on the gateway
-- hot path, so out-of-band group edits (direct SQL, or a crash between the
-- update and app-level invalidation) must not leave cached snapshots serving a
-- stale ladder. Normal admin saves already invalidate via
-- InvalidateAuthCacheByGroupID; this trigger is the durable backstop. Based on
-- the latest function body from
-- 193_group_profit_control_auth_cache_invalidation.sql.
CREATE OR REPLACE FUNCTION enqueue_group_auth_cache_invalidation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    target_group_id BIGINT;
BEGIN
    target_group_id := OLD.id;
    IF TG_OP = 'UPDATE'
       AND OLD.status IS NOT DISTINCT FROM NEW.status
       AND OLD.is_exclusive IS NOT DISTINCT FROM NEW.is_exclusive
       AND OLD.allow_image_generation IS NOT DISTINCT FROM NEW.allow_image_generation
       AND OLD.platform IS NOT DISTINCT FROM NEW.platform
       AND OLD.subscription_type IS NOT DISTINCT FROM NEW.subscription_type
       AND OLD.rate_multiplier IS NOT DISTINCT FROM NEW.rate_multiplier
       AND OLD.peak_rate_enabled IS NOT DISTINCT FROM NEW.peak_rate_enabled
       AND OLD.peak_start IS NOT DISTINCT FROM NEW.peak_start
       AND OLD.peak_end IS NOT DISTINCT FROM NEW.peak_end
       AND OLD.peak_rate_multiplier IS NOT DISTINCT FROM NEW.peak_rate_multiplier
       AND OLD.profit_control_enabled IS NOT DISTINCT FROM NEW.profit_control_enabled
       AND OLD.profit_min_margin IS NOT DISTINCT FROM NEW.profit_min_margin
       AND OLD.profit_safety_buffer IS NOT DISTINCT FROM NEW.profit_safety_buffer
       AND OLD.qos_enabled IS NOT DISTINCT FROM NEW.qos_enabled
       AND OLD.qos_metric IS NOT DISTINCT FROM NEW.qos_metric
       AND OLD.qos_tiers IS NOT DISTINCT FROM NEW.qos_tiers
       AND OLD.deleted_at IS NOT DISTINCT FROM NEW.deleted_at THEN
        RETURN NEW;
    END IF;

    INSERT INTO auth_cache_invalidation_outbox (cache_key)
    SELECT encode(sha256(convert_to(k.key, 'UTF8')), 'hex')
    FROM api_keys AS k
    WHERE k.group_id = target_group_id
      AND k.deleted_at IS NULL
      AND k.key <> '';
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;
