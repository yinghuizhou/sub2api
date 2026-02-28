-- VPN event log for monitoring and alerting
CREATE TABLE IF NOT EXISTS vpn_events (
    id          BIGSERIAL PRIMARY KEY,
    tunnel_name VARCHAR(100) NOT NULL,
    event_type  VARCHAR(50)  NOT NULL,
    details     JSONB        DEFAULT '{}',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_vpn_events_tunnel ON vpn_events(tunnel_name);
CREATE INDEX IF NOT EXISTS idx_vpn_events_type ON vpn_events(event_type);
CREATE INDEX IF NOT EXISTS idx_vpn_events_time ON vpn_events(created_at DESC);
