-- Migration 057: Add vendor_id to groups table
-- Allows a group to be associated with a vendor for upstream forwarding

ALTER TABLE groups ADD COLUMN IF NOT EXISTS vendor_id BIGINT;

-- Add index for vendor_id lookups
CREATE INDEX IF NOT EXISTS idx_groups_vendor_id ON groups (vendor_id) WHERE vendor_id IS NOT NULL;

-- Add foreign key constraint
ALTER TABLE groups ADD CONSTRAINT fk_groups_vendor_id
    FOREIGN KEY (vendor_id) REFERENCES vendors(id) ON DELETE SET NULL;
