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

// PaymentOrder holds the schema definition for the PaymentOrder entity.
type PaymentOrder struct{ ent.Schema }

func (PaymentOrder) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "payment_orders"}}
}

func (PaymentOrder) Fields() []ent.Field {
	return []ent.Field{
		field.String("order_no").MaxLen(64).NotEmpty().Unique(),
		field.Int64("user_id"),
		field.Float("amount_cny").SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("exchange_rate").SchemaType(map[string]string{dialect.Postgres: "decimal(10,6)"}),
		field.Float("amount").SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("bonus").SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).Default(0),
		field.Float("total_credit").SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.String("channel").MaxLen(20),
		field.String("status").MaxLen(20).Default("pending"),
		field.String("commission_status").MaxLen(20).Default("none"),
		field.String("trade_no").MaxLen(128).Optional().Nillable(),
		field.Time("paid_at").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("created_at").Immutable().Default(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (PaymentOrder) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("status"),
		index.Fields("created_at"),
	}
}
