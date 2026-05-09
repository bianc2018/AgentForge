---
name: "OPSX: Bulk Archive"
description: 一次性归档多个已完成的变更
category: Workflow
tags: [workflow, archive, experimental, bulk]
---

在单个操作中归档多个已完成的变更。

此 skill 允许你批量归档变更，通过检查代码库来确定实际实现了什么，从而智能地处理 spec 冲突。

**输入**：无需输入（提示用户选择）

**步骤**

1. **获取活跃变更**

   运行 `openspec list --json` 获取所有活跃变更。

   如果没有活跃变更，通知用户并停止。

2. **提示用户选择变更**

   使用 **AskUserQuestion 工具**并启用多选，让用户选择变更：
   - 显示每个变更及其 schema
   - 包含"所有变更"选项
   - 允许任意数量的选择（1+ 可以，2+ 是典型用例）

   **重要**：不要自动选择。始终让用户选择。

3. **批量验证 — 为所有选定变更收集状态**

   对每个选定的变更，收集：

   a. **产物状态** - 运行 `openspec status --change "<name>" --json`
      - 解析 `schemaName` 和 `artifacts` 列表
      - 记录哪些产物是 `done` 与其他状态

   b. **任务完成** - 读取 `openspec/changes/<name>/tasks.md`
      - 计算 `- [ ]`（未完成）vs `- [x]`（已完成）
      - 如果没有任务文件，记录为"无任务"

   c. **Delta specs** - 检查 `openspec/changes/<name>/specs/` 目录
      - 列出存在哪些 capability specs
      - 对每个，提取需求名称（匹配 `### Requirement: <name>` 的行）

4. **检测 spec 冲突**

   构建 `capability -> [触及它的变更]` 的映射：

   ```
   auth -> [change-a, change-b]  <- 冲突（2+ 个变更）
   api  -> [change-c]            <- 正常（只有 1 个变更）
   ```

   当 2 个以上选定的变更有相同 capability 的 delta specs 时，存在冲突。

5. **通过 agent 解决冲突**

   **对每个冲突**，调查代码库：

   a. **读取**每个冲突变更的 delta specs，了解每个变更声称添加/修改了什么

   b. **搜索代码库**以获取实现证据：
      - 查找实现每个 delta spec 中需求的代码
      - 检查相关文件、函数或测试

   c. **确定解决方案**：
      - 如果只有一个变更实际实现了 → 同步那个变更的 specs
      - 如果两个都实现了 → 按时间顺序应用（旧的先，新的覆盖）
      - 如果都没有实现 → 跳过 spec 同步，警告用户

   d. **记录**每个冲突的解决方案：
      - 应用哪个变更的 specs
      - 按什么顺序（如果两个都应用）
      - 理由（在代码库中发现了什么）

6. **显示合并状态表**

   显示汇总所有变更的表格：

   ```
   | 变更                 | 产物 | 任务 | Specs   | 冲突 | 状态 |
   |---------------------|-----------|-------|---------|-----------|--------|
   | schema-management   | 已完成    | 5/5   | 2 delta | 无       | 就绪  |
   | project-config      | 已完成    | 3/3   | 1 delta | 无       | 就绪  |
   | add-oauth           | 已完成    | 4/4   | 1 delta | auth (!)  | 就绪* |
   | add-verify-skill    | 剩余 1    | 2/5   | 无      | 无       | 警告  |
   ```

   对于冲突，显示解决方案：
   ```
   * 冲突解决方案：
     - auth spec：将先应用 add-oauth 然后 add-jwt（两者都已实现，按时间顺序）
   ```

   对于不完整的变更，显示警告：
   ```
   警告：
   - add-verify-skill：1 个未完成的产物，3 个未完成的任务
   ```

7. **确认批量操作**

   使用 **AskUserQuestion 工具**进行单次确认：

   - "归档 N 个变更？" 选项基于状态
   - 选项可能包括：
     - "归档所有 N 个变更"
     - "仅归档 N 个就绪的变更（跳过不完整的）"
     - "取消"

   如果存在不完整的变更，明确说明它们将带着警告被归档。

8. **为每个确认的变更执行归档**

   按确定的顺序处理变更（尊重冲突解决方案）：

   a. **同步 specs** 如果存在 delta specs：
      - 使用 openspec-sync-specs 方式（agent 驱动的智能合并）
      - 对于冲突，按解决的顺序应用
      - 跟踪同步是否已完成

   b. **执行归档**：
      ```bash
      mkdir -p openspec/changes/archive
      mv openspec/changes/<name> openspec/changes/archive/YYYY-MM-DD-<name>
      ```

   c. **跟踪**每个变更的结果：
      - 成功：已成功归档
      - 失败：归档期间出错（记录错误）
      - 跳过：用户选择不归档（如适用）

9. **显示摘要**

   显示最终结果：

   ```
   ## 批量归档完成

   已归档 3 个变更：
   - schema-management-cli -> archive/2026-01-19-schema-management-cli/
   - project-config -> archive/2026-01-19-project-config/
   - add-oauth -> archive/2026-01-19-add-oauth/

   已跳过 1 个变更：
   - add-verify-skill（用户选择不归档不完整的变更）

   Spec 同步摘要：
   - 4 个 delta specs 已同步到主 specs
   - 1 个冲突已解决（auth：按时间顺序应用了两者）
   ```

   如果有失败：
   ```
   失败 1 个变更：
   - some-change：归档目录已存在
   ```

**冲突解决示例**

示例 1：只有一个实现了
```
冲突：specs/auth/spec.md 被 [add-oauth, add-jwt] 触及

检查 add-oauth：
- Delta 添加了 "OAuth Provider Integration" 需求
- 搜索代码库... 找到 src/auth/oauth.ts 实现了 OAuth 流程

检查 add-jwt：
- Delta 添加了 "JWT Token Handling" 需求
- 搜索代码库... 未找到 JWT 实现

解决方案：只有 add-oauth 已实现。将仅同步 add-oauth specs。
```

示例 2：两者都实现了
```
冲突：specs/api/spec.md 被 [add-rest-api, add-graphql] 触及

检查 add-rest-api（创建于 2026-01-10）：
- Delta 添加了 "REST Endpoints" 需求
- 搜索代码库... 找到 src/api/rest.ts

检查 add-graphql（创建于 2026-01-15）：
- Delta 添加了 "GraphQL Schema" 需求
- 搜索代码库... 找到 src/api/graphql.ts

解决方案：两者都已实现。将先应用 add-rest-api specs，
然后 add-graphql specs（按时间顺序，新的优先）。
```

**成功时输出**

```
## 批量归档完成

已归档 N 个变更：
- <change-1> -> archive/YYYY-MM-DD-<change-1>/
- <change-2> -> archive/YYYY-MM-DD-<change-2>/

Spec 同步摘要：
- N 个 delta specs 已同步到主 specs
- 无冲突（或：M 个冲突已解决）
```

**部分成功时输出**

```
## 批量归档完成（部分）

已归档 N 个变更：
- <change-1> -> archive/YYYY-MM-DD-<change-1>/

已跳过 M 个变更：
- <change-2>（用户选择不归档不完整的变更）

失败 K 个变更：
- <change-3>：归档目录已存在
```

**无变更时输出**

```
## 没有可归档的变更

未找到活跃的变更。使用 `/opsx:new` 创建新变更。
```

**约束**
- 允许任意数量的变更（1+ 可以，2+ 是典型用例）
- 始终提示用户选择，绝不自动选择
- 尽早检测 spec 冲突并通过检查代码库来解决
- 当两个变更都实现了，按时间顺序应用 specs
- 仅当实现缺失时跳过 spec 同步（警告用户）
- 在确认之前显示清晰的每个变更的状态
- 对整个批次使用单次确认
- 跟踪并报告所有结果（成功/跳过/失败）
- 移动到归档时保留 .openspec.yaml
- 归档目录目标使用当前日期：YYYY-MM-DD-<name>
- 如果归档目标已存在，该变更失败但继续处理其他变更
