## Why

项目当前所有输出均通过 `fmt.Println`/`fmt.Printf` 直接写到 stdout/stderr，无持久化日志。调试时需要重现操作才能看到错误详情，排查问题的效率低。需要在配置目录下按日期记录运行日志，支持日志级别过滤，便于事后排查和审计。

## What Changes

- 新增 `internal/shared/logging/` 包，提供文件日志功能
- 日志文件路径：`<config-dir>/log/<YYYY-MM-DD>.log`，按天自动轮转
- 使用 Go 标准库 `log/slog` 实现结构化日志（JSON 格式），支持 Debug/Info/Warn/Error 四个级别
- 通过环境变量 `AGENTFORGE_LOG_LEVEL` 配置日志级别（默认 `info`）
- 在 run、build、endpoint、doctor 等命令的执行路径中集成日志记录

## Capabilities

### New Capabilities

- `file-logging`: 按日期轮转的文件日志系统，支持 JSON 结构化输出和日志级别配置

### Modified Capabilities

<!-- 无已有 specs 需要修改 -->

## Impact

- `internal/shared/logging/`：新增包，封装日志初始化、级别过滤、文件轮转
- `cmd/root.go`：在 `Execute()` 中初始化日志
- `internal/run/runengine/engine.go`：关键步骤替换 `fmt.Errorf` 为带日志的错误返回
- `internal/build/buildengine/buildengine.go`：构建流程添加日志记录
