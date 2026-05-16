## Why

`agent-forge run` 命令在交互模式（agent/bash 模式）下创建并 attach 到容器后，没有将容器的 I/O 流连接到用户终端——`ContainerAttach` 返回的 `HijackedResponse` 被直接丢弃，用户看不到任何容器输出，也无法输入任何内容。此缺陷在所有平台上都存在，导致 run 命令的交互功能完全不可用。

## What Changes

- 在 `runengine.Run()` 的交互分支中实现完整的流拷贝：使用 `stdcopy.StdCopy` 解复用 Docker 多路复用流，将容器 stdout/stderr 定向到用户终端，将用户 stdin 转发到容器
- 添加终端原始模式设置（`term.SetRawTerminal`），确保 Ctrl+C、Tab 等控制键正确传递到容器
- 添加 SIGINT/SIGTERM 信号处理，用户 Ctrl+C 时转发到容器而非直接杀死 CLI 进程
- 添加窗口大小变化监听和转发（`SIGWINCH` → `ContainerResize`）

## Capabilities

### New Capabilities

- `interactive-container-io`: run 命令交互模式下，容器 stdin/stdout/stderr 与用户终端之间的双向流拷贝，支持终端原始模式和信号转发

### Modified Capabilities

<!-- 无已有 specs 需要修改 -->

## Impact

- `internal/run/runengine/engine.go`: 重写 Run() 交互分支（步骤 7-8），添加流拷贝和信号处理
- `go.mod`: 新增依赖 `github.com/moby/term`（终端原始模式设置）
