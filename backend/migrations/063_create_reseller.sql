-- users 表新增代理等级
ALTER TABLE users ADD COLUMN IF NOT EXISTS reseller_level INT NOT NULL DEFAULT 0;

-- 代理商批量购码订单
CREATE TABLE IF NOT EXISTS reseller_orders (
    id               BIGSERIAL PRIMARY KEY,
    reseller_id      BIGINT NOT NULL,
    quantity         INT NOT NULL,
    face_value_usd   DECIMAL(20,8) NOT NULL,
    unit_cost_usd    DECIMAL(20,8) NOT NULL,
    total_cost_usd   DECIMAL(20,8) NOT NULL,
    payment_order_id BIGINT,
    status           VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_reseller_orders_reseller_id ON reseller_orders(reseller_id);
