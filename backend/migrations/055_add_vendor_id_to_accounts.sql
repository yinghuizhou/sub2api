-- 055_add_vendor_id_to_accounts.sql
-- 为 accounts 表添加供应商关联字段

ALTER TABLE accounts ADD COLUMN IF NOT EXISTS vendor_id BIGINT REFERENCES vendors(id);
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS source_type VARCHAR(20) NOT NULL DEFAULT 'owned';

CREATE INDEX IF NOT EXISTS idx_accounts_vendor_id ON accounts (vendor_id);
CREATE INDEX IF NOT EXISTS idx_accounts_source_type ON accounts (source_type);
