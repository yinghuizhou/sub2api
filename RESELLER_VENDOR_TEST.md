# Reseller 供应商测试指南

## 修复内容

修复了 reseller 类型供应商无法测试的问题。

### 问题描述
之前创建 reseller 类型的供应商时，即使填写了 `reseller_api_key` 字段，点击"测试"按钮仍然会失败，因为健康检查服务没有使用这个 key。

### 修复方案
在 `vendor_health_service.go` 中增加了自动注入逻辑：
- 检测供应商类型是否为 `reseller`
- 如果有 `reseller_api_key` 但 `extra_headers` 中没有认证信息
- 根据 API 格式自动添加认证头：
  - **OpenAI 格式**: `Authorization: Bearer <key>`
  - **Anthropic 格式**: `x-api-key: <key>`

## 测试步骤

### 方式一：通过前端界面测试

1. 访问 http://localhost:5174/admin/vendors
2. 点击"添加供应商"
3. 填写表单：
   - **名称**: 测试 Reseller 供应商
   - **渠道类型**: 选择"二次分发"
   - **二次分发平台**: 选择任意平台（如 Sub2API）
   - **分发商 API Key**: 填写测试 key（如 `sk-test-123`）
   - **API 格式**: 选择 OpenAI 或 Anthropic
   - **Base URL**: 填写上游 URL
   - **认证方式**: API Key
   - **计费模式**: Token 计费
   - **健康检查**: 启用
   - **检查模型**: 填写对应的模型名
4. 点击"创建"
5. 在列表中找到刚创建的供应商，点击"测试"按钮
6. 观察测试结果

### 方式二：通过 API 测试

使用提供的测试脚本：

```bash
# 设置管理员 API Key
export ADMIN_API_KEY="your-admin-api-key"

# 运行测试脚本
./test_reseller_vendor.sh
```

## 预期结果

### 修复前
- 测试会失败，提示认证错误（401 Unauthorized）
- 必须在 `extra_headers` 中手动配置认证信息才能测试

### 修复后
- 只需填写 `reseller_api_key`，测试会自动使用这个 key
- 如果 key 有效，测试成功
- 如果 key 无效，会返回上游的错误信息（如 invalid API key）
- 不再需要手动配置 `extra_headers`

## 注意事项

1. **优先级**: 如果 `extra_headers` 中已经配置了认证信息（`Authorization` 或 `x-api-key`），会优先使用 `extra_headers` 中的配置
2. **格式匹配**: 认证头格式会根据 `api_format` 字段自动选择
3. **向后兼容**: 不影响已有的供应商配置，已经在 `extra_headers` 中配置认证的供应商继续正常工作

## 代码变更

文件: `backend/internal/service/vendor_health_service.go`

```go
// 对于 reseller 类型的供应商，如果有 reseller_api_key 但 extra_headers 中没有认证信息，自动添加
if vendor.VendorType == "reseller" && vendor.ResellerAPIKey != nil && *vendor.ResellerAPIKey != "" {
    // 检查是否已经有认证头
    hasAuth := false
    for k := range vendor.ExtraHeaders {
        lowerKey := strings.ToLower(k)
        if lowerKey == "authorization" || lowerKey == "x-api-key" {
            hasAuth = true
            break
        }
    }
    // 如果没有认证头，根据 API 格式自动添加
    if !hasAuth {
        if vendor.APIFormat == VendorAPIFormatAnthropic {
            req.Header.Set("x-api-key", *vendor.ResellerAPIKey)
        } else {
            req.Header.Set("Authorization", "Bearer "+*vendor.ResellerAPIKey)
        }
    }
}
```

## 提交信息

```
fix(vendors): support reseller_api_key in health check

- Auto-inject reseller_api_key as auth header when testing reseller vendors
- Anthropic format: use x-api-key header
- OpenAI format: use Authorization Bearer header
- extra_headers takes precedence if auth is already configured
- Fixes issue where reseller vendors couldn't be tested without manually configuring extra_headers
```
