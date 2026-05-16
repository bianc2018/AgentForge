## 1. 依赖与导入

- [x] 1.1 将 `github.com/moby/term` 添加到 `go.mod`（`go get github.com/moby/term`）

## 2. 核心实现：交互模式流拷贝

- [x] 2.1 在 `runengine.Run()` 的交互分支（`ContainerAttach` 后）实现双向流拷贝：`stdcopy.StdCopy` 解复用 stdout/stderr → 用户终端，`io.Copy` 转发 stdin → 容器
- [x] 2.2 添加终端原始模式设置（`term.SetRawTerminal` / `term.RestoreTerminal`），处理 Ctrl+C 等控制字符
- [x] 2.3 添加 SIGINT/SIGTERM 信号监听，用户中断时转发到容器

## 3. 验证

- [x] 3.1 运行 `go test ./internal/run/...` 确保测试通过
- [x] 3.2 执行 `go run . run` 验证交互模式可看到 bash 输出
