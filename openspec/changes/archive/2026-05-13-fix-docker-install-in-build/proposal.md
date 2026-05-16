## Why

Docker 镜像构建在安装 `docker` 依赖时失败：`depsmodule` 中硬编码了 `yum install -y docker`，经 `adaptCommandForFamily` 翻译为 `apt-get install -y docker` 后，在 Debian 系基础镜像上因包名错误（应为 `docker.io`）而失败；在 CentOS 7 上也可能因仓库缺失而失败。需要一个跨发行版、不依赖特定包管理器的 Docker CLI 安装方案。

## What Changes

- 将 `docker` 依赖的安装方式从系统包管理器安装改为 Docker 官方静态二进制下载，消除对 `yum`/`apt-get` 及发行版包名的依赖
- 在 `dockerfilegen` 的 `adaptCommandForFamily` 中增加包名映射，确保同类问题不再发生
- 添加 `docker` 依赖安装的单元测试覆盖

## Capabilities

### New Capabilities

- `multi-family-docker-install`: 跨发行版 Docker CLI 静态二进制安装，不依赖系统包管理器，兼容 CentOS 7/8、Ubuntu/Debian、Fedora 等所有 Linux 发行版

### Modified Capabilities

<!-- 无已有 specs 需要修改 -->

## Impact

- `internal/build/depsmodule/depsmodule.go`: 修改 `docker` 依赖的安装命令
- `internal/build/dockerfilegen/dockerfilegen.go`: 增强 `adaptCommandForFamily` 包名映射
- `internal/build/buildengine/buildengine_test.go`: 新增安装命令生成的测试用例
