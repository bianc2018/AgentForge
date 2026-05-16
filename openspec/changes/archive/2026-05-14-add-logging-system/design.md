## Context

项目当前使用 `fmt.Println`/`fmt.Printf` 输出到 stdout，cobra 框架将 error 输出到 stderr。无任何持久化日志，排查问题时需重现操作。`configresolver` 已提供配置目录解析，默认 `$(pwd)/coding-config`。

Go 1.21+ 标准库包含 `log/slog`，提供结构化日志（JSON 格式）、级别过滤、Handler 接口。项目 go.mod 声明 `go 1.23`，可直接使用。

## Goals / Non-Goals

**Goals:**
- 日志写入文件 `<config-dir>/log/<YYYY-MM-DD>.log`，按天自动轮转
- 支持 Debug/Info/Warn/Error 四个日志级别
- 通过环境变量 `AGENTFORGE_LOG_LEVEL` 配置级别（默认 `info`）
- 日志为 JSON 结构化格式，便于 grep/jq 分析
- 日志初始化失败不阻塞命令执行（容错降级）

**Non-Goals:**
- 不替换 stderr 错误输出（cobra 的错误机制保持不变）
- 不添加日志采样、压缩、过期清理
- 不实现远程日志收集
- 不修改已有 `fmt.Println` 的用户可见输出（日志是补充，不是替代）

## Decisions

### Decision 1: 使用 `log/slog` 而非第三方库

**选择**: Go 标准库 `log/slog` + 自定义 `slog.Handler`。

**原因**: 无需新增依赖，`slog` 自 Go 1.21 起内置，支持 JSON 输出和级别过滤。通过实现 `io.WriteCloser` 接口的文件轮转 Writer 嵌入 `slog.NewJSONHandler` 即可。

**替代方案考虑**:
- `logrus`: 功能丰富但已进入维护模式，且增加外部依赖
- `zap`: 高性能但 API 复杂度高，对 CLI 工具过度

### Decision 2: 每日文件轮转通过 Writer 包装实现

**选择**: 自定义 `dailyWriter` 实现 `io.Writer`，每次 `Write` 前检查当前日期，若日期变化则关闭旧文件、打开新文件。

**原因**: 轻量级，无需定时器或 goroutine。CLI 命令生命周期短，在 Write 时检查日期即可覆盖按天场景。

### Decision 3: 日志包初始化接受 configDir 参数

**选择**: `logging.Init(configDir)` 在 `cmd.Execute()` 开始时调用，所有子命令共享同一个 logger 实例。

**原因**: 配置目录由 `-c` 参数决定，需在命令参数解析后才能确定路径。在 `Execute()` 中初始化保证所有命令可用。
