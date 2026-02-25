# 01 - 双节点三层架构

## 核心设计原则

1. **成都做接入，香港做一切** — 成都仅负责用户接入加速，所有业务逻辑在香港
2. **代理出口在香港本机** — VPN 隧道终止在香港服务器，SOCKS5 代理本地访问
3. **无状态应用，有状态存储** — Sub2API 进程无状态，PostgreSQL + Redis 在香港

## 架构拓扑

```
中国大陆用户
    │
    ▼
┌─────────────────────────────────┐
│  成都服务器（接入层）             │
│  ┌────────────────────────────┐ │
│  │ Nginx / Caddy              │ │
│  │  - SSL 终止                │ │
│  │  - proxy_pass → 香港:8080  │ │
│  │  - proxy_buffering off     │ │
│  │  - TCP BBR 拥塞控制        │ │
│  └────────────────────────────┘ │
│  延迟：用户→成都 5-20ms         │
└─────────────┬───────────────────┘
              │ 成都→香港 30-50ms（CN2/精品线路）
              ▼
┌─────────────────────────────────────────────────────┐
│  香港服务器（业务层 + 出口层）                         │
│                                                     │
│  ┌──────────────┐  ┌────────────┐  ┌──────────────┐ │
│  │ Sub2API      │  │ PostgreSQL │  │ Redis        │ │
│  │ (Go :8080)   │──│ (:5432)    │  │ (:6379)      │ │
│  │  - Gateway   │  └────────────┘  └──────────────┘ │
│  │  - Scheduler │                                   │
│  │  - Billing   │                                   │
│  │  - Admin UI  │                                   │
│  └──────┬───────┘                                   │
│         │                                           │
│  ┌──────▼──────────────────────────────────────┐    │
│  │  出口层（VPN 隧道 + SOCKS5 代理）            │    │
│  │                                             │    │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  │    │
│  │  │ tun-ue1  │  │ tun-uw1  │  │ tun-ew1  │  │    │
│  │  │ OpenVPN  │  │ OpenVPN  │  │ OpenVPN  │  │    │
│  │  │ ↓        │  │ ↓        │  │ ↓        │  │    │
│  │  │ 3proxy   │  │ 3proxy   │  │ 3proxy   │  │    │
│  │  │ :10801   │  │ :10802   │  │ :10803   │  │    │
│  │  │ US-East  │  │ US-West  │  │ EU-West  │  │    │
│  │  └──────────┘  └──────────┘  └──────────┘  │    │
│  └─────────────────────────────────────────────┘    │
│                                                     │
│  延迟：香港→美国 API  140-170ms                       │
│  端到端：175-240ms（用户→成都→香港→美国API）           │
└─────────────────────────────────────────────────────┘
```

## 为什么不把 Sub2API 部署在成都？

| 因素 | 部署在成都 | 部署在香港 |
|------|----------|----------|
| 到 AI API 延迟 | 成都→美国 200ms+ | 香港→美国 140-170ms |
| 代理管理 | 需跨服务器管理 VPN | VPN 本地 127.0.0.1 直连 |
| 数据出境 | 用户请求直接跨境 | 仅在香港出境，成都只做转发 |
| 合规风险 | 国内服务器直连海外 AI | 成都只是"加速节点"，模糊化 |
| SSE 延迟 | 多一跳 | 少一跳中转 |

**结论：业务逻辑 + 出口必须在香港，成都仅做 Nginx 反代。**

## 网络优化要点

### 成都 Nginx 核心配置

```nginx
upstream sub2api_hk {
    server <香港IP>:8080;
    keepalive 64;
}

server {
    listen 443 ssl http2;
    server_name api.example.com;

    # SSL 证书
    ssl_certificate     /etc/nginx/ssl/cert.pem;
    ssl_certificate_key /etc/nginx/ssl/key.pem;

    location / {
        proxy_pass http://sub2api_hk;
        proxy_http_version 1.1;
        proxy_set_header Connection '';

        # SSE 生命线 —— 必须
        proxy_buffering off;
        proxy_cache off;
        add_header X-Accel-Buffering no;

        # 超时配置（LLM 长输出）
        proxy_read_timeout 600s;
        proxy_send_timeout 60s;
        proxy_connect_timeout 10s;

        # 透传客户端信息
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 两台服务器都要做的系统优化

```bash
# TCP BBR 拥塞控制
echo "net.core.default_qdisc=fq" >> /etc/sysctl.d/99-bbr.conf
echo "net.ipv4.tcp_congestion_control=bbr" >> /etc/sysctl.d/99-bbr.conf

# TCP Fast Open
echo "net.ipv4.tcp_fastopen=3" >> /etc/sysctl.d/99-bbr.conf

# 禁用空闲后慢启动
echo "net.ipv4.tcp_slow_start_after_idle=0" >> /etc/sysctl.d/99-bbr.conf

sysctl -p /etc/sysctl.d/99-bbr.conf
```

## 域名与 SSL 方案

| 用途 | 域名 | 解析 | SSL |
|------|------|------|-----|
| API 接入 | `api.example.com` | → 成都服务器 IP | Let's Encrypt |
| 管理后台 | `admin.example.com` | → 香港服务器 IP（或成都转发） | Let's Encrypt |
| 备用直连 | `hk.example.com` | → 香港服务器 IP | Let's Encrypt |

建议 ICP 备案域名用于成都接入（备案后可接入阿里云 GA）。
未备案期间使用香港 BGP 精品线路直连。
