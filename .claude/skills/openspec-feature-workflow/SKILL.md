<!-- generatedBy: sot@2.5.7 -->
---
name: openspec-feature-workflow
description: "当 feature 需要 OpenSpec 变更（包含 proposal、design、specs 和 tasks）并在编码前完成时使用。当仓库策略要求对非平凡的 feature 使用 OpenSpec、当用户要求先做 proposal/design/tasks、或当行为变更需要持久的变更产物时触发。"
model_hint: sonnet
tags:
  - openspec
  - specs
  - proposal
category: engineering
---


# OpenSpec Feature Workflow

## 概述

使用此 skill 处理 feature 交付的 OpenSpec 部分。它创建并完成实现前所需的变更产物。

这是一个明确的 opt-in 工作流。默认情况下不要使用。仅在用户明确要求此工作流、命名了此 skill、或仓库策略明确要求时使用。

## 工作流

1. 推导或确认 kebab-case 格式的变更名称。
2. 在 `openspec/changes/<change-name>/` 下创建变更。
3. 运行 `openspec status --change "<change-name>" --json` 检查产物顺序。
4. 在编写每个产物之前，读取 `openspec instructions <artifact> --change "<change-name>" --json`。
5. 按依赖顺序完成产物：
   - `proposal.md`
   - `design.md`
   - `specs/.../spec.md`
   - `tasks.md`
6. 反复检查状态，直到所有 apply-required 产物完成后再开始实现。

## 何时使用

- 用户明确要求在编码前先做 proposal/design/tasks
- 用户明确命名了 `$openspec-feature-workflow`
- `AGENTS.md` 或团队策略明确要求此工作流

## 产物预期

- `proposal.md`：为什么变更以及变更了什么
- `design.md`：技术方案和权衡
- `specs/.../spec.md`：规范性需求和场景
- `tasks.md`：可执行的实现清单

## 约束

- 不要跳过 `openspec status` 中的依赖顺序
- 不要将指令元数据复制到产物文件中
- 在所需产物就绪之前不要开始编码

<!-- checksum: sha256:d0d115f74f27761c1efd3ddec290c376af602f27af41d1e7091aeecd49c60d0c -->