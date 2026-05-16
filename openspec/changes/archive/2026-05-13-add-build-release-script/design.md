## Context

当前项目使用 goreleaser（`.goreleaser.yaml`）进行多平台构建，开发者需手动执行 `goreleaser release --snapshot --clean`。`scripts/` 目录下已有 `build-msi.sh`（MSI 包生成器）作为脚本化构建的先例。

上一步修复中确认：系统需 Go ≥1.21（`go-winio` 依赖要求），goreleaser 需在 PATH 中。这些前置条件目前靠开发者自行保证，新成员或 CI 接入时容易踩坑。

## Goals / Non-Goals

**Goals:**
- 提供 `scripts/build-release.sh`（Linux/macOS）和 `scripts/build-release.bat`（Windows），实现一键构建发布
- 自动校验前置条件（Go 版本 ≥1.21、goreleaser 可用），不满足时给出可操作的错误信息和安装指引
- 支持参数：`--version`（版本号）、`--snapshot`（快照模式，默认开启）、`--skip-tests`（跳过测试）
- 与现有 goreleaser 配置和 `build-msi.sh` 协作，不重复实现

**Non-Goals:**
- 不替代 goreleaser — 核心构建逻辑仍由 goreleaser 执行
- 不提供 macOS 专属适配（macOS 直接使用 Linux shell 脚本）
- 不修改 `.goreleaser.yaml` 配置

## Decisions

### Decision 1: Shell 脚本 + Batch 脚本双文件方案

**选择**: 分别编写 `build-release.sh`（bash）和 `build-release.bat`（cmd.exe），保持两种脚本等效但独立。

**原因**:
- bash 在 Linux/macOS/WSL 通用，无需额外依赖
- batch 是 Windows 原生支持，无需 PowerShell 执行策略配置
- 两个脚本各自处理平台特定的 PATH、命令语法差异
- 保持简单，不引入跨平台脚本运行时（如 Python/Node）

**替代方案考虑**:
- 单一 Makefile：Windows 原生不支持 make
- PowerShell 单脚本：需额外配置执行策略，且 Linux 上需安装 PowerShell
- Python 脚本：增加 Python 运行时依赖

### Decision 2: 默认 Snapshot 模式

**选择**: 脚本默认以 `--snapshot` 模式运行 goreleaser，通过 `--release` 参数切换到正式发布模式。

**原因**: snapshot 是日常开发最常用的模式（跳过 tag 校验和发布上传），降低误操作风险。

### Decision 3: 环境校验前置

**选择**: 在执行 goreleaser 之前，先校验 Go 版本和 goreleaser 可用性。

**原因**: 本项目的 go.mod 声明 `go 1.23`，`go-winio` 依赖要求 ≥1.19。提前校验比 goreleaser 报 `no main function` 模糊错误更有帮助。

## Risks / Trade-offs

- **batch 脚本能力有限**：字符串处理、错误处理不如 bash → Mitigation: 保持 batch 逻辑简单，仅做环境校验 + 参数转发
- **Go 版本检测可能与实际编译版本不一致**（多版本共存时） → Mitigation: 脚本开头打印 `go version` 和 `go env GOPATH` 让用户可见
