## Context

当前 `--gh-proxy` 默认值为空字符串（不启用代理），在国内网络环境下 GitHub 下载经常超时/失败。同时 `applyGHProxy` 的 URL 拼接逻辑在代理 URL 无尾部斜杠时产生畸形 URL。

需同时解决三个关联问题：默认值、禁用机制、URL 拼接。

## Goals / Non-Goals

**Goals:**
- 不传 `--gh-proxy` 时默认使用 `https://ghproxy.net`
- 传 `--gh-proxy ""` 时禁用代理
- 自定义代理 URL 时自动标准化（兼容带/不带尾部斜杠）
- 所有受影响的代码位置同步更新

**Non-Goals:**
- 不扩展代理覆盖范围到 setup 阶段的 GitHub URL（如 Windows git 安装）
- 不引入 `HTTP_PROXY`/`HTTPS_PROXY` 标准环境变量设置
- 不引入 git 代理配置

## Decisions

### 决策 1：利用 Cobra 默认值机制区分"未传"与"显式传空"

Cobra 的 `Flags().GetString()` 行为：
- 用户未传参 → 返回 flag 定义的默认值
- 用户传 `--gh-proxy ""` → 返回空字符串 `""`
- 用户传 `--gh-proxy https://custom.com` → 返回该 URL

因此将 flag default 设为 `"https://ghproxy.net"`，下游逻辑不变：`GHProxy == ""` 仍表示"不使用代理"。无需引入 `*string` 指针或 `changed` 标记。

**替代方案**：用 `cmd.Flags().Changed("gh-proxy")` 判断是否传入。但当前 `ParseBuild` 无法感知 cobra 的 changed 状态，且最终都要落到 "空串=禁用" 这一语义，方案 1 更简洁。

### 决策 2：在 `applyGHProxy` 中标准化代理 URL

```go
normalized := strings.TrimRight(ghProxy, "/") + "/"
cmd = strings.ReplaceAll(cmd, "https://github.com/", normalized + "https://github.com/")
```

| 输入 | 标准化后 | 最终拼接 |
|------|---------|---------|
| `https://ghproxy.net` | `https://ghproxy.net/` | `https://ghproxy.net/https://github.com/...` |
| `https://ghproxy.net/` | `https://ghproxy.net/` | `https://ghproxy.net/https://github.com/...` |
| `https://ghproxy.net//` | `https://ghproxy.net/` | `https://ghproxy.net/https://github.com/...` |

**替代方案**：在 CLI 层标准化后传入。但生成器层是最底层、唯一使用该值进行 URL 拼接的地方，在此修复最健壮（防御性编程）。

### 决策 3：受影响的代码位置

1. `cmd/build.go:65` — cobra flag default: `""` → `"https://ghproxy.net"`
2. `argsparser/params.go:38-43` — `DefaultBuildParams().GHProxy`: `""` → `"https://ghproxy.net"`，注释更新
3. `argsparser/parser.go` — `ParseBuild` 从 `DefaultBuildParams()` 获取默认值，无需额外修改
4. `dockerfilegen/dockerfilegen.go:432` — `applyGHProxy` URL 标准化

## Risks / Trade-offs

- [极低] 原本不需代理就能访问 GitHub 的用户，现在默认走代理可能略慢。→ 用户可传 `--gh-proxy ""` 显式禁用。
- [极低] `ghproxy.net` 服务不可用时构建失败。→ 与当前 GitHub 不可达时的后果相同，且代理比直连 GitHub 更可靠（国内）；用户可传空字符串切回直连。
