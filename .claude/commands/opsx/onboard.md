---
name: "OPSX: Onboard"
description: 引导式入门 — 通过完整的 OpenSpec 工作流循环，带讲解和真实代码库工作
category: Workflow
tags: [workflow, onboarding, tutorial, learning]
---

引导用户完成他们的第一个完整 OpenSpec 工作流循环。这是一次教学体验 — 你将在他们的代码库中做真实的工作，同时解释每一步。

---

## 预检

开始之前，检查 OpenSpec CLI 是否已安装：

```bash
# Unix/macOS
openspec --version 2>&1 || echo "CLI_NOT_INSTALLED"
# Windows (PowerShell)
# if (Get-Command openspec -ErrorAction SilentlyContinue) { openspec --version } else { echo "CLI_NOT_INSTALLED" }
```

**如果 CLI 未安装：**
> OpenSpec CLI 未安装。请先安装它，然后返回 `/opsx:onboard`。

如果未安装，在此停止。

---

## 阶段 1：欢迎

显示：

```
## 欢迎来到 OpenSpec！

我将带你完成一个完整的变更循环 — 从想法到实现 — 使用你代码库中的真实任务。在此过程中，你将通过实践学习工作流。

**我们将要做的：**
1. 在你的代码库中挑选一个小而真实的任务
2. 简要探索问题
3. 创建一个变更（我们工作的容器）
4. 构建产物：proposal → specs → design → tasks
5. 实现任务
6. 归档已完成的变更

**时间：** 约 15-20 分钟

让我们先找点事情做。
```

---

## 阶段 2：任务选择

### 代码库分析

扫描代码库寻找小的改进机会。寻找：

1. **TODO/FIXME 注释** - 在代码文件中搜索 `TODO`、`FIXME`、`HACK`、`XXX`
2. **缺失的错误处理** - 吞没错误的 `catch` 块、没有 try-catch 的风险操作
3. **没有测试的函数** - 交叉引用 `src/` 与测试目录
4. **类型问题** - TypeScript 文件中的 `any` 类型（`: any`、`as any`）
5. **调试产物** - 非调试代码中的 `console.log`、`console.debug`、`debugger` 语句
6. **缺失的验证** - 没有验证的用户输入处理器

同时检查最近的 git 活动：
```bash
# Unix/macOS
git log --oneline -10 2>/dev/null || echo "No git history"
# Windows (PowerShell)
# git log --oneline -10 2>$null; if ($LASTEXITCODE -ne 0) { echo "No git history" }
```

### 呈现建议

从你的分析中，呈现 3-4 个具体建议：

```
## 任务建议

基于对你的代码库的扫描，以下是一些好的入门任务：

**1. [最有希望的任务]**
   位置：`src/path/to/file.ts:42`
   范围：约 1-2 个文件，约 20-30 行
   为什么好：[简要理由]

**2. [第二个任务]**
   位置：`src/another/file.ts`
   范围：约 1 个文件，约 15 行
   为什么好：[简要理由]

**3. [第三个任务]**
   位置：[位置]
   范围：[估计]
   为什么好：[简要理由]

**4. 其他？**
   告诉我你想做什么。

你对哪个任务感兴趣？（选择一个编号或描述你自己的）
```

**如果什么都没找到：** 退回到询问用户想构建什么：
> 我在你的代码库中没有找到明显的快速改进机会。你有没有一直想做或修复的小事情？

### 范围约束

如果用户选择或描述了太大的内容（主要功能、多日工作）：

```
那是个有价值的任务，但对于你第一次 OpenSpec 演练来说可能太大了。

对于学习工作流，越小越好 — 它能让你看到完整的循环而不会陷入实现细节。

**选项：**
1. **切分得更小** - [他们的任务] 中最小的有用部分是什么？也许只是 [具体的切分]？
2. **选择其他的** - 其他建议中的一个，或另一个小任务？
3. **无论如何都做** - 如果你真的想处理这个，我们可以。只是会花更长时间。

你更倾向于哪个？
```

让用户覆盖如果坚持 — 这是软约束。

---

## 阶段 3：探索演示

一旦选定了任务，简要演示探索模式：

```
在我们创建变更之前，让我快速向你展示**探索模式** — 它是你在确定方向之前思考问题的方式。
```

花 1-2 分钟调查相关代码：
- 读取涉及的文件
- 如果有帮助，画一个快速的 ASCII 图
- 记录任何考量

```
## 快速探索

[你的简要分析 — 你发现了什么，任何考量]

┌─────────────────────────────────────────┐
│   [可选：如果有帮助的 ASCII 图]          │
└─────────────────────────────────────────┘

探索模式（`/opsx:explore`）就是用于这种思考 — 在实现之前进行调查。你可以在任何时候用它来思考问题。

现在让我们创建一个变更来容纳我们的工作。
```

**暂停** - 等待用户确认后再继续。

---

## 阶段 4：创建变更

**解释：**
```
## 创建变更

OpenSpec 中的"变更"是围绕一项工作的所有思考和规划的容器。它位于 `openspec/changes/<name>/` 并保存你的产物 — proposal、specs、design、tasks。

让我为我们的任务创建一个。
```

**执行：** 使用推导的 kebab-case 名称创建变更：
```bash
openspec new change "<derived-name>"
```

**展示：**
```
已创建：`openspec/changes/<name>/`

文件夹结构：
```
openspec/changes/<name>/
├── proposal.md    ← 我们为什么要做这个（空的，我们将填充）
├── design.md      ← 我们将如何构建它（空的）
├── specs/         ← 详细需求（空的）
└── tasks.md       ← 实现清单（空的）
```

现在让我们填充第一个产物 — proposal。
```

---

## 阶段 5：Proposal

**解释：**
```
## Proposal

Proposal 记录了**为什么**要做这个更改以及它**在高层面上涉及什么**。它是工作的"电梯演讲"。

我将基于我们的任务起草一个。
```

**执行：** 起草 proposal 内容（先不保存）：

```
这是一个草案 proposal：

---

## Why

[1-2 句话解释问题/机会]

## What Changes

[将要改变什么的 bullet points]

## Capabilities

### New Capabilities
- `<capability-name>`：[简要描述]

### Modified Capabilities
<!-- 如果修改现有行为 -->

## Impact

- `src/path/to/file.ts`：[改变了什么]
- [其他文件，如适用]

---

这抓住了意图吗？我可以在保存之前调整。
```

**暂停** - 等待用户批准/反馈。

批准后，保存 proposal：
```bash
openspec instructions proposal --change "<name>" --json
```
然后将内容写入 `openspec/changes/<name>/proposal.md`。

```
Proposal 已保存。这是你的"为什么"文档 — 随着理解的深入，随时可以回来完善。

下一步：specs。
```

---

## 阶段 6：Specs

**解释：**
```
## Specs

Specs 以精确、可测试的术语定义**我们正在构建什么**。它们使用需求/场景格式，使预期行为变得清晰。

对于像这样的小任务，我们可能只需要一个 spec 文件。
```

**执行：** 创建 spec 文件：
```bash
# Unix/macOS
mkdir -p openspec/changes/<name>/specs/<capability-name>
# Windows (PowerShell)
# New-Item -ItemType Directory -Force -Path "openspec/changes/<name>/specs/<capability-name>"
```

起草 spec 内容：

```
这是 spec：

---

## ADDED Requirements

### Requirement: <Name>

<描述系统应该做什么>

#### Scenario: <场景名称>

- **WHEN** <触发条件>
- **THEN** <预期结果>
- **AND** <附加结果，如需要>

---

这种 WHEN/THEN/AND 格式使需求可测试。你实际上可以把它们当作测试用例来阅读。
```

保存到 `openspec/changes/<name>/specs/<capability>/spec.md`。

---

## 阶段 7：Design

**解释：**
```
## Design

Design 记录了**我们如何构建它** — 技术决策、权衡、方法。

对于小变更，这可能很简短。这完全没问题 — 并非每个变更都需要深入的设计讨论。
```

**执行：** 起草 design.md：

```
这是 design：

---

## Context

[关于当前状态的简要上下文]

## Goals / Non-Goals

**Goals：**
- [我们试图实现什么]

**Non-Goals：**
- [明确不在范围内什么]

## Decisions

### Decision 1：[关键决策]

[方法和理由的解释]

---

对于一个小组件，这捕捉了关键决策而不过度工程化。
```

保存到 `openspec/changes/<name>/design.md`。

---

## 阶段 8：Tasks

**解释：**
```
## Tasks

最后，我们将工作分解为实现任务 — 驱动 apply 阶段的复选框。

这些应该小、清晰，并按逻辑顺序排列。
```

**执行：** 基于 specs 和 design 生成任务：

```
这是实现任务：

---

## 1. [类别或文件]

- [ ] 1.1 [具体任务]
- [ ] 1.2 [具体任务]

## 2. 验证

- [ ] 2.1 [验证步骤]

---

每个复选框成为 apply 阶段的一个工作单元。准备好实现了吗？
```

**暂停** - 等待用户确认他们准备好实现。

保存到 `openspec/changes/<name>/tasks.md`。

---

## 阶段 9：Apply（实现）

**解释：**
```
## 实现

现在我们实现每个任务，逐项核对。我将宣布每个任务，并偶尔提及 specs/design 如何影响了方法。
```

**执行：** 对每个任务：

1. 宣布："正在处理任务 N：[描述]"
2. 在代码库中实现更改
3. 自然地引用 specs/design："Spec 说 X，所以我这样实现 Y"
4. 在 tasks.md 中标记完成：`- [ ]` → `- [x]`
5. 简要状态："✓ 任务 N 完成"

保持叙述轻量 — 不要过度解释每一行代码。

所有任务完成后：

```
## 实现完成

所有任务已完成：
- [x] 任务 1
- [x] 任务 2
- [x] ...

变更已实现！还有一步 — 让我们归档它。
```

---

## 阶段 10：归档

**解释：**
```
## 归档

当变更完成时，我们归档它。这会将其从 `openspec/changes/` 移动到 `openspec/changes/archive/YYYY-MM-DD-<name>/`。

已归档的变更成为你项目的决策历史 — 你以后总能找到它们来理解为什么某件事被以特定方式构建。
```

**执行：**
```bash
openspec archive "<name>"
```

**展示：**
```
已归档到：`openspec/changes/archive/YYYY-MM-DD-<name>/`

变更现在是你项目历史的一部分。代码在你的代码库中，决策记录被保留下来。
```

---

## 阶段 11：回顾与后续步骤

```
## 恭喜！

你刚刚完成了一个完整的 OpenSpec 循环：

1. **Explore** - 思考了问题
2. **New** - 创建了变更容器
3. **Proposal** - 记录了 WHY
4. **Specs** - 详细定义了 WHAT
5. **Design** - 决定了 HOW
6. **Tasks** - 分解为步骤
7. **Apply** - 实现了工作
8. **Archive** - 保留了记录

这个节奏适用于任何规模的变更 — 小修复或大功能。

---

## 命令参考

| 命令 | 功能 |
|---------|--------------|
| `/opsx:explore` | 在工作前/工作中思考问题 |
| `/opsx:new` | 开始新变更，逐步创建产物 |
| `/opsx:ff` | 快速推进：一次性创建所有产物 |
| `/opsx:continue` | 继续处理现有变更 |
| `/opsx:apply` | 实现变更中的任务 |
| `/opsx:verify` | 验证实现是否与产物匹配 |
| `/opsx:archive` | 归档已完成的变更 |

---

## 下一步？

在你真正想构建的东西上尝试 `/opsx:new` 或 `/opsx:ff`。你现在已经掌握了节奏！
```

---

## 优雅退出处理

### 用户想中途停止

如果用户说他们需要停止、想暂停或显得不投入：

```
没问题！你的变更已保存在 `openspec/changes/<name>/`。

稍后继续：
- `/opsx:continue <name>` - 继续产物创建
- `/opsx:apply <name>` - 跳到实现（如果存在任务）

工作不会丢失。准备好了随时回来。
```

优雅退出，不施压。

### 用户只想要命令参考

如果用户说他们只想看命令或跳过教程：

```
## OpenSpec 快速参考

| 命令 | 功能 |
|---------|--------------|
| `/opsx:explore` | 思考问题（不更改代码） |
| `/opsx:new <name>` | 开始新变更，逐步进行 |
| `/opsx:ff <name>` | 快速推进：一次性创建所有产物 |
| `/opsx:continue <name>` | 继续现有变更 |
| `/opsx:apply <name>` | 实现任务 |
| `/opsx:verify <name>` | 验证实现 |
| `/opsx:archive <name>` | 完成后归档 |

尝试 `/opsx:new` 开始你的第一个变更，或 `/opsx:ff` 如果你想快速推进。
```

优雅退出。

---

## 约束

- **在关键过渡点遵循 EXPLAIN → DO → SHOW → PAUSE 模式**（探索之后、proposal 草案之后、tasks 之后、归档之后）
- **保持叙述轻量** — 教学但不说教
- **不要跳阶段**，即使变更很小 — 目标是教授工作流
- **在标记点暂停以确认**，但不要过度暂停
- **优雅处理退出** — 绝不对用户施压继续
- **使用真实代码库任务** — 不要模拟或使用假示例
- **温和调整范围** — 引导向更小的任务但尊重用户选择
