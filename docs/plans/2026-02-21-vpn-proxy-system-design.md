# VPN 代理管理系统设计文档

> 日期：2026-02-21
> 状态：已批准

## 目录

- [1. 概述](#1-概述)
- [2. 架构设计](#2-架构设计)
- [3. 模块详细设计](#3-模块详细设计)
- [4. 平台适配规则](#4-平台适配规则)
- [5. 数据模型变更](#5-数据模型变更)
- [6. 实施计划](#6-实施计划)

## 1. 概述

### 1.1 背景

Sub2API 管理 50+ AI 账号（Claude/Gemini/OpenAI），当前多账号共享同一出口 IP，导致：
- 同 IP 多账号被平台检测封号/限速
- 账号注册地与访问 IP 地区不匹配触发风控

### 1.2 目标

- 每个 AI 账号绑定稳定的代理出口 IP（按平台分组）
- 从 Astrill Web 后台批量抓取 OpenVPN 配置
- 不同 AI 平台适配不同的代理策略
- 自动健康检查和故障转移
- 支持专用 IP 作为高级付费服务

### 1.3 约束

- 单台阿里云服务器（成都 47.108.158.227）
- 1 个 Astrill 账号 + 1 个芝加哥专用 IP
- 运维自动化程度要高（用户为运维小白）

## 2. 架构设计

### 2.1 总体架构

```
┌─────────────────────────────────────────────────────┐
│  阿里云服务器 (47.108.158.227)                        │
│                                                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐           │
│  │ OpenVPN  │  │ OpenVPN  │  │ OpenVPN  │  ...      │
│  │ Client 1 │  │ Client 2 │  │ Client 3 │           │
│  │ tun-ue1  │  │ tun-uw1  │  │ tun-ew1  │           │
│  └─────┬────┘  └─────┬────┘  └─────┬────┘           │
│        │             │             │                 │
│  ┌─────┴────┐  ┌─────┴────┐  ┌─────┴────┐           │
│  │ 3proxy   │  │ 3proxy   │  │ 3proxy   │  ...      │
│  │ SOCKS5   │  │ SOCKS5   │  │ SOCKS5   │           │
│  │ :10801   │  │ :10802   │  │ :10803   │           │
│  └─────┬────┘  └─────┴────┘  └─────┬────┘           │
│        │             │             │                 │
│  ┌─────┴─────────────┴─────────────┴──────────────┐  │
│  │              Sub2API Gateway                    │  │
│  │                                                 │  │
│  │  ProxyGroupService → 按规则选择 SOCKS5 代理      │  │
│  │  ProxyHealthService → 定时检测代理健康           │  │
│  │  PlatformRuleEngine → 平台适配自动分配          │  │
│  └─────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────┘
```

### 2.2 数据流

```
新账号创建 → PlatformRuleEngine 自动分配 proxy_group
         → 管理员可手动覆盖 → 覆盖后触发连通性测试

请求转发 → SelectAccount → SelectProxyForAccount
        → 获取 socks5://127.0.0.1:108XX
        → HTTP Upstream 通过代理发送请求

健康检查 → 每 60s 检测所有活跃代理
        → 更新 health_status/latency_ms/vpn_exit_ip
        → 不健康 → 自动重连 VPN → 3 次失败标记 error
```

## 3. 模块详细设计

### 3.1 Astrill 配置抓取器

**技术栈**：Python 3 + Playwright

**功能**：
1. 自动登录 Astrill Web 后台
2. 枚举所有可用 OpenVPN 服务器节点
3. 批量下载 .ovpn 配置文件
4. 输出结构化 JSON 清单
5. 可选自动调用 Sub2API Admin API 注册代理

**输出格式**：
```json
[
  {
    "server_name": "US - Chicago #1",
    "region": "us-central",
    "country": "US",
    "city": "Chicago",
    "ovpn_file": "/etc/openvpn/configs/us-chicago-1.ovpn",
    "is_dedicated": true
  }
]
```

**部署位置**：`/opt/sub2api/tools/astrill-scraper/`
**运行方式**：手动触发或 cron 定期更新

### 3.2 OpenVPN 多实例管理

**systemd 模板**：`/etc/systemd/system/openvpn-client@.service`

每个实例：
- 独立 .conf 文件：`/etc/openvpn/clients/{name}.conf`
- 独立 tun 设备：`tun-{name}`
- `route-nopull` 阻止修改默认路由
- up/down 脚本管理独立路由表

**路由隔离方案**：
```bash
# up.sh（VPN 连接成功后执行）
TABLE_ID=$((100 + INSTANCE_NUM))
ip route add default via $VPN_GATEWAY dev $TUN_DEV table $TABLE_ID
ip rule add from $VPN_LOCAL_IP table $TABLE_ID

# down.sh（VPN 断开后执行）
ip rule del from $VPN_LOCAL_IP table $TABLE_ID
ip route flush table $TABLE_ID
```

**管理脚本**：`/opt/sub2api/tools/vpn-manager.sh`
- `list` — 列出所有实例及状态
- `start/stop/restart <name>` — 控制单个实例
- `deploy <json>` — 从抓取器输出批量部署
- `status <name>` — 详情（出口IP、延迟）

### 3.3 SOCKS5 代理层

**选型**：3proxy（单二进制，轻量，~2MB）

**配置模板**：`/etc/3proxy/instances/{name}.cfg`
```
nscache 65536
nserver 8.8.8.8
auth none
allow * 127.0.0.1
socks -p{PORT} -i127.0.0.1 -e{TUN_IP}
```

**端口分配**：基础端口 10800，按实例序号递增
**systemd 模板**：`3proxy@.service`，与 OpenVPN 实例联动

### 3.4 代理健康检查服务

**后端新增**：`ProxyHealthService`（Wire 注入，自动启动）

```go
type ProxyHealthService struct {
    proxyRepo    ProxyRepository
    ticker       *time.Ticker  // 60s interval
}

// CheckOne: 单个代理健康检查
// 1. 通过 SOCKS5 代理请求 https://httpbin.org/ip → 获取出口 IP
// 2. 测量 TCP 连接延迟
// 3. 比对出口 IP 是否与预期一致
// 4. 更新 DB: health_status, latency_ms, vpn_exit_ip, last_health_at
// 5. 连续 3 次失败 → unhealthy → 触发 systemctl restart

// 告警阈值
HealthCheckInterval = 60s
MaxConsecutiveFailures = 3
LatencyWarningMs = 500
LatencyCriticalMs = 2000
```

### 3.5 平台适配规则引擎

**存储**：数据库 `platform_proxy_rules` 表（或配置文件，初期用配置简单）

```go
type PlatformProxyRule struct {
    Platform           string   // anthropic, gemini, openai
    PreferredRegions   []string // ["us-east", "us-central"]
    MaxAccountsPerIP   int      // 每 IP 最大账号数
    StickyIP           bool     // 是否绑定固定 IP
    RequireResidential bool     // 是否需要住宅 IP
}
```

**自动分配流程**：
1. 新账号创建 → 查找平台规则
2. 列出该平台偏好地区的代理分组
3. 按 MaxAccountsPerIP 负载找到最空的分组
4. 设置 account.proxy_group
5. 管理员可手动覆盖，覆盖后触发连通性测试

### 3.6 管理后台 UI 扩展

**新增/增强页面**：

1. **代理管理页**（已存在，增强）
   - 增加：VPN 连接状态（connected/disconnected/error）
   - 增加：实时出口 IP 显示
   - 增加：延迟 badge（绿<100ms/黄<500ms/红>500ms）
   - 增加：批量操作（启停、重连、健康检查）

2. **代理分组概览**（新增 Tab 或页面）
   - 分组列表 + 成员代理 + 绑定账号数/上限
   - 负载百分比可视化

3. **账号编辑**（增强）
   - proxy_group 字段：下拉选择 + 「自动分配」按钮
   - 修改后即时连通性测试 + 结果反馈

## 4. 平台适配规则（初始配置）

| 平台 | 偏好地区 | 每IP上限 | 固定IP | 说明 |
|------|---------|---------|--------|------|
| Claude (Anthropic) | us-east, us-central | 5 | 是 | 美国 IP 最稳定 |
| Gemini (Google) | us-east, us-west, eu-west | 10 | 是 | 地区限制较松 |
| OpenAI | us-east, us-west | 8 | 是 | 标准策略 |

## 5. 数据模型变更

### 5.1 现有 Proxy 表（无需变更）

已有字段完全满足需求：
- `ovpn_config`, `ovpn_username`, `ovpn_password` — VPN 配置
- `vpn_status`, `vpn_exit_ip` — VPN 状态
- `health_status`, `latency_ms`, `last_health_at`, `health_check_failures`
- `region`, `group_name`, `is_dedicated`

### 5.2 新增：平台代理规则（配置文件）

初期使用 Go 配置结构体，不新增数据库表。
后期可迁移到 DB 实现动态配置。

### 5.3 Account 表（无需变更）

已有 `proxy_id` 和 `proxy_group` 字段。

## 6. 实施计划

### Phase 1：服务器基础设施（运维）
1. 在阿里云服务器安装 OpenVPN 客户端和 3proxy
2. 创建 systemd 模板（openvpn-client@, 3proxy@）
3. 编写 vpn-manager.sh 管理脚本
4. 部署 up.sh/down.sh 路由隔离脚本

### Phase 2：Astrill 抓取器
5. 编写 Python Playwright 抓取脚本
6. 测试登录和配置下载
7. 批量部署 OpenVPN 配置到服务器

### Phase 3：后端增强
8. 实现 ProxyHealthService（健康检查 + 自动重连）
9. 实现 PlatformRuleEngine（自动分配逻辑）
10. 新增 API 端点：手动触发健康检查、自动分配
11. Wire 注入新服务

### Phase 4：前端增强
12. 代理管理页增强（VPN 状态、延迟、出口IP）
13. 账号编辑页增强（proxy_group 选择 + 即时测试）
14. 代理分组概览页面
