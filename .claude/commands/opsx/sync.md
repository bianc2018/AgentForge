---
name: "OPSX: Sync"
description: 将变更中的 delta specs 同步到主 specs
category: Workflow
tags: [workflow, specs, experimental]
---

将变更中的 delta specs 同步到主 specs。

这是一个 **agent 驱动**的操作 — 你将读取 delta specs 并直接编辑主 specs 以应用更改。这允许智能合并（例如，添加一个场景而不复制整个需求）。

**输入**：可选地在 `/opsx:sync` 之后指定变更名称（如：`/opsx:sync add-auth`）。如果省略，检查是否可以从会话上下文中推断。如果模糊或歧义，你必须提示用户可用的变更。

**步骤**

1. **如果未提供变更名称，提示用户选择**

   运行 `openspec list --json` 获取可用变更。使用 **AskUserQuestion 工具**让用户选择。

   显示有 delta specs（在 `specs/` 目录下）的变更。

   **重要**：不要猜测或自动选择变更。始终让用户选择。

2. **找到 delta specs**

   在 `openspec/changes/<name>/specs/*/spec.md` 中查找 delta spec 文件。

   每个 delta spec 文件包含如下 section：
   - `## ADDED Requirements` - 要添加的新需求
   - `## MODIFIED Requirements` - 对现有需求的更改
   - `## REMOVED Requirements` - 要删除的需求
   - `## RENAMED Requirements` - 要重命名的需求（FROM:/TO: 格式）

   如果未找到 delta specs，通知用户并停止。

3. **对每个 delta spec，将更改应用到主 specs**

   对于在 `openspec/changes/<name>/specs/<capability>/spec.md` 有 delta spec 的每个 capability：

   a. **读取 delta spec** 以了解预期的更改

   b. **读取主 spec** 在 `openspec/specs/<capability>/spec.md`（可能尚不存在）

   c. **智能应用更改**：

      **ADDED Requirements：**
      - 如果需求在主 spec 中不存在 → 添加它
      - 如果需求已存在 → 更新它以匹配（视为隐式 MODIFIED）

      **MODIFIED Requirements：**
      - 在主 spec 中找到该需求
      - 应用更改 — 可以是：
        - 添加新场景（不需要复制已有场景）
        - 修改现有场景
        - 更改需求描述
      - 保留 delta 中未提及的场景/内容

      **REMOVED Requirements：**
      - 从主 spec 中删除整个需求块

      **RENAMED Requirements：**
      - 找到 FROM 需求，重命名为 TO

   d. **如果 capability 尚不存在则创建新的主 spec**：
      - 创建 `openspec/specs/<capability>/spec.md`
      - 添加 Purpose section（可以简短，标记为 TBD）
      - 添加包含 ADDED 需求的 Requirements section

4. **显示摘要**

   应用所有更改后，总结：
   - 更新了哪些 capabilities
   - 做出了哪些更改（需求已添加/修改/删除/重命名）

**Delta Spec 格式参考**

```markdown
## ADDED Requirements

### Requirement: New Feature
系统 SHALL 做某些新的事情。

#### Scenario: 基本案例
- **WHEN** 用户执行 X
- **THEN** 系统执行 Y

## MODIFIED Requirements

### Requirement: Existing Feature
#### Scenario: 要添加的新场景
- **WHEN** 用户执行 A
- **THEN** 系统执行 B

## REMOVED Requirements

### Requirement: Deprecated Feature

## RENAMED Requirements

- FROM: `### Requirement: Old Name`
- TO: `### Requirement: New Name`
```

**关键原则：智能合并**

与程序化合并不同，你可以应用**部分更新**：
- 要添加一个场景，只需将该场景包含在 MODIFIED 下 — 不要复制现有场景
- Delta 代表*意图*，而非整体替换
- 使用你的判断来合理地合并更改

**成功时输出**

```
## Specs 已同步：<change-name>

已更新主 specs：

**<capability-1>**：
- 已添加需求："New Feature"
- 已修改需求："Existing Feature"（添加了 1 个场景）

**<capability-2>**：
- 已创建新的 spec 文件
- 已添加需求："Another Feature"

主 specs 已更新。变更保持活跃 — 在实现完成后归档。
```

**约束**
- 在做出更改之前读取 delta 和主 specs
- 保留 delta 中未提及的现有内容
- 如果某事不清楚，要求澄清
- 在进行更改时显示你在做什么
- 操作应该是幂等的 — 运行两次应给出相同结果
