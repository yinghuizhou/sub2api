-- Add force_claude_code_headers to vendors table
-- When enabled, the gateway injects Claude Code CLI headers (User-Agent, X-App, etc.)
-- for upstream providers that require Claude Code client identity.
ALTER TABLE vendors ADD COLUMN IF NOT EXISTS force_claude_code_headers BOOLEAN NOT NULL DEFAULT FALSE;
