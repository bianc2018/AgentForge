## Why

当前项目构建依赖手动执行 `goreleaser release --snapshot --clean`，开发者需要记住命令、确认环境配置正确（Go 版本、PATH 等）。缺少一个统一入口脚本，能自动校验环境、处理跨平台差异（Linux `.sh` / Windows `.bat`），让新开发者或 CI 系统一键完成构建发布。

## What Changes

- 新增 `scripts/build-release.sh`（Linux/macOS），提供一键编译发布能力
- 新增 `scripts/build-release.bat`（Windows），提供等效的一键编译发布能力
- 脚本自动校验前置条件（Go ≥1.21、goreleaser 可用），环境不满足时给出明确指引
- 支持参数化控制：版本号、是否跳过测试、是否生成 snapshot

## Capabilities

### New Capabilities

- `cross-platform-build-script`: 跨平台一键编译发布脚本，自动校验环境、调用 goreleaser 完成多平台构建，在 Linux 和 Windows 上提供一致的操作体验

### Modified Capabilities

<!-- 无已有 specs 需要修改 -->

## Impact

- `scripts/build-release.sh`：新增，Linux/macOS 构建脚本
- `scripts/build-release.bat`：新增，Windows 构建脚本
- 不修改任何已有代码或配置
