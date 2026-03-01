package domain

const (
	VendorTypeOfficial = "official"
	VendorTypeReseller = "reseller"

	OfficialPlatformClaude = "claude"
	OfficialPlatformOpenAI = "openai"
	OfficialPlatformGemini = "gemini"

	ResellerPlatformSub2API = "sub2api"
	ResellerPlatformNewAPI  = "newapi"
	ResellerPlatformOther   = "other"
)

// VendorHelper 提供 Vendor 相关的辅助方法
type VendorHelper struct{}

// IsOfficial 判断是否为官方渠道
func (VendorHelper) IsOfficial(vendorType string) bool {
	return vendorType == VendorTypeOfficial
}

// IsReseller 判断是否为二次分发渠道
func (VendorHelper) IsReseller(vendorType string) bool {
	return vendorType == VendorTypeReseller
}

// GetPlatformName 获取平台显示名称
func (VendorHelper) GetPlatformName(vendorType, officialPlatform, resellerPlatform string) string {
	if vendorType == VendorTypeOfficial && officialPlatform != "" {
		return officialPlatform
	}
	if vendorType == VendorTypeReseller && resellerPlatform != "" {
		return resellerPlatform
	}
	return "unknown"
}
