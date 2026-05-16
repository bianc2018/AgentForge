## Why

当前项目整体测试覆盖率仅 68.9%，多个核心模块（`dockerhelper` 34%、`cmd` 33%、`configresolver` 52%、`logging` 64%）远低于健康标准。同时项目缺少 CI/CD 流水线，代码提交无自动化质量门禁。低覆盖率导致重构风险高、回归 bug 难以及时发现。本次变更将覆盖率提升至各模块 >90%、总体 >90%，并建立覆盖率门禁防止后续劣化。

## What Changes

- 为 `dockerhelper`、`configresolver`、`logging`、`argsparser`、`cmd`、`runengine`、`endpointmanager`、`applysyncer`、`diagnosticengine`、`packagemanager`、`progress`、`dockerfilegen` 等低覆盖模块补充单元测试，使各模块覆盖率达到 90% 以上
- 添加 GitHub Actions CI 流水线配置，包含 lint、test、coverage 检查
- 配置覆盖率门禁：PR 合并前必须通过覆盖率阈值检查（各模块 ≥90%，总体 ≥90%）
- 在根目录添加 `.github/workflows/ci.yml`

## Capabilities

### New Capabilities

- `unit-test-coverage`: 为当前低于 90% 覆盖率的模块编写单元测试，将各模块覆盖率提升至 ≥90%，总体覆盖率提升至 ≥90%
- `ci-coverage-gate`: 配置 GitHub Actions CI/CD 流水线，在 PR 和 push 时自动运行测试并检查覆盖率阈值，不达标则阻断合并

### Modified Capabilities

<!-- 本次不修改现有 spec，仅补充测试和 CI 配置 -->

## Impact

- 受影响代码：`internal/shared/dockerhelper/`、`internal/shared/configresolver/`、`internal/shared/logging/`、`internal/shared/argsparser/`、`internal/shared/progress/`、`cmd/`、`internal/run/runengine/`、`internal/endpoint/endpointmanager/`、`internal/endpoint/applysyncer/`、`internal/doctor/diagnosticengine/`、`internal/doctor/packagemanager/`、`internal/build/dockerfilegen/`
- 新增文件：各模块的 `*_test.go` 补充测试文件、`.github/workflows/ci.yml`
- 依赖：无新增外部依赖，使用 Go 标准 `testing` 包和现有 test 框架
- CI/CD：新增 GitHub Actions 配置，需要仓库开启 Actions 功能
