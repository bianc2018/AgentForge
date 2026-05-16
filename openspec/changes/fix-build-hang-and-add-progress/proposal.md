## Why

`agent-forge build -d all` 执行时用户端无任何输出，构建过程完全黑盒——`io.ReadAll(resp.Body)` 阻塞式读取 Docker 构建流，直到构建结束或失败才一次性输出全部日志。用户按 Ctrl+C 中断时，context 取消信号仅在重试循环顶部检查，无法中断正在进行的 `io.ReadAll` 阻塞调用。长时间运行的耗时任务（Docker 镜像构建、依赖安装等）缺乏进度显示，用户体验极差。

## What Changes

- **修复构建无响应**：将 `io.ReadAll` 阻塞读取改为流式读取，实时输出 Docker 构建日志到 stdout，同时收集完整日志用于错误分析
- **Context 取消响应**：流式读取循环中检查 context 取消，确保 Ctrl+C 能及时中断构建
- **通用进度组件**：新增 `internal/shared/progress` 包，提供进度日志、进度条和 spinner 动画三种进度显示模式
- **构建引擎集成进度显示**：在 `buildengine.Build()` 中集成流式输出和进度显示，让用户实时感知构建状态
- **Run 引擎同步修复**：排查 `runengine` 中类似的阻塞读取问题，一并修复

## Capabilities

### New Capabilities

- `streaming-build-output`: 构建过程实时流式输出，用户可立即看到 Docker 构建日志，同时收集完整输出用于错误分析和重试判断
- `context-cancellation-during-io`: 流式读取循环中响应 context 取消，Ctrl+C 可即时中断长时间运行的 I/O 操作
- `progress-indicator`: 通用进度显示组件，支持进度日志（文本行）、进度条（百分比/步骤）和 spinner（动画旋转指示器）三种模式，供构建、运行等耗时命令复用

### Modified Capabilities

- `build-command`: 构建命令的行为从"构建完成后一次性输出"变为"构建过程中实时流式输出"，进度信息通过 progress 组件展示

## Impact

- `internal/build/buildengine/buildengine.go` — 核心修改：将 `io.ReadAll` 改为流式 `io.Copy` + context 检查
- `cmd/build.go` — 移除一次性 `fmt.Print(output)`，改为直接写入 stdout
- `internal/run/runengine/engine.go` — 排查类似阻塞读取问题
- `internal/shared/progress/` — 新增通用进度组件包
- 不影响 Docker API 调用方式、Dockerfile 生成逻辑、依赖展开逻辑
