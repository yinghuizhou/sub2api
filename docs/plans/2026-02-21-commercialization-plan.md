# Sub2API 商业化实现计划（主索引）

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 Sub2API 实现完整的商业化能力：支付充值、邀请返佣、阶梯优惠、分销代理。

**Architecture:** 复用现有 Ent ORM + Gin + Wire 体系，新增 payment/referral/reseller 三个领域模块，每个模块独立 service/handler/repository，通过 Wire 注入。支付使用 gopay 库对接微信 Native + 支付宝 PC 扫码。

**Tech Stack:** Go 1.25.7, Gin, Ent ORM, gopay, PostgreSQL, Redis, Vue 3 + TypeScript

---

## 分批计划文件索引

| 批次 | 文件 | 内容 |
|------|------|------|
| Batch 1 | [plan-batch1-payment.md](./2026-02-21-commercialization-plan-batch1-payment.md) | 支付核心：DB schema + gopay 集成 + 充值套餐 |
| Batch 2 | [plan-batch2-payment-api.md](./2026-02-21-commercialization-plan-batch2-payment-api.md) | 支付 API：handler + 路由 + 回调 + Wire |
| Batch 3 | [plan-batch3-referral.md](./2026-02-21-commercialization-plan-batch3-referral.md) | 邀请返佣：邀请码 + 返佣记录 + 注册集成 |
| Batch 4 | [plan-batch4-growth.md](./2026-02-21-commercialization-plan-batch4-growth.md) | 增长工具：阶梯优惠 + 免费试用 + 分销代理 |
| Batch 5 | [plan-batch5-frontend.md](./2026-02-21-commercialization-plan-batch5-frontend.md) | 前端：充值页 + 邀请页 + 账单页 |

---

## 关键约定

- 迁移文件编号从 `056` 开始（当前最高为 `055`）
- 货币：DB 存 USD，前端展示 CNY（汇率从 settings 读取，key=`payment.cny_to_usd_rate`）
- 支付密钥存 settings 表（加密），不写 config.yaml
- 幂等键：`payment_orders.order_no` 唯一约束
- 返佣仅一级，被邀请人首次充值触发
- 每次完成一个 Task 后立即 commit
