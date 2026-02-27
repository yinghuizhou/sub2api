package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// RechargePackage holds the schema definition for the RechargePackage entity.
type RechargePackage struct{ ent.Schema }

func (RechargePackage) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "recharge_packages"}}
}

func (RechargePackage) Fields() []ent.Field {
	return []ent.Field{
		field.Float("amount").SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("bonus_rate").SchemaType(map[string]string{dialect.Postgres: "decimal(5,4)"}).Default(0),
		field.Float("bonus_fixed").SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).Default(0),
		field.String("label").MaxLen(50).Optional().Nillable(),
		field.Bool("is_active").Default(true),
		field.Int("sort_order").Default(0),
		field.Time("created_at").Immutable().Default(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}
