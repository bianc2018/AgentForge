## MODIFIED Requirements

### Requirement: 交互模式下容器输出可见

系统 SHALL 在 run 命令的交互模式（agent/bash/powershell 模式）下，将容器 attach 流的 stdout/stderr 解复用并输出到用户终端。

#### Scenario: bash 模式下看到容器 shell 提示符

- **WHEN** 用户执行 `agent-forge run`（默认 bash 模式）
- **THEN** 用户终端显示容器内 bash 的输出（提示符、命令回显等）
- **AND** 用户可在提示符后输入命令

#### Scenario: agent 模式下看到 agent 交互界面

- **WHEN** 用户执行 `agent-forge run -a claude`
- **THEN** 用户终端显示 claude 的交互式 TUI 界面
- **AND** 用户可正常与 agent 交互

#### Scenario: Windows PowerShell 模式下看到容器 shell 提示符

- **WHEN** 用户执行 `agent-forge run -b mcr.microsoft.com/powershell:lts-nanoserver-1809`
- **THEN** 用户终端显示容器内 PowerShell 的输出（提示符 `PS C:\>`、命令回显等）
- **AND** 用户可在提示符后输入 PowerShell 命令

### Requirement: 交互模式下用户输入传递到容器

系统 SHALL 在交互模式下将用户终端的 stdin 转发到容器 attach 连接。

#### Scenario: 用户在容器 bash 中输入命令

- **WHEN** 用户在 bash 模式下输入 `echo hello`
- **THEN** `echo hello` 被发送到容器内执行
- **AND** 终端显示 `hello` 输出

#### Scenario: 用户在容器 PowerShell 中输入命令

- **WHEN** 用户在 PowerShell 模式下输入 `Write-Host "hello"`
- **THEN** `Write-Host "hello"` 被发送到容器内执行
- **AND** 终端显示 `hello` 输出

### Requirement: 终端原始模式支持

系统 SHALL 在交互模式下将终端设置为原始模式（raw mode），使控制字符能正确传递。Linux 平台使用 Unix PTY raw mode，Windows 平台通过 Docker SDK 的 TTY 模式处理 ConPTY。

#### Scenario: Ctrl+C 终止容器内进程 (Linux)

- **WHEN** 用户在 Linux 容器交互模式下按 Ctrl+C
- **THEN** Ctrl+C 信号被转发为 `ContainerKill(containerID, "SIGINT")`
- **AND** CLI 进程不会被意外终止

#### Scenario: Ctrl+C 终止容器内进程 (Windows)

- **WHEN** 用户在 Windows 容器交互模式下按 Ctrl+C
- **THEN** 系统调用 `ContainerStop(containerID)` 停止容器
- **AND** CLI 进程不会被意外终止

### Requirement: 后台命令模式不受影响

系统 SHALL 保持 `--run` 后台命令模式的现有行为不变，无论平台。

#### Scenario: 后台命令模式正常执行

- **WHEN** 用户执行 `agent-forge run --run "echo hello"`
- **THEN** 容器执行命令后正常退出
- **AND** CLI 返回容器退出码

#### Scenario: Windows 后台命令模式正常执行

- **WHEN** 用户执行 `agent-forge run -b mcr.microsoft.com/powershell:lts-nanoserver-1809 --run "Write-Host hello"`
- **THEN** 容器通过 PowerShell 执行命令后正常退出
- **AND** CLI 返回容器退出码

## ADDED Requirements

### Requirement: Windows 平台终端尺寸同步

系统 SHALL 在 Windows 容器交互模式下通过 Docker SDK 的 `ContainerResize` API 同步终端尺寸，与 Linux 行为一致。

#### Scenario: Windows 容器终端尺寸同步

- **WHEN** 用户在 Windows 容器交互模式下调整终端窗口大小
- **THEN** 容器内 PowerShell 的终端尺寸随之更新
- **AND** 行为与 Linux 容器一致
