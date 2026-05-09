<!-- generatedBy: sot@2.5.7 -->
---
name: superpowers-learning-workflow
description: 当用户明确希望从已完成的工作中捕获经验教训、持久化项目知识、或将重复模式转化为可供未来会话复用的学习笔记时使用。
model_hint: haiku
---


# Superpowers Learning Workflow

## 概述

在有意义的工作之后使用此工作流，以捕获应跨越当前会话保留的内容。它是一个轻量级的、仓库自有的学习循环，受反思型 agent 系统启发，但限定在正常项目工作流中安全使用。

这是一个明确的 opt-in 工作流。默认情况下不要使用。仅在用户明确要求此工作流、命名了此 skill、或仓库策略明确要求时使用。

## 工作流

1. 回顾最近的工作、决策和验证证据。
2. 将学到的东西分为四类：
   - 持久的项目事实
   - 当前工作状态
   - 会话结果
   - 可复用方法或重复陷阱
3. 为持久条目添加必需的元数据：
   - `id`
   - `status`
   - `confidence`
   - `source`
   - `last_updated`
   - `review_after`
   如果 `source` 为空，不要将条目标记为 `verified`。
4. 如果存在 `.superpowers-memory/`，更新：
   - `PROJECT_CONTEXT.md` 用于持久事实
   - `CURRENT_STATE.md` 用于活动状态
   - `DECISIONS.md` 用于持久决策
   - `KNOWN_FAILURES.md` 用于重复失败模式
   - `VERIFICATION_BASELINE.md` 用于可信的验证规则
   - `TEAM_PREFERENCES.md` 用于持久的团队约定
   - `USER_PROFILE.md` 用于非项目事实的持久用户偏好
   - `AGENT_NOTES.md` 用于非项目事实的持久执行提醒
   - `session-journal/` 用于会话摘要
   - `LEARNING_BACKLOG.md` 用于可能值得未来工作流或 skill 的复用模式
5. 如果不存在 `.superpowers-memory/`，告知用户安装内存脚手架或将学习摘要保存在普通项目文档中。
6. 检查是否有任何 backlog 项足够强，可以推荐升级为检查清单、项目规则、工作流步骤、脚本或 skill 草稿。
7. 在完成学习捕获之前，审查 `.superpowers-memory/SESSION_CLOSE_CHECKLIST.md`。
8. 如果不清楚当前会话信号应更新哪些内存面，使用 `scripts/suggest-superpowers-memory-updates.ps1`。
9. 当您希望通过一个命令来审查检查清单、获取更新建议并可选地运行验证时，优先使用 `scripts/run-superpowers-memory-closeout.ps1` 作为标准结算辅助工具。
10. 当内存文件已更新时，运行 `scripts/validate-superpowers-memory.ps1` 并将结果包含在摘要中。
11. 当需要确认某个模式是否已存在于持久内存或近期日志中时，使用 `scripts/search-superpowers-memory.ps1`。
12. 总结学到了什么，以及什么（如果有的话）应该成为未来的规则、检查清单、脚本或 skill。

## 何时使用

- 用户明确要求从当前会话中捕获经验教训
- 用户明确命名了 `$superpowers-learning-workflow`
- 用户希望为未来会话持久化知识
- 用户希望将重复模式转化为可复用的学习笔记
- 仓库策略明确要求在有意义的工作后进行反思性捕获

## 输出

- 当持久事实变更时，更新 `.superpowers-memory/PROJECT_CONTEXT.md`
- 更新 `.superpowers-memory/CURRENT_STATE.md`
- 当持久决策变更时，更新 `.superpowers-memory/DECISIONS.md`
- 当识别到重复失败模式时，更新 `.superpowers-memory/KNOWN_FAILURES.md`
- 当可信验证规则变更时，更新 `.superpowers-memory/VERIFICATION_BASELINE.md`
- 当持久团队约定变更时，更新 `.superpowers-memory/TEAM_PREFERENCES.md`
- 当持久用户偏好变更时，更新 `.superpowers-memory/USER_PROFILE.md`
- 当持久执行提醒变更时，更新 `.superpowers-memory/AGENT_NOTES.md`
- 新的或更新的会话日志条目
- 更新 `.superpowers-memory/LEARNING_BACKLOG.md` 用于可复用的经验教训
- 更新 `.superpowers-memory/SESSION_CLOSE_CHECKLIST.md`，仅作为参考检查清单，不作为会话日志
- 当内存更新时，内存验证证据
- 当使用了建议脚本时，可选的内存更新建议证据
- 当使用了结算脚本时，可选的结算辅助工具输出
- 关于下次应记住内容的简短摘要

## 约束

- 不要将临时的 TODO 噪声写入 `PROJECT_CONTEXT.md`
- 如果没有明确的重复模式，不要将一次性修复转变为可复用规则
- 除非用户明确要求该单独步骤，否则不要自动编辑 skill 库本身
- 保持学习笔记简洁且可操作
- 没有足够的重复证据或跨会话价值时，不要提升 backlog 项
- 将 `ready_for_promotion` 视为更高标准：需要重复证据、关联来源和可审查的升级理由

<!-- checksum: sha256:b8235d5e2b75047b1177e33d195041f76152904a16280b77ab5f29d0df21538a -->
