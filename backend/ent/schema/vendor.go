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

type Vendor struct {
	ent.Schema
}

func (Vendor) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "vendors"},
	}
}

func (Vendor) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (Vendor) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").MaxLen(100).NotEmpty(),
		field.String("description").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "text"}),

		// 渠道类型
		field.String("vendor_type").
			MaxLen(20).
			NotEmpty().
			Default("official").
			Comment("official | reseller"),

		// 官方渠道字段
		field.String("official_platform").
			MaxLen(50).
			Optional().
			Nillable().
			Comment("claude | openai | gemini (仅 vendor_type=official)"),

		// 二次分发渠道字段
		field.String("reseller_platform").
			MaxLen(100).
			Optional().
			Nillable().
			Comment("sub2api | newapi | other (仅 vendor_type=reseller)"),

		field.String("reseller_api_key").
			MaxLen(500).
			Optional().
			Nillable().
			Comment("渠道商主 API Key (仅 vendor_type=reseller)"),

		// API 配置
		field.String("api_format").MaxLen(20).NotEmpty().Comment("anthropic | openai"),
		field.String("base_url").MaxLen(500).NotEmpty().Comment("供应商 API 地址"),
		field.String("auth_type").MaxLen(20).NotEmpty().Default("api_key").Comment("api_key | session | bearer"),
		field.String("api_path_override").MaxLen(500).Optional().Nillable().Comment("自定义 API 路径覆盖"),
		field.JSON("extra_headers", map[string]string{}).Default(map[string]string{}).SchemaType(map[string]string{dialect.Postgres: "jsonb"}).Comment("额外请求头"),

		// 计费信息
		field.String("billing_type").MaxLen(20).NotEmpty().Default("token").Comment("token | quota | subscription"),
		field.Float("cost_per_1k_input").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("cost_per_1k_output").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("total_quota_usd").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("used_quota_usd").Default(0).SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("balance_usd").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Time("expires_at").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),

		// 健康监控
		field.String("status").MaxLen(20).Default("active").Comment("active | suspended | depleted | error"),
		field.Bool("health_check_enabled").Default(false),
		field.Int("health_check_interval").Default(300).Comment("健康检查间隔（秒）"),
		field.String("health_check_model").MaxLen(50).Default("claude-sonnet-4-20250514"),
		field.Time("last_health_check_at").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.String("last_health_status").MaxLen(20).Optional().Nillable().Comment("ok | slow | error | timeout"),
		field.Int("last_health_latency").Optional().Nillable().Comment("最近一次健康检查延迟（ms）"),
		field.String("error_message").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Int("consecutive_failures").Default(0),

		// 自动采购
		field.Bool("auto_purchase_enabled").Default(false),
		field.JSON("auto_purchase_config", map[string]any{}).Default(map[string]any{}).SchemaType(map[string]string{dialect.Postgres: "jsonb"}),

		// 余额预警
		field.Bool("balance_alert_enabled").Default(false),
		field.Float("balance_alert_threshold").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),

		// 请求伪装
		field.Bool("force_claude_code_headers").
			Default(false).
			Comment("强制注入 Claude Code CLI 请求头（User-Agent、X-App 等），用于上游要求 Claude Code 客户端身份的渠道"),

		// ========== 调度相关字段 ==========
		// priority: 供应商优先级（数值越小越优先）
		// 用于调度排序，与 Account.priority 一起参与优先级比较
		field.Int("priority").
			Default(50).
			Comment("供应商优先级（数值越小越优先，用于调度排序）"),

		// concurrency: 供应商最大并发请求数
		// 限制同一时间对该供应商发起的请求数量
		field.Int("concurrency").
			Default(3).
			Comment("供应商最大并发请求数"),
	}
}

func (Vendor) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("accounts", Account.Type).Ref("vendor"),
		// proxy_accounts: 该 Vendor 的代理账号（反向关系）
		// 一个 Vendor 最多有一个代理账号
		edge.From("proxy_accounts", Account.Type).Ref("vendor_proxy"),
	}
}

func (Vendor) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("vendor_type"),
		index.Fields("status"),
		index.Fields("api_format"),
		index.Fields("billing_type"),
		index.Fields("deleted_at"),
		index.Fields("priority"), // 调度优先级排序
	}
}
