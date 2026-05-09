# Tasks 示例 — 密码恢复

> **注意：** 注释示例。`<!-- -->` 注释解释了为什么每个部分满足检查清单。在实际 tasks 中删除它们。
>
> 来源产物是 feature `recuperacao-de-senha` 的：
> `requirements.md`、`nf-requirements.md`、`scenarios.feature` 和 `design.md`。

---

## REQ-1 — 发送验证码

> When a registered delivery driver submits a valid contact detail and requests a verification code, the system shall send a verification code to the chosen contact channel.

### T-01: 创建实体 `VerificationCode`

- [ ] 在 ORM 中定义 `VerificationCode` 实体，字段包括：`id` (UUID), `driver_id` (UUID, FK), `code` (string, 6 位), `channel` (enum: email|sms), `expires_at` (timestamp), `used_at` (timestamp 可为空), `created_at` (timestamp)。

**可追溯性：** REQ-1 · REQ-9 · REQ-10 · NFR-2
**依赖：** —
**完成标准：** 实体在 ORM 中已定义，所有字段、类型和指向 `delivery_drivers` 的 FK 已正确配置。

---

### T-02: 创建表 `verification_codes` 的 migration

- [ ] 创建 migration，建立 `verification_codes` 表，包含指向 `delivery_drivers` 的 FK 和必填字段的 NOT NULL 约束。

**可追溯性：** REQ-1 · REQ-10
**依赖：** T-01
**完成标准：** Migration 在开发环境中无错误执行，且表以正确结构存在于数据库中。

---

### T-03: 实现 `PasswordRecoveryDomain.generateCode()`

- [ ] 实现生成随机 6 位数字 OTP 验证码并计算过期时间戳为 `now() + 5 分钟` 的方法。

**可追溯性：** REQ-1 · NFR-2 · DT-1
**依赖：** —
**完成标准：** 方法生成恰好 6 位数字的验证码，返回的 `expires_at` 始终为 `now() + 5min`。

---

### T-04: 实现 `VerificationCodeRepository.save()`

- [ ] 实现 `save(driverId, code, channel, expiresAt)` 方法，将新 `VerificationCode` 记录持久化到数据库。

**可追溯性：** REQ-1
**依赖：** T-01 · T-02
**完成标准：** 方法持久化包含所有字段的记录，且记录可通过 `driverId` 检索。

---

### T-05: 实现 `DeliveryDriverRepository.findByContact()`

- [ ] 实现 `findByContact(channel, contact)` 方法，按提供的邮箱或电话查找配送员。未找到联系方式时应返回 `null`（不抛出错误）。

**可追溯性：** REQ-1 · REQ-6 · NFR-3
**依赖：** —
**完成标准：** 联系方式存在时返回配送员实体；不存在时静默返回 `null`。

---

### T-06: 实现 `EmailNotificationAdapter.send()`

- [ ] 实现通过项目中配置的外部邮件服务发送 OTP 验证码的 adapter。

**可追溯性：** REQ-1 · NFR-1
**依赖：** —
**完成标准：** Adapter 发送包含 OTP 验证码的邮件，且在测试环境中可接收。

---

### T-07: 实现 `SmsNotificationAdapter.send()` （同上，SMS）

### T-08: 创建 `POST /request` 的载荷验证 schema

- [ ] 为请求载荷创建验证 schema（如 Zod）：`channel`（enum: email|sms，必填）和 `contact`（string，必填，格式根据渠道验证 — 邮箱或电话）。

**可追溯性：** REQ-1 · REQ-5
**依赖：** —
**完成标准：** Schema 拒绝缺失的 `channel`、空的 `contact` 以及格式对指定渠道无效的 `contact`。

---

### T-09: 实现 `POST /api/v1/password-recovery/request`

- [ ] 实现完整 endpoint：使用 T-08 验证载荷，检查速率限制，查找配送员，如已注册则生成并持久化 OTP，通过对应渠道发送，无论联系方式是否已注册始终返回中性确认消息（200 OK）。

**可追溯性：** REQ-1 · REQ-6 · NFR-3 · Scenario: "使用邮件和有效验证码恢复密码"
**依赖：** T-03 · T-04 · T-05 · T-06 · T-07 · T-08
**完成标准：** Endpoint 对已注册和未注册的联系方式均返回 200 及中性消息；仅对已注册联系方式发送验证码。

---

### T-10: 覆盖 Scenario "使用邮件和有效验证码恢复密码"（E2E）

### ↳ NFR-1 — 95% 在 30 秒内投递

### T-11: 验证码投递性能测试（NFR-1）

---

## REQ-2 — 有效验证码后推进到重置步骤

> When the delivery driver submits a valid and unexpired verification code, the system shall advance the driver to the password reset step.

### T-12: 实现 `PasswordRecoveryDomain.validateCode()`

- [ ] 实现 `validateCode(code, entry)` 方法，验证验证码是否存在、`expires_at > now()` 以及 `used_at` 是否为 null。对每种失败条件返回描述性错误。

**可追溯性：** REQ-2 · REQ-9 · REQ-10 · NFR-2
**依赖：** T-01
**完成标准：** 方法接受有效验证码，拒绝已过期和已使用的验证码，每种情况有明确的错误。

---

### T-13: 实现 `VerificationCodeRepository.findValid()`

### T-14: 实现 `POST /api/v1/password-recovery/verify`

---

## REQ-3 至 REQ-12（类似结构）

（为简洁起见省略了中间 task — 完整示例遵循相同模式，包含实体、migration、领域方法、repository 方法、adapter、schema、endpoint 和测试的 task）

---

### ↳ NFR-6 — 尝试日志记录，保留 1 年

### T-37: 配置验证码请求事件的日志记录

- [ ] 配置 `POST /request` 所有事件的日志记录：成功发送验证码、未注册联系方式（不泄露是哪个）、无效格式和速率限制阻止。日志必须包含 timestamp、事件类型、渠道和联系方式哈希（不含明文 PII）。最低保留期配置为：1 年。

**可追溯性：** NFR-6 · REQ-11
**依赖：** T-09
**完成标准：** 所有列出的事件在开发环境日志中可见，且 1 年保留策略在项目日志基础设施中已配置。

---

## 依赖图

（从每个 task 的"依赖"字段自动推导的 Mermaid flowchart）
