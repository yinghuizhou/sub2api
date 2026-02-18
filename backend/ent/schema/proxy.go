package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Proxy holds the schema definition for the Proxy entity.
type Proxy struct {
	ent.Schema
}

func (Proxy) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "proxies"},
	}
}

func (Proxy) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (Proxy) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(100).
			NotEmpty(),
		field.String("protocol").
			MaxLen(20).
			NotEmpty(),
		field.String("host").
			MaxLen(255).
			NotEmpty(),
		field.Int("port"),
		field.String("username").
			MaxLen(100).
			Optional().
			Nillable(),
		field.String("password").
			MaxLen(100).
			Optional().
			Nillable(),
		field.String("status").
			MaxLen(20).
			Default("active"),

		// ========== OpenVPN / 代理分组相关字段 ==========

		// region: 地区标识，如 us-east, us-west, eu-west
		field.String("region").
			MaxLen(50).
			Optional().
			Nillable(),

		// group_name: 代理分组名，如 "美东共享组"、"专用-账号A"
		field.String("group_name").
			MaxLen(100).
			Optional().
			Nillable(),

		// ovpn_config: .ovpn 文件完整内容
		field.String("ovpn_config").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),

		// ovpn_username: OpenVPN 认证用户名（如 Astrill 账号）
		field.String("ovpn_username").
			MaxLen(255).
			Optional().
			Nillable(),

		// ovpn_password: OpenVPN 认证密码
		field.String("ovpn_password").
			MaxLen(255).
			Optional().
			Nillable(),

		// is_dedicated: 是否专用 IP（专用=1账号独占）
		field.Bool("is_dedicated").
			Default(false),

		// vpn_status: VPN 隧道状态 connected / disconnected / error
		field.String("vpn_status").
			MaxLen(20).
			Optional().
			Nillable(),

		// vpn_exit_ip: 实际出口 IP（连接后自动检测）
		field.String("vpn_exit_ip").
			MaxLen(45).
			Optional().
			Nillable(),

		// health_status: healthy / degraded / unhealthy
		field.String("health_status").
			MaxLen(20).
			Optional().
			Nillable(),

		// latency_ms: 最近延迟(ms)
		field.Int("latency_ms").
			Optional().
			Nillable(),

		// last_health_at: 最后健康检查时间
		field.Time("last_health_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),

		// health_check_failures: 连续健康检查失败次数（用于故障转移判断）
		field.Int("health_check_failures").
			Default(0),
	}
}

// Edges 定义代理实体的关联关系。
func (Proxy) Edges() []ent.Edge {
	return []ent.Edge{
		// accounts: 使用此代理的账户（反向边）
		edge.From("accounts", Account.Type).
			Ref("proxy"),
	}
}

func (Proxy) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
		index.Fields("deleted_at"),
		index.Fields("group_name"),
		index.Fields("region"),
		index.Fields("health_status"),
		index.Fields("is_dedicated"),
	}
}
