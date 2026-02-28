-- VPN alert rules for automated monitoring notifications
CREATE TABLE IF NOT EXISTS vpn_alert_rules (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    condition   VARCHAR(50)  NOT NULL,
    threshold   INT          NOT NULL,
    webhook_url TEXT         NOT NULL DEFAULT '',
    enabled     BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Add config tracking fields to proxies
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS config_version    INT DEFAULT 1;
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS last_connected_at TIMESTAMPTZ;
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS config_stale      BOOLEAN DEFAULT FALSE;
