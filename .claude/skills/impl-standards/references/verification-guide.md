# 验证指南 — 常见情况

## 如何识别要运行的test

1. 读取当前 task 的 `可追溯性` 字段 — UT/IT/GH/PT/ST ID 就是索引
2. 使用 `grep` 仅提取 `test-strategy.md` 中的相关块 — 不要加载整个文件：
   ```bash
   grep -A 15 "### UT-1:" docs/features/<slug>/test-strategy.md
   grep -A 15 "### IT-3:" docs/features/<slug>/test-strategy.md
   ```
3. 对于 `design.md` 中的组件，仅提取组件的 section：
   ```bash
   grep -A 20 "### ComponentName" docs/features/<slug>/design.md
   ```
4. 对于 `.feature` 中的 Scenarios，仅提取相关场景：
   ```bash
   grep -A 15 'Scenario: "场景名称"' docs/features/<slug>/scenarios.feature
   ```

**规则：** 如果你加载整个文件只为使用 10 行，就用 grep。

---

## 阻塞情况及如何处理

### test因缺失依赖而失败

**症状：** IT 失败因为test数据库未运行；GH 失败因为 Redis 不可用。

**操作：**
1. 检查 `CLAUDE.md` — 是否有关于如何启动test依赖的说明？
2. 如果有：按照说明操作并重新执行
3. 如果没有：使用 `AskUserQuestion` 一次性询问："为了运行集成test，我需要test数据库/Redis。如何启动环境？"

---

### test因组件尚未实现而失败

**症状：** GH-3 失败因为它依赖尚未实现的 T-09。

**操作：**
- 不要因此阻塞 task
- 实现 GH 的 step definition 但在报告中记录：
  ```
  ⚠️  GH-3 已实现但未完全执行 — 依赖尚未实现的 T-09
      （POST /request）。
  ```
- GH 将在 T-09 实现后再次验证

---

### Lint 因未记录的代码约定而失败

**症状：** ESLint/TSC 因 `CLAUDE.md` 中没有的规则而报错。

**操作：**
1. 按错误模式修正（如：添加显式类型、调整导入）
2. 如果修正显著改变了实现设计，在报告中记录
3. 不要就每条 lint 规则询问用户 — 解决并继续

---

### Constitution.md 违反 task 的自然设计

**症状：** 实现 task 最直接的方式会违反 constitution.md 的规则（如：在领域中直接调用 Prisma 会更简单）。

**操作：**
1. **不要违反 constitution.md** — 它优先于便利性
2. 以正确方式实现（如：通过 port 注入 repository）
3. 在报告中使用 ⚠️ 记录："实现使用依赖注入，符合 constitution.md 规则 N，而非直接访问"

---

### Task 的完成标准不明确

**症状：** "完成标准：系统正常工作" — 不可验证。

**操作：**
1. 使用 `可追溯性` 字段推断实际标准（REQ/NFR 描述了预期行为）
2. 针对该行为实现和test
3. 在报告中记录："标准解释为：<已验证的内容>"

---

### Design.md 和 task.md 存在分歧

**症状：** Task 说要实现 `findByEmail()` 但设计说 `findByContact(channel, contact)`。

**操作：**
1. `design.md` 优先 — 它更详细且生成更晚
2. 按 `design.md` 实现
3. 在报告中记录："按 design.md 实现为 `findByContact()` — task.md 使用旧名称"

---

## 呈现报告前的快速检查清单

在使用 `AskUserQuestion` 之前，确认：

- [ ] Lint/typecheck 无错误通过？
- [ ] 所有追踪的test通过？
- [ ] "完成标准"已满足？
- [ ] Task 在 tasks.md 中已标记 `- [x]`？
- [ ] 报告提及了所有创建/编辑的文件？
- [ ] 报告列出了所有已执行test及其结果？
- [ ] 如果因 constitution.md 做出调整，已使用 ⚠️ 记录？

如果任何项目待处理：在呈现报告前解决。
