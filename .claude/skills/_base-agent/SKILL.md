---
name: _base-agent
description: >
    所有 SDD agent 的共享行为：slug 推导、
    前置条件检查模板、增量构建循环
    （宣布 → 草稿 → 呈现 → 评估 → 提问 → 保存）以及
    通用的访谈和文件规则。
---

# SDD Agent 基础行为

## 语言

读取 `docs/sdd/sdd-config.md` 并将 `language` 的值用于所有用户沟通和文档生成。如果文件不存在，使用项目上下文中最自然的语言。

绝不翻译 IT 技术术语 — 无论配置的语言如何，它们始终保持英文：API, backend, frontend, endpoint, deploy, branch, commit, pull request, merge, cache, token, bug, framework, pipeline, build, release, feature, sprint, backlog, mock, stub, refactor, hotfix, rollback, CI/CD, log, test, debug。

---

## Slug 推导

将收到的参数转换为 slug：
- 小写，空格和下划线 → 连字符，移除重音和特殊字符
- 示例：`"Cadastro de Usuário"` → `cadastro-de-usuario` | `"Login de entregador"` → `login-entregador`

---

## 前置条件检查模板

如果必需的产物不存在，以下列方式结束：
```
[<agent-名称>] 错误：在 docs/features/<slug>/<文件> 中未找到 <产物>。
在继续之前请执行 /create-<命令> <feature 名称>。
```

---

## 增量构建循环

对每个项目（section、Story、需求、屏幕等），按顺序执行循环：

**A. 宣布** — 显示 `[项目 X/N：<标题>]`。

**B. 草稿** — 从产物和已构建的内容中推导。不要询问可以推导的内容。

**C. 呈现**：
```
草稿：
---
<内容>
---
```

**D. 评估** — 检查参考 skill 的检查清单。识别最关键但仍然缺失的项目。

**E. 决定：**
- 所有项目已覆盖 → 转到 G
- 有项目缺失 → 转到 F

**F. 提问** — 通过 `AskUserQuestion` 就最关键的项目提一个问题。纳入回答并返回到 C。

**G. 定稿** — 使用 `Edit` 写入，宣布 `✅ 项目 X 已完成。`，前进到下一个。

---

## 收尾模板

完成所有项目后：
1. 使用 `Read` 读取最终文件
2. 检查一致性（矛盾、覆盖率、统一词汇）
3. 如有必要使用 `Edit` 修正并通知用户
4. 宣布：agent 名称、文件路径、指标（项目总数）和建议的后续步骤

---

## 通用访谈规则

- **一次一个问题。** 如果有两个项目缺失，选择最关键的。
- **进行上下文关联。** 不要问产物已经回答的问题。
- **接受"N/A"。** 如果不适用，记录并前进。
- **推导先于提问。** 问题是为了填补真正的空白，不是从零构建。
- 在用户每次回答后展示**更新后的草稿**。

---

## 通用文件规则

- 每个项目完成后**立即**写入 — 不要累积到最后一次性写入。
- 使用 `Edit` 向文件添加 sections/项目 — 绝不使用 `Write`（会覆盖已保存的内容）。
- 格式严格遵循 agent 参考 skill 的标准。

---

## Mermaid 图表

每当 section 涉及流程、组件间关系或状态转换时，包含适当的 Mermaid 图表：

- 控制或用户流程 → `flowchart LR` 或 `flowchart TD`
- 组件间调用序列 → `sequenceDiagram`
- 数据实体间关系 → `erDiagram`
- 屏幕状态转换 → `stateDiagram-v2`
- tasks 或组件间的依赖 → `flowchart TD`

**规则：** 图表补充文本 — 不重复。如果文本已经自解释且图表不会增加理解，则省略。
