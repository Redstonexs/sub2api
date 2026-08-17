-- Group QoS degradation snapshot carried by batch image jobs from admission
-- (Submit) through async settlement to the usage row.
--
-- The settlement worker runs outside the request lifecycle, so the admission
-- snapshot must be persisted on the job itself. The three columns mirror
-- usage_logs.group_qos_* and are written together and are NULL together:
--   NULL          = no tier was in effect at submit time (undegraded / fail-open)
--   mask = 0      = a tier was in effect but changed nothing material
--   mask > 0      = bit field: model=1, reasoning=2, rpm=4
-- No default/backfill: existing jobs stay NULL.
ALTER TABLE batch_image_jobs
    ADD COLUMN IF NOT EXISTS group_qos_tier SMALLINT,
    ADD COLUMN IF NOT EXISTS group_qos_window VARCHAR(16),
    ADD COLUMN IF NOT EXISTS group_qos_effect_mask SMALLINT;
