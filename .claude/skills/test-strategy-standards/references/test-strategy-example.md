# 测试策略示例 — 密码恢复

> **注意：** 注释示例。`<!-- -->` 注释解释了为什么每个部分满足检查清单。在实际策略中删除它们。

---

## 1. 单元测试

### UT-1: `PasswordRecoveryDomain.generateCode()`

- **测试内容：** 生成带正确过期时间戳的数字 OTP 验证码
- **覆盖案例：**
  - 正常路径：返回恰好 6 位数字的字符串
  - 过期：`expires_at` 是 `now() + 5 分钟`（± 1 秒容差）
  - 随机性：100 次执行不产生连续相同的验证码
- **所需 Mock：** 模拟时钟以在 `expires_at` 断言中控制 `now()`
- **可追溯性：** REQ-1 · NFR-2

---

### UT-2: `PasswordRecoveryDomain.validateCode()`

- **测试内容：** 根据有效性规则验证 OTP 验证码
- **覆盖案例：**
  - 正常路径：`expires_at` 在未来且 `used_at` 为 null 的验证码 → 接受
  - 验证码已过期：`expires_at` 在过去 → 拒绝，错误 `CODE_EXPIRED`
  - 验证码已使用：`used_at` 已填充 → 拒绝，错误 `CODE_ALREADY_USED`
  - 验证码不存在：条目 `null` → 拒绝，错误 `CODE_NOT_FOUND`
- **所需 Mock：** 模拟时钟以在过期测试中控制 `now()`
- **可追溯性：** REQ-2 · REQ-9 · REQ-10 · NFR-2

---

### UT-3: `PasswordRecoveryDomain.validatePasswordStrength()`

- **测试内容：** 根据 5 个最低要求验证密码强度
- **覆盖案例：**
  - 正常路径：满足所有要求的密码 → 接受，返回空列表
  - 无大写：返回 `['MISSING_UPPERCASE']`
  - 无小写：返回 `['MISSING_LOWERCASE']`
  - 无数字：返回 `['MISSING_NUMBER']`
  - 无特殊字符：返回 `['MISSING_SPECIAL']`
  - 少于 8 字符：返回 `['MIN_LENGTH']`
  - 多个缺失：所有违反的要求同时返回
- **所需 Mock：** 无 — 纯领域
- **可追溯性：** REQ-8

---

### UT-4: `PasswordRecoveryDomain.validatePasswordMatch()`

- **测试内容：** 密码与确认密码的比较
- **覆盖案例：**
  - 字段相同 → 接受
  - 字段不同 → 拒绝，错误 `PASSWORDS_DO_NOT_MATCH`
  - 一个字段为空 → 拒绝
- **所需 Mock：** 无 — 纯领域
- **可追溯性：** REQ-7

---

## 2. 集成测试

### IT-1: `VerificationCodeRepository` — 验证码生命周期操作

- **测试内容：** 数据库中 OTP 验证码的持久化、检索和失效
- **使用的真实依赖：** 测试 PostgreSQL
- **覆盖案例：**
  - `save()`：持久化包含所有字段的验证码；`used_at` 初始为 null
  - `findValid()`：验证码有效时返回验证码；已过期验证码返回 null；`used_at` 已填充的验证码返回 null
  - `markAsUsed()`：填充 `used_at`；后续调用 `findValid()` 返回 null
- **所需 Setup：** 每个测试清理数据库（teardown 时清空 `verification_codes`）；预插入测试配送员
- **可追溯性：** REQ-1 · REQ-2 · REQ-9 · REQ-10

---

### IT-2: `RateLimiterAdapter` — Redis 中的尝试控制

- **测试内容：** 计数器递增、阻止验证、TTL 归零和成功重置
- **使用的真实依赖：** 测试 Redis
- **覆盖案例：**
  - `increment()` + `check()`：计数器正确增长；第 5 次调用 `check()` 返回已阻止
  - TTL：键在 30 分钟后过期（通过测试中 Redis 的 `TTL` 验证）
  - `reset()`：归零计数器；之后 `check()` 立即返回未阻止
  - 键隔离：联系方式的阻止不阻止 IP，反之亦然
- **所需 Setup：** 每个 IT 前清空测试 Redis
- **可追溯性：** REQ-11 · REQ-12 · NFR-4

---

### IT-3: `DeliveryDriverRepository` — 按联系方式读取和更新

- **测试内容：** 按渠道/联系方式查找配送员并更新密码哈希
- **使用的真实依赖：** 测试 PostgreSQL
- **覆盖案例：**
  - `findByContact('email', 存在邮箱)`：返回完整实体
  - `findByContact('email', 不存在邮箱)`：返回 null 不抛出错误
  - `findByContact('sms', 存在电话)`：返回完整实体
  - `updatePasswordHash(driverId, 新哈希)`：哈希在数据库中已更新
- **所需 Setup：** 预插入测试配送员；teardown 时清理
- **可追溯性：** REQ-1 · REQ-3 · REQ-6

---

### IT-4: `EmailNotificationAdapter` — 通过邮件发送验证码
### IT-5: `SmsNotificationAdapter` — 通过 SMS 发送验证码

（结构与 IT-4 相同，适用于 SMS）

---

## 3. E2E Gherkin 测试

### GH-1 至 GH-8

每个 `.feature` 文件的 Scenario 一个 GH。涵盖正常路径、无效格式、未注册联系人、过期验证码、密码不一致、弱密码、过度尝试阻止和验证码重用。

---

## 4. 性能测试

### PT-1: 30 秒内投递验证码到渠道（NFR-1）
- **测量内容：** `POST /request` 调用到渠道（邮件或 SMS）实际投递之间的时间
- **阈值：** p95 ≤ 30 秒（正常操作条件下）
- **可追溯性：** NFR-1

### PT-2: 密码哈希执行时间 ≥ 100ms（NFR-5）
- **测量内容：** 每次单独操作 `PasswordService.hash()` 的执行时间
- **阈值：** 每次执行 ≥ 100ms（最低限制）
- **可追溯性：** NFR-5

---

## 5. 安全测试

### ST-1: 中性响应不泄露账户存在（NFR-3 / 反枚举）
- **验证内容：** 攻击者无法通过系统响应确定邮箱是否已注册
- **模拟的攻击向量：** 账户枚举
- **可追溯性：** NFR-3 · PRD 风险

### ST-2: 速率限制在 5 次尝试后阻止（NFR-4）
- **验证内容：** 速率限制机制按联系方式和 IP 独立工作
- **模拟的攻击向量：** 滥用 SMS/邮件发送的验证码请求垃圾邮件
- **可追溯性：** NFR-4 · PRD 风险

### ST-3: OTP 验证码使用后失效（REQ-10 / 重放风险）
### ST-4: 密码哈希抵抗离线暴力破解（NFR-5）

---

## 覆盖率摘要

| 需求 | 单元 | 集成 | E2E Gherkin | 性能 | 安全 |
|-----------|----------|------------|-------------|-------------|-----------|
| REQ-1 | UT-1 | IT-1, IT-4, IT-5 | GH-1 | PT-1 | — |
| REQ-2 | UT-2 | IT-1 | GH-1 | — | — |
| REQ-3 | — | IT-3 | GH-1 | — | ST-4 |
| REQ-4 | — | — | GH-1 | — | — |
| REQ-5 | — | — | GH-2 | — | — |
| REQ-6 | — | IT-3 | GH-3 | — | ST-1 |
| REQ-7 | UT-4 | — | GH-5 | — | — |
| REQ-8 | UT-3 | — | GH-6 | — | — |
| REQ-9 | UT-2 | IT-1 | GH-4 | — | — |
| REQ-10 | UT-2 | IT-1 | GH-8 | — | ST-3 |
| REQ-11 | — | IT-2 | GH-7 | — | ST-2 |
| REQ-12 | — | IT-2 | GH-7 | — | — |
| NFR-1 | — | IT-4, IT-5 | — | PT-1 | — |
| NFR-2 | UT-2 | IT-1 | GH-4, GH-8 | — | ST-3 |
| NFR-3 | — | IT-3 | GH-3 | — | ST-1 |
| NFR-4 | — | IT-2 | GH-7 | — | ST-2 |
| NFR-5 | — | — | — | PT-2 | ST-4 |
| NFR-6 | — | — | — | — | — |
