## 1. 日志核心包

- [x] 1.1 创建 `internal/shared/logging/` 包：实现 `dailyWriter`（日期轮转）、`Init(configDir)` 初始化函数、级别解析、全局 logger
- [x] 1.2 添加日志包单元测试：级别解析、dailyWriter 轮转、降级行为

## 2. 命令集成

- [x] 2.1 在 `cmd/root.go` 的 `Execute()` 中调用 `logging.Init()` 初始化日志
- [x] 2.2 在 `internal/run/runengine/engine.go` 关键步骤添加日志记录（容器创建、启动、attach、信号）
- [x] 2.3 在 `internal/build/buildengine/buildengine.go` 关键步骤添加日志记录（构建开始、重试、完成）

## 3. 验证

- [x] 3.1 运行 `go test ./internal/shared/logging/...` 确保测试通过
- [x] 3.2 执行 `go run . run --run "echo hello"` 验证日志文件生成且格式正确
