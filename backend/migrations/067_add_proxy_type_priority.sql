-- Add proxy_type and priority fields for multi-layer proxy pool support
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS proxy_type VARCHAR(20) NOT NULL DEFAULT 'static';
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS priority INTEGER NOT NULL DEFAULT 100;
CREATE INDEX IF NOT EXISTS idx_proxies_proxy_type ON proxies (proxy_type) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_proxies_priority ON proxies (priority) WHERE deleted_at IS NULL;
UPDATE proxies SET proxy_type = 'astrill', priority = 90 WHERE ovpn_config IS NOT NULL AND ovpn_config != '';
