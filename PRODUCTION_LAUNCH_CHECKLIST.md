# Sub2API 生产环境上线准备清单

## 🔴 P0 - 必须立即处理（上线前）

### 1. 安全加固

#### 1.1 修改默认密码
```bash
# PostgreSQL 密码
# 当前：sub2api
# 建议：使用 32 位随机密码
openssl rand -base64 32

# Redis 密码
# 当前：无密码
# 建议：启用 requirepass

# 管理员密码
# 当前：password（太弱！）
# 建议：至少 16 位，包含大小写字母、数字、特殊字符
```

**操作步骤**：
1. 生成新密码：`openssl rand -base64 32`
2. 更新 `deploy/.env` 文件
3. 重启服务
4. 验证连接

#### 1.2 JWT Secret
```bash
# 检查当前配置
grep JWT_SECRET deploy/.env

# 如果是默认值或太短，重新生成
openssl rand -hex 64
```

#### 1.3 防火墙配置
```bash
# 只开放必要端口
# 8080: API 服务（通过 Nginx 反代，不直接暴露）
# 443: HTTPS
# 22: SSH（建议改端口 + 密钥认证）

# 禁止直接访问数据库端口
# 5432: PostgreSQL（仅本地）
# 6379: Redis（仅本地）
```

#### 1.4 HTTPS 配置
```bash
# 确认 SSL 证书有效
curl -I https://llm.全民ai.cc

# 配置 HSTS
# 在 Nginx 配置中添加：
# add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
```

### 2. 数据库备份

#### 2.1 自动备份脚本
见 `scripts/backup-database.sh`

#### 2.2 备份策略
- **全量备份**：每天 2:00 AM
- **增量备份**：每小时
- **保留策略**：
  - 最近 7 天：每日备份
  - 最近 30 天：每周备份
  - 超过 30 天：每月备份

#### 2.3 异地备份
- 使用 rsync 或 rclone 同步到其他服务器
- 或使用云存储（阿里云 OSS、腾讯云 COS）

### 3. 监控和告警

#### 3.1 健康检查监控
见 `scripts/health-monitor.sh`

#### 3.2 关键指标
- API 响应时间 > 2s
- 错误率 > 5%
- 数据库连接数 > 80%
- Redis 内存使用 > 80%
- 磁盘空间 > 85%
- 账号余额 < 阈值

#### 3.3 告警渠道
- 邮件
- 企业微信/钉钉
- 短信（紧急情况）

## 🟡 P1 - 上线后 24 小时内完成

### 4. 性能优化

#### 4.1 数据库索引
```sql
-- 检查慢查询
SELECT * FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;

-- 添加必要索引
```

#### 4.2 Redis 缓存
- 热点数据缓存
- 限流计数器
- Session 存储

#### 4.3 连接池调优
```yaml
# config.yaml
postgres:
  max_open_conns: 100
  max_idle_conns: 10
  conn_max_lifetime: 1h

redis:
  pool_size: 50
  min_idle_conns: 10
```

### 5. 日志管理

#### 5.1 日志轮转
```bash
# /etc/logrotate.d/sub2api
/var/log/sub2api/*.log {
    daily
    rotate 30
    compress
    delaycompress
    notifempty
    create 0640 sub2api sub2api
    sharedscripts
    postrotate
        systemctl reload sub2api
    endscript
}
```

#### 5.2 错误日志告警
- 监控 ERROR 级别日志
- 异常堆栈收集
- 错误趋势分析

### 6. 限流和防护

#### 6.1 API 限流
```yaml
# 已有配置，确认合理性
rate_limit:
  enabled: true
  requests_per_minute: 60
  burst: 10
```

#### 6.2 IP 黑白名单
- 管理后台限制 IP 访问
- API 网关支持 IP 白名单

#### 6.3 DDoS 防护
- 使用 CDN（Cloudflare、阿里云 CDN）
- 配置 Nginx 限流
- 启用 fail2ban

## 🟢 P2 - 持续优化

### 7. 用户体验

#### 7.1 API 文档
- Swagger/OpenAPI 文档
- 使用示例
- SDK 支持

#### 7.2 错误提示
- 友好的错误信息
- 错误码规范
- 多语言支持

### 8. 运营准备

#### 8.1 定价策略
- 确认价格合理性
- 对比竞品
- 优惠活动

#### 8.2 支付集成
- 支付宝
- 微信支付
- 银行卡

#### 8.3 客服支持
- 工单系统
- 在线客服
- FAQ 文档

### 9. 合规和法律

#### 9.1 用户协议
- 服务条款
- 隐私政策
- 退款政策

#### 9.2 ICP 备案
- 成都服务器需要备案
- 备案期间使用香港服务器

#### 9.3 数据安全
- 用户数据加密
- 日志脱敏
- 定期安全审计

## 📋 上线检查清单

### 上线前（必须全部完成）

- [ ] 所有默认密码已修改
- [ ] HTTPS 证书有效
- [ ] 防火墙规则配置正确
- [ ] 数据库备份脚本已测试
- [ ] 监控告警已配置
- [ ] 所有功能测试通过
- [ ] 性能测试通过（并发 100+）
- [ ] 安全扫描无高危漏洞
- [ ] 文档已更新
- [ ] 回滚方案已准备

### 上线时

- [ ] 数据库备份
- [ ] 代码部署
- [ ] 服务重启
- [ ] 健康检查通过
- [ ] 功能冒烟测试
- [ ] 监控指标正常

### 上线后 1 小时

- [ ] 错误日志检查
- [ ] 性能指标检查
- [ ] 用户反馈收集
- [ ] 数据库性能检查

### 上线后 24 小时

- [ ] 完整功能测试
- [ ] 性能报告
- [ ] 用户使用情况分析
- [ ] 问题汇总和优化计划

## 🚨 应急预案

### 服务不可用
1. 检查服务状态：`systemctl status sub2api`
2. 查看错误日志：`tail -f /var/log/sub2api/error.log`
3. 检查数据库连接：`psql -h localhost -U sub2api`
4. 检查 Redis 连接：`redis-cli ping`
5. 如无法快速修复，执行回滚

### 数据库故障
1. 检查磁盘空间
2. 检查连接数
3. 查看慢查询
4. 必要时从备份恢复

### 性能下降
1. 检查 CPU、内存、磁盘 IO
2. 查看数据库慢查询
3. 检查 Redis 内存使用
4. 分析访问日志，识别异常流量

### 安全事件
1. 立即隔离受影响系统
2. 保留日志证据
3. 修改所有密码
4. 通知受影响用户
5. 进行安全审计

## 📞 联系方式

- **技术负责人**：[填写]
- **运维负责人**：[填写]
- **紧急联系电话**：[填写]
- **监控告警群**：[填写]

## 📚 相关文档

- [部署文档](./DEPLOY.md)
- [API 文档](./API.md)
- [故障排查手册](./TROUBLESHOOTING.md)
- [安全最佳实践](./SECURITY.md)
