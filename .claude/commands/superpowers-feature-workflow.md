<!-- generatedBy: sot@2.5.7 -->
---
name: superpowers-feature-workflow
description: 当 feature 工作在实现之前或期间需要 Superpowers 阶段时使用：brainstorming、设计确认、实现规划、worktree 设置、测试驱动开发和验证。当用户要求先 brainstorm、希望在编码前有计划、或希望使用 TDD 和验证进行规范执行时触发。
model_hint: sonnet
---


# Superpowers Feature Workflow

## 概述

使用此 skill 处理 feature 交付中的 Superpowers 部分。它涵盖需求澄清、设计、计划、worktree、TDD、验证和仓库持久化内存，但不管理 OpenSpec 产物。

这是一个明确的 opt-in 工作流。默认情况下不要使用。仅在用户明确要求此工作流、命名了此 skill、或仓库策略明确要求时使用。

## 工作流

1. 在提出解决方案之前探索项目上下文。如果存在 `.superpowers-memory/`，首先读取 `PROJECT_CONTEXT.md`、`CURRENT_STATE.md`、`DECISIONS.md`、`KNOWN_FAILURES.md`、`VERIFICATION_BASELINE.md`、`TEAM_PREFERENCES.md`、`USER_PROFILE.md`、`AGENT_NOTES.md` 以及最新的会话日志条目。
2. 一次一个问题地澄清需求。
3. 提出 2-3 种方案并给出推荐。
4. 将批准的设计写入 `docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md`。
5. 在继续之前，请求用户对书面设计进行确认。
6. 将实现计划写入 `docs/superpowers/plans/YYYY-MM-DD-<topic>.md`。
7. 优先使用仓库本地的 worktree 进行实现。
8. 使用 TDD 实现：先编写失败的测试，然后是最小代码，最后变绿。
9. 在任何完成声明之前，运行全新的验证命令。
10. 在结束会话之前，运行一次快速的内存结算检查：持久性事实是否变更、当前状态是否变更、重要决策是否添加、失败模式是否发现、验证规则是否变更、以及可复用经验是否识别。
11. 当您希望通过一个命令来审查检查清单、获取更新建议并可选地运行验证时，优先使用 `scripts/run-superpowers-memory-closeout.ps1`。
12. 当实现或验证后正确的内存面仍不明确时，使用 `scripts/suggest-superpowers-memory-updates.ps1`。
13. 当仓库使用 Superpowers memory 时，更新相关的内存文件，包括 `.superpowers-memory/CURRENT_STATE.md` 和 `.superpowers-memory/session-journal/` 下的简短会话记录。
14. 当内存质量对任务很重要时，在最终完成声明之前运行 `scripts/validate-superpowers-memory.ps1`。

## 何时使用

- 用户明确要求先 brainstorm
- 用户明确命名了 `$superpowers-feature-workflow`
- 用户明确要求书面计划、TDD 或验证工作流
- 仓库策略明确要求此工作流

## 输出

- 已确认的设计文档
- 实现计划
- 已验证的实现证据
- 当存在 `.superpowers-memory/` 时，更新的 Superpowers memory
- 当使用了结算辅助工具时，可选的结算辅助输出
- 当内存更新是工作流的一部分时，内存验证证据

## OMC 团队加速（可选）

当安装了 oh-my-claudecode 并且任务足够大可以从并行代理中受益时，您可以使用 OMC 的 `/team` 来加速执行。这是可选的 — 顺序单代理执行始终有效。

**何时考虑团队：**
- 实现计划有 3 个以上独立任务
- 验证可以与下一个实现任务并行运行
- 用户明确要求并行或基于团队的执行

**如何使用：**
1. 在计划批准后（步骤 6），使用 `TeamCreate` 建立一个团队
2. 通过 `TaskCreate` 将计划分解为具有依赖链的任务
3. 通过 `Agent` 工具使用 `team_name` 生成团队成员 — 使用诸如 `executor`、`verifier`、`critic` 等角色
4. 团队负责人（您）进行协调；团队成员执行任务并通过 `SendMessage` 报告
5. 在所有任务完成后，使用 `TeamDelete` 进行清理

**此工作流的推荐团队模式：**
- **步骤 1-6（顺序）：** 澄清、设计、计划 — 单个 agent，需要判断力
- **步骤 7-8（并行）：** 实现任务 — 多个 `executor` agent 处理独立任务
- **步骤 9（并行）：** `verifier` agent 运行验证，同时 `executor` 处理下一个任务
- **步骤 10-14（顺序）：** 内存结算 — 单个 agent，需要完整上下文

如果未安装 OMC 或任务较小，则按上述顺序进行。

## 约束

- 在设计批准之前不要编写生产代码
- 对于新行为，不要跳过失败测试的步骤
- 没有全新的命令输出时，不要报告成功
- 不要用临时记录覆盖稳定的项目内存；将长期事实保存在 `PROJECT_CONTEXT.md` 中，将会话特定的更新保存在 journal 中
- 当仓库使用 Superpowers memory 时，不要将新的持久性决策、已知失败或验证规则仅留在聊天历史中

<!-- checksum: sha256:d63a2c615bc39b98f861ed589a3b00d63bad01dc2abb7583a0c8d143fe916d1e -->
