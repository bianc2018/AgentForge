## ADDED Requirements

### Requirement: 从基础镜像名称推断平台

系统 SHALL 提供 `InferPlatform(baseImage string) string` 函数，根据基础镜像名称推断目标平台：
- 镜像名称含 `windows`、`nanoserver`、`servercore` 不区分大小写 → 返回 `"windows"`
- 镜像来自 `mcr.microsoft.com` 且路径含 `windows` 关键词 → 返回 `"windows"`
- 其它所有情况 → 返回 `""`（空值表示 Linux 默认）

#### Scenario: Windows PowerShell 镜像推断

- **WHEN** baseImage 为 `mcr.microsoft.com/powershell:lts-nanoserver-1809`
- **THEN** `InferPlatform` 返回 `"windows"`

#### Scenario: Windows ServerCore 镜像推断

- **WHEN** baseImage 为 `mcr.microsoft.com/windows/servercore:ltsc2022`
- **THEN** `InferPlatform` 返回 `"windows"`

#### Scenario: Linux 镜像返回空

- **WHEN** baseImage 为 `docker.1ms.run/centos:7` 或 `ubuntu:22.04`
- **THEN** `InferPlatform` 返回 `""`

#### Scenario: 大小写不敏感

- **WHEN** baseImage 为 `Mcr.Microsoft.Com/Windows/ServerCore:ltsc2022`
- **THEN** `InferPlatform` 返回 `"windows"`

### Requirement: 未指定基础镜像时根据 Docker daemon 推断

系统 SHALL 在 `-b` 参数为空时调用 `dockerhelper.Client.Info()` 读取 `OSType` 字段：
- `OSType == "windows"` → 默认 BaseImage 为 `mcr.microsoft.com/powershell:lts-nanoserver-1809`，Platform 为 `"windows"`
- `OSType == "linux"` → 默认 BaseImage 为 `docker.1ms.run/centos:7`，Platform 为空（Linux）

#### Scenario: 连上 Windows Docker daemon 时自动选择 Windows 镜像

- **WHEN** Docker daemon OSType 为 `windows` 且用户未指定 `-b`
- **THEN** 系统自动使用 `mcr.microsoft.com/powershell:lts-nanoserver-1809` 作为 BaseImage
- **AND** Platform 设为 `"windows"`

#### Scenario: 连上 Linux Docker daemon 时保持默认

- **WHEN** Docker daemon OSType 为 `linux` 且用户未指定 `-b`
- **THEN** 系统使用 `docker.1ms.run/centos:7` 作为 BaseImage（行为不变）

### Requirement: 平台兼容性校验

系统 SHALL 在容器创建前校验推断的平台与 Docker daemon 是否兼容。Windows 容器只能在 Windows Docker daemon 上运行。

#### Scenario: Linux Docker daemon 拒绝 Windows 容器

- **WHEN** Docker daemon OSType 为 `linux` 且用户通过 `-b` 指定了 Windows 镜像
- **THEN** 系统返回错误 `当前 Docker daemon 不支持 Windows 容器，请在 Windows Docker 主机上运行`
- **AND** 以非零退出码退出

#### Scenario: Windows Docker daemon 接受 Linux 容器

- **WHEN** Docker daemon OSType 为 `windows` 且用户指定了 Linux 镜像
- **THEN** 系统正常创建 Linux 容器（Docker Desktop 支持双平台）
