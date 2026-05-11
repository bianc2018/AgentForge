## Context

AgentForge 是一个 Go 语言 CLI 工具（`github.com/agent-forge/cli`），基于 Cobra 框架，通过 Docker 管理 AI Coding Agent 容器生命周期。当前版本号通过 `-ldflags` 手动注入（`Version`、`GitHash`），无自动化构建流水线，无 Makefile，无 CI/CD 发布配置。

目标是为项目引入 GoReleaser，实现一键多平台构建与发布，覆盖 Windows/Linux、x86_64/ARM64 四大目标，产出归档包、Linux 系统包（deb/rpm）和 Windows MSI 安装包，所有安装方式均需支持全新安装、版本升级和卸载。

## Goals / Non-Goals

**Goals:**
- 通过 `.goreleaser.yaml` 定义多平台构建矩阵（GOOS × GOARCH）
- 自动注入版本号、GitHash、构建时间戳到二进制
- 生成 tar.gz（Linux）、zip（Windows）归档包
- 生成 .deb/.rpm 包用于 Linux 系统级安装，支持安装/升级/卸载
- 生成 .msi 包用于 Windows 系统级安装，支持安装/升级/卸载
- 支持 `snapshot` 模式本地验证和 `release` 模式正式发布
- 提供 RELEASE.md 文档说明发布操作及各平台安装/升级/卸载流程

**Non-Goals:**
- Docker 镜像构建与发布
- Homebrew tap、Snap、Flatpak、AUR、Chocolatey、Scoop、Winget
- Notarization（macOS 签名公证）
- CI/CD 流水线集成（GitHub Actions）—— 本次仅提供本地可执行的配置

## Decisions

### 1. 选择 GoReleaser vs 手写 Makefile + shell 脚本

**决策：GoReleaser**

GoReleaser 是 Go 生态的事实标准发布工具，提供声明式配置、交叉编译、归档、包管理器生成（nfpm 集成 deb/rpm、wixl 集成 MSI）、GitHub Release 集成等开箱即用的能力。手写脚本需要自行处理交叉编译环境、ldflags、归档命名、checksum 生成等重复性工作，维护成本高。

### 2. 构建矩阵

**决策：GOOS = [linux, windows] × GOARCH = [amd64, arm64]**

- 暂不包含 darwin：项目依赖 Docker SDK，macOS 上 Docker 通过 Desktop 提供，与 Linux 行为有差异，且未测试验证
- amd64 覆盖绝大多数 x86 服务器和桌面环境
- arm64 覆盖 ARM 服务器（AWS Graviton）和 Windows on ARM 设备
- 386（32位）市场占有率极低，不纳入

### 3. ldflags 注入策略

**决策：统一使用 `-s -w` 减小二进制体积 + 注入版本信息**

```yaml
ldflags:
  - -s -w
  - -X github.com/agent-forge/cli/cmd.Version={{.Version}}
  - -X github.com/agent-forge/cli/cmd.GitHash={{.Commit}}
  - -X github.com/agent-forge/cli/cmd.BuildTime={{.Date}}
```

- `-s -w` 去除调试信息和符号表，减小约 30% 体积
- 版本号从 Git tag 推导（`v1.0.0`）
- 需要先确认 `cmd.BuildTime` 变量是否存在，若不存在需新增

### 4. 归档格式

**决策：Linux 用 tar.gz，Windows 用 zip，同时生成 checksum 文件**

- tar.gz 是 Linux 标准分发格式
- zip 在 Windows 上无需额外工具解压
- checksums.txt 提供完整性校验

### 5. Linux 包管理器（deb / rpm）

**决策：通过 nfpm 生成 .deb 和 .rpm，利用原生包管理器的升级机制**

- deb 覆盖 Debian/Ubuntu 系，rpm 覆盖 RHEL/Fedora/CentOS 系
- 安装路径：`/usr/bin/agent-forge`
- 全新安装：`dpkg -i agent-forge_*.deb` 或 `rpm -i agent-forge_*.rpm`
- 升级：deb/rpm 原生支持版本比较 —— 高版本包直接安装即可覆盖升级（`dpkg -i` 新版本 / `rpm -U` 新版本），包管理器自动处理旧版本替换，无需额外 postinst 脚本
- 卸载：`dpkg -r agent-forge` / `rpm -e agent-forge`，二进制即被移除
- 包元数据（description、maintainer、license）从项目 README 提取
- GoReleaser 的 nfpm 配置中设置 `replaces` 和 `conflicts` 字段，确保同包名的新版本能正确替换旧版本

### 6. Windows MSI 安装包

**决策：通过 GoReleaser 的 `msi` 配置 + wixl 工具生成，使用 MajorUpgrade 策略**

MSI 升级策略核心：**固定 UpgradeCode，每版更换 ProductCode**。

| 场景 | MSI 行为 | 实现方式 |
|------|----------|----------|
| 全新安装 | `msiexec /i agent-forge.msi`，写入 `C:\Program Files\AgentForge\`，注册到 PATH | 默认安装流程 |
| 升级 | `msiexec /i agent-forge-v2.msi`，MSI 检测同 UpgradeCode 的旧版本 → 先卸载旧版 → 安装新版（MajorUpgrade） | `<MajorUpgrade>` 元素 + 恒定 UpgradeCode |
| 卸载 | `msiexec /x agent-forge.msi` 或通过控制面板，移除二进制和 PATH 注册 | MSI 标准卸载 |

GoReleaser 配置要点：
```yaml
msi:
  - id: agent-forge
    name: AgentForge
    description: "AI Coding Agent 容器生命周期管理工具"
    upgrade_code: <固定 GUID，生成后永不更改>
    wxs: builds/windows/installer.wxs  # 可选，自定义 UI
```

- 首次生成时随机生成一个 `upgrade_code` GUID，后续版本保持不变
- GoReleaser 自动为每版生成新的 ProductCode
- `<MajorUpgrade>` 由 GoReleaser 自动注入 wxs 模板
- 构建依赖：`wixl` 工具（`apt install wixl` 或 `dnf install msitools`）

#### 6.1 自定义安装目录

**决策：MSI 支持 `INSTALLDIR` 公开属性，tar.gz/zip 天然便携，deb/rpm 不做。**

| 格式 | 自定义目录 | 机制 |
|------|:--:|------|
| tar.gz / zip | ✅ | 本身就是便携分发，用户解压到任意目录即可 |
| MSI | ✅ | 通过 `INSTALLDIR` 公开属性——命令行 `msiexec /i agent-forge.msi INSTALLDIR="D:\Tools\AgentForge"` 或 GUI 目录选择页。需要在自定义 `.wxs` 模板中声明该属性 |
| deb / rpm | ❌ | 遵循 FHS 规范，二进制必须放 `/usr/bin/`。如需自定义路径应使用 tar.gz |

MSI 自定义目录的关键是提供 `builds/windows/installer.wxs` 模板：
- 声明 `<Property Id="INSTALLDIR" Secure="yes">`，默认值 `C:\Program Files\AgentForge\`
- 通过 `<CustomAction>` 将 `INSTALLDIR` 追加到系统 PATH（而非硬编码路径）
- 升级时新版本读取旧版本的安装目录，覆盖写入同一位置

### 7. 发布模式

**决策：本地 `snapshot` 模式为主，GitHub Release 为可选**

- `goreleaser release --snapshot --clean`：在本地生成所有产物到 `dist/`，不发 GitHub Release，用于日常验证
- `goreleaser release --clean`：正式发布，需 `GITHUB_TOKEN` 环境变量，自动创建 GitHub Release 并上传产物

## Risks / Trade-offs

- **Git tag 依赖**：GoReleaser 依赖 Git tag 确定版本号 → 制定明确的 tag 规范（`vMAJOR.MINOR.PATCH`），在 RELEASE.md 中说明
- **CGO 交叉编译**：若项目或依赖使用了 CGO，ARM64 交叉编译可能失败 → 当前项目为纯 Go，`.goreleaser.yaml` 中显式设置 `CGO_ENABLED=0`
- **deb/rpm 生成需要额外工具**：rpm 生成在非 RPM 系统上需要 `rpmbuild` → 在 RELEASE.md 中说明依赖，snapshot 模式可使用 `--skip=nfpm` 跳过
- **MSI 交叉编译依赖 wixl**：wixl 在 Linux 上可交叉生成 MSI，但生成的 MSI 无法在 Linux 上测试安装 → 本地 snapshot 只验证 MSI 生成成功（文件存在且大小合理），真实安装/升级/卸载测试需在 Windows 环境中手动执行
- **upgrade_code 管理**：MSI 的 UpgradeCode 丢失会导致新旧版本无法关联升级 → 将 UpgradeCode 明确记录在 `.goreleaser.yaml` 注释和 RELEASE.md 中，作为项目常量永不修改
