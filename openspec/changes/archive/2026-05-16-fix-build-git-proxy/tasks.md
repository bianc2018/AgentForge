# Tasks: fix-build-git-proxy

> 关联：REQ-5（`--gh-proxy` 参数）

## 1. 修改默认值

- [x] 1.1 `cmd/build.go:65`：将 `--gh-proxy` flag 默认值从 `""` 改为 `"https://ghproxy.net"`
- [x] 1.2 `internal/shared/argsparser/params.go:38-43`：`DefaultBuildParams()` 中 `GHProxy` 字段设为 `"https://ghproxy.net"`，更新第 33 行注释
- [x] 1.3 运行 `go test -short ./internal/shared/argsparser/...` 确认默认值相关测试通过（如有失败则更新断言）

## 2. 修复 applyGHProxy URL 拼接

- [x] 2.1 `internal/build/dockerfilegen/dockerfilegen.go:432`：在 `applyGHProxy` 中对 `ghProxy` 做 `strings.TrimRight(ghProxy, "/") + "/"` 标准化后再拼接
- [x] 2.2 运行 `go test -short ./internal/build/dockerfilegen/...` 确认全部通过

## 3. 补充测试

- [x] 3.1 添加 `TestGenerate_DefaultGHProxy`：不传 `GHProxy`（空 Options），验证生成的 Dockerfile 不含代理配置
- [x] 3.2 添加 `TestApplyGHProxy_NoTrailingSlash`：传入 `https://ghproxy.example.com`（无斜杠），验证生成 URL 正确包含 `/https://github.com/`
- [x] 3.3 添加 `TestApplyGHProxy_TrailingSlash`：传入 `https://ghproxy.example.com/`（有斜杠），验证不会双写斜杠
- [x] 3.4 更新 `internal/shared/argsparser/parser_test.go` 中涉及 `GHProxy` 默认值的断言

## 4. 验证

- [x] 4.1 运行 `go test -short ./internal/build/dockerfilegen/... ./internal/shared/argsparser/... ./cmd/...` 确认全部通过
- [x] 4.2 运行覆盖率门禁 `go test -short -coverprofile=coverage.out -covermode=atomic ./... && bash scripts/check-coverage.sh` 确认通过（修改包全部 ≥90%，总体 93.8%）
