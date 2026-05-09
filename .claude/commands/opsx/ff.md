---
name: "OPSX: Fast Forward"
description: 创建一个变更并一次性生成实现所需的所有产物
category: Workflow
tags: [workflow, artifacts, experimental]
---

快速推进产物创建 — 生成开始实现所需的一切。

**输入**：`/opsx:ff` 后面的参数是变更名称（kebab-case），或者是用户想要构建什么的描述。

**步骤**

1. **如果未提供输入，询问他们想构建什么**

   使用 **AskUserQuestion 工具**（开放式，无预设选项）提问：
   > "你想做什么变更？描述一下你想构建或修复的内容。"

   从他们的描述中推导出 kebab-case 名称（例如，"添加用户认证" → `add-user-auth`）。

   **重要**：在理解用户想要构建什么之前，不要继续。

2. **创建变更目录**
   ```bash
   openspec new change "<name>"
   ```
   这会在 `openspec/changes/<name>/` 创建一个脚手架变更。

3. **获取产物构建顺序**
   ```bash
   openspec status --change "<name>" --json
   ```
   解析 JSON 以获取：
   - `applyRequires`：实现之前需要的产物 ID 数组（如：`["tasks"]`）
   - `artifacts`：所有产物及其状态和依赖的列表

4. **按顺序创建产物直到 apply 就绪**

   使用 **TodoWrite 工具**跟踪产物进度。

   按依赖顺序循环处理产物（没有待处理依赖的产物优先）：

   a. **对于每个状态为 `ready`（依赖已满足）的产物**：
      - 获取说明：
        ```bash
        openspec instructions <artifact-id> --change "<name>" --json
        ```
      - 说明 JSON 包含：
        - `context`：项目背景（给你的约束 — 不要包含在输出中）
        - `rules`：产物特定的规则（给你的约束 — 不要包含在输出中）
        - `template`：输出文件使用的结构
        - `instruction`：此产物类型的 schema 特定指导
        - `outputPath`：产物写入位置
        - `dependencies`：需要读取上下文的已完成产物
      - 读取所有已完成的依赖文件以获取上下文
      - 使用 `template` 作为结构创建产物文件
      - 应用 `context` 和 `rules` 作为约束 — 但不要将它们复制到文件中
      - 显示简要进度："✓ 已创建 <artifact-id>"

   b. **继续直到所有 `applyRequires` 产物完成**
      - 创建每个产物后，重新运行 `openspec status --change "<name>" --json`
      - 检查 `applyRequires` 中的每个产物 ID 在产物数组中是否都有 `status: "done"`
      - 当所有 `applyRequires` 产物都完成时停止

   c. **如果产物需要用户输入**（上下文不清楚）：
      - 使用 **AskUserQuestion 工具**澄清
      - 然后继续创建

5. **显示最终状态**
   ```bash
   openspec status --change "<name>"
   ```

**输出**

完成所有产物后，总结：
- 变更名称和位置
- 已创建的产物列表及简要描述
- 就绪状态："所有产物已创建！准备就绪可以实现。"
- 提示："运行 `/opsx:apply` 开始实现。"

**产物创建指南**

- 遵循 `openspec instructions` 中每个产物类型的 `instruction` 字段
- Schema 定义了每个产物应包含的内容 — 遵循它
- 创建新产物之前，读取依赖产物以获取上下文
- 使用 `template` 作为起点，根据上下文填充

**约束**
- 创建实现所需的所有产物（由 schema 的 `apply.requires` 定义）
- 创建新产物之前始终读取依赖产物
- 如果上下文严重不明确，询问用户 — 但优先做出合理决策以保持动力
- 如果该名称的变更已存在，询问用户是想继续还是创建新的
- 在进入下一个之前，写入后验证每个产物文件是否存在
