## MODIFIED Requirements

### Requirement: 交互模式下容器输出可见

系统 SHALL 在 run 命令的交互模式（agent/bash 模式）下，将容器 attach 流的 stdout/stderr 解复用并输出到用户终端。系统 SHALL 在附加到容器后立即同步终端尺寸并注入 TERM 环境变量，确保 bash 提示符在附加完成后立即可见。

#### Scenario: bash 模式下看到容器 shell 提示符

- **WHEN** 用户执行 `agent-forge run`（默认 bash 模式）
- **THEN** 用户终端在附加完成后立即可见容器内 bash 的提示符
- **AND** 用户可在提示符后输入命令

#### Scenario: agent 模式下看到 agent 交互界面

- **WHEN** 用户执行 `agent-forge run -a claude`
- **THEN** 用户终端显示 claude 的交互式 TUI 界面
- **AND** 用户可正常与 agent 交互

### Requirement: 终端原始模式支持

系统 SHALL 在交互模式下将终端设置为原始模式（raw mode），使控制字符（Ctrl+C、Tab 等）能正确传递。系统 SHALL 在设置原始模式前通过 ContainerResize 将当前终端尺寸同步到容器 PTY。

#### Scenario: Ctrl+C 终止容器内进程

- **WHEN** 用户在交互模式下按 Ctrl+C
- **THEN** Ctrl+C 信号被传递到容器内前台进程
- **AND** CLI 进程不会被意外终止

#### Scenario: 终端尺寸在附加时同步

- **WHEN** 系统成功附加到交互容器
- **THEN** 系统调用 ContainerResize 将主机终端的行列数同步到容器 PTY
- **AND** 容器内程序（如 bash）能正确获取窗口尺寸

## ADDED Requirements

### Requirement: TERM 环境变量自动检测与注入

系统 SHALL 在交互模式下从主机环境变量 `TERM` 读取终端类型并注入容器，若主机 `TERM` 为空则 fallback 到 `xterm-256color`。用户通过 `-e TERM=...` 显式指定时始终覆盖自动检测值。

#### Scenario: 主机 TERM 存在时透传

- **WHEN** 用户执行 `agent-forge run`，主机环境 `TERM=screen-256color`，且未提供 `-e TERM=...`
- **THEN** 容器环境变量包含 `TERM=screen-256color`
- **AND** 容器内程序使用与用户终端匹配的终端类型

#### Scenario: 主机 TERM 为空时 fallback

- **WHEN** 用户执行 `agent-forge run`，主机环境 `TERM` 为空或未设置，且未提供 `-e TERM=...`
- **THEN** 容器环境变量包含 `TERM=xterm-256color`

#### Scenario: 用户显式设置 TERM 时使用用户值

- **WHEN** 用户执行 `agent-forge run -e TERM=vt100`
- **THEN** 容器环境变量 `TERM` 值为 `vt100`（用户值覆盖自动检测值）
