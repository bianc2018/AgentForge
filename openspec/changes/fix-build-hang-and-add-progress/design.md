## Context

当前 `buildengine.Build()` 使用 `io.ReadAll(resp.Body)` 一次性阻塞读取 Docker 构建响应流，导致：

1. 整个构建期间（`-d all` 可达数十分钟）用户看不到任何输出
2. context 取消（Ctrl+C）只在重试循环顶部检查，无法中断正在进行的 `io.ReadAll`
3. `runengine` 中存在类似的容器 attach 流读取，同样缺乏进度显示

项目已有 `internal/shared/logging` 文件日志系统和 `cobra` 命令行框架。Docker Engine API 的 `ImageBuild` 返回流式 JSON 行，每行包含 `stream` 或 `aux` 字段。

## Goals / Non-Goals

**Goals:**
- 构建输出从"一次性打印"改为"实时流式打印"，同时保留完整输出用于错误分析和重试判断
- 流式读取过程中响应 context 取消，Ctrl+C 可即时中断
- 提供通用进度显示组件（进度日志、进度条、spinner），供 build/run 等命令复用
- 修复 `runengine` 中类似的长时阻塞无响应问题

**Non-Goals:**
- 不改变 Docker API 调用方式或构建选项
- 不修改依赖展开、Dockerfile 生成、重试/退避逻辑
- 不引入第三方进度条库（使用标准库 + ANSI 控制序列）

## Decisions

### 1. 流式输出：`io.Copy` + `contextAwareWriter`

**选择**：使用 `io.Copy` 将 `resp.Body` 复制到同时写入 stdout 和 `bytes.Buffer` 的 `io.MultiWriter` 组合，包装一个检查 context 的自定义 writer。

**替代方案**：
- `bufio.Scanner` 逐行读取：增加不必要的行缓冲开销，且 Docker 构建输出是 JSON 流不是纯文本行
- 保持 `io.ReadAll` + goroutine 定时打印进度：无法解决"只有构建结束才能看到输出"的根本问题

**理由**：`io.Copy` 是最简单的流式复制方式，`io.MultiWriter` 天然支持双路写入（stdout + buffer），context 检查在 writer 层面无侵入。

### 2. 进度组件：三级模式，TTY 自动检测

**选择**：创建 `internal/shared/progress` 包，提供三种模式：

| 模式 | 适用场景 | 终端 | 非终端 |
|------|----------|------|--------|
| `Log` | 实时文本行流（Docker 构建日志） | 直接写入 | 直接写入 |
| `Bar` | 已知步骤数（依赖安装、文件下载） | ANSI 进度条 | 每步一行文本 |
| `Spinner` | 未知耗时（等待 daemon、健康检查） | 动画旋转器 | 定期文本更新 |

**TTY 检测**：`os.Stdout` 是否指向终端。非 TTY 环境（CI、重定向）自动降级为纯文本模式。

**替代方案**：
- 使用第三方库如 `bubbletea`、`cheggaaa/pb`：增加依赖，且当前需求简单不需要全功能 TUI 框架
- 保持纯日志输出：无法满足"进度显示"需求

**理由**：自建轻量组件足够满足需求，不引入外部依赖，与项目现有代码风格一致。

### 3. BuildEngine 重构：输出模式切换

**选择**：在 `BuildParams` 中增加 `ProgressWriter io.Writer` 字段，`cmd/build.go` 传入 `os.Stdout`。构建引擎将 Docker 流数据同时写入 `ProgressWriter` 和内部 `outputBuf`。

**理由**：通过依赖注入让构建引擎不直接依赖 `os.Stdout`，便于测试（测试时可注入 `bytes.Buffer`）。

### 4. RunEngine 同步修复

**选择**：在 `runengine` 的所有长时间 I/O 操作（容器 attach 流读取、容器等待）中采用相同的 context 检查 writer 模式。

## Risks / Trade-offs

- **[风险] 流式输出 + 重试逻辑交互**：流式输出可能打印了失败的构建日志，然后重试又打印一次。→ **缓解**：重试前打印分隔行 `[重试 N/M]` 区分每次尝试
- **[风险] TTY 检测在某些环境误判**（如 tmux、某些 CI）→ **缓解**：非 TTY 降级为纯文本，功能不受影响，仅失去动画效果
- **[权衡] 进度条需要知道总步骤数**：Docker 构建无法预先知道总步骤数 → 进度条模式适用于 `depsmodule` 依赖下载等已知步骤的场景，构建本身使用 spinner + 流式日志

## Open Questions

- 无
