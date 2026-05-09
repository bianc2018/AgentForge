---
name: "OPSX: Continue"
description: 继续处理变更 — 创建下一个产物（实验性）
category: Workflow
tags: [workflow, artifacts, experimental]
---

通过创建下一个产物来继续处理变更。

**输入**：可选地在 `/opsx:continue` 之后指定变更名称（如：`/opsx:continue add-auth`）。如果省略，检查是否可以从会话上下文中推断。如果模糊或歧义，你必须提示用户可用的变更。

**步骤**

1. **如果未提供变更名称，提示用户选择**

   运行 `openspec list --json` 获取可用变更，按最近修改时间排序。然后使用 **AskUserQuestion 工具**让用户选择要处理的变更。

   将最近修改的 3-4 个变更作为选项呈现，显示：
   - 变更名称
   - Schema（来自 `schema` 字段，如果存在；否则为 "spec-driven"）
   - 状态（如："0/5 任务"、"完成"、"无任务"）
   - 最近修改时间（来自 `lastModified` 字段）

   将最近修改的变更标记为"（推荐）"，因为它很可能是用户想要继续的。

   **重要**：不要猜测或自动选择变更。始终让用户选择。

2. **检查当前状态**
   ```bash
   openspec status --change "<name>" --json
   ```
   解析 JSON 以了解当前状态。响应包括：
   - `schemaName`：正在使用的工作流 schema（如："spec-driven"）
   - `artifacts`：产物数组及其状态（"done"、"ready"、"blocked"）
   - `isComplete`：布尔值，指示所有产物是否已完成

3. **根据状态执行操作**：

   ---

   **如果所有产物已完成（`isComplete: true`）**：
   - 祝贺用户
   - 显示最终状态，包括使用的 schema
   - 建议："所有产物已创建！你现在可以使用 `/opsx:apply` 实现此变更，或使用 `/opsx:archive` 归档。"
   - 停止

   ---

   **如果有产物可以创建**（状态显示有 `status: "ready"` 的产物）：
   - 从状态输出中选择第一个 `status: "ready"` 的产物
   - 获取其说明：
     ```bash
     openspec instructions <artifact-id> --change "<name>" --json
     ```
   - 解析 JSON。关键字段是：
     - `context`：项目背景（给你的约束 — 不要包含在输出中）
     - `rules`：产物特定的规则（给你的约束 — 不要包含在输出中）
     - `template`：输出文件使用的结构
     - `instruction`：Schema 特定的指导
     - `outputPath`：产物写入位置
     - `dependencies`：需要读取上下文的已完成产物
   - **创建产物文件**：
     - 读取所有已完成的依赖文件以获取上下文
     - 使用 `template` 作为结构 — 填充其各个 section
     - 应用 `context` 和 `rules` 作为写入时的约束 — 但不要将它们复制到文件中
     - 写入到说明中指定的输出路径
   - 显示创建了什么以及现在解锁了什么
   - 在创建**一个**产物后停止

   ---

   **如果没有产物就绪（全部被阻塞）**：
   - 这在有效的 schema 下不应该发生
   - 显示状态并建议检查问题

4. **创建产物后，显示进度**
   ```bash
   openspec status --change "<name>"
   ```

**输出**

每次调用后，显示：
- 创建了哪个产物
- 正在使用的 Schema 工作流
- 当前进度（N/M 已完成）
- 哪些产物现在已解锁
- 提示："运行 `/opsx:continue` 创建下一个产物"

**产物创建指南**

产物类型及其目的取决于 schema。使用说明输出中的 `instruction` 字段来了解要创建什么。

常见的产物模式：

**spec-driven schema**（proposal → specs → design → tasks）：
- **proposal.md**：如果不清楚变更内容，询问用户。填写 Why、What Changes、Capabilities、Impact。
  - Capabilities section 至关重要 — 列出的每个 capability 都需要一个 spec 文件。
- **specs/<capability>/spec.md**：为 proposal 的 Capabilities section 中列出的每个 capability 创建一个 spec（使用 capability 名称，不是变更名称）。
- **design.md**：记录技术决策、架构和实现方法。
- **tasks.md**：将实现分解为带复选框的任务。

对于其他 schema，遵循 CLI 输出中的 `instruction` 字段。

**约束**
- 每次调用创建一个产物
- 创建新产物之前始终读取依赖产物
- 绝不跳过产物或无序创建
- 如果上下文不清楚，在创建之前询问用户
- 在标记进度之前，写入后验证产物文件是否存在
- 使用 schema 的产物序列，不要假设特定的产物名称
- **重要**：`context` 和 `rules` 是给你的约束，不是文件的内容
  - 不要将 `<context>`、`<rules>`、`<project_context>` 块复制到产物中
  - 这些指导你写什么，但绝不应出现在输出中
