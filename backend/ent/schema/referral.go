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

// Referral holds the schema definition for the Referral entity.
type Referral struct{ ent.Schema }

func (Referral) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "referrals"}}
}

func (Referral) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("inviter_id"),
		field.Int64("invitee_id").Unique(),
		field.Time("created_at").Immutable().Default(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (Referral) Indexes() []ent.Index {
	return []ent.Index{index.Fields("inviter_id")}
}
