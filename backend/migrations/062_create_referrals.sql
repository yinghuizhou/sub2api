-- 邀请关系
CREATE TABLE referrals (
    id          BIGSERIAL PRIMARY KEY,
    inviter_id  BIGINT NOT NULL,
    invitee_id  BIGINT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_referrals_inviter_id ON referrals(inviter_id);

-- 返佣记录
CREATE TABLE referral_commissions (
    id                BIGSERIAL PRIMARY KEY,
    inviter_id        BIGINT NOT NULL,
    invitee_id        BIGINT NOT NULL,
    order_id          BIGINT NOT NULL,
    order_amount_usd  DECIMAL(20,8) NOT NULL,
    commission_rate   DECIMAL(5,4) NOT NULL,
    commission_amount DECIMAL(20,8) NOT NULL,
    status            VARCHAR(20) NOT NULL DEFAULT 'settled',
    settled_at        TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_referral_commissions_inviter_id ON referral_commissions(inviter_id);

-- users 表新增邀请码字段
ALTER TABLE users ADD COLUMN IF NOT EXISTS invite_code VARCHAR(16) UNIQUE;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_invite_code ON users(invite_code) WHERE invite_code IS NOT NULL;
