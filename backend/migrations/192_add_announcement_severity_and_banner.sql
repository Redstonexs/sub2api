-- 公告新增严重级别与常驻横幅开关。
-- severity 与 notify_mode 正交：notify_mode 决定投递渠道，severity 只影响展示样式。
-- show_banner 同样是正交开关，用布尔值而不是新增 notify_mode 枚举值，
-- 这样“横幅 + 邮件”等组合仍然可以表达。
ALTER TABLE announcements ADD COLUMN IF NOT EXISTS severity VARCHAR(16) NOT NULL DEFAULT 'info';
ALTER TABLE announcements ADD COLUMN IF NOT EXISTS show_banner BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN announcements.severity IS '严重级别: info, warning, critical';
COMMENT ON COLUMN announcements.show_banner IS '是否在页面顶部展示常驻横幅（与 notify_mode 正交）';
