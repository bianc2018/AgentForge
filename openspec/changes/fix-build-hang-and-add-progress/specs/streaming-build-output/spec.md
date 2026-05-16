## ADDED Requirements

### Requirement: Streaming build output to stdout

系统 SHALL 在 Docker 镜像构建期间将构建日志实时流式输出到 stdout，而非在构建完成后一次性输出。

#### Scenario: Build with all deps shows real-time progress

- **WHEN** 用户执行 `agent-forge build -d all`
- **THEN** 系统 SHALL 在 Docker 构建过程中持续输出构建日志行到 stdout，用户可在构建过程中看到实时进度

#### Scenario: Build output is still collected for error analysis

- **WHEN** Docker 构建失败
- **THEN** 系统 SHALL 在错误信息中包含完整的构建输出（与实时流式输出相同的内容），用于错误定位和重试判断

#### Scenario: Streaming output works with rebuild mode

- **WHEN** 用户执行 `agent-forge build -R`
- **THEN** 系统 SHALL 在重建模式下同样实时流式输出构建日志

#### Scenario: Streaming output during retry

- **WHEN** Docker 构建因网络错误触发重试
- **THEN** 系统 SHALL 在每次重试前输出 `[重试 N/M]` 分隔行，区分各次尝试的输出
