## Why

AgentForge 目前没有自动化发布流水线，所有构建依赖手动 `go build`，无法高效产出 Windows/Linux、x86/ARM 多平台发布包。每次发版需要人工在多个平台上分别编译，流程繁琐且易出错。引入 GoReleaser 可实现一键多平台构建与发布，标准化交付流程。

## What Changes

- 新增 `.goreleaser.yaml` 配置文件，定义多平台构建矩阵（Windows/Linux x amd64/arm64）
- 配置 ldflags 自动注入版本号、GitHash、构建时间到二进制
- 生成 tar.gz/zip 归档包，附带 LICENSE 和 README
- 生成 Linux 包管理器格式（deb/rpm）用于系统级安装，支持全新安装、升级和卸载
- 生成 Windows MSI 安装包，支持全新安装、升级（UpgradeCode 恒定 + MajorUpgrade 策略）和卸载
- 通过 GitHub Release 或本地 snapshot 模式发布产物
- 新增发布流程文档（RELEASE.md），说明如何执行一键发布及各平台安装/升级/卸载方法

## Capabilities

### New Capabilities

- `goreleaser-config`: GoReleaser 构建配置，定义多平台构建矩阵、归档格式、ldflags 注入、deb/rpm/MSI 包管理器集成及升级策略
- `release-automation`: 一键发布自动化流程，支持 snapshot 本地验证和正式 release 发布，覆盖全新安装、升级和卸载验证

### Modified Capabilities

<!-- 无现有 spec 需要修改 -->

## Impact

- 新增文件：`.goreleaser.yaml`、`RELEASE.md`
- 影响范围：无现有代码修改，仅新增构建配置和文档
- 依赖：GoReleaser CLI + wixl（MSI 构建），无需新增 Go 依赖
- 构建产物：`dist/` 目录（已加入 `.gitignore`）
