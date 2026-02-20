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

// ReferralCommission holds the schema definition for the ReferralCommission entity.
type ReferralCommission struct{ ent.Schema }

func (ReferralCommission) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "referral_commissions"}}
}

func (ReferralCommission) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("inviter_id"),
		field.Int64("invitee_id"),
		field.Int64("order_id"),
		field.Float("order_amount_usd").SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("commission_rate").SchemaType(map[string]string{dialect.Postgres: "decimal(5,4)"}),
		field.Float("commission_amount").SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.String("status").MaxLen(20).Default("settled"),
		field.Time("settled_at").Optional().Nillable(),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

func (ReferralCommission) Indexes() []ent.Index {
	return []ent.Index{index.Fields("inviter_id")}
}
