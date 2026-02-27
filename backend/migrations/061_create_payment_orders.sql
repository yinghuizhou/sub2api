-- 充值套餐配置
CREATE TABLE recharge_packages (
    id          BIGSERIAL PRIMARY KEY,
    amount      DECIMAL(20,8) NOT NULL,
    bonus_rate  DECIMAL(5,4) NOT NULL DEFAULT 0,
    bonus_fixed DECIMAL(20,8) NOT NULL DEFAULT 0,
    label       VARCHAR(50),
    is_active   BOOLEAN NOT NULL DEFAULT true,
    sort_order  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 支付订单
CREATE TABLE payment_orders (
    id           BIGSERIAL PRIMARY KEY,
    order_no     VARCHAR(64) UNIQUE NOT NULL,
    user_id      BIGINT NOT NULL,
    amount       DECIMAL(20,8) NOT NULL,
    bonus        DECIMAL(20,8) NOT NULL DEFAULT 0,
    total_credit DECIMAL(20,8) NOT NULL,
    channel      VARCHAR(20) NOT NULL,
    status       VARCHAR(20) NOT NULL DEFAULT 'pending',
    trade_no     VARCHAR(128),
    paid_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payment_orders_user_id ON payment_orders(user_id);
CREATE INDEX idx_payment_orders_status ON payment_orders(status);
CREATE INDEX idx_payment_orders_created_at ON payment_orders(created_at);
