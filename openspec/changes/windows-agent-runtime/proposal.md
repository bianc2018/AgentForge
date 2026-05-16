## Why

当前 AgentForge 仅支持 Linux 容器（CentOS 7/Ubuntu），所有运行模式硬编码 bash 作为 shell、Unix 信号处理、PTY 终端交互。Windows 用户需要在自己的 Windows 机器上原生运行 agent 容器，使用 PowerShell 作为交互 shell，而非通过 WSL 间接操作。这是 AgentForge 跨平台能力的关键缺失。

## What Changes

- 新增 Windows 基础镜像家族 `FamilyWindows`，支持 `mcr.microsoft.com/powershell:lts-nanoserver-1809` 等镜像
- Dockerfile 生成器支持 Windows 镜像的 `SHELL ["powershell", "-Command"]` 指令和 PowerShell 命令语法
- RunEngine 根据基础镜像自动选择 shell（bash 或 powershell），构建相应的启动命令
- 终端交互适配 Windows ConPTY（通过 Docker SDK 的 `ConsoleSize` 选项，信号处理使用 `ContainerStop`）
- 路径处理新增 Windows 容器内路径格式（`C:\workspace` 与 `/workspace` 自动适配）
- 依赖安装命令适配 Windows（`Invoke-WebRequest` 替代 `curl`，`Expand-Archive` 替代 `tar`）
- **不新增 `--platform` flag**：平台从 `-b`/`--base-image` 参数自动推断，未指定 `-b` 时根据 Docker daemon OSType 自动选择默认镜像和平台
- `deps` 命令：检测脚本适配 PowerShell，支持 Windows 容器内依赖检测
- `doctor` 命令：诊断引擎新增 Windows 平台检查项（Windows Docker daemon 状态、镜像兼容性）
- `update` 命令：自更新适配 Windows 二进制（`.exe` 文件替换、Windows 路径）

## Capabilities

### New Capabilities

- `windows-container-build`: 构建 Windows 容器镜像（Dockerfile 生成支持 PowerShell shell、Windows 包管理、基础镜像推断平台）
- `windows-container-run`: 在 Windows 容器中运行 agent，PowerShell 作为交互 shell，Windows 终端信号和路径适配
- `image-based-platform-inference`: 从 `-b` 基础镜像名称推断目标平台（Linux/Windows），未指定时回退到 Docker daemon OSType 推断默认镜像
- `windows-deps-detection`: `deps` 命令适配——根据镜像自动选择 bash 或 PowerShell 检测脚本
- `windows-doctor-diagnostics`: `doctor` 命令适配——新增第四层平台兼容性诊断（镜像-Daemon 匹配、Windows 容器支持检查）
- `windows-self-update`: `update` 命令适配——Windows 宿主上处理 `.exe` 路径和二进制替换

### Modified Capabilities

- `interactive-container-io`: 终端交互需区分 Unix PTY 和 Windows ConPTY，ContainerResize 和信号处理需平台感知
- `build-command`: 构建命令根据 `-b` 镜像自动生成对应平台的 Dockerfile，无需额外 flag
- `multi-family-docker-install`: 依赖安装新增 Windows 家族——用 `Invoke-WebRequest` + `Expand-Archive` 替代 `yum`/`apt`

## Impact

- **BREAKING**: `AssembleContainerConfig` 签名增加 `platform` 参数，所有调用方需更新
- `dockerfilegen.Generate` Options 的 `BaseImage` 字段承载平台推断职责
- `dockerhelper.Client` ContainerCreate 调用从 `nil` platform 变为根据基础镜像配置
- `depsmodule` 需新增 Windows 系的安装命令映射（PowerShell 语法）
- `depsinspector` 检测脚本需 Windows 版本
- `diagnosticengine` 需新增 Windows 平台诊断项
- `update/engine` 需处理 `.exe` 二进制替换
- 新增对 `mcr.microsoft.com` 镜像仓库的网络可达性依赖
- 测试需在 Windows Docker 环境下验证（Windows 容器需 Windows Docker 主机）
