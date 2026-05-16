## Why

1. `--gh-proxy` 当前默认值为空（不启用代理），在国内网络环境下构建经常失败，用户每次都需要手动传 `--gh-proxy https://ghproxy.net`，体验差
2. 当用户传入不带尾部斜杠的代理 URL（如 `https://ghproxy.net`）时，`applyGHProxy` 拼接出畸形 URL `https://ghproxy.nethttps://github.com/...`（缺少 `/` 分隔符），导致 curl 下载失败
3. 用户无法通过传空字符串 `--gh-proxy ""` 来禁用代理（当前默认值本身就是空串，无法区分"未传"和"显式禁用"）

## What Changes

- **默认值变更**：`--gh-proxy` 默认值从 `""` 改为 `"https://ghproxy.net"`，未传参时自动启用 GitHub 代理
- **显式禁用**：传 `--gh-proxy ""` 表示不使用代理
- **自定义代理**：传 `--gh-proxy <url>` 使用指定代理 URL
- **修复 URL 拼接**：`applyGHProxy` 在拼接代理 URL 与 GitHub URL 时标准化处理，确保无论用户传入的 URL 是否带尾部斜杠，都能正确生成 `https://<代理>/https://github.com/...` 格式
- 所有涉及默认值的代码位置同步更新（cobra flag default、`DefaultBuildParams()`、argsparser）

## Capabilities

### Modified Capabilities
- `build-command`: `--gh-proxy` 参数行为变更 — 默认启用 `https://ghproxy.net`，传空禁用，传自定义 URL 使用指定代理；同时修复代理 URL 拼接 bug

## Impact

- `cmd/build.go:65`：cobra flag 默认值
- `internal/shared/argsparser/params.go:34,38-43`：`GHProxy` 注释 + `DefaultBuildParams()`
- `internal/build/dockerfilegen/dockerfilegen.go:422-436`：`applyGHProxy` URL 标准化
- `internal/build/dockerfilegen/dockerfilegen_test.go`：补充无尾部斜杠场景、默认值场景
- `internal/shared/argsparser/parser_test.go`：默认值相关断言更新
- 无 API 变更，无破坏性变更（原本不传参 = 无代理 → 现在不传参 = 默认代理，行为更友好）
