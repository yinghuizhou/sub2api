---
name: deploy-and-vpn-ops
description: Sub2API 零停机生产部署 + VPN Agent 运维。使用 Team Agents 协调构建、部署和验证，管理 VPN 隧道、排查代理故障。
trigger: 当用户提到"部署"、"deploy"、"上线"、"发布"、"更新生产"、"滚动部署"、"零停机"、"VPN"、"隧道"、"tunnel"、"代理不通"、"proxy"、"重启服务"时触发。
---

# Sub2API 部署 & VPN 运维 Skill

## 快速参考

| 项目 | 值 |
|------|------|
| HK 服务器 | `47.76.82.51` |
| SSH PEM | `$HOME/work/sub2api.pem` |
| Sub2API 容器 | `:8888`（Nginx 负载均衡），Docker Compose at `/opt/sub2api/` |
| VPN Agent | systemd `vpn-agent`，binary `/usr/local/bin/vpn-agent`，`:9090` |
| VPN Agent API Key | `b3254abb8e85844c1989e398276d39c7` |
| Docker→Host 网桥 | `10.255.1.1` |
| 代理端口范围 | `10801-10899` |

## 常用场景

### 场景 1：我要部署到生产环境

**推荐方式：使用现有脚本（最快）**
```bash
./scripts/deploy-production.sh
```

这个脚本会自动完成：
- ✓ 检查前置条件（Docker、SSH、服务器健康）
- ✓ 构建前端 + 后端二进制（AMD64 + 嵌入前端）
- ✓ 构建 Docker 镜像（使用 Dockerfile.prod）
- ✓ 上传镜像到服务器
- ✓ 执行滚动部署（逐个更新实例，始终保持至少 2 个实例运行）
- ✓ 验证部署结果

**使用 Team Agents 方式（更细粒度控制）**

当需要更细粒度控制或自定义流程时，使用 Team Agents：

1. 创建部署团队
2. 分配任务给各个 agent：
   - `build-engineer` - 构建前端、后端和 Docker 镜像
   - `deploy-engineer` - 上传镜像并执行滚动部署
   - `qa-engineer` - 验证部署结果
3. `deployment-orchestrator` 协调整个流程

详见：[agents/deployment-orchestrator.md](agents/deployment-orchestrator.md)

### 场景 2：VPN 隧道不通 / 代理连接失败

**快速诊断**
```bash
# 使用健康检查脚本
./.claude/skills/deploy-and-vpn-ops/commands/vpn-health-check.sh
```

**手动排查**

1. 检查 VPN Agent 服务状态
2. 列出所有隧道
3. 测试代理连接
4. 查看日志

详见：[agents/vpn-ops-engineer.md](agents/vpn-ops-engineer.md) 的 Troubleshooting 部分

### 场景 3：部署失败需要回滚

```bash
# 使用回滚脚本
./.claude/skills/deploy-and-vpn-ops/commands/rollback.sh
```

### 场景 4：更新 VPN Agent

```bash
# 1. 编译
cd backend && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /tmp/vpn-agent-linux ./cmd/vpn-agent/

# 2. 传输并重启（需要 dangerouslyDisableSandbox: true）
scp -i ~/work/sub2api.pem /tmp/vpn-agent-linux root@47.76.82.51:/opt/sub2api/vpn-agent
ssh -i ~/work/sub2api.pem root@47.76.82.51 \
  "systemctl stop vpn-agent && cp /opt/sub2api/vpn-agent /usr/local/bin/vpn-agent && systemctl start vpn-agent"
```

详见：[agents/vpn-ops-engineer.md](agents/vpn-ops-engineer.md)

## Agent 配置

本技能使用以下 agents 协调工作：

| Agent | 职责 | 配置文件 |
|-------|------|----------|
| deployment-orchestrator | 协调整个部署流程，管理版本和回滚 | [agents/deployment-orchestrator.md](agents/deployment-orchestrator.md) |
| build-engineer | 构建前端、后端和 Docker 镜像 | [agents/build-engineer.md](agents/build-engineer.md) |
| deploy-engineer | 上传镜像并执行滚动部署 | [agents/deploy-engineer.md](agents/deploy-engineer.md) |
| qa-engineer | 验证部署结果，确保服务正常 | [agents/qa-engineer.md](agents/qa-engineer.md) |
| vpn-ops-engineer | 管理 VPN 隧道和代理 | [agents/vpn-ops-engineer.md](agents/vpn-ops-engineer.md) |

## 便捷命令

所有便捷命令位于 `.claude/skills/deploy-and-vpn-ops/commands/`：

| 命令 | 说明 |
|------|------|
| `check-prerequisites.sh` | 检查部署前置条件 |
| `quick-deploy.sh` | 快速部署（调用 deploy-production.sh） |
| `vpn-health-check.sh` | VPN 隧道健康检查 |
| `rollback.sh` | 回滚到上一个版本 |

## 工作流程图

### 部署流程
```
用户请求部署
  ↓
deployment-orchestrator 检查前置条件
  ↓
build-engineer 构建前端 + 后端 + Docker 镜像
  ↓
deploy-engineer 上传镜像并执行滚动部署
  ↓
qa-engineer 验证部署结果
  ↓
部署成功 / 失败回滚
```

### 滚动部署流程
```
初始: 实例1(旧) + 实例2(旧) + 实例3(旧)
  ↓
步骤1: 停止实例1 → 启动实例1(新) → 等待20s → 健康检查
  ↓ (2个旧实例继续服务)
步骤2: 停止实例2 → 启动实例2(新) → 等待20s → 健康检查
  ↓ (1个旧实例 + 1个新实例继续服务)
步骤3: 停止实例3 → 启动实例3(新) → 等待20s → 健康检查
  ↓ (2个新实例继续服务)
完成: 实例1(新) + 实例2(新) + 实例3(新)
```

## 重要提醒

### 沙箱限制
**所有 Docker 和 SSH 命令都必须使用 `dangerouslyDisableSandbox: true`**

原因：
- Docker 命令需要访问 `/var/run/docker.sock`（沙箱禁止）
- SSH 命令需要访问 `~/.ssh/` 和 PEM 文件（沙箱禁止）

### 部署前检查清单
- [ ] Docker Desktop 已启动（`docker info`）
- [ ] SSH 连接正常（`ssh -i ~/work/sub2api.pem root@47.76.82.51 "echo ok"`）
- [ ] 服务器健康（`curl -sf http://localhost:8888/`）
- [ ] 至少 2 个实例在运行（零停机要求）

### 禁止操作
- **禁止在服务器上 docker build** → 1.8GB 内存，OOM 会杀其他容器
- **禁止在服务器上 git pull** → 服务器只接收构建好的镜像，不拉取代码
- HK 服务器没有 rsync，用 scp（镜像 ~50MB 可接受）

## 故障排查快速入口

### 部署相关
- **Docker 未运行** → `open /Applications/Docker.app`
- **服务器连接失败** → 检查 VPN、验证 PEM 文件权限
- **实例数量不足** → 先扩展到 3 个实例再部署
- **健康检查失败** → 检查容器日志、验证健康检查配置
- **频繁重启** → 检查 cron 任务、健康检查配置、应用错误

详见：[agents/qa-engineer.md](agents/qa-engineer.md)

### VPN 相关
- **代理不通（connection refused）** → 检查 3proxy 监听状态
- **代理不通（EOF）** → 检查策略路由和 VPN 隧道
- **隧道 not found** → 检查 OpenVPN 日志、重新创建隧道
- **AUTH_FAILED** → Astrill 限流，等 1-2 分钟

详见：[agents/vpn-ops-engineer.md](agents/vpn-ops-engineer.md) 的 Troubleshooting 部分

## 访问地址

部署成功后，服务可通过以下地址访问：
- https://llm.全民ai.cc
- https://llm.mindabc.ai

## 相关文档

- **部署指南**: `/Users/zhouyinghui/.claude/projects/-Users-zhouyinghui-work-ai-sub2api/memory/deployment.md`
- **VPN 运维手册**: `/Users/zhouyinghui/.claude/projects/-Users-zhouyinghui-work-ai-sub2api/memory/vpn-ops.md`
- **故障排查**: `/Users/zhouyinghui/.claude/projects/-Users-zhouyinghui-work-ai-sub2api/memory/troubleshooting.md`
- **服务器信息**: `/Users/zhouyinghui/.claude/projects/-Users-zhouyinghui-work-ai-sub2api/memory/servers.md`
