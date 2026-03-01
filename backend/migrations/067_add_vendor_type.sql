-- 添加渠道类型字段
ALTER TABLE vendors ADD COLUMN vendor_type VARCHAR(20) NOT NULL DEFAULT 'official';
ALTER TABLE vendors ADD COLUMN official_platform VARCHAR(50);
ALTER TABLE vendors ADD COLUMN reseller_platform VARCHAR(100);
ALTER TABLE vendors ADD COLUMN reseller_api_key VARCHAR(500);

-- 添加索引
CREATE INDEX idx_vendors_vendor_type ON vendors(vendor_type);

-- 添加注释
COMMENT ON COLUMN vendors.vendor_type IS '渠道类型：official (官方渠道) | reseller (二次分发渠道)';
COMMENT ON COLUMN vendors.official_platform IS '官方平台：claude | openai | gemini (仅 vendor_type=official 时使用)';
COMMENT ON COLUMN vendors.reseller_platform IS '渠道平台：sub2api | newapi | other (仅 vendor_type=reseller 时使用)';
COMMENT ON COLUMN vendors.reseller_api_key IS '渠道商主 API Key，用于查询余额、自动采购等 (仅 vendor_type=reseller 时使用)';

-- 更新现有数据（所有现有 Vendor 标记为官方渠道）
UPDATE vendors SET vendor_type = 'official' WHERE vendor_type IS NULL OR vendor_type = '';
