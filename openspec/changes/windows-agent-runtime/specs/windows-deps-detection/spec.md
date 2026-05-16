## ADDED Requirements

### Requirement: deps 命令根据镜像选择检测脚本

系统 SHALL 在 `depsinspector.RunDetection` 中根据传入的 `-i` 镜像引用名称推断平台，生成并执行对应 shell 的检测脚本：
- 推断为 Windows 时生成 PowerShell 检测脚本（`Get-Command`、`Get-ItemProperty` 检测工具）
- 推断为 Linux 时使用现有 bash 检测脚本

#### Scenario: Windows 镜像使用 PowerShell 检测脚本

- **WHEN** 用户执行 `agent-forge deps -i mcr.microsoft.com/powershell:lts-nanoserver-1809`
- **THEN** 系统生成 PowerShell 脚本在临时容器中执行
- **AND** 输出 agent/runtime/tool 分类的安装状态

#### Scenario: Linux 镜像行为不变

- **WHEN** 用户执行 `agent-forge deps`（默认 agent-forge:latest 镜像）
- **THEN** 系统使用现有 bash 检测脚本（行为不变）

### Requirement: Windows 依赖检测覆盖同等范围

系统 SHALL 保证 Windows PowerShell 检测脚本覆盖与 Linux bash 脚本相同的三类依赖：agent、runtime、tool。

#### Scenario: Windows 容器中检测 Node.js

- **WHEN** 在 Windows 容器中运行 deps 检测
- **THEN** 检测脚本通过 `Get-Command node` 检查 Node.js 是否安装
- **AND** 通过 `node --version` 获取版本号

#### Scenario: Windows 容器中检测 Git

- **WHEN** 在 Windows 容器中运行 deps 检测
- **THEN** 检测脚本通过 `Get-Command git` 检查 Git 是否安装
- **AND** 通过 `git --version` 获取版本号
