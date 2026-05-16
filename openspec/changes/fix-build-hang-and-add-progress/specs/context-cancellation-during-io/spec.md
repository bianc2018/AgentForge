## ADDED Requirements

### Requirement: Context cancellation interrupts streaming I/O

系统 SHALL 在流式读取 Docker 构建输出（或任何长时间 I/O 操作）过程中响应 context 取消信号，及时中断操作。

#### Scenario: Ctrl+C interrupts a running build

- **WHEN** 用户在构建过程中按 Ctrl+C（发送 SIGINT）
- **THEN** 系统 SHALL 在 1 秒内终止 Docker 构建并退出，而非继续阻塞直到 Docker daemon 超时

#### Scenario: Context cancellation during retry backoff

- **WHEN** 构建重试正在等待指数退避，用户按 Ctrl+C
- **THEN** 系统 SHALL 立即终止等待并退出，输出"构建被中断"消息

#### Scenario: Graceful cleanup on cancellation

- **WHEN** 构建被 Ctrl+C 中断
- **THEN** 系统 SHALL 输出已收集的构建日志（截断前已有部分），并以非零退出码退出
