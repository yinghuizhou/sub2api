-- Account proxy assignments: persistent binding between accounts and proxies.
-- Once assigned, an account stays on the same proxy (region) permanently.
CREATE TABLE IF NOT EXISTS account_proxy_assignments (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    proxy_id BIGINT NOT NULL REFERENCES proxies(id) ON DELETE CASCADE,
    proxy_group VARCHAR(100) NOT NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_by VARCHAR(50) NOT NULL DEFAULT 'auto',
    UNIQUE(account_id)
);
CREATE INDEX IF NOT EXISTS idx_apa_proxy_id ON account_proxy_assignments(proxy_id);
CREATE INDEX IF NOT EXISTS idx_apa_proxy_group ON account_proxy_assignments(proxy_group);
