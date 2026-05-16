## Context

当前 `depsmodule.ResolveInstallMethod` 对 `docker` 依赖硬编码了 `yum install -y docker` 命令。在 `dockerfilegen` 中，`adaptCommandForFamily` 对 Debian 系基础镜像执行字符串替换，将 `yum install -y` 转为 `apt-get install -y`，但不处理包名差异 — Debian/Ubuntu 的正确包名是 `docker.io` 而非 `docker`。此外，CentOS 7 的默认仓库已 EOL，`docker` 包可能不可用。

当前调用链：`buildengine.Build()` → `depsmodule.ExpandDeps()` → `dockerfilegen.Generate()` → 对每个 dep 调用 `depsmodule.ResolveInstallMethod()` → `adaptCommandForFamily()` → 生成 RUN 指令。

## Goals / Non-Goals

**Goals:**
- `docker` 依赖在所有支持的 Linux 发行版上可成功安装（CentOS 7/8/9, RHEL, Fedora, Ubuntu, Debian）
- 不再依赖系统包管理器（yum/apt）来安装 Docker CLI
- 修复 `adaptCommandForFamily` 中包名不跟随翻译的问题，防止其他系统包依赖遇到同类问题

**Non-Goals:**
- 不在容器内启动 Docker daemon（DinD），仅安装 Docker CLI
- 不改变 `depsmodule.ResolveInstallMethod` 的函数签名
- 不修改其他已知依赖的安装方式

## Decisions

### Decision 1: Docker CLI 使用官方静态二进制安装

**选择**: 在 `depsmodule` 的 `docker` 依赖中，用 curl 下载 Docker 官方静态二进制 tarball 替代 `yum install -y docker`。

**原因**: 
- Docker 官方提供静态编译的二进制文件，不依赖任何系统库（musl/glibc 通用），真正跨发行版
- 避开了包名差异（`docker` vs `docker.io` vs `docker-ce`）和仓库配置问题
- 与 `golang` 依赖的安装模式一致（curl + tar + cleanup）

**替代方案考虑**:
- 在 `adaptCommandForFamily` 中加包名映射表：只解决了 Debian 系问题，CentOS 7 仓库缺失 `docker` 包的问题依然存在
- 调用 `get.docker.com` 脚本：依赖网络且脚本行为不可控

**命令设计**:
```
curl -fsSL https://download.docker.com/linux/static/stable/x86_64/docker-<VERSION>.tgz -o /tmp/docker.tgz
tar -C /usr/local/bin -xzf /tmp/docker.tgz docker/docker --strip-components=1
rm -f /tmp/docker.tgz
chmod +x /usr/local/bin/docker
```

### Decision 2: 版本固定为 24.0.7

**选择**: 使用 Docker 24.0.7（Docker Engine 24.x 的最后稳定版本）。

**原因**: 
- 与 CentOS 7 的 3.10 内核和 glibc 2.17 兼容
- 静态二进制已验证可运行
- 不过新以至于引入未知问题

### Decision 3: adaptCommandForFamily 增加包名映射

**选择**: 在 `adaptCommandForFamily` 中增加常见系统包的名称映射（`docker` → `docker.io`）。

**原因**: 防御性修复 — 即使 docker 已改为静态二进制，其他系统包依赖（通过 `DepSystemPkg` 类型）仍可能遇到同类问题。这个映射确保 `yum install -y <pkg>` 被正确翻译为 Debian 等效包名。

## Risks / Trade-offs

- **静态二进制体积** (~25MB) → Mitigation: 构建后立即删除 tarball 缓存
- **版本硬编码** → Mitigation: 24.0.7 是长期稳定版本，且 Docker CLI 向后兼容 Docker daemon
- **ARM64 架构未处理** → Mitigation: 当前所有构建目标均为 x86_64，ARM64 支持可在后续迭代添加
