## 1. Progress 组件包

- [ ] 1.1 创建 `internal/shared/progress` 包结构和接口（Writer 接口定义 + TTY 检测函数）
- [ ] 1.2 实现 `ProgressLog` 模式 —— 将文本行直接透传写入底层 io.Writer，无格式化
- [ ] 1.3 实现 `ProgressBar` 模式 —— 基于步骤数的百分比进度条（TTY 下 ANSI 原地刷新，非 TTY 下每步一行文本）
- [ ] 1.4 实现 `Spinner` 模式 —— 旋转动画指示器（TTY 下 `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏` 帧动画 100ms 间隔，非 TTY 下每 5s 一次文本更新）

## 2. 构建引擎流式输出重构

- [ ] 2.1 在 `BuildParams` 中新增 `ProgressWriter io.Writer` 字段，用于接收外部注入的输出 writer
- [ ] 2.2 实现 `io.Copy` + `io.MultiWriter` 流式复制，将 Docker 构建流同时写入 ProgressWriter 和内部 outputBuf
- [ ] 2.3 包装自定义 `contextAwareWriter`，在每次 Write 前检查 context 是否取消，取消则返回错误中断 io.Copy
- [ ] 2.4 将 `Build()` 方法中的 `io.ReadAll(resp.Body)` 替换为流式 `io.Copy` + context 检查模式
- [ ] 2.5 在重试循环中确保每次重试间隔（`time.After`）前都检查 context 取消

## 3. Run 引擎 context 响应修复

- [ ] 3.1 排查 `runengine` 中 `ContainerAttach` 返回的 `HijackedResponse` 流读取路径，识别所有阻塞 I/O 点
- [ ] 3.2 对长时间容器输出流读取增加 context 检查和流式输出

## 4. 命令层集成

- [ ] 4.1 更新 `cmd/build.go`：向 `BuildParams` 传入 `os.Stdout` 作为 ProgressWriter，移除 `fmt.Print(output)` 一次性输出
- [ ] 4.2 确保构建失败时错误输出仍包含完整日志（从 outputBuf 获取）

## 5. 测试与验证

- [ ] 5.1 编写 `progress` 包的单元测试（TTY/非 TTY 模式切换、Bar/Spinner/Log 行为验证）
- [ ] 5.2 更新 `buildengine` 单元测试和安全性测试：验证流式输出到注入的 writer、context 取消能中断 io.Copy
- [ ] 5.3 编写 `contextAwareWriter` 的单元测试（正常写入 + context 取消场景）
- [ ] 5.4 运行 `go test ./...` 确认所有测试通过
- [ ] 5.5 端到端手动验证：执行 `go run main.go build -d all` 观察实时输出，Ctrl+C 验证即时中断
