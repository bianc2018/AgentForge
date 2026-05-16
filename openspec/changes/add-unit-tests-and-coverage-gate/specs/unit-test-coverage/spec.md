## ADDED Requirements

### Requirement: 各模块单元测试覆盖率达到 90%

系统 SHALL 确保每个 Go 包的单元测试覆盖率 ≥90%，通过 `go test -cover` 测量语句覆盖率。

#### Scenario: 纯函数模块覆盖率达标

- **WHEN** 运行 `go test -cover ./internal/shared/argsparser/`
- **THEN** 语句覆盖率 ≥90%

#### Scenario: Docker 依赖模块覆盖率达标（接口抽象）

- **WHEN** 运行 `go test -cover ./internal/shared/dockerhelper/`
- **THEN** 通过接口 stub 替代真实 Docker，语句覆盖率 ≥90%

#### Scenario: CLI 入口模块覆盖率达标

- **WHEN** 运行 `go test -cover ./cmd/`
- **THEN** 各命令的参数解析、错误处理逻辑覆盖率 ≥90%

### Requirement: 整体项目覆盖率达到 90%

系统 SHALL 确保项目总体语句覆盖率 ≥90%，通过 `go test -coverprofile` 汇总计算。

#### Scenario: 全量测试覆盖率汇总

- **WHEN** 运行 `go test -short ./... -coverprofile=coverage.out -covermode=atomic`
- **THEN** `go tool cover -func=coverage.out` 显示的 total 覆盖率 ≥90%

### Requirement: 测试断言完整性

系统 SHALL 确保每个单元测试包含明确的断言，而非仅调用函数不检查返回值。

#### Scenario: 函数返回值断言

- **WHEN** 测试调用一个返回 `(string, error)` 的函数
- **THEN** 测试 MUST 同时检查返回值内容和 error 状态

#### Scenario: 边界条件断言

- **WHEN** 测试调用参数校验函数
- **THEN** 测试 MUST 覆盖有效输入、无效输入、空输入和边界值

### Requirement: 接口抽象隔离外部依赖

系统 SHALL 使用 Go 接口抽象 Docker、文件系统等外部依赖，在单元测试中使用 stub/fake 实现替代。

#### Scenario: Docker 操作通过接口注入

- **WHEN** 编写 dockerhelper 模块的单元测试
- **THEN** 测试不使用真实 Docker daemon，使用实现相同接口的 stub 客户端

#### Scenario: 文件系统操作通过接口注入

- **WHEN** 编写 logging 或 configresolver 模块的单元测试
- **THEN** 测试使用临时目录（`t.TempDir()`）而非真实配置路径
