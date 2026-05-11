## ADDED Requirements

### Requirement: GoReleaser Configuration File

项目根目录 SHALL 包含一个 `.goreleaser.yaml` 配置文件，定义多平台构建矩阵、ldflags 注入、归档格式和包管理器生成规则。

#### Scenario: Build matrix covers all target platforms

- **WHEN** 执行 `goreleaser build` 或 `goreleaser release`
- **THEN** GoReleaser SHALL 为以下目标编译二进制：linux/amd64、linux/arm64、windows/amd64、windows/arm64

#### Scenario: Version info injected at build time

- **WHEN** 二进制被编译
- **THEN** 二进制 SHALL 内嵌 `Version`（来自 Git tag）、`GitHash`（短 commit hash）和 `BuildTime`（ISO 8601 时间戳）

#### Scenario: Stripped binaries for smaller size

- **WHEN** 二进制被编译
- **THEN** ldflags 中 SHALL 包含 `-s -w` 以去除调试符号表和符号信息

#### Scenario: Linux archives as tar.gz

- **WHEN** 目标 OS 为 linux
- **THEN** GoReleaser SHALL 将二进制打包为 `tar.gz` 格式，内含二进制文件、LICENSE 和 README.md

#### Scenario: Windows archives as zip

- **WHEN** 目标 OS 为 windows
- **THEN** GoReleaser SHALL 将二进制打包为 `zip` 格式，内含 `.exe` 二进制文件、LICENSE 和 README.md

#### Scenario: Checksum file generated

- **WHEN** 归档包生成完毕
- **THEN** GoReleaser SHALL 生成 `checksums.txt` 文件，包含所有归档包的 SHA256 校验和

### Requirement: Linux Package Manager (deb/rpm)

系统 SHALL 通过 nfpm 生成 `.deb` 和 `.rpm` 包，支持全新安装、版本升级和卸载。

#### Scenario: Package installs binary to system path

- **WHEN** 用户执行 `dpkg -i agent-forge_*.deb` 或 `rpm -i agent-forge_*.rpm`
- **THEN** `agent-forge` 二进制 SHALL 安装到 `/usr/bin/agent-forge`

#### Scenario: Fresh install on clean system

- **WHEN** 目标系统未安装过任何版本的 agent-forge
- **THEN** 安装命令 SHALL 成功完成，`agent-forge --version` 可正常执行

#### Scenario: Upgrade from previous version via deb

- **WHEN** 已通过 deb 安装旧版本 `agent-forge`，用户执行 `dpkg -i agent-forge_<新版本>.deb`
- **THEN** 新版本 SHALL 覆盖旧版本，二进制替换为最新版，`agent-forge --version` 显示新版本号

#### Scenario: Upgrade from previous version via rpm

- **WHEN** 已通过 rpm 安装旧版本 `agent-forge`，用户执行 `rpm -U agent-forge_<新版本>.rpm`
- **THEN** 新版本 SHALL 替换旧版本，`rpm -q agent-forge` 显示新版本号

#### Scenario: Uninstall deb package

- **WHEN** 用户执行 `dpkg -r agent-forge`
- **THEN** `/usr/bin/agent-forge` SHALL 被移除，`which agent-forge` 无输出

#### Scenario: Uninstall rpm package

- **WHEN** 用户执行 `rpm -e agent-forge`
- **THEN** `/usr/bin/agent-forge` SHALL 被移除，`which agent-forge` 无输出

#### Scenario: Package metadata includes release info

- **WHEN** 用户执行 `dpkg -I agent-forge_*.deb` 或 `rpm -qi agent-forge`
- **THEN** 包信息 SHALL 包含项目描述、版本号、维护者和开源许可证

### Requirement: Windows MSI Installer

系统 SHALL 通过 wixl 生成 `.msi` 安装包，支持全新安装、版本升级和卸载。

#### Scenario: MSI is generated for each Windows build

- **WHEN** 目标 OS 为 windows
- **THEN** GoReleaser SHALL 为每个 windows/amd64 和 windows/arm64 生成对应的 `.msi` 安装包

#### Scenario: Fresh MSI install

- **WHEN** 用户在未安装过 agent-forge 的 Windows 系统上执行 `msiexec /i agent-forge_*.msi`
- **THEN** 二进制 SHALL 安装到 `C:\Program Files\AgentForge\agent-forge.exe`，并注册到系统 PATH

#### Scenario: MSI upgrade via MajorUpgrade strategy

- **WHEN** 已安装旧版本 agent-forge MSI，用户执行 `msiexec /i agent-forge_<新版本>.msi`
- **THEN** MSI 安装程序 SHALL 检测到相同的 UpgradeCode → 自动卸载旧版本 → 安装新版本（MajorUpgrade 策略）

#### Scenario: MSI uninstall

- **WHEN** 用户通过控制面板或执行 `msiexec /x <ProductCode>` 卸载 agent-forge
- **THEN** 二进制文件 SHALL 被移除，PATH 注册 SHALL 被清理，不再能在终端中调用 `agent-forge`

#### Scenario: UpgradeCode remains constant across versions

- **WHEN** 项目发布新版本 MSI
- **THEN** `.goreleaser.yaml` 中的 `upgrade_code` SHALL 与之前版本保持相同，确保升级检测能正常工作

#### Scenario: MSI supports custom install directory

- **WHEN** 用户执行 `msiexec /i agent-forge.msi INSTALLDIR="D:\Tools\AgentForge"`
- **THEN** 二进制 SHALL 安装到 `D:\Tools\AgentForge\agent-forge.exe`，PATH 注册指向该自定义路径

#### Scenario: MSI defaults to standard directory

- **WHEN** 用户未指定 `INSTALLDIR` 属性
- **THEN** 二进制 SHALL 安装到默认路径 `C:\Program Files\AgentForge\agent-forge.exe`

#### Scenario: Upgrade preserves custom install directory

- **WHEN** 已通过自定义目录安装旧版本，用户升级到新版本 MSI
- **THEN** 新版本 SHALL 读取旧版本的安装目录，覆盖安装到同一自定义路径，而非回退到默认路径

#### Scenario: MSI can be built on Linux via wixl

- **WHEN** 在 Linux 构建环境中执行 `goreleaser release`
- **THEN** wixl 工具 SHALL 成功交叉生成 Windows MSI 包，无需 Windows 构建机

### Requirement: CGO Disabled for Cross-Compilation

所有平台构建 SHALL 禁用 CGO 以确保纯静态链接和跨平台兼容性。

#### Scenario: CGO disabled for all builds

- **WHEN** 任何平台构建
- **THEN** 构建环境 SHALL 设置 `CGO_ENABLED=0`

### Requirement: Version Variable Consistency

构建配置中的 ldflags 变量路径 SHALL 与源代码中实际定义的变量保持一致。

#### Scenario: Version variable exists in source

- **WHEN** `.goreleaser.yaml` 引用 `cmd.Version`、`cmd.GitHash`、`cmd.BuildTime`
- **THEN** `cmd/version.go` 中 SHALL 存在对应的 `var Version`、`var GitHash`、`var BuildTime` 声明

#### Scenario: Missing BuildTime variable is created

- **WHEN** `cmd/version.go` 中不存在 `BuildTime` 变量
- **THEN** 实现代码 SHALL 新增 `var BuildTime = "unknown"` 声明
