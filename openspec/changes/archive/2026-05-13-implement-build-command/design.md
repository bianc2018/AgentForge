## Context

`internal/build/buildengine` 已完整实现构建引擎（`Build()` 方法），`internal/shared/argsparser` 已定义 `BuildParams` 参数集，但 `cmd/build.go` 的 `buildCmd` 是占位实现。项目内 `cmd/run.go` 和 `cmd/deps.go` 已建立了清晰的 CLI 命令 → Engine 调用模式。

## Goals / Non-Goals

**Goals:**
- `cmd/build.go` 从 cobra 标志解析 `BuildParams`
- 创建 `dockerhelper.Client` 并传入 `buildengine.Engine`
- 调用 `engine.Build(ctx, params)` 执行构建

**Non-Goals:**
- 不修改 buildengine 的任何逻辑
- 不修改 BuildParams 结构体
- 不新增 Docker 操作

## Decisions

### 1. 遵循现有模式

**决策：完全照搬 `cmd/deps.go` 和 `cmd/run.go` 的编排模式。**

```go
// cmd/build.go RunE:
params := buildengine.BuildParams{
    Deps:      flagVal("deps"),
    BaseImage: flagVal("base-image"),
    Config:    flagVal("config"),
    NoCache:   flagVal("no-cache"),
    Rebuild:   flagVal("rebuild"),
    MaxRetry:  flagVal("max-retry"),
    GHProxy:   flagVal("gh-proxy"),
}
helper, err := dockerhelper.NewClient()
engine := buildengine.New(helper)
return engine.Build(cmd.Context(), params)
```

无替代方案，因为引擎和参数模型都已就绪，仅缺接线代码。

### 2. 错误处理

**决策：引擎返回的 error 直接向上传播给 cobra，由 `cmd/root.go` 的 `Execute()` 统一处理退出码。**

与其他命令（run、deps、export）保持一致。

## Risks / Trade-offs

- **Docker daemon 不可达**：若 daemon 未运行，`dockerhelper.NewClient()` 不会报错，但 `engine.Build()` 将返回 Docker 连接错误 → `cmd/root.go` 的 `Execute()` 会捕获并以退出码 1 退出，与 run 命令行为一致
- **构建超时**：镜​像构建可能需要较长时间，特别是首次拉取基础镜像 → `context.Background()` 无超时；后续可考虑 `--timeout` 参数（不在本次 scope）
