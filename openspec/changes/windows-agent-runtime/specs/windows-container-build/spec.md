## ADDED Requirements

### Requirement: Dockerfile 生成器支持 Windows 基础镜像

系统 SHALL 在 `ImageFamily` 枚举中新增 `FamilyWindows`，当检测到基础镜像名称包含 `windows`、`nanoserver`、`servercore` 关键词，或镜像来源为 `mcr.microsoft.com` 且包含上述关键词时，归类为 Windows 家族。

#### Scenario: 检测 Windows Nanoserver 镜像

- **WHEN** BaseImage 为 `mcr.microsoft.com/powershell:lts-nanoserver-1809`
- **THEN** detectImageFamily 返回 FamilyWindows

#### Scenario: 检测 Windows ServerCore 镜像

- **WHEN** BaseImage 为 `mcr.microsoft.com/windows/servercore:ltsc2022`
- **THEN** detectImageFamily 返回 FamilyWindows

#### Scenario: 普通 Linux 镜像不受影响

- **WHEN** BaseImage 为 `docker.1ms.run/centos:7`
- **THEN** detectImageFamily 返回 FamilyRHEL

### Requirement: Windows Dockerfile 使用 PowerShell Shell 指令

系统 SHALL 为 FamilyWindows 镜像在 Dockerfile 中 FROM 之后、第一个 RUN 之前生成 `SHELL ["powershell", "-Command"]` 指令。

#### Scenario: Windows Dockerfile 包含 SHELL 指令

- **WHEN** 为 FamilyWindows 镜像生成 Dockerfile
- **THEN** Dockerfile 在 FROM 行之后立即包含 `SHELL ["powershell", "-Command"]`

### Requirement: Windows 基础工具安装使用 PowerShell 命令

系统 SHALL 为 Windows 镜像生成等价的 PowerShell 命令：
- `Invoke-WebRequest -Uri <url> -OutFile <path>` 替代 `curl -fsSL`
- `Expand-Archive` 替代 `tar -xzf` / `unzip`
- 环境变量设置使用 `$env:VAR_NAME = "value"` 语法

#### Scenario: Windows Dockerfile 安装基础工具

- **WHEN** 为 FamilyWindows 且无依赖生成 Dockerfile
- **THEN** Dockerfile 包含 `Invoke-WebRequest` 下载 Git for Windows 的 RUN 指令
- **AND** 不包含 `apt-get` 或 `yum` 命令

### Requirement: Windows Dockerfile 支持 Node.js 和 Python 安装

系统 SHALL 为需要 npm/pip 的 Windows 依赖生成 PowerShell 安装命令：
- Node.js: 通过 `Invoke-WebRequest` 下载 `.msi` 并静默安装（`msiexec /i /qn`）
- Python: 通过 `Invoke-WebRequest` 下载官方 `python.exe` 安装器

#### Scenario: Windows 镜像安装 Node.js 依赖

- **WHEN** 依赖列表包含需要 npm 的依赖，且基础镜像为 FamilyWindows
- **THEN** Dockerfile 包含下载并静默安装 Node.js MSI 的 PowerShell RUN 指令
- **AND** npm 镜像源通过 `$env:npm_config_registry` 设置

### Requirement: Windows Dockerfile CMD 使用 PowerShell

系统 SHALL 为 FamilyWindows 镜像生成 `CMD ["powershell"]` 而非 `CMD ["/bin/bash"]`。

#### Scenario: Windows Dockerfile 默认入口为 PowerShell

- **WHEN** 为 FamilyWindows 镜像生成 Dockerfile
- **THEN** 最后一行指令为 `CMD ["powershell"]`

### Requirement: 构建引擎根据基础镜像设置 platform

系统 SHALL 在 `buildengine.Build` 中根据 `BaseImage` 参数推断平台。当推断为 Windows 平台时，在 `ImageBuild` API 调用中设置 `ImageBuildOptions.Platform` 为 `"windows/amd64"`，镜像标签追加 `-windows` 后缀。

#### Scenario: 构建 Windows 镜像时自动设置 platform

- **WHEN** BuildParams.BaseImage 为 `mcr.microsoft.com/powershell:lts-nanoserver-1809`
- **THEN** 平台推断为 Windows，ImageBuild API 的 Platform 为 `"windows/amd64"`
- **AND** 镜像构建成功后标签为 `agent-forge:latest-windows`

#### Scenario: Linux 构建不变

- **WHEN** BuildParams.BaseImage 为空或 Linux 镜像
- **THEN** ImageBuild API 的 Platform 为空字符串（Docker daemon 默认）
- **AND** 镜像标签为 `agent-forge:latest`
