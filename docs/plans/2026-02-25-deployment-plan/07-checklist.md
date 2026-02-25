# 07 - 实施检查清单

## Phase 0 检查清单（立即可执行）

### 香港服务器 - 基础部署

- [ ] 系统优化：TCP BBR + Fast Open + 禁用空闲慢启动
- [ ] 安装 Docker + Docker Compose
- [ ] 构建 Sub2API Docker 镜像（或从开发机传镜像）
- [ ] 配置 `.env`（数据库、Redis、JWT 密钥等）
- [ ] `docker compose -f docker-compose.local.yml up -d`
- [ ] 验证：`curl http://localhost:8080/health` 返回 ok
- [ ] 访问管理后台，完成初始化设置

### 香港服务器 - VPN 代理层

- [ ] 上传 `deploy/vpn/` 到服务器
- [ ] 执行 `sudo bash install.sh`
- [ ] 上传 Astrill .ovpn 配置文件
- [ ] 部署第一个 VPN 实例：`sudo vpn-manager deploy <ovpn> us-chicago 10801`
- [ ] 验证 VPN 健康：`sudo vpn-manager health`
- [ ] 验证 SOCKS5 代理：`curl --socks5 127.0.0.1:10801 https://api.ipify.org`
- [ ] 验证到 API 的连通性：`curl --socks5 127.0.0.1:10801 https://api.anthropic.com/...`
- [ ] 配置防火墙：SOCKS5 端口仅允许 127.0.0.1

### 香港服务器 - Sub2API 配置

- [ ] 通过 Admin API 注册 SOCKS5 代理
- [ ] 创建上游 AI 账号（Claude/OpenAI/Gemini）
- [ ] 绑定账号到代理组
- [ ] 创建测试用户 + API Key
- [ ] 端到端测试：使用 API Key 发送请求，验证代理和路由正确

### 成都服务器 - 接入层

- [ ] 安装 Nginx
- [ ] 配置 SSL（Let's Encrypt / 自签证书）
- [ ] 配置反代到香港（proxy_buffering off！）
- [ ] 系统优化：TCP BBR
- [ ] 验证：从成都发请求到 AI API，端到端正常
- [ ] 测试 SSE 流式输出是否实时（无缓冲卡顿）

### 域名与网络

- [ ] 注册域名（如未有）
- [ ] DNS 解析 `api.example.com` → 成都服务器 IP
- [ ] DNS 解析 `admin.example.com` → 香港服务器 IP
- [ ] 考虑 ICP 备案申请（为 Phase 2 GA 准备）

---

## Phase 1 检查清单（用户 > 50 后）

### 代理池扩展

- [ ] 从 Astrill 获取更多 .ovpn 配置（至少 5 个不同地区）
- [ ] 批量部署 VPN 实例
- [ ] 在 Sub2API 注册所有代理
- [ ] 验证代理自动分配正常工作

### 稳定性

- [ ] 配置运维告警规则（代理健康率、错误率、延迟）
- [ ] 配置邮件通知
- [ ] 设置数据库自动备份（pg_dump + cron）
- [ ] 部署 Cloudflare Workers 备用出口

### 商业化

- [ ] 实现支付集成（微信/支付宝）
- [ ] 配置充值套餐和费率
- [ ] 上线用户注册 + 充值流程
- [ ] 配置新用户免费试用额度

---

## Phase 2 检查清单（用户 > 500 后）

### 扩展

- [ ] Sub2API 多实例 + Nginx 负载均衡
- [ ] PostgreSQL 升级（RDS 或独立服务器）
- [ ] Redis Sentinel（如需高可用）
- [ ] ICP 备案完成 → 切换到阿里云 GA
- [ ] VPN 代理池扩展到 15+

### 商业化完善

- [ ] 邀请返佣系统
- [ ] 代理商分销体系
- [ ] 阶梯充值优惠
- [ ] 用户自助服务（使用统计、账单查询）

---

## 关键决策点

| 决策 | 建议 | 何时决定 |
|------|------|---------|
| 是否 ICP 备案 | 是（为 GA 和合规） | Phase 1 中期启动 |
| 数据库自建 vs 托管 | Phase 1 自建，Phase 2 托管 | 用户 > 300 |
| Astrill vs 自建 VPN | Astrill 起步，规模化后评估 WireGuard | 用户 > 1000 |
| 单机 vs 多机 | 单机到极限后多机 | CPU > 60% 持续 |
| 支付通道 | 虎皮椒/YunGouOS 起步，后切官方 | Phase 1 初期 |

## 立即可以做的 3 件事

1. **在香港服务器上部署 Sub2API Docker**（30 分钟）
2. **安装 VPN 基础设施并部署第一个 Astrill 实例**（1 小时）
3. **配置成都 Nginx 反代并端到端测试**（30 分钟）
