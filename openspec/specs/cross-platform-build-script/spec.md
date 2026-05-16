## Purpose

定义跨平台一键编译发布脚本的功能行为，覆盖 Linux/macOS（bash）和 Windows（batch）两种实现。

## Requirements

### Requirement: Linux 一键构建发布

`scripts/build-release.sh` SHALL 在 Linux/macOS 上提供一键构建发布能力，自动校验环境并调用 goreleaser 完成多平台构建。

#### Scenario: 默认 snapshot 构建成功

- **WHEN** 用户在项目根目录执行 `bash scripts/build-release.sh`
- **THEN** 脚本校验 Go ≥1.21 和 goreleaser 可用后，调用 `goreleaser release --snapshot --clean` 完成构建
- **AND** `dist/` 目录下生成所有平台的二进制包和归档

#### Scenario: Go 版本不满足时报错

- **WHEN** 系统 Go 版本低于 1.21
- **THEN** 脚本输出明确错误信息 "Go 版本过低，需要 Go ≥1.21，当前版本: X.Y.Z" 并以非零退出码退出
- **AND** 给出安装指引链接

#### Scenario: goreleaser 不可用时给出安装指引

- **WHEN** `goreleaser` 不在 PATH 中
- **THEN** 脚本输出 "goreleaser 未安装" 并给出安装命令

#### Scenario: 指定版本号构建

- **WHEN** 用户执行 `bash scripts/build-release.sh --version v1.2.3 --release`
- **THEN** 脚本将版本号注入环境变量 `GORELEASER_CURRENT_TAG=v1.2.3` 并执行正式发布构建

### Requirement: Windows 一键构建发布

`scripts/build-release.bat` SHALL 在 Windows 上提供与 Linux 脚本等效的一键构建发布能力。

#### Scenario: Windows 默认 snapshot 构建

- **WHEN** 用户在项目根目录执行 `scripts\build-release.bat`
- **THEN** 脚本校验环境后调用 `goreleaser release --snapshot --clean` 完成构建
- **AND** `dist\` 目录下生成所有平台的二进制包

#### Scenario: Windows 下 Go 未安装时报错

- **WHEN** `go` 命令不可用
- **THEN** 脚本输出 "Go 未安装，请从 https://go.dev/dl/ 下载安装" 并以错误码退出

### Requirement: 脚本可执行权限

Linux 脚本 SHALL 具有可执行权限，支持直接 `./scripts/build-release.sh` 运行，无需显式指定 bash 解释器。

#### Scenario: 直接执行脚本

- **WHEN** 用户执行 `./scripts/build-release.sh`
- **THEN** 脚本正常运行，行为与 `bash scripts/build-release.sh` 一致
