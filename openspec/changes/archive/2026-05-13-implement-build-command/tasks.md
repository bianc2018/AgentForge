## 1. 接线实现

- [x] 1.1 修改 `cmd/build.go`：在 `RunE` 中从 cobra 标志解析 `buildengine.BuildParams`，创建 `dockerhelper.Client` 和 `buildengine.Engine`，调用 `engine.Build(cmd.Context(), params)`
- [x] 1.2 修复 `Build()` 返回值处理：`Build()` 返回 `(string, error)`，需同时处理 output 和 err
- [x] 1.3 修复错误输出丢失：即使构建失败，也先打印构建输出（含 Docker 日志），再返回 error

## 2. Dockerfile 生成器修复

- [x] 2.1 新增 `detectImageFamily()` 函数，根据基础镜像名称自动识别 Debian/Ubuntu 或 RHEL/CentOS 家族
- [x] 2.2 新增 `writeDebianSetup()` 函数，生成 apt 镜像源配置（阿里云）和 apt-get 基础工具安装
- [x] 2.3 修改 `Generate()` 根据镜像家族分支：编译工具（build-essential vs gcc）、Node.js（nodesource deb vs rpm）、Python3（apt vs yum）、缓存清理
- [x] 2.4 修复 Ubuntu 镜像源替换：同时替换 `security.ubuntu.com` 和 `archive.ubuntu.com` 为阿里云镜像

## 3. 构建引擎重试修复

- [x] 3.1 修复重试循环：当构建内容（非连接层）包含网络错误（SSL timeout、EOF 等）时，触发重试而非直接 break

## 4. 编译+运行验证

- [x] 4.1 执行 `go build ./...` 确认项目编译通过
- [x] 4.2 执行 `go vet ./...` 确认无静态分析警告
- [x] 4.3 `agent-forge build -b docker.1ms.run/ubuntu:22.04 -d claude` 构建成功，生成 `agent-forge:latest`（905MB），`agent-forge deps` 验证 claude + node 正确安装
