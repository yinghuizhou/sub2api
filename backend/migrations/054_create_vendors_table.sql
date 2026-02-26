-- 054_create_vendors_table.sql
-- 创建供应商表，用于管理第三方 API 转发服务供应商

CREATE TABLE IF NOT EXISTS vendors (
    id BIGSERIAL PRIMARY KEY,

    -- 基本信息
    name VARCHAR(100) NOT NULL,
    description TEXT,

    -- API 配置
    api_format VARCHAR(20) NOT NULL,          -- anthropic | openai
    base_url VARCHAR(500) NOT NULL,           -- 供应商 API 地址
    auth_type VARCHAR(20) NOT NULL DEFAULT 'api_key', -- api_key | session | bearer
    api_path_override VARCHAR(500),           -- 自定义 API 路径覆盖
    extra_headers JSONB NOT NULL DEFAULT '{}', -- 额外请求头

    -- 计费信息
    billing_type VARCHAR(20) NOT NULL DEFAULT 'token', -- token | quota | subscription
    cost_per_1k_input DECIMAL(20,8),
    cost_per_1k_output DECIMAL(20,8),
    total_quota_usd DECIMAL(20,8),
    used_quota_usd DECIMAL(20,8) NOT NULL DEFAULT 0,
    balance_usd DECIMAL(20,8),
    expires_at TIMESTAMPTZ,

    -- 健康监控
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    health_check_enabled BOOLEAN NOT NULL DEFAULT false,
    health_check_interval INTEGER NOT NULL DEFAULT 300,
    health_check_model VARCHAR(50) NOT NULL DEFAULT 'claude-sonnet-4-20250514',
    last_health_check_at TIMESTAMPTZ,
    last_health_status VARCHAR(20),
    last_health_latency INTEGER,
    error_message TEXT,
    consecutive_failures INTEGER NOT NULL DEFAULT 0,

    -- 自动采购
    auto_purchase_enabled BOOLEAN NOT NULL DEFAULT false,
    auto_purchase_config JSONB NOT NULL DEFAULT '{}',

    -- 余额预警
    balance_alert_enabled BOOLEAN NOT NULL DEFAULT false,
    balance_alert_threshold DECIMAL(20,8),

    -- 时间戳
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- 索引
CREATE INDEX idx_vendors_status ON vendors (status);
CREATE INDEX idx_vendors_api_format ON vendors (api_format);
CREATE INDEX idx_vendors_billing_type ON vendors (billing_type);
CREATE INDEX idx_vendors_deleted_at ON vendors (deleted_at);

-- 部分唯一索引：name 在未删除记录中唯一
CREATE UNIQUE INDEX idx_vendors_name_unique ON vendors (name) WHERE deleted_at IS NULL;
