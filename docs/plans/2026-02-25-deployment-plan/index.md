# Sub2API 商业化部署方案（成都+香港双节点）

> 日期：2026-02-25
> 基于：`docs/API中转服务商业化部署全方案.md` + 项目实际实现

## 资源现状

| 资源 | 详情 |
|------|------|
| **成都服务器** | 阿里云（国内接入点） |
| **香港服务器** | 核心中转 + 出口节点 |
| **项目实现** | Go 后端 + Vue 前端，Docker 多阶段构建，PostgreSQL + Redis |
| **代理系统** | 已实现：ProxyGroupService / ProxyHealthService / PlatformProxyRules / ProxyAssignmentService |
| **VPN 部署工具** | 已就绪：OpenVPN systemd 模板 + 3proxy + vpn-manager.sh |
| **Astrill** | 1 个 .ovpn 配置（TCP-USA-Chicago-10GB-Private），需扩展 |
| **计费** | Token 计费 + 三层费率已实现，支付集成未做 |

## 架构概览

详见 [01-architecture.md](./01-architecture.md)

## 分阶段实施

| 阶段 | 目标用户 | 月成本 | 详情 |
|------|---------|-------|------|
| Phase 0：MVP | <50 | ~¥200 | [02-phase0-mvp.md](./02-phase0-mvp.md) |
| Phase 1：增长 | 50-500 | ¥500-2000 | [03-phase1-growth.md](./03-phase1-growth.md) |
| Phase 2：规模化 | 500-3000 | ¥2000-8000 | [04-phase2-scale.md](./04-phase2-scale.md) |

## VPN 代理部署

详见 [05-vpn-proxy-deploy.md](./05-vpn-proxy-deploy.md)

## 成本与盈利分析

详见 [06-cost-analysis.md](./06-cost-analysis.md)

## 实施检查清单

详见 [07-checklist.md](./07-checklist.md)
