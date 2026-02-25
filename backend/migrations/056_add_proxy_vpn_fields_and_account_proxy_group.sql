-- 056: Add proxy VPN/group fields and account proxy_group
-- Adds new columns to proxies table for VPN management and grouping,
-- and adds proxy_group column to accounts table.

-- ========== proxies table: new VPN & grouping fields ==========

ALTER TABLE proxies ADD COLUMN IF NOT EXISTS region VARCHAR(50);
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS group_name VARCHAR(100);
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS ovpn_config TEXT;
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS ovpn_username VARCHAR(255);
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS ovpn_password VARCHAR(255);
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS is_dedicated BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS vpn_status VARCHAR(20);
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS vpn_exit_ip VARCHAR(45);
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS health_status VARCHAR(20);
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS latency_ms INT;
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS last_health_at TIMESTAMPTZ;
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS health_check_failures INT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_proxies_group_name ON proxies(group_name);
CREATE INDEX IF NOT EXISTS idx_proxies_region ON proxies(region);
CREATE INDEX IF NOT EXISTS idx_proxies_health_status ON proxies(health_status);
CREATE INDEX IF NOT EXISTS idx_proxies_is_dedicated ON proxies(is_dedicated);

-- ========== accounts table: add proxy_group ==========

ALTER TABLE accounts ADD COLUMN IF NOT EXISTS proxy_group VARCHAR(100);

CREATE INDEX IF NOT EXISTS idx_accounts_proxy_group ON accounts(proxy_group);
