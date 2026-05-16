## 0. 基础设施准备

- [x] 0.1 安装 `go.uber.org/mock`：`go get go.uber.org/mock`，添加至 `go.mod`
- [x] 0.2 为需要 mock 的模块定义接口（`dockerhelper` 已有 `Client` 结构体，需提取 `DockerAPIClient` 接口供 mockgen 生成）
- [x] 0.3 在各模块目录创建 `generate.go`，通过 `//go:generate mockgen` 指令自动生成 mock 文件

## 1. Layer 1：纯函数模块测试补充（无外部依赖，纯单元测试）

- [x] 1.1 `shared/configresolver`：补充配置路径解析测试，覆盖默认路径、自定义路径、不存在路径、环境变量覆盖等场景（52% → 100%）
- [x] 1.2 `shared/logging`：补充日志级别过滤、日志格式化、多 logger、Init 函数边界条件等测试（64% → 100%）
- [x] 1.3 `shared/argsparser`：补充参数解析默认值、短/长标志、参数数组、边界值校验等测试（69% → 100%）
- [x] 1.4 `shared/progress`：补充进度条边界值、spinner 符号渲染、TTY 模式分支等测试（77% → 97%）
- [x] 1.5 `build/dockerfilegen`：补充镜像家族检测、运行时依赖分析、Debian/RHEL 两系 Dockerfile 生成差异等测试（78% → 99%）
- [x] 1.6 `build/depsmodule`：补充依赖展开、安装方式解析、无效依赖错误处理等测试（83% → 100%）

## 2. Layer 2：Docker 依赖模块 — 接口抽象 + mockgen 驱动测试

- [x] 2.1 `shared/dockerhelper`：提取 `DockerAPIClient` 接口，用 mockgen 生成 mock；补充 Ping、Info、ImageList、ImageBuild、ContainerCreate、ImageTag、ImageRemove 等所有方法的单元测试（34% → ≥90%）
- [x] 2.2 `build/buildengine`：基于 dockerhelper mock，补充构建参数校验、重试逻辑、重建模式、错误类型、构建输出解析等测试（84% → 92.8%）
- [x] 2.3 `run/runengine`：基于 dockerhelper mock，补充容器配置组装、端口映射、挂载生成、环境变量处理、Wrapper 脚本注入等测试（71% → 92.7%）
- [x] 2.4 `endpoint/applysyncer`：补充端点同步、配置差异检测、错误恢复等测试（73% → 97%）
- [x] 2.5 `endpoint/endpointmanager`：补充端点 CRUD 操作、provider 过滤、健康检查结果处理等测试（73% → 99%）
- [x] 2.6 `doctor/diagnosticengine`：补充各诊断项检查逻辑、结果汇总、阈值判定等测试（70% → 100%）
- [x] 2.7 `doctor/packagemanager`：补充包管理器检测、版本解析、已安装包列表解析等测试（71% → 100%）
- [x] 2.8 `deps/depsinspector`：补充依赖扫描、嵌套解析、配置解析、错误处理等测试（89% → 100%）
- [x] 2.9 `distribution/engine`：补充镜像导出/导入流程、校验逻辑、压缩处理等测试（86% → 100%）

## 3. Layer 3：CLI 入口层测试补充

- [x] 3.1 `cmd`：补充各子命令参数校验、错误信息输出、退出码、帮助文本完整性等测试（33% → 90.7%）

## 4. CI/CD 覆盖率门禁配置

- [x] 4.1 创建 `.github/workflows/ci.yml`，配置 push 和 PR 触发（`on: [push, pull_request]`）
- [x] 4.2 配置 Go 环境安装、module 缓存（`actions/setup-go@v5` + `actions/cache`）
- [x] 4.3 添加 `go test -short -race -coverprofile=coverage.out -covermode=atomic ./...` step
- [x] 4.4 编写 `scripts/check-coverage.sh`，解析 `go tool cover -func=coverage.out` 输出，按包校验 ≥90% 阈值，任一不达标则退出码 1
- [x] 4.5 CI 中 `scripts/check-coverage.sh` 作为 test 之后的下一个 step 运行，不达标时阻断流水线（exit code 1）
- [x] 4.6 编写分支保护规则说明（`.github/COVERAGE_GATE.md`），指导维护者如何在 GitHub repo settings 中配置 required status check

## 5. 验证与收尾

- [ ] 5.1 运行 `go generate ./...` 验证所有 mockgen 生成的 mock 文件是最新的
- [ ] 5.2 运行 `go test -short -race ./...` 确保所有新测试通过
- [ ] 5.3 运行 `go test -short ./... -coverprofile=coverage.out && go tool cover -func=coverage.out` 验证各模块 ≥90%、总体 ≥90%
- [ ] 5.4 本地执行 `bash scripts/check-coverage.sh` 验证门禁脚本返回 0
- [ ] 5.5 提交所有变更并验证 git status 干净
