## 1. Linux/macOS 构建脚本

- [x] 1.1 创建 `scripts/build-release.sh`：包含环境校验（Go ≥1.21、goreleaser 可用）、参数解析（--version、--release、--skip-tests）、goreleaser 调用
- [x] 1.2 设置 `build-release.sh` 可执行权限（chmod +x）

## 2. Windows 构建脚本

- [x] 2.1 创建 `scripts/build-release.bat`：包含环境校验（go 可用、goreleaser 可用）、参数解析、goreleaser 调用

## 3. 验证

- [x] 3.1 在 Linux 上执行 `./scripts/build-release.sh` 确认 snapshot 构建成功
- [x] 3.2 执行 `./scripts/build-release.sh --help` 确认帮助信息正确
