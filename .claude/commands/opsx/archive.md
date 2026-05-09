---
name: "OPSX: Archive"
description: 在实验性工作流中归档已完成的变更
category: Workflow
tags: [workflow, archive, experimental]
---

在实验性工作流中归档已完成的变更。

**输入**：可选地在 `/opsx:archive` 之后指定变更名称（如：`/opsx:archive add-auth`）。如果省略，检查是否可以从会话上下文中推断。如果模糊或歧义，你必须提示用户可用的变更。

**步骤**

1. **如果未提供变更名称，提示用户选择**

   运行 `openspec list --json` 获取可用变更。使用 **AskUserQuestion 工具**让用户选择。

   仅显示活跃的变更（非已归档的）。
   如果可用，包含每个变更使用的 schema。

   **重要**：不要猜测或自动选择变更。始终让用户选择。

2. **检查产物完成状态**

   运行 `openspec status --change "<name>" --json` 检查产物完成情况。

   解析 JSON 以了解：
   - `schemaName`：正在使用的工作流
   - `artifacts`：产物列表及其状态（`done` 或其他）

   **如果任何产物不是 `done`：**
   - 显示警告，列出未完成的产物
   - 提示用户确认是否继续
   - 如果用户确认则继续

3. **检查任务完成状态**

   读取任务文件（通常是 `tasks.md`）检查未完成的任务。

   计算标记为 `- [ ]`（未完成）与 `- [x]`（已完成）的任务数量。

   **如果发现未完成的任务：**
   - 显示警告，显示未完成任务的计数
   - 提示用户确认是否继续
   - 如果用户确认则继续

   **如果没有任务文件存在：** 在没有任务相关警告的情况下继续。

4. **评估 delta spec 同步状态**

   检查 `openspec/changes/<name>/specs/` 处的 delta specs。如果不存在，在没有同步提示的情况下继续。

   **如果存在 delta specs：**
   - 将每个 delta spec 与其在 `openspec/specs/<capability>/spec.md` 处对应的主 spec 进行比较
   - 确定将应用哪些更改（添加、修改、删除、重命名）
   - 在提示之前显示合并摘要

   **提示选项：**
   - 如果需要更改："立即同步（推荐）"，"不同步直接归档"
   - 如果已经同步："立即归档"，"仍然同步"，"取消"

   如果用户选择同步，使用 Task 工具（subagent_type: "general-purpose"，prompt: "使用 Skill 工具调用变更 '<name>' 的 openspec-sync-specs。Delta spec 分析：<包含分析的 delta spec 摘要>"）。无论选择如何，继续归档。

5. **执行归档**

   如果不存在则创建归档目录：
   ```bash
   mkdir -p openspec/changes/archive
   ```

   使用当前日期生成目标名称：`YYYY-MM-DD-<change-name>`

   **检查目标是否已存在：**
   - 如果是：失败并报错，建议重命名现有归档或使用不同日期
   - 如果否：将变更目录移动到归档

   ```bash
   mv openspec/changes/<name> openspec/changes/archive/YYYY-MM-DD-<name>
   ```

6. **显示摘要**

   显示归档完成摘要，包括：
   - 变更名称
   - 使用的 Schema
   - 归档位置
   - Spec 同步状态（已同步 / 跳过同步 / 无 delta specs）
   - 关于任何警告的说明（不完整的产物/任务）

**成功时输出**

```
## 归档完成

**变更：** <change-name>
**Schema：** <schema-name>
**归档到：** openspec/changes/archive/YYYY-MM-DD-<name>/
**Specs：** ✓ 已同步到主 specs

所有产物已完成。所有任务已完成。
```

**成功时输出（无 Delta Specs）**

```
## 归档完成

**变更：** <change-name>
**Schema：** <schema-name>
**归档到：** openspec/changes/archive/YYYY-MM-DD-<name>/
**Specs：** 无 delta specs

所有产物已完成。所有任务已完成。
```

**成功时输出（有警告）**

```
## 归档完成（有警告）

**变更：** <change-name>
**Schema：** <schema-name>
**归档到：** openspec/changes/archive/YYYY-MM-DD-<name>/
**Specs：** 同步已跳过（用户选择跳过）

**警告：**
- 归档时有 2 个未完成的产物
- 归档时有 3 个未完成的任务
- Delta spec 同步已跳过（用户选择跳过）

如果这不是故意的，请检查归档。
```

**错误时输出（归档已存在）**

```
## 归档失败

**变更：** <change-name>
**目标：** openspec/changes/archive/YYYY-MM-DD-<name>/

目标归档目录已存在。

**选项：**
1. 重命名现有归档
2. 如果是重复的则删除现有归档
3. 等到另一天再归档
```

**约束**
- 如果未提供变更名称，始终提示用户选择
- 使用产物图（openspec status --json）检查完成情况
- 不要因为警告而阻止归档 — 仅通知并确认
- 移动到归档时保留 .openspec.yaml（随目录一起移动）
- 显示清晰的发生了什么摘要
- 如果请求同步，使用 Skill 工具调用 `openspec-sync-specs`（agent 驱动）
- 如果存在 delta specs，始终运行同步评估并在提示之前显示合并摘要
