<!-- generatedBy: sot@2.5.7 -->
---
name: superpowers-learning-workflow
description: "当用户明确希望从已完成的工作中捕获经验教训、持久化项目知识、或将重复模式转化为可供未来会话复用的学习笔记时使用。"
model_hint: haiku
tags:
  - learning
  - memory
  - standalone
category: learning
---


# Superpowers Learning Workflow

## 概述

在重要工作完成后使用此工作流来捕获应当保留到当前会话之后的内容。它是一个轻量级的、仓库自有的学习循环，受反思式 agent 系统启发，但范围限定在正常项目工作流中的安全使用。

这是一个明确的 opt-in 工作流。默认情况下不要使用。仅在用户明确要求此工作流、命名了此 skill、或仓库策略明确要求时使用。

## 工作流

1. 审查最近的工作、决策和验证证据。
2. 将所学内容分为四类：
   - 持久性项目事实
   - 当前工作状态
   - 会话结果
   - 可复用的方法或重复出现的陷阱
3. 为持久性条目添加所需的元数据：
   - `id`
   - `status`
   - `confidence`
   - `source`
   - `last_updated`
   - `review_after`
   如果 `source` 为空，不要将条目标记为 `verified`。
4. 如果 `.superpowers-memory/` 存在，更新：
   - `PROJECT_CONTEXT.md` — 持久性事实
   - `CURRENT_STATE.md` — 当前状态
   - `DECISIONS.md` — 持久性决策
   - `KNOWN_FAILURES.md` — 重复的失败模式
   - `VERIFICATION_BASELINE.md` — 可信的验证规则
   - `TEAM_PREFERENCES.md` — 持久性团队约定
   - `USER_PROFILE.md` — 非项目事实的持久性用户偏好
   - `AGENT_NOTES.md` — 非项目事实的持久性执行提醒
   - `session-journal/` — 会话摘要
   - `LEARNING_BACKLOG.md` — 可能值得未来工作流或 skill 的可复用模式
5. 如果 `.superpowers-memory/` 不存在，告诉用户安装内存脚手架或将学习摘要保存在正常的项目文档中。
6. 检查是否有 backlog 条目足够强以推荐升级为清单、项目规则、工作流步骤、脚本或 skill 草案。
7. 在完成学习捕获之前审查 `.superpowers-memory/SESSION_CLOSE_CHECKLIST.md`。
8. 如果从当前会话信号中不清楚应更新哪些内存面，使用 `scripts/suggest-superpowers-memory-updates.ps1`。
9. 当想用一个命令来审查清单、获取更新建议并可选地运行验证时，优先使用 `scripts/run-superpowers-memory-closeout.ps1` 作为标准结算辅助工具。
10. 当内存文件被更新后，运行 `scripts/validate-superpowers-memory.ps1` 并将结果包含在摘要中。
11. 当需要确认某个模式是否已存在于持久性内存或最近的 journal 中时，使用 `scripts/search-superpowers-memory.ps1`。
12. 总结所学内容，以及哪些（如果有）应成为未来的规则、清单、脚本或 skill。

## 何时使用

- 用户明确要求从当前会话中捕获经验教训
- 用户明确命名了 `$superpowers-learning-workflow`
- 用户希望为未来会话持久化知识
- 用户希望将重复模式转化为可复用的学习笔记
- 仓库策略明确要求在有意义的工作后进行反思性捕获

## 输出

- 当持久性事实变更时，更新 `.superpowers-memory/PROJECT_CONTEXT.md`
- 更新 `.superpowers-memory/CURRENT_STATE.md`
- 当持久性决策变更时，更新 `.superpowers-memory/DECISIONS.md`
- 当识别出重复的失败模式时，更新 `.superpowers-memory/KNOWN_FAILURES.md`
- 当可信的验证规则变更时，更新 `.superpowers-memory/VERIFICATION_BASELINE.md`
- 当持久性团队约定变更时，更新 `.superpowers-memory/TEAM_PREFERENCES.md`
- 当持久性用户偏好变更时，更新 `.superpowers-memory/USER_PROFILE.md`
- 当持久性执行提醒变更时，更新 `.superpowers-memory/AGENT_NOTES.md`
- 新的或更新的会话日志条目
- 更新 `.superpowers-memory/LEARNING_BACKLOG.md` 中的可复用经验
- 更新 `.superpowers-memory/SESSION_CLOSE_CHECKLIST.md`（仅作为参考清单，不作为会话日志）
- 当内存被更新时的内存验证证据
- 当使用了建议脚本时，可选的内存更新建议证据
- 当使用了结算脚本时，可选的结算辅助输出
- 关于下次应记住内容的简短摘要

## 约束

- 不要将临时 TODO 噪音写入 `PROJECT_CONTEXT.md`
- 没有明确的重复模式时，不要将一次性修复转化为可复用规则
- 不要自动编辑 skill 库本身，除非用户明确要求这一单独步骤
- 保持学习笔记简洁且可操作
- 没有足够的重复证据或跨会话价值时，不要提升 backlog 条目
- 将 `ready_for_promotion` 视为更高的门槛：期望有重复证据、链接的来源和可审查的升级理由

<!-- checksum: sha256:5660eeb6123b0b66434a10b3e00eda6347f0db1ddd20c5ffeb365768b99fa0c2 -->