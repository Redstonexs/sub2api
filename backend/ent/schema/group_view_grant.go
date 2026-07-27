package schema

import (
	"time"

	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GroupViewGrant holds the schema definition for the group_view_grants table.
// A group_view_grant records whether a regular user is allowed to view a group's
// quota dashboard card — separate from user_allowed_groups (API key binding).
type GroupViewGrant struct {
	ent.Schema
}

func (GroupViewGrant) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "group_view_grants"},
	}
}

func (GroupViewGrant) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (GroupViewGrant) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.Int64("group_id"),
		field.Int64("granted_by_user_id"),
		field.Time("granted_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (GroupViewGrant) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("group_view_grants").
			Field("user_id").
			Required().
			Unique(),
		edge.From("group", Group.Type).
			Ref("group_view_grants").
			Field("group_id").
			Required().
			Unique(),
	}
}

func (GroupViewGrant) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "group_id").
			Unique().
			Annotations(entsql.IndexWhere("deleted_at IS NULL")),
		index.Fields("group_id"),
	}
}
