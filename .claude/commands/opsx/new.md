---
name: "OPSX: New"
description: 使用实验性产物工作流 (OPSX) 开始一个新的变更
category: Workflow
tags: [workflow, artifacts, experimental]
---

使用实验性的产物驱动方法开始一个新的变更。

**输入**：`/opsx:new` 后面的参数是变更名称（kebab-case），或者是用户想要构建什么的描述。

**步骤**

1. **如果未提供输入，询问他们想构建什么**

   使用 **AskUserQuestion 工具**（开放式，无预设选项）提问：
   > "你想做什么变更？描述一下你想构建或修复的内容。"

   从他们的描述中推导出 kebab-case 名称（例如，"添加用户认证" → `add-user-auth`）。

   **重要**：在理解用户想要构建什么之前，不要继续。

2. **确定工作流 schema**

   使用默认 schema（省略 `--schema`），除非用户明确请求不同的工作流。

   **仅在用户提到以下情况时使用不同的 schema：**
   - 特定的 schema 名称 → 使用 `--schema <name>`
   - "显示工作流"或"有哪些工作流" → 运行 `openspec schemas --json` 并让他们选择

   **否则**：省略 `--schema` 以使用默认值。

3. **创建变更目录**
   ```bash
   openspec new change "<name>"
   ```
   仅当用户请求了特定工作流时才添加 `--schema <name>`。
   这会在 `openspec/changes/<name>/` 创建一个带有所选 schema 的脚手架变更。

4. **显示产物状态**
   ```bash
   openspec status --change "<name>"
   ```
   这显示哪些产物需要创建以及哪些已就绪（依赖已满足）。

5. **获取第一个产物的说明**
   第一个产物取决于 schema。检查状态输出以找到状态为"ready"的第一个产物。
   ```bash
   openspec instructions <first-artifact-id> --change "<name>"
   ```
   输出创建第一个产物的模板和上下文。

6. **停止并等待用户指示**

**输出**

完成步骤后，总结：
- 变更名称和位置
- 正在使用的 Schema/工作流及其产物序列
- 当前状态（0/N 个产物已完成）
- 第一个产物的模板
- 提示："准备好创建第一个产物了吗？运行 `/opsx:continue` 或者直接描述这个变更的内容，我来起草。"

**约束**
- 现在还不要创建任何产物 — 仅显示说明
- 不要超出显示第一个产物模板的范围
- 如果名称无效（不是 kebab-case），要求提供有效名称
- 如果该名称的变更已存在，建议改用 `/opsx:continue`
- 如果使用非默认工作流，传递 --schema
