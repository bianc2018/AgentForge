## ADDED Requirements

### Requirement: Progress log mode for streaming text

系统 SHALL 提供进度日志模式（`ProgressLog`），将文本行直接写入输出，适用于 Docker 构建日志等无固定步骤数的流式输出场景。

#### Scenario: Progress log writes lines directly

- **WHEN** 系统使用 `ProgressLog` 模式写入一行文本
- **THEN** 文本行 SHALL 直接写入底层的 `io.Writer`，无任何前缀或格式化

### Requirement: Progress bar mode for step-based tasks

系统 SHALL 提供进度条模式（`ProgressBar`），以步骤数为基础显示百分比进度，适用于依赖下载等已知步骤数的场景。

#### Scenario: Progress bar updates as steps complete

- **WHEN** 系统使用 `ProgressBar` 模式，总步骤数为 5，每完成一个步骤调用 `Tick()`
- **THEN** 在终端环境下 SHALL 渲染 ANSI 进度条（如 `[==>   ] 3/5  60%`），在非终端环境下 SHALL 每步输出一行文本（如 `[3/5] <description>`）

#### Scenario: Progress bar completion message

- **WHEN** 所有步骤完成
- **THEN** 进度条 SHALL 显示 100% 完成状态并换行

### Requirement: Spinner mode for unknown-duration tasks

系统 SHALL 提供 spinner 动画模式（`Spinner`），显示旋转动画和描述文本，适用于未知耗时的等待任务。

#### Scenario: Spinner animates during long operation

- **WHEN** 系统使用 `Spinner` 模式启动一个耗时不确定的操作
- **THEN** 在终端环境下 SHALL 持续渲染旋转字符（`⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`）和描述文本，帧间隔 SHALL 为 100ms；在非终端环境下 SHALL 定期（每 5 秒）输出文本状态更新

#### Scenario: Spinner stops on completion

- **WHEN** 操作完成，调用 `Spinner.Stop()` 或 `Spinner.Success(message)`
- **THEN** 系统 SHALL 停止动画，终端环境下清除 spinner 行并输出完成消息，非终端环境下输出最终状态消息

### Requirement: TTY auto-detection and fallback

系统 SHALL 自动检测输出是否为终端（TTY），终端环境使用 ANSI 控制序列渲染动画/进度条，非终端环境降级为纯文本输出。

#### Scenario: TTY environment uses ANSI rendering

- **WHEN** `os.Stdout` 指向终端
- **THEN** `ProgressBar` 和 `Spinner` SHALL 使用 ANSI 控制序列（回车 `\r` 和清除行 `\033[K`）进行原地刷新

#### Scenario: Non-TTY environment falls back to plain text

- **WHEN** `os.Stdout` 被重定向到文件或管道
- **THEN** `ProgressBar` SHALL 每步输出一行文本而非原地刷新，`Spinner` SHALL 定期输出文本状态行而非动画
