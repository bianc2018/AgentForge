## 1. dockerhelper: 新增 ContainerResize 方法

- [x] 1.1 在 `internal/shared/dockerhelper/dockerhelper.go` 中新增 `ContainerResize(ctx context.Context, containerID string, height, width uint) error` 方法，调用 Docker Engine API `POST /containers/{id}/resize?h=<height>&w=<width>`

## 2. runengine: 修复默认工作目录

- [x] 2.1 在 `internal/run/runengine/engine.go` 的 `AssembleContainerConfig()` 中实现工作目录自动挂载逻辑：确定路径（`-w` > `os.Getwd()` > `/workspace`），将该路径以读写方式绑定挂载到容器同路径（1:1），设为 WorkingDir
- [x] 2.2 在 `AssembleContainerConfig()` 中自动检测主机 `os.Getenv("TERM")` 并注入容器，主机 TERM 为空时 fallback 到 `xterm-256color`；用户通过 `-e TERM=...` 显式覆盖时使用用户值

## 3. runengine: 修复交互模式终端尺寸同步

- [x] 3.1 在 `internal/run/runengine/engine.go` 的 `Run()` 方法中，`ContainerAttach` 成功后、`makeRaw` 前调用 `ContainerResize` 同步终端尺寸（使用 `golang.org/x/term` 获取当前终端行列数）
- [x] 3.2 ContainerResize 调用失败时记录日志但不阻塞主流程（降级处理）

## 4. 验证

- [x] 4.1 构建并运行 `go run main.go run`，确认提示符立即可见且工作目录为主机当前目录（由 `TestAssembleContainerConfig_EmptyParams` + `TestST3_NoMountNoExtra` 覆盖）
- [x] 4.2 运行 `go run main.go run -w /custom`，确认工作目录为 `/custom`（由 `TestAssembleContainerConfig_WorkingDirectory` + `TestIT6_WorkingDirectory` 覆盖）
- [x] 4.3 运行 `go run main.go run -e TERM=vt100`，确认 TERM 为用户指定值（由 `TestAssembleContainerConfig_EnvironmentVariables` + `TestIT6_EnvironmentVariables` 覆盖）
- [x] 4.4 运行 `go run main.go run -m /host/ref`，确认 `/host/ref` 为只读挂载（1:1 路径映射）且工作目录不受其影响（由 `TestAssembleContainerConfig_ReadOnlyMount` + `TestST3_SingleMountReadOnly` 覆盖）
- [ ] 4.5 交互验证：实际 `agent-forge run` 进入容器，肉眼确认 bash 提示符在附加后立即可见（需 Docker 环境 + 人工观察）
