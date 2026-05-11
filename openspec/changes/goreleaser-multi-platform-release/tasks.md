## 1. 源代码准备

- [x] 1.1 在 `cmd/version.go` 中新增 `BuildTime` 变量声明，用于接收 ldflags 注入的构建时间戳
- [x] 1.2 确保 `.gitignore` 中包含 `dist/` 目录，防止构建产物被提交

## 2. GoReleaser 构建配置

- [x] 2.1 创建 `.goreleaser.yaml`，配置构建矩阵（GOOS=[linux,windows] × GOARCH=[amd64,arm64]），设置 `CGO_ENABLED=0`
- [x] 2.2 配置 ldflags：注入 `-s -w`（缩减体积）+ `cmd.Version`、`cmd.GitHash`、`cmd.BuildTime` 变量
- [x] 2.3 配置归档规则：Linux 输出 `tar.gz`、Windows 输出 `zip`，均包含 LICENSE 和 README.md
- [x] 2.4 配置 checksum 生成，产出 `checksums.txt`

## 3. Linux 包管理器（deb/rpm）

- [x] 3.1 在 `.goreleaser.yaml` 中配置 nfpm：生成 `.deb` 和 `.rpm`，安装路径 `/usr/bin/agent-forge`，填写包元数据（描述、维护者、许可证）
- [x] 3.2 配置 nfpm 的 `replaces` 和 `conflicts` 字段，确保同包名新旧版本正确替换

## 4. Windows MSI 安装包

- [x] 4.1 生成固定的 UpgradeCode GUID，在 `.goreleaser.yaml` 中配置 `msi` 块（`upgrade_code`、产品名称、描述），引用 `builds/windows/installer.wxs`，启用 MajorUpgrade 策略
- [x] 4.2 创建 `builds/windows/installer.wxs` 模板：声明 `INSTALLDIR` 公开属性（默认 `C:\Program Files\AgentForge\`）、`<CustomAction>` 将安装目录追加到系统 PATH、支持升级时读取旧版安装路径
- [x] 4.3 验证 wixl 工具可用（`which wixl`），确保 Linux 上可交叉生成 MSI

## 5. 发布文档

- [x] 5.1 创建 `RELEASE.md`，包含：
  - 前置条件（GoReleaser CLI、wixl、GitHub Token）
  - Git tag 规范（`vMAJOR.MINOR.PATCH`）
  - 分步发布流程（打 tag → snapshot 验证 → 正式 release）
  - 各平台安装/升级/卸载命令（Linux: deb/rpm/tar.gz, Windows: msi/zip）
  - MSI UpgradeCode 记录及"永不修改"警示
  - 常见问题排错

## 6. 本地验证

- [x] 6.1 执行 `goreleaser release --snapshot --clean`，确认所有平台（linux/windows × amd64/arm64）二进制编译通过
- [x] 6.2 验证输出二进制的 ldflags 注入（`./dist/agent-forge_linux_amd64_v1/agent-forge --version` 显示版本号、GitHash 和 BuildTime）
- [x] 6.3 验证归档包内容：`tar.gz` 和 `zip` 内均含二进制 + LICENSE + README
- [x] 6.4 验证 deb/rpm 包生成成功，包元数据正确（`dpkg -I` / `rpm -qi`）
- [x] 6.5 验证 MSI 文件生成成功且非空（`ls -lh dist/*.msi`）
- [x] 6.6 模拟升级验证：修改版本号后再次 snapshot，确认新版本 deb/rpm 可覆盖安装旧版本（`dpkg -i` 新版 / `rpm -U` 新版）
