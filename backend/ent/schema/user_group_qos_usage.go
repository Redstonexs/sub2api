package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
)

// UserGroupQoSUsage holds per-user per-group rolling-window consumption that
// drives the group QoS degradation ladder.
//
// Deliberately separate from user_subscriptions: that counter only exists for
// subscription-type groups and accumulates charged cost, whereas QoS must work
// for balance groups too and defaults to undiscounted list cost. One table here
// keeps a single code path for both billing modes.
type UserGroupQoSUsage struct {
	ent.Schema
}

func (UserGroupQoSUsage) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "user_group_qos_usages"},
	}
}

func (UserGroupQoSUsage) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (UserGroupQoSUsage) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.Int64("group_id"),

		// 当前窗口已用量（USD）。口径由 groups.qos_metric 决定。
		field.Float("daily_usage_usd").
			Default(0).
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}),
		field.Float("weekly_usage_usd").
			Default(0).
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}),
		field.Float("monthly_usage_usd").
			Default(0).
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}),

		// 窗口起点（NULL = 尚未初始化，由首次累加时填充）
		field.Time("daily_window_start").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("weekly_window_start").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("monthly_window_start").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (UserGroupQoSUsage) Indexes() []ent.Index {
	return []ent.Index{
		// 软删除友好：只对未删记录唯一
		index.Fields("user_id", "group_id").
			Unique().
			Annotations(entsql.IndexWhere("deleted_at IS NULL")),
		index.Fields("group_id"),
	}
}
