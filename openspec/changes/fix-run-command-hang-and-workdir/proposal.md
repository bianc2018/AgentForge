## Why

`agent-forge run`（默认交互 bash 模式）存在两个用户体验缺陷：1) 启动后终端无输出，看起来像"卡住了"，直到用户按 Ctrl+C 才出现提示符；2) 进入容器后工作目录是 `/` 而非预期的挂载目录，用户需手动 `cd` 到工作路径。这两个问题严重影响首次使用体验。

## What Changes

- 修复交互模式下容器启动后 bash 提示符不立即显示的问题（根因：缺少终端尺寸同步 + 无初始输出触发）
- 为容器设置合理默认工作目录：`-w` 显式指定时使用 `-w` 值，否则自动将主机 `$PWD` 绑定挂载到容器同路径并设为工作目录（`os.Getwd()` 失败时 fallback `/workspace`）。`-m` 只读挂载不参与工作目录推导
- 在容器配置中注入 `TERM` 环境变量以确保 bash 输出正确的转义序列

## Capabilities

### New Capabilities

- `default-container-workdir`: 为交互容器提供合理的默认工作目录逻辑（`-w` 显式指定 > 主机 PWD 自动挂载 > `/workspace` 兜底；`-m` 只读挂载不参与）

### Modified Capabilities

- `interactive-container-io`: 修改容器的交互 I/O 初始化流程 —— 附加后立即同步终端尺寸，注入 `TERM` 环境变量，确保 bash 提示符在附加完成后立即可见

## Impact

- `internal/run/runengine/engine.go`：AssembleContainerConfig（设置 WorkingDir/TERM）、Run（附加后 ContainerResize）
- `internal/shared/dockerhelper/dockerhelper.go`：可能需新增 ContainerResize 封装（若未实现）
