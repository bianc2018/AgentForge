## ADDED Requirements

### Requirement: RunEngine 根据基础镜像选择 Shell

系统 SHALL 在 `buildCmd` 函数中根据 `RunParams.BaseImage` 推断平台，自动选择 shell：
- 推断为 Windows 时使用 `powershell -Command "<cmd>"`
- 推断为 Linux 时使用 `bash -c "<cmd>"`（保持现有行为）
- `RunParams` 新增 `Platform` 字段存储推断结果用于后续流程

#### Scenario: Windows 镜像使用 PowerShell 启动容器

- **WHEN** RunParams.BaseImage 为 `mcr.microsoft.com/powershell:lts-nanoserver-1809`
- **THEN** 容器 Cmd 为 `["powershell", "-Command", "..."]`
- **AND** wrapper 脚本使用 PowerShell 语法（`Function Invoke-AgentWrapper { ... }`）

#### Scenario: Linux 镜像行为不变

- **WHEN** RunParams.BaseImage 为 Linux 镜像（如 `docker.1ms.run/centos:7`）
- **THEN** 容器 Cmd 为 `["bash", "-c", "eval \"$AGENTFORGE_WRAPPER\"; exec bash"]`

### Requirement: ContainerCreate 传递 platform 参数

系统 SHALL 在推断为 Windows 平台时，调用 `dockerhelper.ContainerCreate` 时传递 `&specs.Platform{OS: "windows", Architecture: "amd64"}`。

#### Scenario: Windows 容器创建指定 platform

- **WHEN** 推断平台为 Windows
- **THEN** ContainerCreate 的 platform 参数为 `&specs.Platform{OS: "windows", Architecture: "amd64"}`

#### Scenario: Linux 容器 platform 为 nil

- **WHEN** 推断平台为 Linux
- **THEN** ContainerCreate 的 platform 参数为 nil

### Requirement: Windows 容器路径适配（RunEngine 侧）

系统 SHALL 在 `toContainerPath` 中新增 Windows 容器内路径转换：当 Platform 为 `"windows"` 时，将 WSL 路径转为 Windows 容器内 `C:\` 格式用于 WorkingDir 和 mount Target。

#### Scenario: Linux 路径转为 Windows 容器路径

- **WHEN** 主机路径为 `/home/user/project` 且 Platform 为 `"windows"`
- **THEN** 容器 WorkingDir 为 `C:\home\user\project`

#### Scenario: WSL 路径转为 Windows 容器路径

- **WHEN** 主机路径为 `/mnt/d/code/AgentForge` 且 Platform 为 `"windows"`
- **THEN** 容器 WorkingDir 为 `D:\code\AgentForge`

### Requirement: Windows 不支持 Docker-in-Docker 模式

系统 SHALL 在 Windows 平台下忽略 `--docker`/`--dind` flag 并输出警告信息。

#### Scenario: Windows 平台 DIND 模式输出警告

- **WHEN** 推断平台为 Windows 且 RunParams.Docker 为 true
- **THEN** 系统输出警告 `Windows 容器不支持 Docker-in-Docker 模式，已忽略 --docker 参数`
- **AND** 容器仍正常启动（非特权模式）

### Requirement: 交互模式下 Windows 容器信号处理

系统 SHALL 在 Windows 容器交互模式下使用 `ContainerStop`（而非 `ContainerKill`）处理用户退出信号，因 Windows 容器不支持 POSIX 信号转发。

#### Scenario: Windows 容器 Ctrl+C 处理

- **WHEN** 用户在 Windows 容器交互模式下按 Ctrl+C
- **THEN** 系统调用 `ContainerStop(containerID)` 停止容器
- **AND** 不调用 `ContainerKill`（避免 Windows 下未知信号错误）
