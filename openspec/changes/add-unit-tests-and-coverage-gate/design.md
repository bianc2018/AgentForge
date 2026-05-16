## Context

当前项目有 20 个 Go 包，测试覆盖率分布不均：4 个包已达到 ≥90%，但 12 个包低于 90%，其中 `dockerhelper`(34%)、`cmd`(33%)、`configresolver`(52%) 严重不足。项目当前无 CI/CD 流水线，无自动化代码质量门禁。

测试基础设施：项目使用 Go 标准 `testing` 包，部分模块已有表驱动测试（table-driven tests）模式。Docker 相关模块的测试需要 Docker daemon 运行，已通过 `testing.Short()` 和 `t.Skip()` 处理集成测试隔离。

## Goals / Non-Goals

**Goals:**
- 将每个 Go 包的单元测试覆盖率提升至 ≥90%
- 将整体项目覆盖率提升至 ≥90%
- 建立 GitHub Actions CI 流水线，在 PR 时自动运行测试和覆盖率检查
- 覆盖率不达标时阻断 PR 合并

**Non-Goals:**
- 不修改生产代码逻辑（仅补充测试）
- 不添加集成测试或 E2E 测试（本次聚焦单元测试）
- 不修改 Docker 相关的集成测试策略

## Decisions

### 1. 使用 go.uber.org/mock (mockgen) 自动生成 mock

**选择**：使用 `go.uber.org/mock` + `mockgen` 自动生成接口 mock 实现。
**原因**：
- `go.uber.org/mock` 是 Go 官方维护的 mock 框架（原 `golang/mock`），社区标准，专业成熟
- `mockgen` 从接口定义自动生成 mock 代码，类型安全，支持 EXPECT() 调用序列验证
- 手写 stub 需维护大量样板代码，mockgen 一行命令即可重新生成
- 与 Go 标准 `testing` 包无缝集成，无需额外运行时依赖
**替代方案**：
- `testify/mock`：需手动编写 mock 方法，代码量大，类型安全性弱
- 手写 stub：无依赖但维护成本随接口数量线性增长，无法验证调用参数和次数

### 2. 分层测试策略

**选择**：按模块依赖层级从底向上补充测试：
- Layer 1（纯函数/无外部依赖）：`argsparser`、`configresolver`、`logging`、`depsmodule`、`dockerfilegen`
- Layer 2（依赖 Docker API 接口）：`dockerhelper`、`buildengine`、`runengine`、`endpointmanager`、`applysyncer`
- Layer 3（CLI 入口层）：`cmd`

Layer 1 可直接写纯单元测试。Layer 2 需要抽象 Docker 交互接口，通过 mockgen 生成 mock 注入测试。Layer 3 测试 CLI 参数解析和错误处理逻辑。

### 3. CI/CD 平台选择

**选择**：GitHub Actions。
**原因**：仓库托管在 GitHub，原生集成，社区生态成熟。无需额外配置凭证。
**替代方案**：Jenkins（需要自建服务器）、GitLab CI（不在 GitLab 上）。

### 4. 覆盖率门禁配置

**选择**：使用 `go test -cover` 输出 + `scripts/check-coverage.sh` 解析覆盖率，在 CI 中校验各包阈值。
**原因**：Go 原生 `-cover` 标志无需额外工具。脚本按包检查覆盖率，任一包低于 90% 则失败。
**替代方案**：`goveralls` + Coveralls 服务（引入外部服务依赖）、`golangci-lint` 的 `govet` 规则（不直接支持覆盖率）。

**`scripts/check-coverage.sh` 运行时机**：
1. **CI/GitHub Actions**：在 `go test -coverprofile` 之后作为独立 step 运行，校验覆盖率阈值，不达标返回非零退出码阻断流水线
2. **本地 pre-commit**：开发者可手动执行 `bash scripts/check-coverage.sh` 在提交前自查（可选，不强制 hook）
3. **PR 合并门禁**：作为 GitHub Branch Protection 规则中的 required status check，必须通过才能 merge

### 5. 覆盖率基线

| 包 | 当前 | 目标 |
|---|---|---|
| `provideragentmatrix` | 100% | 100% (保持) |
| `wrapperloader` | 100% | 100% (保持) |
| `argspersistence` | 96.2% | 96%+ (保持) |
| `update/engine` | 91.0% | 91%+ (保持) |
| `depsinspector` | 88.5% | ≥90% |
| `distribution/engine` | 85.7% | ≥90% |
| `build/buildengine` | 84.2% | ≥90% |
| `build/depsmodule` | 82.5% | ≥90% |
| `build/dockerfilegen` | 77.5% | ≥90% |
| `shared/progress` | 77.3% | ≥90% |
| `endpoint/applysyncer` | 72.9% | ≥90% |
| `endpoint/endpointmanager` | 72.7% | ≥90% |
| `run/runengine` | 71.4% | ≥90% |
| `doctor/packagemanager` | 70.8% | ≥90% |
| `doctor/diagnosticengine` | 69.9% | ≥90% |
| `shared/argsparser` | 68.9% | ≥90% |
| `shared/logging` | 64.3% | ≥90% |
| `shared/configresolver` | 52.4% | ≥90% |
| `shared/dockerhelper` | 34.1% | ≥90% |
| `cmd` | 33.3% | ≥90% |

## Risks / Trade-offs

- **[时间成本] 12 个模块需要大量测试代码** → 按优先级分阶段：先修最影响总体覆盖率的低覆盖大模块（dockerhelper、cmd、configresolver、logging）
- **[假阳性] 高覆盖率但低质量测试（仅调用不断言）** → 每个测试必须有明确断言，CI 中启用 `-race` 检测数据竞争
- **[Docker 依赖] 部分模块如 dockerhelper 需要 Docker daemon** → 通过接口抽象隔离，单元测试用 mockgen 生成的 mock 替代真实 Docker 调用
- **[CI 运行时间] 全量测试可能较慢** → 使用 `-short` 跳过集成测试，单元测试保持毫秒级
