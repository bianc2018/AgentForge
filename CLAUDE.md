# CLAUDE.md

本文件为 Claude Code 在此仓库中工作时提供行为约束。

## 语言

输出（代码注释、文档、回复、思考过程）均使用简体中文，以下除外：

- IT 技术术语始终保留英文：API, backend, frontend, endpoint, deploy, branch, commit, pull request, merge, cache, token, bug, framework, pipeline, build, release, feature, sprint, backlog, mock, stub, refactor, hotfix, rollback, CI/CD, log, test, debug。

## 提交规范

每完成一个任务或功能，必须进行一次代码提交。提交信息遵循约定式提交（Conventional Commits）格式，正文须包含以下四个字段：

```
<type>[scope]: <description>

变更功能描述：<本次变更实现了什么>
变更影响：<对现有功能、接口、数据的影响>
变更理由：<为什么这样改，而非其他方案>
关联文档：<spec 文档路径 + 章节，如 docs/features/<slug>/requirements.md#REQ-1>
```

其中 `<type>` 取值为 `feat | fix | docs | chore | refactor | test`。

## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

Rules:
- ALWAYS read graphify-out/GRAPH_REPORT.md before reading any source files, running grep/glob searches, or answering codebase questions. The graph is your primary map of the codebase.
- IF graphify-out/wiki/index.md EXISTS, navigate it instead of reading raw files
- For cross-module "how does X relate to Y" questions, prefer `graphify query "<question>"`, `graphify path "<A>" "<B>"`, or `graphify explain "<concept>"` over grep — these traverse the graph's EXTRACTED + INFERRED edges instead of scanning files
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).
