# 实现循环示例 — 密码恢复

> `impl-agent` 如何执行两个连续 task 的注释示例，
> 展示每个 task 的完整循环，包括呈现给用户的报告。

---

## 会话启动

```
[impl-agent] 正在实现 feature：密码恢复
已加载产物：tasks.md, test-strategy.md, design.md, constitution.md, CLAUDE.md

下一个 task：T-01 — 创建实体 `VerificationCode`
已完成 task：0/37（37 个待处理）

开始实现。
```

---

## T-01：创建实体 `VerificationCode`

### A. 读取 task

```
可追溯性：REQ-1 · REQ-9 · REQ-10 · NFR-2
依赖：—
完成标准：实体在 ORM 中定义，包含所有字段和指向 delivery_drivers 的 FK
```

### B. 规划

- 要创建的文件：`prisma/schema.prisma`（添加 model）
- 根据 `CLAUDE.md`：实体在 Prisma schema 中定义
- 检查 constitution.md：数据模型不包含业务逻辑 ✅

### C. 实现

编辑 `prisma/schema.prisma` 添加 VerificationCode model。

### D. 静态验证

```bash
npx prisma validate && npx tsc --noEmit
```
✅ 无错误

### E. 测试
Model task — 无直接 UT/IT。执行 migration smoke test。

### G. 标记 task
`- [ ] T-01` → `- [x] T-01`

### H. 呈现给用户的报告
```
✅ T-01 已完成 — 创建实体 `VerificationCode`

已完成内容：
- prisma/schema.prisma：添加了包含 7 个字段和指向 DeliveryDriver 的 FK 的 VerificationCode model

已执行测试：
- Migration smoke test：✅ 无错误应用
- TypeScript typecheck：✅ 无错误

已覆盖可追溯性：REQ-1 · REQ-9 · REQ-10 · NFR-2

已完成 Task：1/37
下一个：T-02 — 创建表 `verification_codes` 的 migration

继续吗？
```

---

## T-03：实现 `PasswordRecoveryDomain.generateCode()`

### A. 读取 task
可追溯性：REQ-1 · NFR-2 · DT-1，测试：UT-1

### B. 规划
- 要创建的文件：`src/domain/password-recovery/generate-code.ts` 及 `.spec.ts`
- Constitution.md 已验证：纯领域，无基础设施导入 ✅

### C. 实现
创建 `generateCode.ts`（6 位 OTP，5 分钟过期）和测试文件。

### D-E. 验证 + 测试
✅ typecheck 通过，3 个 UT 用例全部通过。

### H. 报告
```
✅ T-03 已完成 — 实现 `PasswordRecoveryDomain.generateCode()`

已完成内容：
- src/domain/password-recovery/generate-code.ts：生成 6 位 OTP，过期时间为 now() + 5min
- src/domain/password-recovery/generate-code.spec.ts：覆盖 UT-1 的 3 个测试用例

已执行测试：
- UT-1 (generateCode)：✅ 3 个用例通过

已覆盖可追溯性：REQ-1 · NFR-2

已完成 Task：3/37
下一个：T-04 — 实现 `VerificationCodeRepository.save()`

继续吗？
```

---

## ⚠️ 报告示例（因 constitution.md 做出调整）

```
✅ T-09 已完成 — 实现 POST /api/v1/password-recovery/request

⚠️  注意：速率限制验证逻辑保留在 service 中（application 层），
    而非 controller，符合 constitution.md 规则 2
    （"业务逻辑在领域之外"）。

继续吗？
```

---

## 🔴 阻塞报告示例

```
🔴 T-15 已阻塞 — 实现 `PasswordService.hash()`

尝试修正：3 次
持续错误：
  PT-2 失败：测量的哈希时间 = 45ms（阈值：≥ 100ms）

已尝试：
- Cost factor 10：45ms（低于阈值）
- Cost factor 12：87ms（低于阈值）
- Cost factor 13：210ms（通过但 PT-2 因超出 500ms 上限而失败）

我需要指导才能继续。该怎么办？
```
