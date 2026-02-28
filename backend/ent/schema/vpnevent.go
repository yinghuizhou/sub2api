package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// VpnEvent holds the schema definition for VPN monitoring events.
// This is an append-only log table — no updated_at or soft delete.
type VpnEvent struct {
	ent.Schema
}

func (VpnEvent) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "vpn_events"},
	}
}

func (VpnEvent) Fields() []ent.Field {
	return []ent.Field{
		field.String("tunnel_name").
			MaxLen(100).
			NotEmpty(),
		field.String("event_type").
			MaxLen(50).
			NotEmpty(),
		field.JSON("details", map[string]interface{}{}).
			Optional().
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (VpnEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tunnel_name"),
		index.Fields("event_type"),
		index.Fields("created_at"),
	}
}
