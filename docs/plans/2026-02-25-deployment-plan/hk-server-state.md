# HK Server State Snapshot

> Last updated: 2026-02-25
> Server: 47.76.82.51 (Alibaba Cloud Hong Kong)

## OS & Base

- **OS**: Alibaba Cloud Linux 3
- **Kernel**: alinux3 (based on CentOS/RHEL stream)

## Docker Services

All containers healthy, managed by Docker Compose.

| Service | Port | Status |
|---------|------|--------|
| sub2api | 8080 | healthy |
| postgres | 5432 (internal) | healthy |
| redis | 6379 (internal) | healthy |

## Nginx

- Listening: ports 80, 443
- SSL domain: `llm.xn--ai-lz4c442h.cc`
- Proxies `/` to `127.0.0.1:8080`

## VPN Configuration

| Property | Value |
|----------|-------|
| Provider | Astrill (OpenVPN TCP) |
| Instance name | us-chicago |
| TUN device | tun-us-chicago |
| TUN IP | 198.18.70.86 |
| MTU | 1000 |
| Exit IP | 204.188.253.109 (US Chicago) |

### Key config tweaks
- `tun-mtu 1000` in OpenVPN config
- MSS clamping to 900 via iptables mangle rules
- `net.ipv4.ip_forward = 1` in sysctl

## SOCKS5 Proxy

| Property | Value |
|----------|-------|
| Software | microsocks (NOT 3proxy) |
| Listen | 0.0.0.0:10801 |
| Bind IP | 198.18.70.86 (TUN) |
| systemd unit | microsocks@us-chicago.service |
| Env file | /run/sub2api-vpn/us-chicago.env |

### Docker access
- Docker containers reach proxy at `10.255.1.1:10801` (macvlan bridge)
- Configured in Sub2API as proxy host `10.255.1.1`

## Sub2API Proxy Registration

| Field | Value |
|-------|-------|
| ID | 1 |
| Name | astrill-us-chicago |
| Protocol | socks5 |
| Host | 10.255.1.1 |
| Port | 10801 |
| Region | us-central |
| Accounts bound | 13 |

## Admin Access

- Admin API key: `admin-4fb8428b8402d27b3e4cec45a6877f369fb1ab2f0a621f14beb04f3fc44299f3`
- API base: `https://llm.xn--ai-lz4c442h.cc/api/admin`

## Key Files on Server

| Path | Purpose |
|------|---------|
| `/etc/systemd/system/microsocks@.service` | SOCKS5 proxy systemd template |
| `/etc/openvpn/clients/us-chicago.conf` | OpenVPN client config |
| `/etc/openvpn/scripts/post-up.sh` | Dynamic TUN IP env generator |
| `/etc/openvpn/scripts/mss-clamp.sh` | MSS clamping iptables rules |
| `/etc/sysctl.d/99-sub2api-vpn.conf` | Kernel params (ip_forward) |
| `/run/sub2api-vpn/us-chicago.env` | Runtime TUN_IP + SOCKS_PORT |

## Verification Commands

```bash
# Check exit IP (should return US IP)
curl -s --socks5 127.0.0.1:10801 https://api.ipify.org

# Check Claude API (should return 401, NOT 403)
curl -s --socks5 127.0.0.1:10801 https://api.anthropic.com/v1/messages

# Check from Docker network
curl -s --socks5 10.255.1.1:10801 https://api.ipify.org
```
