## Why

`cmd/build.go` 的 `buildCmd` 当前是占位实现（仅打印"build 命令已调用（占位实现）"并返回），而 `internal/build/buildengine` 已完整实现了 Docker 镜像构建引擎（依赖展开、Dockerfile 生成、构建执行、重试、重建模式）。用户执行 `agent-forge build -b docker.1ms.run/ubuntu:22.04 -d all` 时构建并未实际执行。

## What Changes

- 修改 `cmd/build.go`：从 cobra 标志解析参数，创建 `dockerhelper.Client`，调用 `buildengine.Engine.Build()` 执行实际构建
- 构建流程遵循项目现有模式（参考 `cmd/run.go` 和 `cmd/deps.go` 的编排方式）

## Capabilities

### New Capabilities

- `build-command`: build CLI 命令接入构建引擎，支持 `-d`（依赖列表）、`-b`（基础镜像）、`--no-cache`、`-R`（重建模式）、`--max-retry`、`--gh-proxy` 参数

### Modified Capabilities

<!-- 无现有 spec 需要修改 -->

## Impact

- 修改文件：`cmd/build.go`（从占位实现替换为引擎调用）
- 依赖：`internal/build/buildengine`、`internal/shared/dockerhelper`（均已存在）
- 无新增依赖，无 API 变更
