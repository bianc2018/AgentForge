## ADDED Requirements

### Requirement: 交互模式下容器输出可见

系统 SHALL 在 run 命令的交互模式（agent/bash 模式）下，将容器 attach 流的 stdout/stderr 解复用并输出到用户终端。

#### Scenario: bash 模式下看到容器 shell 提示符

- **WHEN** 用户执行 `agent-forge run`（默认 bash 模式）
- **THEN** 用户终端显示容器内 bash 的输出（提示符、命令回显等）
- **AND** 用户可在提示符后输入命令

#### Scenario: agent 模式下看到 agent 交互界面

- **WHEN** 用户执行 `agent-forge run -a claude`
- **THEN** 用户终端显示 claude 的交互式 TUI 界面
- **AND** 用户可正常与 agent 交互

### Requirement: 交互模式下用户输入传递到容器

系统 SHALL 在交互模式下将用户终端的 stdin 转发到容器 attach 连接。

#### Scenario: 用户在容器 bash 中输入命令

- **WHEN** 用户在 bash 模式下输入 `echo hello`
- **THEN** `echo hello` 被发送到容器内执行
- **AND** 终端显示 `hello` 输出

### Requirement: 终端原始模式支持

系统 SHALL 在交互模式下将终端设置为原始模式（raw mode），使控制字符（Ctrl+C、Tab 等）能正确传递。

#### Scenario: Ctrl+C 终止容器内进程

- **WHEN** 用户在交互模式下按 Ctrl+C
- **THEN** Ctrl+C 信号被传递到容器内前台进程
- **AND** CLI 进程不会被意外终止

### Requirement: 后台命令模式不受影响

系统 SHALL 保持 `--run` 后台命令模式的现有行为不变。

#### Scenario: 后台命令模式正常执行

- **WHEN** 用户执行 `agent-forge run --run "echo hello"`
- **THEN** 容器执行命令后正常退出
- **AND** CLI 返回容器退出码
