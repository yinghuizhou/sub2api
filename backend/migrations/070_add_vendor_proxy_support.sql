-- Migration 070: Add vendor proxy support
-- Enables vendors to act as independent scheduling channels alongside accounts
-- Part of Phase 1: Hybrid scheduling implementation

-- ============================================================
-- Part 1: Add vendor proxy fields to accounts table
-- ============================================================

-- is_vendor_proxy: Marks an account as a proxy for a vendor
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS is_vendor_proxy BOOLEAN NOT NULL DEFAULT FALSE;

-- vendor_proxy_id: Links proxy account to its vendor (only valid when is_vendor_proxy=true)
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS vendor_proxy_id BIGINT;

-- Add indexes for vendor proxy queries
CREATE INDEX IF NOT EXISTS idx_accounts_is_vendor_proxy ON accounts (is_vendor_proxy);
CREATE INDEX IF NOT EXISTS idx_accounts_vendor_proxy_id ON accounts (vendor_proxy_id) WHERE vendor_proxy_id IS NOT NULL;

-- Composite index for scheduling queries that include proxy accounts
CREATE INDEX IF NOT EXISTS idx_accounts_scheduling_with_proxy
    ON accounts (platform, schedulable, priority, is_vendor_proxy)
    WHERE deleted_at IS NULL;

-- Optimized index for vendor proxy lookups
CREATE INDEX IF NOT EXISTS idx_accounts_vendor_proxy_lookup
    ON accounts (is_vendor_proxy, vendor_proxy_id)
    WHERE is_vendor_proxy = TRUE;

-- Add foreign key constraint for vendor_proxy_id (idempotent)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_accounts_vendor_proxy_id') THEN
        ALTER TABLE accounts ADD CONSTRAINT fk_accounts_vendor_proxy_id
            FOREIGN KEY (vendor_proxy_id) REFERENCES vendors(id) ON DELETE CASCADE;
    END IF;
END $$;

-- Add comments
COMMENT ON COLUMN accounts.is_vendor_proxy IS '是否为 Vendor 的代理账号（虚拟账号）';
COMMENT ON COLUMN accounts.vendor_proxy_id IS '关联的 Vendor ID（仅 is_vendor_proxy=true 时有效）';

-- ============================================================
-- Part 2: Add scheduling fields to vendors table
-- ============================================================

-- priority: Vendor scheduling priority (lower number = higher priority)
ALTER TABLE vendors ADD COLUMN IF NOT EXISTS priority INT NOT NULL DEFAULT 50;

-- concurrency: Maximum concurrent requests for this vendor
ALTER TABLE vendors ADD COLUMN IF NOT EXISTS concurrency INT NOT NULL DEFAULT 3;

-- Add index for priority-based scheduling
CREATE INDEX IF NOT EXISTS idx_vendors_priority ON vendors (priority);

-- Add comments
COMMENT ON COLUMN vendors.priority IS '供应商优先级（数值越小越优先，用于调度排序）';
COMMENT ON COLUMN vendors.concurrency IS '供应商最大并发请求数';

-- ============================================================
-- Part 3: Add validation constraints
-- ============================================================

-- Ensure vendor_proxy_id is only set when is_vendor_proxy is true
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_accounts_vendor_proxy_consistency') THEN
        ALTER TABLE accounts ADD CONSTRAINT chk_accounts_vendor_proxy_consistency
            CHECK (
                (is_vendor_proxy = FALSE AND vendor_proxy_id IS NULL) OR
                (is_vendor_proxy = TRUE AND vendor_proxy_id IS NOT NULL)
            );
    END IF;
END $$;

-- Ensure each vendor has at most one proxy account
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_accounts_vendor_proxy_id') THEN
        CREATE UNIQUE INDEX uq_accounts_vendor_proxy_id
            ON accounts (vendor_proxy_id)
            WHERE is_vendor_proxy = TRUE AND deleted_at IS NULL;
    END IF;
END $$;

-- ============================================================
-- Part 4: Migration notes and rollback instructions
-- ============================================================

-- Migration Summary:
-- 1. Added is_vendor_proxy and vendor_proxy_id to accounts table
-- 2. Added priority and concurrency to vendors table
-- 3. Created indexes for efficient vendor proxy queries
-- 4. Added validation constraints for data consistency
--
-- Rollback Instructions (if needed):
-- ALTER TABLE accounts DROP CONSTRAINT IF EXISTS chk_accounts_vendor_proxy_consistency;
-- DROP INDEX IF EXISTS uq_accounts_vendor_proxy_id;
-- DROP INDEX IF EXISTS idx_accounts_vendor_proxy_lookup;
-- DROP INDEX IF EXISTS idx_accounts_scheduling_with_proxy;
-- DROP INDEX IF EXISTS idx_accounts_vendor_proxy_id;
-- DROP INDEX IF EXISTS idx_accounts_is_vendor_proxy;
-- DROP INDEX IF EXISTS idx_vendors_priority;
-- ALTER TABLE accounts DROP CONSTRAINT IF EXISTS fk_accounts_vendor_proxy_id;
-- ALTER TABLE accounts DROP COLUMN IF EXISTS vendor_proxy_id;
-- ALTER TABLE accounts DROP COLUMN IF EXISTS is_vendor_proxy;
-- ALTER TABLE vendors DROP COLUMN IF EXISTS concurrency;
-- ALTER TABLE vendors DROP COLUMN IF EXISTS priority;
