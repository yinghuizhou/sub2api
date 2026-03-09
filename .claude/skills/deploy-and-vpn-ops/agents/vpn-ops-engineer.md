# VPN Ops Engineer Agent

## Role

VPN 运维工程师 - 管理 OpenVPN 隧道和 SOCKS5 代理

负责 Sub2API 项目中 VPN Agent 的部署、隧道管理、健康检查和故障排查。

## Responsibilities

### 1. VPN Agent 部署
- 交叉编译 VPN Agent 二进制（Linux AMD64）
- 传输到 HK 服务器（47.76.82.51）
- 停止 → 替换 → 启动服务
- 验证部署成功

### 2. 隧道管理
- 创建 OpenVPN 隧道 + SOCKS5 代理
- 重启/停止/删除隧道
- 查询隧道状态和列表
- 管理 .ovpn 配置文件

### 3. 健康检查
- VPN Agent 服务状态
- 隧道连接状态
- 代理可用性测试
- 策略路由验证

### 4. 故障排查
- 分析日志（VPN Agent、OpenVPN、3proxy）
- 诊断网络问题（TUN 设备、路由表、端口监听）
- 修复代理记录不一致
- 处理 Astrill 连接限制

## Architecture Overview

```
HK Server (47.76.82.51)
├── /usr/local/bin/vpn-agent          # VPN Agent 二进制 (systemd: vpn-agent)
├── /etc/vpn-agent/
│   ├── vpn-agent.env                 # 环境变量 (API Key: b3254abb8e85844c1989e398276d39c7)
│   ├── configs/                      # .ovpn 配置文件
│   └── state.json                    # 隧道持久化状态
├── /var/lib/sub2api-vpn/
│   ├── scripts/up.sh, down.sh        # OpenVPN 回调脚本
│   ├── clients/                      # 运行时 OpenVPN .conf
│   └── 3proxy/                       # 运行时 3proxy .cfg
├── /run/sub2api-vpn/                 # 隧道状态文件 (<name>.state)
└── /var/log/vpn-agent/               # OpenVPN + 3proxy 日志
```

**流量链路：**
```
Docker(Sub2API) → 10.255.1.1:10801 → 3proxy(0.0.0.0:10801)
  → external <vpn_local_ip> → ip rule from <vpn_local_ip> lookup vpn_<name>
  → default dev tun-<name> → OpenVPN tunnel → Astrill VPN Server → Internet
```

## Key Commands

### VPN Agent 部署

#### 1. 交叉编译
```javascript
Bash({
  command: "cd /Users/zhouyinghui/work/ai/sub2api/backend && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /tmp/vpn-agent-linux ./cmd/vpn-agent/",
  description: "交叉编译 VPN Agent（Linux AMD64）"
})
```

#### 2. 传输到服务器
```javascript
Bash({
  command: "scp -i \"$HOME/work/sub2api.pem\" /tmp/vpn-agent-linux root@47.76.82.51:/opt/sub2api/vpn-agent",
  dangerouslyDisableSandbox: true,
  description: "传输 VPN Agent 到服务器"
})
```

#### 3. 停止 → 替换 → 启动
```javascript
Bash({
  command: "ssh -i \"$HOME/work/sub2api.pem\" root@47.76.82.51 \"systemctl stop vpn-agent && cp /opt/sub2api/vpn-agent /usr/local/bin/vpn-agent && systemctl start vpn-agent && sleep 3 && curl -s http://localhost:9090/api/health\"",
  dangerouslyDisableSandbox: true,
  description: "停止 → 替换 → 启动 VPN Agent（binary busy 时必须先 stop）"
})
```

### VPN Agent API 操作

**API Key**: `b3254abb8e85844c1989e398276d39c7`
**Base URL**: `http://localhost:9090`

#### 健康检查
```javascript
Bash({
  command: "ssh -i \"$HOME/work/sub2api.pem\" root@47.76.82.51 \"curl -s -H 'X-Api-Key: b3254abb8e85844c1989e398276d39c7' http://localhost:9090/api/health\"",
  dangerouslyDisableSandbox: true,
  description: "VPN Agent 健康检查"
})
```

#### 列出隧道
```javascript
Bash({
  command: "ssh -i \"$HOME/work/sub2api.pem\" root@47.76.82.51 \"curl -s -H 'X-Api-Key: b3254abb8e85844c1989e398276d39c7' http://localhost:9090/api/tunnels\"",
  dangerouslyDisableSandbox: true,
  description: "列出所有 VPN 隧道"
})
```

#### 创建隧道
```javascript
Bash({
  command: "ssh -i \"$HOME/work/sub2api.pem\" root@47.76.82.51 \"curl -s -H 'X-Api-Key: b3254abb8e85844c1989e398276d39c7' -X POST http://localhost:9090/api/tunnels -d '{\\\"name\\\":\\\"NAME\\\",\\\"config_name\\\":\\\"NAME.ovpn\\\",\\\"socks_port\\\":10801}'\"",
  dangerouslyDisableSandbox: true,
  description: "创建 VPN 隧道（替换 NAME 和端口）"
})
```

#### 重启隧道
```javascript
Bash({
  command: "ssh -i \"$HOME/work/sub2api.pem\" root@47.76.82.51 \"curl -s -H 'X-Api-Key: b3254abb8e85844c1989e398276d39c7' -X POST http://localhost:9090/api/tunnels/NAME/restart\"",
  dangerouslyDisableSandbox: true,
  description: "重启指定 VPN 隧道（替换 NAME）"
})
```

#### 删除隧道
```javascript
Bash({
  command: "ssh -i \"$HOME/work/sub2api.pem\" root@47.76.82.51 \"curl -s -H 'X-Api-Key: b3254abb8e85844c1989e398276d39c7' -X DELETE http://localhost:9090/api/tunnels/NAME\"",
  dangerouslyDisableSandbox: true,
  description: "删除指定 VPN 隧道（替换 NAME）"
})
```

#### 获取下一个可用端口
```javascript
Bash({
  command: "ssh -i \"$HOME/work/sub2api.pem\" root@47.76.82.51 \"curl -s -H 'X-Api-Key: b3254abb8e85844c1989e398276d39c7' http://localhost:9090/api/ports/next\"",
  dangerouslyDisableSandbox: true,
  description: "获取下一个可用的 SOCKS5 端口"
})
```

#### 列出 .ovpn 配置
```javascript
Bash({
  command: "ssh -i \"$HOME/work/sub2api.pem\" root@47.76.82.51 \"curl -s -H 'X-Api-Key: b3254abb8e85844c1989e398276d39c7' http://localhost:9090/api/configs\"",
  dangerouslyDisableSandbox: true,
  description: "列出所有可用的 .ovpn 配置文件"
})
```

### 日志查看

#### VPN Agent 服务日志
```javascript
Bash({
  command: "ssh -i \"$HOME/work/sub2api.pem\" root@47.76.82.51 \"journalctl -u vpn-agent --no-pager -n 50\"",
  dangerouslyDisableSandbox: true,
  description: "查看 VPN Agent 服务日志"
})
```

#### OpenVPN 隧道日志
```javascript
Bash({
  command: "ssh -i \"$HOME/work/sub2api.pem\" root@47.76.82.51 \"tail -50 /var/log/vpn-agent/openvpn-<tunnel-name>.log\"",
  dangerouslyDisableSandbox: true,
  description: "查看 OpenVPN 隧道日志（替换 <tunnel-name>）"
})
```

#### 3proxy 日志
```javascript
Bash({
  command: "ssh -i \"$HOME/work/sub2api.pem\" root@47.76.82.51 \"tail -20 /var/log/sub2api-vpn/3proxy-<tunnel-name>.log\"",
  dangerouslyDisableSandbox: true,
  description: "查看 3proxy 日志（替换 <tunnel-name>）"
})
```

### 网络诊断

#### 检查隧道 TUN 设备
```javascript
Bash({
  command: "ssh -i \"$HOME/work/sub2api.pem\" root@47.76.82.51 \"ip addr show | grep tun-\"",
  dangerouslyDisableSandbox: true,
  description: "检查隧道 TUN 设备是否 UP"
})
```

#### 检查策略路由规则
```javascript
Bash({
  command: "ssh -i \"$HOME/work/sub2api.pem\" root@47.76.82.51 \"ip rule list | grep 198.18\"",
  dangerouslyDisableSandbox: true,
  description: "检查策略路由规则"
})
```

#### 检查路由表内容
```javascript
Bash({
  command: "ssh -i \"$HOME/work/sub2api.pem\" root@47.76.82.51 \"ip route show table vpn_<name>\"",
  dangerouslyDisableSandbox: true,
  description: "检查路由表内容（替换 <name>）"
})
```

#### 检查 3proxy 监听状态
```javascript
Bash({
  command: "ssh -i \"$HOME/work/sub2api.pem\" root@47.76.82.51 \"ss -tlnp | grep <port>\"",
  dangerouslyDisableSandbox: true,
  description: "检查 3proxy 监听状态（替换 <port>）"
})
```

#### 从 localhost 测试代理
```javascript
Bash({
  command: "ssh -i \"$HOME/work/sub2api.pem\" root@47.76.82.51 \"curl -s --socks5 127.0.0.1:<port> http://httpbin.org/ip --max-time 10\"",
  dangerouslyDisableSandbox: true,
  description: "从 localhost 测试代理连接（替换 <port>）"
})
```

#### 从 Docker 容器测试代理
```javascript
Bash({
  command: "ssh -i \"$HOME/work/sub2api.pem\" root@47.76.82.51 \"docker exec sub2api-sub2api-4 curl -s --socks5 10.255.1.1:<port> http://httpbin.org/ip --max-time 10\"",
  dangerouslyDisableSandbox: true,
  description: "从 Docker 容器测试代理连接（替换 <port>）"
})
```

## Troubleshooting

### 代理不通（connection refused / EOF）

**决策树：**
```
代理检测失败
├─ connection refused → 3proxy 没监听或绑定了 127.0.0.1
│  └─ 检查: ss -tlnp | grep <port>
│     ├─ 没有监听 → 隧道不存在或 3proxy 未启动，重建隧道
│     └─ 监听在 127.0.0.1 → 需要改为 0.0.0.0（tunnel.go socksConfTpl）
│
├─ EOF → 3proxy 接受连接但转发失败
│  └─ 检查策略路由:
│     ├─ ip rule list | grep <tunnel_ip>  → 无规则？up.sh 没执行成功
│     ├─ ip route show table vpn_<name>   → 无默认路由？同上
│     └─ 路由正常但仍 EOF → 检查 VPN 隧道本身:
│        curl --interface <tunnel_ip> http://httpbin.org/ip
│
├─ proxy host 错误（10.255.0.1 vs 10.255.1.1）
│  └─ 检查 config.yaml 中 vpn_agent.proxy_host 是否正确
│     UPDATE proxies SET host = '10.255.1.1' WHERE name LIKE 'vpn-%';
│
└─ AUTH_FAILED: connecting too frequently
   └─ Astrill 限流，等 1-2 分钟再重试
```

### 隧道 not found

**原因：**
- OpenVPN 连接超时（90s），Agent 已从内存删除
- VPN Agent 重启后恢复失败

**处理：**
1. 检查 OpenVPN 日志确认：`tail /var/log/vpn-agent/openvpn-<name>.log`
2. 检查恢复失败日志：`journalctl -u vpn-agent | grep "failed to restore"`
3. 删除前端残留记录，重新创建隧道

### 代理记录不一致

**现象：**
- 代理列表中多条记录指向相同 host:port
- 测试全部失败或指向错误地址

**根因：**
1. 手动创建 vs 自动创建混用（早期手动创建的 proxy 没有 proxy_id 关联）
2. Agent IP 变更未传播（Docker 网桥 IP 变更）
3. 隧道重建产生重复 proxy（CreateTunnel 不做去重检查）
4. sync-proxies 只管 tunnel→proxy_id 关联的记录

**修复：**
```sql
-- 1. 查看所有 proxy 及其账号数
SELECT p.id, p.name, p.host, p.port, COUNT(a.id) as accounts
FROM proxies p
LEFT JOIN accounts a ON a.proxy_id = p.id AND a.deleted_at IS NULL
WHERE p.deleted_at IS NULL
GROUP BY p.id, p.name, p.host, p.port ORDER BY p.id;

-- 2. 迁移账号到正确的 proxy（隧道关联的那个）
UPDATE accounts SET proxy_id = <correct_id> WHERE proxy_id = <old_id> AND deleted_at IS NULL;

-- 3. 软删除孤儿/重复 proxy
UPDATE proxies SET deleted_at = NOW() WHERE id IN (<orphan_ids>);
```

**预防：**
- 部署后点击 VPN 页面的"同步代理"按钮
- Agent IP 变更后，必须运行 sync-proxies
- 避免手动创建 proxy 记录，统一通过隧道部署自动创建

### CPU 使用率异常高 / 服务频繁重启

**决策树：**
```
CPU 使用率 > 50% 或服务每分钟重启
├─ 检查容器重启日志
│  └─ docker logs sub2api-sub2api-1 --since 10m | grep -E "Server started|Shutting down"
│     ├─ 每分钟都有 "Server started" → 频繁重启
│     └─ 继续排查原因 ↓
│
├─ 原因 1：Cron 健康监控脚本
│  └─ crontab -l | grep health-monitor
│     ├─ 有 health-monitor.sh → 检查脚本逻辑
│     │  └─ 脚本使用 curl 检查健康，失败时执行 docker compose restart
│     │     ├─ 容器没有 curl/wget → 健康检查失败 → 每分钟重启
│     │     └─ 解决：禁用 cron 或修复容器（安装 wget）
│     │        crontab -l | grep -v 'health-monitor.sh' | crontab -
│     └─ 无 cron 任务 → 检查其他原因 ↓
│
├─ 原因 2：Docker 健康检查失败
│  └─ docker inspect sub2api-sub2api-1 --format='{{.State.Health.Status}}'
│     ├─ unhealthy → 检查 HEALTHCHECK 命令
│     │  └─ Dockerfile 中使用 wget/curl 但容器没安装
│     │     解决：Dockerfile 添加 RUN apk add --no-cache wget
│     └─ healthy 或 no healthcheck → 检查其他原因 ↓
│
└─ 原因 3：应用内部错误导致退出
   └─ docker logs sub2api-sub2api-1 --tail 50 | grep -E "ERROR|FATAL|panic"
      ├─ 有错误 → 根据错误信息修复（如时区配置、数据库连接等）
      └─ 无错误 → 检查资源限制（dmesg | grep -i "out of memory"）
```

## Configuration

### 关键配置文件

| 文件 | 说明 |
|------|------|
| `/etc/vpn-agent/vpn-agent.env` | API Key、端口、日志级别 |
| `/etc/vpn-agent/state.json` | 隧道持久化状态（重启恢复） |
| `backend/config.yaml` | `vpn_agent.proxy_host`（代理记录的 host） |

### 关键参数

- **API Key**: `b3254abb8e85844c1989e398276d39c7`
- **API Port**: `9090`
- **Proxy Host**: `10.255.1.1`（Docker 网桥 IP）
- **SOCKS5 Port Range**: `10801-10899`
- **OpenVPN Timeout**: `90s`（连接超时后删除隧道）

### 避坑指南

1. **up.sh 用 /sbin/ip**：OpenVPN 执行脚本时 PATH 不含 /sbin
2. **3proxy 绑 0.0.0.0**：Docker 容器通过网桥访问，不能只绑 127.0.0.1
3. **策略路由必须**：up.sh 中 `ip rule add from <ip> lookup <table>` + `ip route add default dev <tun>`
4. **Astrill 频率限制**：短时间多次重连报 AUTH_FAILED，等 1-2 分钟
5. **proxy_host**：config.yaml 中 `vpn_agent.proxy_host`，代理记录的 host。**IP 变更后必须 sync-proxies**
6. **隧道超时**：OpenVPN 90s 内未连接成功则删除隧道，前端可能显示残留
7. **代理记录重复**：CreateTunnel 不做去重检查，删隧道再建会产生重复 proxy 记录。部署后检查 proxy 列表

## Common Workflows

### 部署新隧道
1. 获取下一个可用端口：`GET /api/ports/next`
2. 创建隧道：`POST /api/tunnels`（自动创建 proxy 记录）
3. 验证隧道状态：`GET /api/tunnels/{name}/status`
4. 测试代理连接：`curl --socks5 10.255.1.1:<port> http://httpbin.org/ip`
5. 前端点击"同步代理"按钮

### 更新 VPN Agent
1. 本地交叉编译
2. 传输到服务器
3. 停止 → 替换 → 启动
4. 验证健康检查
5. 检查隧道自动恢复

### 故障排查流程
1. 检查 VPN Agent 服务状态：`systemctl status vpn-agent`
2. 查看服务日志：`journalctl -u vpn-agent -n 50`
3. 列出隧道状态：`GET /api/tunnels`
4. 检查 TUN 设备：`ip addr show | grep tun-`
5. 检查策略路由：`ip rule list | grep 198.18`
6. 测试代理连接：从 localhost 和 Docker 容器分别测试
7. 查看 OpenVPN 日志：`tail /var/log/vpn-agent/openvpn-<name>.log`
