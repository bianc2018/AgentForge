## 1. 核心修复

- [x] 1.1 修改 `depsmodule.go` 中 `docker` 依赖的安装命令：用 Docker 官方静态二进制下载（docker-24.0.7.tgz）替代 `yum install -y docker`
- [x] 1.2 在 `dockerfilegen.go` 的 `adaptCommandForFamily` 中增加系统包名映射表，确保 `docker` → `docker.io` 等常见差异被正确处理

## 2. 验证

- [x] 2.1 为 `adaptCommandForFamily` 的包名映射添加单元测试
- [x] 2.2 为 `docker` 依赖的 `ResolveInstallMethod` 返回值添加单元测试
- [x] 2.3 运行 `go test ./internal/build/...` 确保所有测试通过
- [x] 2.4 使用默认基础镜像执行 `go run . build -d docker --no-cache` 验证构建成功
