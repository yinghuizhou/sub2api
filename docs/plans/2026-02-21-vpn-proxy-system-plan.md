# VPN 代理管理系统实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 OpenVPN 多实例 + SOCKS5 代理层，为 50+ AI 账号提供独立 IP 出口和智能代理分配。

**Architecture:** 每个 OpenVPN 实例绑定独立 tun 设备，配套 3proxy SOCKS5 代理监听 127.0.0.1。Sub2API 后端通过现有 ProxyGroupService 选择代理，新增 PlatformRuleEngine 自动分配和 VPN 管理 API。

**Tech Stack:** OpenVPN client, 3proxy, systemd templates, Python 3 + Playwright (scraper), Go (backend services), Vue 3 (frontend)

---

## Phase 1: 服务器基础设施

### Task 1: OpenVPN 客户端 systemd 模板

**Files:**
- Create: `deploy/vpn/openvpn-client@.service`
- Create: `deploy/vpn/scripts/up.sh`
- Create: `deploy/vpn/scripts/down.sh`
- Create: `deploy/vpn/README.md`

**Step 1: 创建 systemd 模板文件**

```ini
# deploy/vpn/openvpn-client@.service
[Unit]
Description=OpenVPN Client - %i
After=network-online.target
Wants=network-online.target
PartOf=sub2api-vpn.target

[Service]
Type=notify
ExecStart=/usr/sbin/openvpn --config /etc/openvpn/clients/%i.conf
ExecReload=/bin/kill -HUP $MAINPID
Restart=on-failure
RestartSec=10
LimitNPROC=10
DeviceAllow=/dev/null rw
DeviceAllow=/dev/net/tun rw

[Install]
WantedBy=multi-user.target
```

**Step 2: 创建路由隔离 up.sh 脚本**

```bash
#!/bin/bash
# deploy/vpn/scripts/up.sh
# Called by OpenVPN after tunnel is established
# Environment variables set by OpenVPN: dev, ifconfig_local, route_vpn_gateway

INSTANCE_NAME="${dev#tun-}"
TABLE_ID=$((100 + ${INSTANCE_NAME##*-}))

# Add default route through VPN tunnel in dedicated routing table
ip route add default via "$route_vpn_gateway" dev "$dev" table "$TABLE_ID"
ip rule add from "$ifconfig_local" table "$TABLE_ID" priority "$TABLE_ID"

echo "$(date): VPN up: dev=$dev local=$ifconfig_local gw=$route_vpn_gateway table=$TABLE_ID" \
  >> /var/log/sub2api-vpn.log
```

**Step 3: 创建 down.sh 清理脚本**

```bash
#!/bin/bash
# deploy/vpn/scripts/down.sh
INSTANCE_NAME="${dev#tun-}"
TABLE_ID=$((100 + ${INSTANCE_NAME##*-}))

ip rule del from "$ifconfig_local" table "$TABLE_ID" 2>/dev/null
ip route flush table "$TABLE_ID" 2>/dev/null

echo "$(date): VPN down: dev=$dev table=$TABLE_ID" >> /var/log/sub2api-vpn.log
```

**Step 4: 创建部署说明 README**

**Step 5: Commit**

```bash
git add deploy/vpn/
git commit -m "infra: add OpenVPN client systemd template and routing scripts"
```

---

### Task 2: 3proxy SOCKS5 systemd 模板

**Files:**
- Create: `deploy/vpn/3proxy@.service`
- Create: `deploy/vpn/3proxy-template.cfg`

**Step 1: 创建 3proxy systemd 模板**

```ini
# deploy/vpn/3proxy@.service
[Unit]
Description=3proxy SOCKS5 - %i
After=openvpn-client@%i.service
BindsTo=openvpn-client@%i.service

[Service]
Type=simple
ExecStart=/usr/local/bin/3proxy /etc/3proxy/instances/%i.cfg
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

**Step 2: 创建配置模板**

```
# deploy/vpn/3proxy-template.cfg
# Variables: {{PORT}}, {{TUN_IP}}
nscache 65536
nserver 8.8.8.8
nserver 1.1.1.1
timeouts 1 5 30 60 180 1800 15 60
auth none
allow * 127.0.0.1
socks -p{{PORT}} -i127.0.0.1 -e{{TUN_IP}}
```

**Step 3: Commit**

```bash
git add deploy/vpn/3proxy@.service deploy/vpn/3proxy-template.cfg
git commit -m "infra: add 3proxy SOCKS5 systemd template"
```

---

### Task 3: VPN 管理脚本 (vpn-manager.sh)

**Files:**
- Create: `deploy/vpn/vpn-manager.sh`

**Step 1: 编写管理脚本**

功能清单：
- `vpn-manager.sh list` — 列出所有实例及状态
- `vpn-manager.sh deploy <ovpn_file> <instance_name> <socks_port>` — 部署单个实例
- `vpn-manager.sh deploy-batch <json_file>` — 批量部署
- `vpn-manager.sh start|stop|restart <instance_name>` — 控制实例
- `vpn-manager.sh status <instance_name>` — 详细状态
- `vpn-manager.sh remove <instance_name>` — 移除实例
- `vpn-manager.sh health` — 检查所有实例健康

每个命令自动处理：
1. OpenVPN 配置文件放到 `/etc/openvpn/clients/{name}.conf`
2. 注入 `route-nopull`、`dev tun-{name}`、`script-security 2`、`up/down` 脚本路径
3. 生成 3proxy 配置到 `/etc/3proxy/instances/{name}.cfg`
4. 启动 systemd 服务

**Step 2: Commit**

```bash
git add deploy/vpn/vpn-manager.sh
git commit -m "infra: add vpn-manager.sh for OpenVPN+SOCKS5 instance lifecycle"
```

---

### Task 4: 服务器安装脚本

**Files:**
- Create: `deploy/vpn/install.sh`

**Step 1: 编写安装脚本**

安装清单：
1. 安装 OpenVPN 客户端 (`yum install openvpn` / `apt install openvpn`)
2. 编译安装 3proxy（从 GitHub release 下载）
3. 创建目录结构：`/etc/openvpn/clients/`, `/etc/3proxy/instances/`, `/var/log/sub2api-vpn/`
4. 复制 systemd 模板到 `/etc/systemd/system/`
5. 复制 up.sh/down.sh 到 `/etc/openvpn/scripts/` 并 chmod +x
6. 配置 `/etc/iproute2/rt_tables` 添加路由表 (100-200)
7. `systemctl daemon-reload`

**Step 2: Commit**

```bash
git add deploy/vpn/install.sh
git commit -m "infra: add server installation script for VPN infrastructure"
```

---

## Phase 2: Astrill 配置抓取器

### Task 5: Astrill Scraper 基础框架

**Files:**
- Create: `tools/astrill-scraper/requirements.txt`
- Create: `tools/astrill-scraper/scraper.py`
- Create: `tools/astrill-scraper/config.example.json`

**Step 1: 创建 requirements.txt**

```
playwright>=1.40.0
httpx>=0.25.0
```

**Step 2: 创建配置模板**

```json
{
  "astrill_username": "",
  "astrill_password": "",
  "output_dir": "/etc/openvpn/configs",
  "sub2api_url": "http://127.0.0.1:8080",
  "sub2api_api_key": "",
  "auto_register": false
}
```

**Step 3: 编写抓取脚本核心**

功能：
1. `login(page, username, password)` — 登录 Astrill Web 后台
2. `list_servers(page)` — 枚举所有可用 OpenVPN 服务器节点
3. `download_config(page, server)` — 下载单个 .ovpn 配置文件
4. `batch_download(page, servers, output_dir)` — 批量下载
5. `register_proxies(configs, api_url, api_key)` — 调用 Sub2API Admin API 注册代理
6. 输出 JSON 清单供 vpn-manager.sh deploy-batch 使用

**注意**：具体的页面选择器需要在实际登录 Astrill 后台后根据 DOM 结构确定。初始版本提供框架 + 占位选择器，用户需要在首次运行时协助确认。

**Step 4: Commit**

```bash
git add tools/astrill-scraper/
git commit -m "feat: add Astrill OpenVPN config scraper (Playwright)"
```

---

## Phase 3: 后端增强

### Task 6: 平台代理规则配置

**Files:**
- Create: `backend/internal/service/platform_proxy_rules.go`
- Test: `backend/internal/service/platform_proxy_rules_test.go`

**Step 1: 写失败测试**

```go
//go:build unit
package service

func TestPlatformProxyRules_GetRule(t *testing.T) {
    rules := NewPlatformProxyRules()
    rule, ok := rules.GetRule("anthropic")
    assert.True(t, ok)
    assert.Equal(t, 5, rule.MaxAccountsPerIP)
    assert.True(t, rule.StickyIP)
    assert.Contains(t, rule.PreferredRegions, "us-east")
}

func TestPlatformProxyRules_UnknownPlatform(t *testing.T) {
    rules := NewPlatformProxyRules()
    _, ok := rules.GetRule("unknown-platform")
    assert.False(t, ok)
}
```

Run: `cd backend && go test -tags=unit -run TestPlatformProxyRules ./internal/service/...`
Expected: FAIL

**Step 2: 实现规则引擎**

```go
package service

type PlatformProxyRule struct {
    Platform         string
    PreferredRegions []string
    MaxAccountsPerIP int
    StickyIP         bool
}

type PlatformProxyRules struct {
    rules map[string]PlatformProxyRule
}

func NewPlatformProxyRules() *PlatformProxyRules {
    return &PlatformProxyRules{
        rules: map[string]PlatformProxyRule{
            "anthropic": {
                Platform:         "anthropic",
                PreferredRegions: []string{"us-east", "us-central", "us-west"},
                MaxAccountsPerIP: 5,
                StickyIP:         true,
            },
            "gemini": {
                Platform:         "gemini",
                PreferredRegions: []string{"us-east", "us-west", "eu-west"},
                MaxAccountsPerIP: 10,
                StickyIP:         true,
            },
            "openai": {
                Platform:         "openai",
                PreferredRegions: []string{"us-east", "us-west"},
                MaxAccountsPerIP: 8,
                StickyIP:         true,
            },
        },
    }
}

func (r *PlatformProxyRules) GetRule(platform string) (PlatformProxyRule, bool) {
    rule, ok := r.rules[platform]
    return rule, ok
}
```

**Step 3: 运行测试确认通过**

Run: `cd backend && go test -tags=unit -run TestPlatformProxyRules ./internal/service/...`
Expected: PASS

**Step 4: Commit**

```bash
git add backend/internal/service/platform_proxy_rules*.go
git commit -m "feat: add platform proxy rules engine for auto-assignment"
```

---

### Task 7: 代理自动分配服务

**Files:**
- Create: `backend/internal/service/proxy_assignment_service.go`
- Test: `backend/internal/service/proxy_assignment_service_test.go`

**Step 1: 写失败测试**

```go
//go:build unit
package service

func TestProxyAssignmentService_AutoAssign(t *testing.T) {
    // Setup: mock proxyRepo returns groups with account counts
    // Test: anthropic account gets assigned to us-east/us-central group
    // Assert: returned group name matches preferred region
}

func TestProxyAssignmentService_RespectMaxAccountsPerIP(t *testing.T) {
    // Setup: us-east group already has 5 anthropic accounts (at limit)
    // Test: new anthropic account auto-assigns
    // Assert: assigned to next best group (us-central or us-west)
}
```

**Step 2: 实现自动分配**

```go
type ProxyAssignmentService struct {
    proxyRepo ProxyRepository
    rules     *PlatformProxyRules
}

// AutoAssign finds the best proxy group for an account based on platform rules
func (s *ProxyAssignmentService) AutoAssign(
    ctx context.Context,
    platform string,
) (string, error) {
    rule, ok := s.rules.GetRule(platform)
    if !ok {
        return "", nil // no rule = no auto-assignment
    }

    // List all active proxy groups with account counts
    groups, err := s.proxyRepo.ListActiveWithAccountCount(ctx)
    // ... filter by preferred regions, respect MaxAccountsPerIP ...
    // Return best group name
}
```

**Step 3: 运行测试**

**Step 4: Commit**

```bash
git add backend/internal/service/proxy_assignment_service*.go
git commit -m "feat: add proxy auto-assignment service with platform rules"
```

---

### Task 8: ProxyRepository 新增方法

**Files:**
- Modify: `backend/internal/service/proxy_service.go` (ProxyRepository interface)
- Modify: `backend/internal/repository/proxy_repo.go`

**Step 1: 添加接口方法**

在 `ProxyRepository` 接口添加：
```go
// CountAccountsByGroupName returns account count per proxy group
CountAccountsByGroupName(ctx context.Context) (map[string]int64, error)
```

**Step 2: 实现方法**

```go
func (r *proxyRepository) CountAccountsByGroupName(ctx context.Context) (map[string]int64, error) {
    rows, err := r.sql.QueryContext(ctx, `
        SELECT proxy_group, COUNT(*)
        FROM accounts
        WHERE proxy_group IS NOT NULL
          AND proxy_group != ''
          AND deleted_at IS NULL
        GROUP BY proxy_group
    `)
    // ... scan into map ...
}
```

**Step 3: 更新所有 test stubs 实现新接口方法**

**Step 4: Commit**

```bash
git add backend/internal/service/proxy_service.go backend/internal/repository/proxy_repo.go
git commit -m "feat: add CountAccountsByGroupName to ProxyRepository"
```

---

### Task 9: VPN 管理 API 端点

**Files:**
- Modify: `backend/internal/handler/admin/proxy_handler.go`
- Modify: router registration file

**Step 1: 添加新端点**

```go
// POST /api/v1/admin/proxies/:id/health-check — 手动触发单个代理健康检查
// POST /api/v1/admin/proxies/health-check-all — 触发全量健康检查
// POST /api/v1/admin/accounts/:id/auto-assign-proxy — 自动分配代理给账号
// POST /api/v1/admin/accounts/:id/test-proxy — 测试账号当前代理的连通性
```

**Step 2: 实现 handler 方法**

**Step 3: 注册路由**

**Step 4: Commit**

```bash
git commit -m "feat: add VPN management and auto-assign API endpoints"
```

---

### Task 10: Wire DI 注入新服务

**Files:**
- Modify: `backend/internal/service/wire.go`
- Modify: `backend/cmd/server/wire.go`

**Step 1: 在 service ProviderSet 添加**

```go
NewPlatformProxyRules,
NewProxyAssignmentService,
```

**Step 2: 运行 Wire 代码生成**

Run: `cd backend && go generate ./cmd/server`

**Step 3: 验证编译**

Run: `cd backend && go build -o /dev/null ./cmd/server`

**Step 4: Commit**

```bash
git add backend/internal/service/wire.go backend/cmd/server/wire.go backend/cmd/server/wire_gen.go
git commit -m "feat: wire inject PlatformProxyRules and ProxyAssignmentService"
```

---

## Phase 4: 前端增强

### Task 11: 代理管理页 VPN 状态增强

**Files:**
- Modify: `frontend/src/views/admin/ProxiesView.vue`

**Step 1: 增强健康状态列**

- VPN 状态指示灯（connected=绿, disconnected=灰, error=红）
- 出口 IP 显示（带复制按钮）
- 延迟 badge（<100ms 绿, <500ms 黄, >500ms 红, 失败 灰）
- 悬浮提示显示上次检查时间

**Step 2: 添加批量操作**

- 「全部健康检查」按钮 — 调用 POST /health-check-all
- 「导入 Astrill 配置」— 上传 JSON 清单，调用 batch create

**Step 3: Commit**

```bash
git commit -m "feat(frontend): enhance proxy list with VPN status indicators"
```

---

### Task 12: 账号编辑页代理自动分配

**Files:**
- Modify: account create/edit modal/form component

**Step 1: 增强 proxy_group 字段**

- 下拉选择现有分组
- 「自动分配」按钮 — 调用 POST /auto-assign-proxy
- 修改后即时测试 — 调用 POST /test-proxy
- 测试结果实时反馈（成功/失败 + 延迟 + 出口IP）

**Step 2: Commit**

```bash
git commit -m "feat(frontend): add proxy auto-assign and test in account editor"
```

---

## 实施顺序和依赖

```
Phase 1 (Tasks 1-4): 服务器基础设施 — 独立，可并行开发
Phase 2 (Task 5): Astrill 抓取器 — 依赖 Phase 1 的目录结构
Phase 3 (Tasks 6-10): 后端增强 — 核心业务逻辑
  Task 6 → Task 7 (规则引擎先于分配服务)
  Task 8 可并行于 Task 6-7
  Task 9 依赖 Task 7
  Task 10 依赖 Task 6-9
Phase 4 (Tasks 11-12): 前端增强 — 依赖 Phase 3 API
```

## 预期总提交数：12 个 (每 Task 1 个)
