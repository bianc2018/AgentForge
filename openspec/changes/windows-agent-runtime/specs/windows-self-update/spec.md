## ADDED Requirements

### Requirement: update 命令检测宿主平台

系统 SHALL 在 `update/engine.Update` 中通过 `runtime.GOOS` 检测当前宿主操作系统。

#### Scenario: Windows 宿主检测

- **WHEN** 在 Windows 宿主上执行 `agent-forge update`
- **THEN** 系统识别 `runtime.GOOS == "windows"`
- **AND** 使用 `.exe` 后缀查找和替换二进制

#### Scenario: Linux 宿主行为不变

- **WHEN** 在 Linux 宿主上执行 `agent-forge update`
- **THEN** 系统行为与现有逻辑完全一致

### Requirement: Windows 自更新二进制替换

系统 SHALL 在 Windows 宿主上下载新版本后，将 `.exe` 文件替换到当前进程的可执行路径。

#### Scenario: Windows 更新成功

- **WHEN** 在 Windows 上执行 `agent-forge update`，有新版本可用
- **THEN** 系统下载 `agent-forge.exe` 并替换当前运行的 `agent-forge.exe`
- **AND** 备份原文件为 `agent-forge.exe.bak`
- **AND** 更新失败时自动回滚（从 `.bak` 恢复）

#### Scenario: Linux 更新不变

- **WHEN** 在 Linux 上执行 `agent-forge update`
- **THEN** 系统行为不变：下载、替换二进制（无后缀）、备份回滚
