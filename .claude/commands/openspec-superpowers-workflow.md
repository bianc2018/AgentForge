<!-- generatedBy: sot@2.5.7 -->
---
name: openspec-superpowers-workflow
description: 当用户明确需要完整的 OpenSpec 加 Superpowers 路径，从需求澄清到 proposal、design、tasks、实现、验证以及可选的归档时使用。
model_hint: sonnet
---


# OpenSpec + Superpowers Workflow

## 概述

使用此 skill 作为 feature 交付的团队入口。它协调工作顺序；不替代详细的 OpenSpec 或 Superpowers 子 skill。

这是一个明确的 opt-in 工作流。默认情况下不要使用。仅在用户明确要求此工作流、命名了此 skill、或仓库策略明确要求时使用。

如果仓库中存在 `.superpowers-memory/`，将其视为共享项目内存：在规划前读取，在关闭工作流前更新。

## 必需顺序

1. 运行 `$superpowers-feature-workflow` 来澄清需求、比较方案、确认设计并准备实现。
2. 运行 `$openspec-feature-workflow` 来创建变更并完成 `proposal`、`design`、`specs` 和 `tasks`。
3. 返回 Superpowers 轨道进行计划执行、worktree 设置、TDD 和验证。
4. 如果项目使用 OpenSpec 归档流程并且代码、specs 和验证一致，则将变更归档作为 OpenSpec 的最后一步。
5. 在存在验证证据之前，不要声称完成。
6. 如果存在 `.superpowers-memory/`，更新 `CURRENT_STATE.md` 并为会话结果添加简短的日志条目。

## 何时使用

- 用户明确要求 `OpenSpec + Superpowers`
- 用户明确命名了 `$openspec-superpowers-workflow`
- 用户明确要求先 brainstorm，然后是 proposal/design/tasks，再是实现，最后是验证
- 仓库策略明确要求此工作流

## 交付物

- `docs/superpowers/specs/` 中的设计文档
- `openspec/changes/<change-name>/` 中的 OpenSpec 变更产物
- `docs/superpowers/plans/` 中的实现计划
- 代码、测试和全新的验证输出
- 当归档流程是项目工作流一部分时，归档的 OpenSpec 变更
- 当存在 `.superpowers-memory/` 时，更新的 Superpowers memory

## OMC 团队加速（可选）

当安装了 oh-my-claudecode 并且任务足够大可以从并行代理中受益时，您可以使用 OMC 的 `/team` 来加速执行。这是可选的 — 顺序单代理执行始终有效。

**何时考虑团队：**
- feature 在规划后有 3 个以上独立的实现任务
- 验证和实现可以并行运行
- 用户明确要求并行或基于团队的执行

**如何使用：**
1. 在设计批准后（步骤 1），使用 `TeamCreate` 建立一个团队
2. 通过 `TaskCreate` 将实现计划分解为具有依赖链的任务
3. 通过 `Agent` 工具使用 `team_name` 生成团队成员 — 使用诸如 `executor`、`verifier`、`critic` 等角色
4. 团队负责人（您）进行协调；团队成员执行任务并通过 `SendMessage` 报告
5. 在所有任务完成后，使用 `TeamDelete` 进行清理

**此工作流的推荐团队模式：**
- **阶段 1（顺序）：** 设计 + OpenSpec 产物 — 单个 agent，需要判断力
- **阶段 2（并行）：** 实现任务 — 多个 `executor` agent 处理独立任务
- **阶段 3（并行）：** 验证 — `verifier` agent 检查 spec 合规性，同时 `executor` 修复问题

如果未安装 OMC 或任务较小，则按上述顺序进行。

## 约束

- 在设计批准之前不要开始实现
- 对于行为变更，不要跳过 OpenSpec 产物
- 在代码、测试和 specs 一致之前，不要归档变更
- 当请求包含 worktree、TDD 或验证时，不要跳过它们
- 保持 skill 的可移植性：使用仓库本地路径，避免机器特定的假设

<!-- checksum: sha256:db97d18209d2cb6efc847c3df79064fdf76cba6cb8a2db3ad83f1a3826ca9556 -->
