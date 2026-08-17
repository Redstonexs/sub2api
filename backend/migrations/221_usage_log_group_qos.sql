-- Group QoS degradation snapshot on served usage rows.
--
-- The three columns record the admission-time QoS tier in effect for a served
-- request and the material effects it actually had (not merely that a tier was
-- active). They are written together and are NULL together:
--   NULL          = no tier was in effect (undegraded / fail-open / legacy row)
--   mask = 0      = a tier was in effect but changed nothing material
--   mask > 0      = bit field: model=1, reasoning=2, rpm=4
-- No default/backfill: legacy rows stay NULL.
--
-- A hard block is never represented here: blocked requests are rejected at
-- admission and never produce a served usage row.
ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS group_qos_tier SMALLINT,
    ADD COLUMN IF NOT EXISTS group_qos_window VARCHAR(16),
    ADD COLUMN IF NOT EXISTS group_qos_effect_mask SMALLINT;
