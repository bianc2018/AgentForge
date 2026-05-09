---
name: commit
description: 分析 git 仓库的变更，给出清晰简洁的 commit 消息建议，使用中文，遵循 conventional commits 规范。
argument-hint: [-y]
---

分析 git 仓库的变更（使用 git status 和 git diff），给出清晰简洁的 commit 消息建议，
遵循 conventional commits 规范（feat, fix, docs, chore 等）。

重要：也要查看未暂存文件的差异。

如果提供了 -y 标志，则直接执行 commit。
如果未提供 -y 标志，询问是否应该执行 commit。如果用户选择"否"，则中止。

如果确认，先添加文件（使用 git add），然后使用建议的消息执行 commit。
