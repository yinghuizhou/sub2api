-- Add unique constraint on referral_commissions.order_id to prevent duplicate commission settlement
CREATE UNIQUE INDEX IF NOT EXISTS idx_referral_commissions_order_id ON referral_commissions (order_id);
