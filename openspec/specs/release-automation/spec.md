## Purpose

定义项目发布流程规范——snapshot 本地验证、正式 release 发布、发布文档要求。

## Requirements

### Requirement: Snapshot mode for local verification

系统 SHALL 支持 `snapshot` 模式，在本地生成所有发布产物，不依赖 Git 干净状态、Git tag 或 GitHub Token。

#### Scenario: Snapshot builds with fake version

- **WHEN** 执行 `goreleaser release --snapshot --clean`
- **THEN** 系统 SHALL 使用占位版本号生成所有构建产物到 `dist/` 目录

#### Scenario: Snapshot skips GitHub Release

- **WHEN** 使用 `--snapshot` 标志
- **THEN** 系统 SHALL NOT 推送任何内容到 GitHub Release

#### Scenario: Clean flag removes previous dist

- **WHEN** 执行 `--clean` 标志
- **THEN** 系统 SHALL 在构建前删除 `dist/` 目录中的所有旧产物

### Requirement: Release mode for formal publishing

系统 SHALL 支持 `release` 模式，基于 Git tag 创建 GitHub Release 并上传所有构建产物。

#### Scenario: Release requires clean git state

- **WHEN** 执行 `goreleaser release --clean`（不带 `--snapshot`）
- **THEN** 系统 SHALL 验证当前 Git 状态为干净且 HEAD 指向一个 tag，若不满足则 SHALL 中止并报错

#### Scenario: Release requires GitHub token

- **WHEN** 执行正式 release
- **THEN** 系统 SHALL 检查 `GITHUB_TOKEN` 环境变量，若未设置则 SHALL 中止并提示用户

#### Scenario: Release creates GitHub Release with artifacts

- **WHEN** 正式 release 成功完成
- **THEN** 系统 SHALL 在 GitHub 上创建对应 tag 的 Release，并上传所有归档包

### Requirement: Release documentation

项目 SHALL 包含 `RELEASE.md` 文档，说明完整的发布操作流程及各平台安装/升级/卸载方法。

#### Scenario: Document covers prerequisites

- **WHEN** 开发者阅读 RELEASE.md
- **THEN** 文档 SHALL 列出前置条件：GoReleaser CLI 安装、wixl 安装、GitHub Token 配置、Git tag 规范

#### Scenario: Document covers step-by-step workflow

- **WHEN** 开发者需要执行发布
- **THEN** RELEASE.md SHALL 包含完整发布流程：打 tag → snapshot 验证 → 正式 release 的分步命令

#### Scenario: Document covers install/upgrade/uninstall per platform

- **WHEN** 用户需要安装 agent-forge
- **THEN** RELEASE.md SHALL 分别说明 Linux（deb/rpm/tar.gz）和 Windows（msi/zip）的全新安装、版本升级、卸载命令

#### Scenario: Document covers troubleshooting

- **WHEN** 开发者遇到发布问题
- **THEN** RELEASE.md SHALL 包含常见问题及其解决方案

### Requirement: Gitignore dist directory

项目 `.gitignore` 文件 SHALL 包含 `dist/` 目录，防止构建产物被提交到仓库。

#### Scenario: dist directory is ignored

- **WHEN** 构建产物生成到 `dist/`
- **THEN** `dist/` 目录 SHALL 被 `.gitignore` 忽略，不会出现在 `git status` 中
