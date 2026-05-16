## Context

`runengine.Run()` 方法（`engine.go:307-325`）在交互模式下调用 `ContainerAttach` 获取了 `HijackedResponse`，但未执行任何 I/O 操作便立即返回。Docker Engine API 的 attach 端点返回一个多路复用流（multiplexed stream），需要：
1. 从 `attachResp.Reader` 读取并解复用为 stdout/stderr
2. 将用户 stdin 写入 `attachResp.Conn`
3. 在 TTY 模式下切换终端为原始模式（raw mode）以支持控制字符传递

`moby/term` 包提供跨平台（Linux/macOS/Windows）的终端操作，`docker/docker/pkg/stdcopy` 提供 Docker 多路复用流的解复用。

## Goals / Non-Goals

**Goals:**
- 交互模式下用户可以看到容器输出（stdout/stderr）
- 交互模式下用户输入可以传递到容器（stdin）
- 支持终端原始模式，正确处理 Ctrl+C、Tab 等控制键
- 支持终端窗口大小变化转发

**Non-Goals:**
- 不修改后台命令模式（`--run`）——该模式已正常工作
- 不添加 Windows ConPTY 的特殊处理（moby/term 已抽象跨平台差异）
- 不修改 Docker API 客户端层

## Decisions

### Decision 1: 使用 goroutine 进行双向流拷贝

**选择**: 启动两个 goroutine：一个执行 `io.Copy(attachResp.Conn, os.Stdin)`，另一个执行 `stdcopy.StdCopy(os.Stdout, os.Stderr, attachResp.Reader)`。

**原因**: Docker 的 HijackedResponse 的 Conn（net.Conn）和 Reader 需要同时读写，单线程会死锁。

**替代方案考虑**: 使用 `docker/cli/command/container` 包中的 `AttachStreams` —— 但它依赖 `docker/cli` 的大量上下文，引入成本高。

### Decision 2: 使用 moby/term 而非手动处理终端

**选择**: 引入 `github.com/moby/term`，在交互模式下调用 `term.SetRawTerminal(os.Stdin.Fd())`，defer `term.RestoreTerminal()`。

**原因**: moby/term 是 Docker 生态的标准终端库，自动处理跨平台差异（Windows 用 `SetConsoleMode`，Unix 用 `MakeRaw`）。

### Decision 3: 信号处理——SIGINT/SIGTERM 直接退出

**选择**: 监听 SIGINT 和 SIGTERM，收到后调用 `ContainerKill` 发送信号到容器，然后退出。不监听 SIGWINCH（窗口变化转发作为后续增强）。

**原因**: 核心需求是让用户看到输出并与之交互。窗口 resize 是高级特性，可延后实现。信号转发确保 Ctrl+C 行为符合用户预期。

## Risks / Trade-offs

- **Windows ConPTY 兼容性**: Windows 10+ 的 ConPTY 与传统控制台行为不同 → Mitigation: moby/term 已处理此差异
- **goroutine 泄漏**: 如果 attach 连接异常关闭，goroutine 可能泄漏 → Mitigation: 在 function return 时 `attachResp.Close()` 会触发 goroutine 中的 I/O 出错退出
