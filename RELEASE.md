# AgentForge 发布指南

## 前置条件

| 工具 | 用途 | 安装 |
|------|------|------|
| GoReleaser | 构建与发布自动化 | `go install github.com/goreleaser/goreleaser/v2@latest` |
| wixl | Windows MSI 包生成 | `sudo apt install wixl` (Debian/Ubuntu) 或 `sudo dnf install msitools` (Fedora) |
| GitHub Token | 正式发布到 GitHub Release | Settings → Developer settings → Personal access tokens → 勾选 `repo` scope |

## Git Tag 规范

版本号遵循 `vMAJOR.MINOR.PATCH` 格式（Semantic Versioning）：

```
v1.0.0   # 正式版本
v1.0.1   # 补丁版本
v1.1.0   # 次版本
v2.0.0   # 主版本
```

## 发布流程

### 1. 打 Tag

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

### 2. Snapshot 验证（本地测试）

此项不依赖 Git 干净状态或 GitHub Token，可随时执行：

```bash
goreleaser release --snapshot --clean
```

产物生成到 `dist/` 目录，检查：
- 所有平台二进制编译通过
- 归档包内容正确
- deb/rpm 包元数据正确
- MSI 文件正常生成

如需跳过特定生成器（如本地缺少 rpmbuild）：

```bash
goreleaser release --snapshot --clean --skip=nfpm    # 跳过 deb/rpm
goreleaser release --snapshot --clean --skip=msi     # 跳过 MSI
```

### 3. 正式 Release

确保 Git 状态干净且 HEAD 指向 tag，然后：

```bash
export GITHUB_TOKEN="ghp_xxxxxxxxxxxxxxxxxxxx"
goreleaser release --clean
```

此命令将：
- 为所有目标平台编译二进制
- 生成归档包（tar.gz / zip）
- 生成 deb / rpm 包
- 生成 MSI 安装包
- 创建 GitHub Release 并上传所有产物

## 安装方式

### Linux

#### 便携包（无需 root）

```bash
tar -xzf agent-forge_*_linux_amd64.tar.gz
./agent-forge --version
sudo mv agent-forge /usr/local/bin/
```

#### deb（Debian / Ubuntu）

```bash
# 全新安装
sudo dpkg -i agent-forge_*_linux_amd64.deb

# 升级
sudo dpkg -i agent-forge_<新版本>_linux_amd64.deb

# 卸载
sudo dpkg -r agent-forge
```

#### rpm（RHEL / Fedora / CentOS）

```bash
# 全新安装
sudo rpm -i agent-forge_*_linux_amd64.rpm

# 升级
sudo rpm -U agent-forge_<新版本>_linux_amd64.rpm

# 卸载
sudo rpm -e agent-forge
```

### Windows

#### MSI 安装包（推荐）

```bash
# 全新安装（默认 C:\Program Files\AgentForge\）
msiexec /i agent-forge_*_windows_amd64.msi

# 自定义安装目录
msiexec /i agent-forge_*_windows_amd64.msi INSTALLDIR="D:\Tools\AgentForge"

# 升级（MSI 自动检测旧版本并卸载后安装新版）
msiexec /i agent-forge_<新版本>_windows_amd64.msi

# 卸载（通过控制面板或命令行）
msiexec /x agent-forge_*_windows_amd64.msi
```

#### 便携包

解压 zip 到任意目录，将该目录加入 PATH 即可。

## MSI UpgradeCode

```
f5a75dc7-ef43-4cf9-87d3-21fbcfc3ccd1
```

**此值永不修改。** 丢失或变更此 GUID 将导致新旧版本 MSI 无法关联升级，每个旧版本都需要手动卸载后才能安装新版本。

此值记录在 `.goreleaser.yaml` 的 `msi.upgrade_code` 字段和 `builds/windows/installer.wxs` 模板中。

## 常见问题

### CGO 交叉编译错误

错误：`cannot find -l<lib>` 或 `undefined reference to`

原因：某个依赖使用了 CGO 但交叉编译工具链不完整。

解决：确认 `.goreleaser.yaml` 中设置了 `CGO_ENABLED=0`。当前项目为纯 Go，无 CGO 依赖。

### rpmbuild: command not found

错误：`rpmbuild: command not found`

解决：
- 安装 rpm 构建工具：`sudo apt install rpm` (Debian/Ubuntu)
- 或跳过 rpm 生成：`goreleaser release --snapshot --clean --skip=nfpm`

### wixl: command not found

错误：`wixl: command not found` 或 MSI 生成失败

解决：安装 msitools：`sudo apt install wixl` (Debian/Ubuntu) 或 `sudo dnf install msitools` (Fedora)

### GITHUB_TOKEN 未设置

错误：`GITHUB_TOKEN is not set`

解决：正式发布需要 GitHub Token。如果仅做本地验证，使用 `--snapshot` 模式。

### 升级后版本号未变化

现象：安装新版本后 `agent-forge --version` 仍显示旧版本号

原因：可能安装了不同路径的二进制（如 PATH 中同时存在 `/usr/local/bin/agent-forge` 和 `/usr/bin/agent-forge`）

解决：`which agent-forge` 确认实际调用的路径，移除旧版本文件。

### Git 状态不干净

错误：`git state is dirty`

解决：
- 提交或暂存所有变更
- 或使用 `--snapshot` 模式（跳过此检查）
