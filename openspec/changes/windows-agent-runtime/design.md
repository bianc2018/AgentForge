## Context

AgentForge 当前仅支持 Linux 容器（CentOS 7/Ubuntu），所有代码层均硬编码 Linux 行为：
- `dockerfilegen.Generate` 仅识别 RHEL/Debian 家族，CMD 固定为 `/bin/bash`
- `runengine.buildCmd` 始终使用 `bash -c` 执行命令
- `runengine.Engine.Run` 使用 Unix `term.SetRawTerminal` 和 `syscall` 信号处理
- `dockerhelper.Client.ContainerCreate` 传递 `nil` platform（Docker daemon 默认 Linux）
- `toContainerPath` 已有 WSL 路径转换，但假设容器内为 Linux 路径
- `depsinspector` 检测脚本为 bash，无法在 Windows 容器中运行
- `diagnosticengine` 三层诊断全部面向 Linux Docker daemon
- `update/engine` 假设 Linux 二进制路径

本设计目标是在非破坏性前提下引入 Windows 容器支持，所有命令根据 `-b` 基础镜像自动推断目标平台。

## Goals / Non-Goals

**Goals:**
- 根据 `-b`/`--base-image` 参数自动推断目标平台（Linux/Windows），不新增 `--platform` flag
- 未指定 `-b` 时根据 Docker daemon OSType 选择默认基础镜像和平台
- PowerShell 作为 Windows 容器默认交互 shell
- Dockerfile 生成器支持 Windows 基础镜像和 PowerShell 命令语法
- 终端交互在 Windows 平台上正常工作
- `deps`/`doctor`/`update` 三个命令同步适配 Windows
- 现有 Linux 功能 100% 兼容，不破坏任何已有测试

**Non-Goals:**
- Docker-in-Docker 模式在 Windows 上不支持
- 不在本次变更中支持 Windows 宿主机构建（`build` 命令仍在 Linux Docker daemon 上执行，仅生成的镜像目标平台不同）

## Decisions

### Decision 1: 平台推断——从 `-b` 参数而非 `--platform` flag

**选择**: 根据 `-b`/`--base-image` 参数值推断平台：
- 镜像名称含 `windows`、`nanoserver`、`servercore` 关键词 → Windows
- 镜像名称含 `centos`、`ubuntu`、`debian`、`rhel`、`fedora`、`alpine` 关键词 → Linux
- 镜像来自 `mcr.microsoft.com/windows/*` → Windows
- 其他所有情况 → Linux（回退）
- 未指定 `-b` 时以 Docker daemon OSType 为准选择默认镜像

**替代方案**: 独立的 `--platform` flag → 增加用户心智负担，且平台本身是镜像属性，应由镜像推导。

**理由**: 用户选择镜像时已经隐含了平台意图——选 `ubuntu:22.04` 就是 Linux，选 `mcr.microsoft.com/powershell:...` 就是 Windows。让系统自动推断消除冗余输入和出错可能。

### Decision 2: Windows 默认镜像

**选择**: 
- 默认 Linux: 保持 `docker.1ms.run/centos:7`
- 默认 Windows: `mcr.microsoft.com/powershell:lts-nanoserver-1809`

**理由**: Nanoserver 比 ServerCore 小 ~4GB，且已预装 PowerShell。不足的工具（git、curl）在 Dockerfile 生成阶段按需下载。

**备选镜像供用户通过 `-b` 覆盖**:
| 镜像 | 适用场景 |
|------|----------|
| `mcr.microsoft.com/powershell:lts-nanoserver-1809` | 默认，轻量交互 |
| `mcr.microsoft.com/powershell:lts-windowsservercore-1809` | 需要完整工具链 |
| `mcr.microsoft.com/windows/servercore:ltsc2022` | 最兼容，需自装 PowerShell |
| `mcr.microsoft.com/windows/nanoserver:ltsc2022` | 极小镜像，不推荐（无 PowerShell） |

### Decision 3: ImageFamily 扩展

**选择**: 在 `ImageFamily` 枚举中新增 `FamilyWindows`，对应：
- `SHELL ["powershell", "-Command"]` 指令
- `Invoke-WebRequest` + `Expand-Archive` 替代 `curl` + `tar`
- `$env:VAR = "value"` 环境变量语法
- `CMD ["powershell"]` 替代 `CMD ["/bin/bash"]`

**理由**: `ImageFamily` 已用于区分安装命令语法，Windows 是该模式的自然延伸。

### Decision 4: Shell 适配策略

**选择**: 在 `buildCmd` 和 Dockerfile `CMD` 中根据推断的平台选择 shell：
- Linux: `CMD ["/bin/bash"]`, `bash -c "<cmd>"`
- Windows: `CMD ["powershell"]`, `powershell -Command "<cmd>"`

**理由**: Windows 容器的标准 shell 是 PowerShell，`cmd.exe` 功能受限不适合交互式 agent。

### Decision 5: 终端和信号处理

**选择**: 在 RunEngine 中根据平台分支：
- Linux: 保持现有 `term.SetRawTerminal` + `ContainerKill` 信号转发
- Windows: Docker SDK TTY 模式自动处理 ConPTY，`ContainerStop` 替代 `ContainerKill`

**理由**: Docker Windows 容器 attach 流仍支持 TTY 模式。Windows 容器不支持 POSIX 信号，`ContainerStop` 向容器发送 `CtrlBreak` 事件是最接近的等价行为。

### Decision 6: deps/doctor/update 适配范围

**选择**:
- **deps**: `depsinspector.RunDetection` 根据镜像名称生成 bash 或 PowerShell 检测脚本，在临时容器中执行
- **doctor**: `diagnosticengine` 新增第四层检查——平台兼容性（基础镜像与 daemon OS 是否匹配），`Info()` API 已返回 OSType
- **update**: `update/engine` 检测当前 OS，Windows 下处理 `.exe` 路径和文件替换

**理由**: 这三个命令是 build/run 的自然延伸，纳入同一变更避免功能割裂。

## Risks / Trade-offs

- **[R] Windows 容器镜像拉取慢**（首次约 1-5GB）→ M: 构建时给出进度提示
- **[R] Windows 容器仅能在 Windows Docker 主机上运行** → M: 平台推断后在 ContainerCreate 前校验，不兼容组合给出清晰错误
- **[R] Nanoserver 缺少基础工具** → M: Dockerfile 按需添加 `Invoke-WebRequest` 下载
- **[R] 测试需 Windows Docker 环境** → M: 单元测试 mock 隔离，集成/E2E 在 Windows runner 上条件运行
- **[R] `--dind` 模式不兼容** → M: Windows 平台下忽略 `--docker` flag 并输出警告
- **[R] 镜像名推断可能误判**（如自定义镜像名含 windows 但非 Windows 镜像）→ M: 覆盖率 99%+ 的正确推断，边缘情况用户可换镜像名或提 issue
