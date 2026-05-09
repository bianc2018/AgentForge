<!-- generatedBy: sot@2.5.7 -->
---
name: superpowers-openspec-execution-workflow
description: 当团队明确希望为某个 feature 使用 Superpowers 探索、OpenSpec 规范定义、Superpowers 执行和 OpenSpec 归档工作流时使用
model_hint: sonnet
---


# Superpowers -> OpenSpec -> Superpowers Workflow

## 概述

当团队希望采用以下四步交付路径时使用此 skill：

1. 使用 Superpowers 进行探索
2. 使用 OpenSpec 锁定变更
3. 使用 Superpowers 执行并以实现、test 和验证收尾
4. 归档完成的 OpenSpec 变更

此 skill 是一个编排器。它应将具体工作委托给现有的工作流 skill，而不是重复实现它们。

这是一个明确的 opt-in 工作流。默认情况下不要使用。仅在用户明确要求此工作流、命名了此 skill、或仓库策略明确要求时使用。

如果仓库中存在 `.superpowers-memory/`，在开始时读取 `PROJECT_CONTEXT.md`、`CURRENT_STATE.md`、`DECISIONS.md`、`KNOWN_FAILURES.md`、`VERIFICATION_BASELINE.md`、`TEAM_PREFERENCES.md`、`USER_PROFILE.md`、`AGENT_NOTES.md` 以及最新的会话日志条目，然后在最终归档前更新相关文件，以便下一个会话可以在真实上下文中恢复。

## 必需顺序

1. 从 `$superpowers-feature-workflow` 开始。
   使用它来澄清范围、比较方案、确认解决方案形态并捕获设计草案。
2. 转到 `$openspec-feature-workflow`。
   使用它来创建变更并完成 `proposal.md`、`design.md`、`specs/.../spec.md` 和 `tasks.md`。
3. 返回 `$superpowers-feature-workflow`。
   使用它来编写实现计划、优先使用 worktree、使用 TDD 执行并运行全新的验证。
4. 如果验证后实现与 specs 一致，使用 `$openspec-archive-change` 归档完成的变更。
5. 如果存在 `.superpowers-memory/`，在验证和归档决策后执行内存对齐检查：确保持久事实、当前状态、决策、失败模式和会话结果反映在正确的文件中。
6. 当您希望通过一个命令来审查检查清单、获取更新建议并在执行或归档工作后可选地运行验证时，优先使用 `scripts/run-superpowers-memory-closeout.ps1`。
7. 如果在实现、验证或归档工作后仍不清楚应更新哪些内存面，使用 `scripts/suggest-superpowers-memory-updates.ps1`。
8. 当内存质量对项目很重要时，在最终完成声明之前运行 `scripts/validate-superpowers-memory.ps1`。

## OMC 团队加速（可选）

当安装了 oh-my-claudecode 并且任务足够大可以从并行代理中受益时，您可以使用 OMC 的 `/team` 来加速执行。这是可选的 — 顺序单代理执行始终有效。

**何时考虑团队：**
- feature 在规划后有 3 个以上独立的实现任务
- 验证和实现可以并行运行
- 用户明确要求并行或基于团队的执行

**如何使用：**
1. 在设计锁定后（步骤 2），使用 `TeamCreate` 建立一个团队
2. 通过 `TaskCreate` 将实现计划分解为具有依赖链的任务
3. 通过 `Agent` 工具使用 `team_name` 生成团队成员 — 使用诸如 `executor`、`verifier`、`critic` 等角色
4. 团队负责人（您）进行协调；团队成员执行任务并通过 `SendMessage` 报告
5. 在所有任务完成后，使用 `TeamDelete` 进行清理

**此工作流的推荐团队模式：**
- **步骤 1-2（顺序）：** 探索 + OpenSpec — 单个 agent，需要判断力
- **步骤 3（并行）：** 实现任务 — 多个 `executor` agent 处理独立任务
- **步骤 3 验证（并行）：** `verifier` agent 检查 spec 合规性，同时 `executor` 修复问题
- **步骤 4（顺序）：** 归档 — 单个 agent，需要上下文

如果未安装 OMC 或任务较小，则按上述顺序进行。

## 决策关卡

- 在探索阶段不要创建实现代码。
- 在所需的 OpenSpec 产物完成之前，不要开始编码。
- 在存在全新的验证输出之前，不要声称成功。
- 在代码、test 和 specs 一致之前，不要归档变更。
- 当存在 `.superpowers-memory/` 时，不要使内存与最终的归档决策不同步。

## 何时使用

- 用户明确要求"先探索，再规范，后执行"
- 用户明确命名了 `$superpowers-openspec-execution-workflow`
- 用户明确要求 Superpowers 探索、OpenSpec 锁定、然后 Superpowers 执行和归档
- 仓库策略明确要求此工作流

## 交付物

- `docs/superpowers/specs/` 中的设计草案
- `openspec/changes/<change-name>/` 下的 OpenSpec 产物
- `docs/superpowers/plans/` 中的实现计划
- 代码、test 和全新的验证证据
- 当内存正在使用时，更新的 Superpowers memory 和内存验证证据
- 当使用了结算辅助工具时，可选的结算辅助工具输出
- 工作完成时归档的 OpenSpec 变更

## 推荐提示词

```text
Use $superpowers-openspec-execution-workflow for this feature: first explore with Superpowers, then lock the change with OpenSpec, then return to Superpowers for implementation, testing, verification, and archive.
```

<!-- checksum: sha256:5efd5e9125144007690fc9d145d2a53d6c05c61e36595a99b2aa4bcc13b7ab64 -->
