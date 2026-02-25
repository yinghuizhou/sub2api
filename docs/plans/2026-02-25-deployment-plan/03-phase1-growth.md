# 03 - Phase 1：增长期（50-500 用户）

> 目标：稳定服务 + 支付闭环 + 代理池扩展
> 预计月成本：¥500-2,000

## 与 Phase 0 的关键差异

| 维度 | Phase 0 | Phase 1 |
|------|---------|---------|
| 代理出口 | 1-2 个 VPN | 5-10 个 VPN + Cloudflare Workers 备用 |
| 数据库 | Docker 内 PG | 独立 PG（同机但独立进程或容器隔离） |
| 监控 | 基本日志 | 告警规则 + 邮件通知 |
| 支付 | 手动充值 | 微信/支付宝自动充值 |
| 备份 | 无 | 每日自动备份 |
| SSL | Let's Encrypt | 不变 |

## 架构变化

```
用户 → 成都 Nginx(:443) ──→ 香港 Sub2API(:8080)
                                    ↓
                        ┌───── PostgreSQL（独立容器，挂载外部卷）
                        ├───── Redis（Sentinel 可选）
                        ↓
                  ┌─────────────────────────────────┐
                  │  出口层（5-10 个 VPN 实例）        │
                  │  us-east-1/2  :10801-10802       │
                  │  us-central-1 :10803              │
                  │  us-west-1/2  :10804-10805       │
                  │  eu-west-1    :10806              │
                  └─────────────────────────────────┘
                  ┌─────────────────────────────────┐
                  │  Cloudflare Workers（备用出口）    │
                  │  worker.example.com              │
                  └─────────────────────────────────┘
```

## 关键实施项

### 1. 扩展 Astrill VPN 代理池

**获取更多 .ovpn 配置**：

需要从 Astrill 获取 Private IP（专属 IP）的 .ovpn 文件。按平台规则分布：

| 地区 | 数量 | 用途 | Astrill 节点 |
|------|------|------|-------------|
| US East | 2 | Claude 主力 | New York / Washington DC |
| US Central | 1 | Claude 备用 | Chicago（已有） |
| US West | 2 | OpenAI + Claude | Los Angeles / San Jose |
| EU West | 1 | Gemini 备用 | London / Amsterdam |

每个 Private IP $5/月，6 个 = $30/月。

**批量部署命令**：

```bash
# 创建 manifest.json
cat > /tmp/vpn-manifest.json << 'EOF'
[
  {"ovpn_file": "/root/ovpn/us-east-ny.ovpn",    "name": "us-east-1",    "port": 10801},
  {"ovpn_file": "/root/ovpn/us-east-dc.ovpn",    "name": "us-east-2",    "port": 10802},
  {"ovpn_file": "/root/ovpn/us-central-chi.ovpn", "name": "us-central-1", "port": 10803},
  {"ovpn_file": "/root/ovpn/us-west-la.ovpn",    "name": "us-west-1",    "port": 10804},
  {"ovpn_file": "/root/ovpn/us-west-sj.ovpn",    "name": "us-west-2",    "port": 10805},
  {"ovpn_file": "/root/ovpn/eu-west-ams.ovpn",   "name": "eu-west-1",    "port": 10806}
]
EOF

# 批量部署
sudo vpn-manager deploy-batch /tmp/vpn-manifest.json

# 验证全部健康
sudo vpn-manager health
```

### 2. Cloudflare Workers 备用出口

部署一个 Worker 作为备用出口，当 VPN 全部不健康时降级使用：

- 在 Sub2API 中注册为 HTTP 代理类型
- 优先级低于 VPN 出口
- 无需 VPN，直接 HTTP 转发

### 3. 支付系统集成（商业化 Batch 1）

这是 Phase 1 最关键的开发工作。参照 `commercialization-plan-batch1-payment.md`：

- `payment_orders` + `recharge_packages` Ent schema
- gopay SDK 集成（微信 V3 / 支付宝）
- 支付回调 handler
- 前端充值页面

### 4. 数据库备份

```bash
# crontab -e
# 每日凌晨 3 点备份
0 3 * * * docker exec sub2api-postgres pg_dump -U sub2api sub2api | \
  gzip > /backup/sub2api-$(date +\%Y\%m\%d).sql.gz

# 保留最近 30 天
0 4 * * * find /backup -name "sub2api-*.sql.gz" -mtime +30 -delete
```

### 5. 告警配置

通过 Sub2API 运维仪表板配置：

| 指标 | 阈值 | 动作 |
|------|------|------|
| 代理健康率 | < 80% | 邮件告警 |
| 错误率（5xx） | > 5% | 邮件告警 |
| P95 延迟 | > 30s | 邮件告警 |
| 账户可用率 | < 50% | 紧急告警 |
| 429 频率 | > 20% | 告警 + 自动降低调度权重 |

## Phase 1 成本估算

| 项目 | 月成本 |
|------|-------|
| 香港服务器（4核4G，升配） | ¥100-300 |
| 成都服务器 | ¥0（已有） |
| Astrill VPN（6 Private IP） | ¥210（$30/月） |
| Cloudflare Workers（Paid） | ¥35（$5/月） |
| 域名 + SSL | ¥10 |
| 短信/邮件通知 | ¥20 |
| **基础设施合计** | **¥375-575** |
| AI API 成本（50-500 活跃用户，按 ARPU ¥100） | ¥3,700-37,000 |
| **总成本** | **¥4,000-37,500** |

## Phase 1 收入预估（1.5x 倍率）

| 活跃用户 | 月收入 | API 成本(74%) | 基础设施 | 月净利 |
|---------|--------|-------------|---------|-------|
| 50 | ¥7,500 | ¥5,550 | ¥500 | ¥1,450 |
| 200 | ¥30,000 | ¥22,200 | ¥600 | ¥7,200 |
| 500 | ¥75,000 | ¥55,500 | ¥800 | ¥18,700 |
