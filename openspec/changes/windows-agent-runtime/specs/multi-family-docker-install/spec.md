## MODIFIED Requirements

### Requirement: Docker CLI 跨发行版安装

系统 SHALL 在所有支持的基础镜像（CentOS 7/8、RHEL、Fedora、Ubuntu、Debian）上成功安装 Docker CLI，不依赖系统包管理器（yum/apt）。Windows 平台上 Docker CLI 的安装待后续支持（Windows 容器内不支持 Docker）。

#### Scenario: CentOS 7 基础镜像安装 Docker CLI

- **WHEN** 基础镜像为 `centos:7` 且依赖列表包含 `docker`
- **THEN** Docker CLI 通过官方静态二进制下载安装成功
- **AND** `docker --version` 可正常执行

#### Scenario: Ubuntu 基础镜像安装 Docker CLI

- **WHEN** 基础镜像为 `ubuntu:22.04` 且依赖列表包含 `docker`
- **THEN** Docker CLI 通过官方静态二进制下载安装成功
- **AND** 未调用 `apt-get install -y docker`

#### Scenario: Debian 基础镜像安装 Docker CLI

- **WHEN** 基础镜像为 `debian:12` 且依赖列表包含 `docker`
- **THEN** Docker CLI 通过官方静态二进制下载安装成功
- **AND** 未调用 `apt-get install -y docker`

### Requirement: 系统包名跨发行版映射

`adaptCommandForFamily` 函数 SHALL 将常见 RHEL 系包名映射为 Debian 系等效包名，确保 `DepSystemPkg` 类型的未知依赖在 Debian 系基础镜像上使用正确的包名。Windows 平台不使用系统包管理（无映射）。

#### Scenario: 未知系统包名自动映射

- **WHEN** 依赖列表包含未知名称（归为 `DepSystemPkg`）且基础镜像为 Debian 系
- **THEN** `yum install -y <pkg>` 被翻译为 `apt-get install -y <mapped-pkg>`
- **AND** 已知映射表中 `docker` → `docker.io` 生效

#### Scenario: RHEL 系基础镜像不触发包名映射

- **WHEN** 基础镜像为 CentOS/RHEL/Fedora 系
- **THEN** 原始 `yum` 命令保持不变，不进行包名转换

## ADDED Requirements

### Requirement: Windows 依赖安装使用 PowerShell 命令

系统 SHALL 为 FamilyWindows 基础镜像生成 PowerShell 安装命令，不使用系统包管理器：
- 所有下载使用 `Invoke-WebRequest -Uri <url> -OutFile <path>`
- 压缩包解压使用 `Expand-Archive -Path <src> -DestinationPath <dst>`
- 环境变量使用 `$env:VAR = "value"` 语法
- 不生成任何 `yum`、`apt-get`、`choco` 命令

#### Scenario: Windows 镜像安装非系统包依赖

- **WHEN** 基础镜像为 FamilyWindows 且依赖列表包含 `claude`（npm 依赖）
- **THEN** Node.js 通过 `Invoke-WebRequest` 下载 MSI 静默安装
- **AND** `npm install -g @anthropic-ai/claude-code` 通过 `RUN npm install ...` 执行

#### Scenario: Windows 镜像无系统包命令

- **WHEN** 基础镜像为 FamilyWindows
- **THEN** 生成的 Dockerfile 不含任何 `yum`、`apt-get` 命令
- **AND** 所有安装命令均为 PowerShell 语法

### Requirement: adaptCommandForFamily 支持 Windows 命令翻译

`adaptCommandForFamily` 函数 SHALL 在 FamilyWindows 下将 Linux 命令翻译为 PowerShell 等价命令：
- `curl -fsSL <url> -o <path>` → `Invoke-WebRequest -Uri <url> -OutFile <path>`
- `tar -xzf <file>` → `Expand-Archive -Path <file> -DestinationPath <dest>`
- `bash <script>` → `powershell -File <script>`
- `chmod +x <file>` → 跳过（Windows 不需要可执行位）

#### Scenario: curl 命令翻译为 Invoke-WebRequest

- **WHEN** 安装命令为 `curl -fsSL https://example.com/tool.tar.gz -o /tmp/tool.tar.gz` 且 Family 为 FamilyWindows
- **THEN** 翻译为 `Invoke-WebRequest -Uri https://example.com/tool.tar.gz -OutFile C:\tmp\tool.tar.gz`

#### Scenario: Linux 命令在 Linux 家族中保持不变

- **WHEN** 安装命令为 `curl -fsSL ...` 且 Family 为 FamilyDebian 或 FamilyRHEL
- **THEN** 命令原文保留
