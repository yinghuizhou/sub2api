# Sub2API 渗透测试综合报告

**测试日期**: 2026-03-09
**测试方式**: 黑客视角 - 7个并行渗透测试
**测试范围**: 全栈安全评估（前端、后端、基础设施、依赖）

---

## 执行摘要

从攻击者视角对 Sub2API 进行了全面的渗透测试，覆盖 OWASP Top 10 和常见攻击向量。

### 风险统计

| 等级 | 数量 | 占比 |
| ---- | ---- | ---- |
| 🔴 **高危** | 9 | 20% |
| 🟡 **中危** | 14 | 31% |
| 🟢 **低危** | 5 | 11% |
| ✅ **安全** | 17 | 38% |
| **总计** | 45 | 100% |

### 总体评分

**安全评分**: 7.5/10（良好）

**评估**: 系统整体安全性良好，核心认证和授权机制实现正确，但存在一些需要修复的高危漏洞，主要集中在：
- 第三方依赖漏洞（npm 包）
- 业务逻辑缺陷（兑换码过期验证）
- DoS 防护不完整（网关 API 无速率限制）

---

## 🔴 高危漏洞（9个）

### 1. 第三方依赖漏洞（7个）

**来源**: npm 依赖

#### 1.1 xlsx 包 - 原型污染 + ReDoS
- **当前版本**: 0.18.5
- **漏洞**: GHSA-4r6h-8v6p-xvw6, GHSA-5pgg-2g8v-p4x9
- **影响**: 恶意 Excel 文件可触发原型污染或 DoS
- **修复**: 升级到 0.20.2+ 或替换为 exceljs

#### 1.2 minimatch - ReDoS 漏洞（多处）
- **受影响**: eslint、@typescript-eslint、@vue/test-utils
- **漏洞**: GHSA-3ppc-4f35-3m26, GHSA-7r86-cg39-jmmj
- **影响**: 恶意 glob 模式导致 CPU 100%
- **修复**: 更新 eslint 工具链

#### 1.3 rollup - 路径遍历漏洞
- **受影响**: vite > rollup
- **版本**: >= 4.0.0 < 4.59.0
- **漏洞**: GHSA-mw96-cpmx-2vgc
- **影响**: 构建时可能写入任意文件
- **修复**: 更新 vite 到最新版本

**修复命令**:
```bash
cd frontend
pnpm update vite@latest
pnpm update eslint @typescript-eslint/eslint-plugin @typescript-eslint/parser
pnpm update xlsx@latest  # 或替换为 exceljs
pnpm audit
```

---

### 2. 业务逻辑漏洞（1个）

#### 2.1 兑换码缺少过期时间验证
- **位置**: `backend/internal/service/redeem_code.go:24-28`
- **问题**: 兑换码一旦生成，永久有效
- **风险**: 泄露的兑换码可无限期使用
- **影响**: 资金损失、无法实现限时促销

**修复建议**:
```go
type RedeemCode struct {
    // ... 现有字段
    ExpiresAt *time.Time  // 新增
}

func (r *RedeemCode) CanUse() bool {
    if r.Status != StatusUnused {
        return false
    }
    if r.ExpiresAt != nil && time.Now().After(*r.ExpiresAt) {
        return false
    }
    return true
}
```

---

### 3. DoS 防护缺陷（1个）

#### 3.1 核心网关 API 无速率限制
- **位置**: `backend/internal/server/routes/gateway.go`
- **未保护端点**:
  - `/v1/messages` - Claude API
  - `/v1/responses` - OpenAI API
  - `/v1beta/models/*` - Gemini API
  - `/sora/v1/chat/completions` - Sora 视频生成
- **风险**: 攻击者可无限制调用 AI API，导致成本爆炸
- **影响**: 资源耗尽、上游账号被封禁

**修复建议**:
```go
// backend/internal/server/routes/gateway.go
rateLimiter := middleware.NewRateLimiter(redisClient)
gateway.Use(rateLimiter.LimitWithOptions("gateway-api", 100, time.Minute,
    middleware.RateLimitOptions{FailureMode: middleware.RateLimitFailClose}))
```

---

## 🟡 中危漏洞（14个）

### 4. 认证与授权（3个）

#### 4.1 缺少账号锁定机制
- **问题**: 仅有 IP 级速率限制
- **风险**: 分布式暴力破解攻击
- **建议**: 添加账号级锁定（5次失败锁定30分钟）

#### 4.2 密码策略过弱
- **当前**: 最小长度 6 字符，无复杂度要求
- **风险**: 允许弱密码（如 "123456"）
- **建议**: 最小 8-12 字符 + 复杂度要求 + 弱密码黑名单

#### 4.3 生产环境 CORS 配置
- **风险**: 需确认未使用 `allowed_origins: ["*"]`
- **建议**: 审查 `deploy/config.yaml` 中的 CORS 配置

---

### 5. 客户端安全（2个）

#### 5.1 HomeView.vue - Admin-only XSS
- **位置**: `frontend/src/views/HomeView.vue:12`
- **问题**: 管理员可注入任意 HTML/JavaScript
- **风险**: 存储型 XSS，影响所有用户
- **修复**: 使用 `DOMPurify.sanitize()`

#### 5.2 缺少 CSRF Token 机制
- **当前**: 仅依赖 `SameSite=Lax` Cookie
- **风险**: 顶级导航仍有 CSRF 风险
- **建议**: 实现完整的 CSRF Token 验证

---

### 6. 数据泄露（2个）

#### 6.1 用户枚举 - 注册接口
- **问题**: 返回 "EMAIL_EXISTS" 错误
- **风险**: 攻击者可枚举已注册用户
- **建议**: 统一响应或添加延迟

#### 6.2 错误信息泄露
- **位置**: 多处 handler（user_handler.go:64, api_key_handler.go:198 等）
- **问题**: 返回 `err.Error()` 泄露数据库细节
- **示例**: `"pq: duplicate key violates unique constraint \"users_email_key\""`
- **建议**: 统一错误处理，避免泄露内部实现

---

### 7. 注入攻击（1个）

#### 7.1 路径遍历 - Sora 媒体代理
- **位置**: `backend/internal/handler/sora_gateway_handler.go:675`
- **问题**: 缺少最终路径验证
- **风险**: 虽然有 `path.Clean()` + 前缀检查，但建议加强
- **修复**: 添加 `filepath.Abs()` + 前缀检查

---

### 8. DoS 防护（2个）

#### 8.1 分页限制不严格
- **问题**: 未发现全局分页大小上限
- **风险**: 攻击者可请求 `?page_size=999999`
- **建议**: 实现全局 MaxPageSize 限制（如100条）

#### 8.2 邮箱正则 ReDoS 风险
- **位置**: `backend/internal/service/registration_email_policy.go:10`
- **风险**: 嵌套量词可能导致回溯
- **缓解**: 已有速率限制保护，但建议添加长度预检查

---

### 9. 业务逻辑（3个）

#### 9.1 透支设计允许负余额
- **位置**: `backend/internal/repository/user_repo.go:345-350`
- **风险**: 存在小额透支风险
- **建议**: 添加透支上限（如 -$1.00）

#### 9.2 余额检查与扣减的 TOCTOU 问题
- **评估**: 这是有意的设计，允许透支以确保当前请求完成
- **建议**: 监控负余额用户，添加告警

#### 9.3 缓存一致性问题
- **位置**: `backend/internal/service/billing_cache_service.go:342-350`
- **问题**: 异步队列可能导致缓存与数据库不一致
- **缓解**: 已有 circuit breaker 和回退机制

---

### 10. 第三方依赖（1个）

#### 10.1 golang.org/x/crypto - SSH DoS 风险
- **当前版本**: v0.48.0
- **已知问题**: SSH 服务器 DoS、CVE-2024-45337
- **影响**: 本项目不直接暴露 SSH，风险较低
- **建议**: 更新到最新版本

---

## 🟢 低危问题（5个）

### 11. 外部攻击面（1个）

#### 11.1 Nginx 版本信息泄露
- **发现**: `server: nginx/1.20.1`
- **风险**: 攻击者可针对该版本的已知漏洞
- **修复**: 在 nginx.conf 中添加 `server_tokens off;`

---

### 12. 认证与授权（2个）

#### 12.1 bcrypt 成本因子可提升
- **当前**: `bcrypt.DefaultCost = 10`
- **建议**: 提升至 12-14（根据服务器性能）

#### 12.2 缺少密码历史检查
- **问题**: 用户可重复使用旧密码
- **建议**: 记录最近 3-5 个密码哈希

---

### 13. 客户端安全（1个）

#### 13.1 CORS 配置缺失
- **问题**: `deploy/docker-compose.yml` 未设置 `CORS_ALLOWED_ORIGINS`
- **建议**: 在生产环境配置文件中添加

---

### 14. 测试覆盖（1个）

#### 14.1 缺失测试用例
- 兑换码并发兑换测试
- 余额透支边界测试
- 缓存失效场景测试

---

## ✅ 安全实现（17个）

### 15. 认证与授权（15个）

- ✅ JWT Token 安全（算法限制、双重验证、密钥强度、版本控制）
- ✅ API 端点授权（Admin API Key + JWT 双重认证）
- ✅ 暴力破解防护（速率限制 + fail-close 模式）
- ✅ Session 安全（Refresh Token、重用检测、Token 轮转）
- ✅ CORS 配置（Origin 白名单、通配符保护）
- ✅ 安全响应头（CSP、X-Frame-Options、X-Content-Type-Options）
- ✅ 时序攻击防护（constant-time 比较）
- ✅ SQL 注入防护（Ent ORM + 参数化查询）
- ✅ 命令注入防护（无用户可控的命令执行）
- ✅ SSRF 防护（URL 验证、IP 白名单、DNS Rebinding 防护）
- ✅ XSS 防护（DOMPurify、escapeHtml）
- ✅ Clickjacking 防护（X-Frame-Options + CSP）
- ✅ Open Redirect 防护（严格的路径验证）
- ✅ IDOR 防护（API Key、Usage、Sora Generation 访问控制）
- ✅ 兑换码重复使用防护（分布式锁 + 数据库事务 + 乐观锁）

### 16. 基础设施（2个）

- ✅ Docker 镜像版本（Go 1.26.1、Alpine 3.21、PostgreSQL 18、Redis 8）
- ✅ 连接池限制（HTTP、数据库、Redis）

---

## 修复优先级

### P0 - 立即修复（本周内）

1. 🔴 **更新前端依赖** - 修复 xlsx、minimatch、rollup 漏洞
2. 🔴 **添加兑换码过期验证** - 防止永久有效的兑换码
3. 🔴 **网关 API 速率限制** - 防止 DoS 和成本爆炸
4. 🟡 **修复 HomeView XSS** - 添加 DOMPurify.sanitize
5. 🟡 **统一错误处理** - 避免泄露数据库细节

### P1 - 高优先级（2周内）

6. 🟡 **实现 CSRF Token 机制**
7. 🟡 **添加账号锁定机制**
8. 🟡 **修复用户枚举漏洞**
9. 🟡 **加强路径遍历防护**
10. 🟡 **添加全局分页限制**
11. 🔴 **更新 golang.org/x/crypto**

### P2 - 中优先级（1个月内）

12. 🟡 **提升密码策略强度**
13. 🟡 **添加透支上限**
14. 🟡 **审查生产环境 CORS 配置**
15. 🟢 **隐藏 Nginx 版本号**
16. 🟢 **配置 CORS 环境变量**

### P3 - 低优先级（可选）

17. 🟢 **提升 bcrypt 成本因子**
18. 🟢 **添加密码历史检查**
19. 🟢 **补充测试用例**

---

## 修复命令清单

### 前端依赖修复

```bash
cd /Users/zhouyinghui/work/ai/sub2api/frontend

# 1. 更新 vite（修复 rollup 漏洞）
pnpm update vite@latest

# 2. 更新 eslint 工具链（修复 minimatch）
pnpm update eslint @typescript-eslint/eslint-plugin @typescript-eslint/parser

# 3. 更新 xlsx（或替换）
pnpm update xlsx@latest
# 或者替换为 exceljs
# pnpm remove xlsx && pnpm add exceljs

# 4. 运行审计验证
pnpm audit

# 5. 测试构建
pnpm run build
pnpm run test
```

### 后端依赖修复

```bash
cd /Users/zhouyinghui/work/ai/sub2api/backend

# 1. 更新 crypto 库
go get -u golang.org/x/crypto golang.org/x/net golang.org/x/sys golang.org/x/term

# 2. 清理依赖
go mod tidy

# 3. 运行测试
go test ./...

# 4. 检查漏洞（需要 govulncheck）
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

---

## 持续安全建议

### 自动化工具集成

1. **GitHub Dependabot**: 自动 PR 更新依赖
2. **Snyk**: 实时漏洞监控
3. **govulncheck**: Go 漏洞扫描（CI/CD 集成）
4. **pnpm audit**: 前端依赖审计（CI/CD 集成）

### CI/CD 集成示例

```yaml
# .github/workflows/security-audit.yml
name: Security Audit
on:
  schedule:
    - cron: '0 0 * * 1'  # 每周一
  push:
    branches: [main, dev]

jobs:
  audit-go:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
      - run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          cd backend && govulncheck ./...

  audit-npm:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: pnpm/action-setup@v2
      - run: cd frontend && pnpm audit --audit-level=high
```

---

## 总结

### 优势

- ✅ 核心认证和授权机制实现正确
- ✅ 完善的 SSRF、SQL 注入、命令注入防护
- ✅ 良好的速率限制和并发控制
- ✅ 使用现代化的安全实践（constant-time 比较、参数化查询）

### 需改进

- 🔴 第三方依赖存在已知漏洞（主要是 npm 包）
- 🔴 业务逻辑存在缺陷（兑换码过期验证）
- 🔴 核心 API 缺少速率限制
- 🟡 错误处理泄露内部信息
- 🟡 密码策略和账号保护可以加强

### 风险评估

**当前风险等级**: 🟡 中等风险

**修复后预期**: 🟢 低风险（8.5/10）

---

**渗透测试完成**: 2026-03-09
**测试人**: Claude Code (7 Parallel Penetration Testing Agents)
**下次测试**: 2026-06-09（3个月后）或重大功能更新后
**报告版本**: 1.0
