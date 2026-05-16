## ADDED Requirements

### Requirement: CI 流水线自动运行测试

系统 SHALL 在 GitHub Actions 中配置 CI 流水线，在每次 push 和 PR 时自动触发测试。

#### Scenario: Push 触发 CI

- **WHEN** 开发者 push 代码到任意分支
- **THEN** GitHub Actions 自动运行 `go test -short -race ./...`

#### Scenario: PR 触发 CI

- **WHEN** 创建或更新 Pull Request
- **THEN** GitHub Actions 自动运行测试和覆盖率检查，结果作为 PR 状态检查展示

### Requirement: 覆盖率门禁阻断不合标 PR

系统 SHALL 在 CI 流水线中校验代码覆盖率，若任一模块 <90% 或总体 <90%，流水线标记为失败并阻断 PR 合并。

#### Scenario: 覆盖率达标 PR 可合并

- **WHEN** PR 的所有模块覆盖率 ≥90% 且总体覆盖率 ≥90%
- **THEN** CI 覆盖率检查通过，PR 可正常合并

#### Scenario: 覆盖率不达标 PR 被阻断

- **WHEN** PR 中某个模块覆盖率 <90%
- **THEN** CI 覆盖率检查失败，PR 合并按钮被禁用，输出未达标模块列表

#### Scenario: 总体覆盖率不达标 PR 被阻断

- **WHEN** PR 中总体覆盖率 <90%
- **THEN** CI 覆盖率检查失败，PR 合并按钮被禁用

### Requirement: 覆盖率报告可审查

系统 SHALL 在 CI 输出中提供按包的覆盖率明细，便于开发者定位覆盖率不足的模块。

#### Scenario: CI 输出覆盖率明细

- **WHEN** CI 覆盖率检查运行完成
- **THEN** 日志输出每个包的覆盖率百分比和总体覆盖率，未达标的包用醒目标记标注

### Requirement: CI 配置使用 Go 官方工具链

系统 SHALL 使用 Go 官方工具链（`go test -cover`）进行覆盖率检测，不依赖第三方覆盖率服务。

#### Scenario: 覆盖率检测无外部依赖

- **WHEN** CI 运行覆盖率检查
- **THEN** 仅使用 `go test -coverprofile` 和 `go tool cover -func` 命令，无需外部 API 调用

### Requirement: CI 缓存加速构建

系统 SHALL 配置 Go module 缓存和 build cache，减少 CI 运行时间。

#### Scenario: 依赖缓存命中

- **WHEN** CI 第二次运行且 go.mod 未变更
- **THEN** Go 依赖从缓存加载，无需重新下载
