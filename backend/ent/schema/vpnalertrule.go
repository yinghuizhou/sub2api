package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// VpnAlertRule holds the schema definition for VPN alert rules.
// Alert rules define conditions and thresholds for automated VPN monitoring notifications.
type VpnAlertRule struct {
	ent.Schema
}

func (VpnAlertRule) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "vpn_alert_rules"},
	}
}

// Mixin provides created_at and updated_at via TimeMixin.
func (VpnAlertRule) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (VpnAlertRule) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(100).
			NotEmpty(),
		field.String("condition").
			MaxLen(50).
			NotEmpty(),
		field.Int("threshold"),
		field.String("webhook_url").
			Default(""),
		field.Bool("enabled").
			Default(true),
	}
}
