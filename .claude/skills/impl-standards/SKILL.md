---
name: impl-standards
description: >
    SDD 产物引导的实现质量标准。定义
    每个 task 的实现循环（实现 → 静态验证 →
    测试 → 确认标准 → 报告 → 等待批准）、各测试类型的
    验证规则、报告格式和 task 间的推进条件。
    配合 verification-guide 和 impl-example 使用。
---

# 实现标准

## 什么是实现循环

实现循环 **一次一个**地执行 `tasks.md` 中的 task，
每一步都内置验证。用户批准每个 task 后 agent
才推进到下一个。

验证不是可选的，也不仅发生在最后 — 它是每个 task
循环的一部分。

---

## 在 SDD 流程中的位置

```
... → design.md → test-strategy.md → tasks.md → [实现] → 代码
```

`impl-agent` 消费 `tasks.md` 和 `test-strategy.md` 作为输入，
生成验证过的代码作为输出。

---

## 每个 task 的循环

```
读取 task → 规划 → 实现 + 测试 → Lint/typecheck
→ 运行追踪的测试 → 验证完成标准
→ 标记 - [x] → 报告 → 等待批准
```

不能跳过任何步骤。如果某步骤失败，循环返回到
实现步骤 — 不推进。

---

## 各测试类型的验证规则

### 单元测试 (UT-N)

**何时运行：** 领域 task — 实现业务规则的方法

**如何识别要运行的测试：** task 的 `可追溯性` 字段列出了 ID（如：`UT-1 · UT-2`）。在 `test-strategy.md` 中定位对应的 `UT-N` 块，了解每个测试确切验证什么。

**通过标准：**
- [ ] `test-strategy.md` 中 UT 块的所有案例已覆盖
- [ ] 没有测试使用数据库、HTTP 或外部服务（如果有，属于 IT 而非 UT）
- [ ] 时钟和非确定性依赖已 mock

**典型命令：** `npm test -- --testPathPattern="<组件名>.spec"`

---

### 集成测试 (IT-N)

**何时运行：** repository、adapter 或任何基础设施组件的 task

**前置条件：** 测试数据库和测试 Redis 必须可用。如果不可用，在报告中标注并询问用户如何处理。

**通过标准：**
- [ ] `test-strategy.md` 中 IT 块的所有案例已覆盖
- [ ] 外部状态的 setup 和 teardown 正确执行
- [ ] 适用时，依赖失败测试（如：Redis 不可用）通过

**典型命令：** `npm run test:integration -- --testPathPattern="<名称>"`

---

### E2E Gherkin 测试 (GH-N)

**何时运行：** API endpoint task — 实现 controller/handler 之后

**前置条件：** Scenario 流程的所有组件必须已实现。如果某些还不存在，GH 可以实现但不会通过 — 在报告中记录并继续。

**通过标准：**
- [ ] `.feature` 的 Scenario 无错误执行
- [ ] Scenario 所有 steps 的 step definitions 已实现
- [ ] Scenario 的初始状态在 setup/teardown 中正确创建/销毁

**典型命令：** `npm run test:e2e -- --tags "@<scenario-名称>"`

---

### 性能测试 (PT-N)

**何时运行：** 实现带有可测量 NFR 的组件的 task（如：`PasswordService.hash()`、通知 adapters）

**通过标准：**
- [ ] 测量的指标在 NFR 的阈值内
- [ ] `test-strategy.md` 中指定的执行次数已遵守
- [ ] 测试足够确定性以在 CI 中运行

**注意：** 重型负载测试（如：100 rps 持续 60s）不应在每次 task 时运行 — 仅当实现特定的性能 task 时。

---

### 安全测试 (ST-N)

**何时运行：** 实现安全机制的 task（速率限制器、中性响应、token 失效）

**通过标准：**
- [ ] `test-strategy.md` 中描述的攻击向量已模拟
- [ ] 系统按预期行为响应（HTTP 状态码、无敏感数据）
- [ ] 响应中没有本不应暴露的信息

---

## 报告格式

报告在每个 task 结束时通过 `AskUserQuestion` 呈现。应在 30 秒内读完。

```
✅ T-<NN> 已完成 — <标题>

已完成内容：
- <路径/文件.ts>：<一句话>
- <路径/文件.spec.ts>：<一句话>

已执行测试：
- <UT-N | IT-N | GH-N | PT-N | ST-N>：✅ <N> 个案例通过
- <UT-N | IT-N | GH-N | PT-N | ST-N>：✅ <N> 个案例通过

已覆盖可追溯性：<REQ-N> · <NFR-N>

已完成 Task：<N>/<total>
下一个：T-<NN> — <标题>

继续吗？
```

**带有提醒的变体（不阻塞，但告知）：**
```
✅ T-<NN> 已完成 — <标题>

已完成内容：...

已执行测试：...

⚠️  注意：<值得审查的情况，如："实现已调整以遵守
    constitution.md 规则 3 — 领域不直接导入 Prisma">

继续吗？
```

**阻塞变体（在解决前不推进）：**
```
🔴 T-<NN> 阻塞 — <标题>

尝试修正：3 次
持续错误：
  <相关错误消息 — 非完整 dump>

已尝试：
- <尝试 1>
- <尝试 2>
- <尝试 3>

我需要指导才能继续。该怎么办？
```

---

## 推进条件

Agent **仅**在以下情况下推进到下一个 task：

1. Lint/typecheck 无错误通过
2. task 中所有追踪的测试通过
3. "完成标准"客观上已满足
4. task 在 `tasks.md` 中已标记 `- [x]`
5. 用户通过 `AskUserQuestion` 批准

如果任何条件未满足，agent **不推进**。
唯一的例外是持续错误导致的阻塞（3+ 次尝试）— 在此情况下
agent 呈现阻塞报告并等待用户指导。

---

## constitution.md 验证

在实现每个 task 之前，agent 在脑中验证：

| 规则 | 验证 |
|-------|-------------|
| 必须做 — 分层分离 | 组件在正确的层中吗？导入是否跨层？ |
| 必须做 — 错误传播 | 错误是否有类型并被传播，而非被静默？ |
| 必须做 — 输入验证 | 验证是否发生在系统边界（controller/adapter）？ |
| 必须做 — 日志 | 关键操作是否被记录？ |
| 绝不做 — 领域外的逻辑 | 业务规则在领域中，而非 controller 中？ |
| 绝不做 — 静默错误 | 没有空的 catch 或 log-and-continue？ |

如果**在实现期间**检测到违规，在继续之前修正。
在报告的 ⚠️ 字段中记录修正。

---

## 可追溯性覆盖

在每个 task 结束时，确认：

- `可追溯性` 中列出的 REQ 是否被实现的代码覆盖？
- `可追溯性` 中列出的 NFR 是否满足（如：NFR-5 = bcrypt ≥ 100ms）？
- 列出的 Scenarios 是否有 step definitions 已实现？

如果没有：在标记 `- [x]` 之前实现缺失的内容。

---

## 参考

- 各场景的验证指南：`references/verification-guide.md`
- 完整循环的注释示例：`references/impl-example.md`
