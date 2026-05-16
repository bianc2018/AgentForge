# 覆盖率门禁配置指南

## 工作原理

项目使用 GitHub Actions + `scripts/check-coverage.sh` 实现覆盖率门禁：

1. 每次 push 到 `main` 或创建/更新 PR 时，CI 自动运行 `go test -short -race -coverprofile=coverage.out ./...`
2. `scripts/check-coverage.sh` 解析覆盖率文件，校验每个包 ≥90% 且总体 ≥90%
3. 不达标时脚本返回 exit code 1，CI 流水线标记为失败
4. 配合分支保护规则，PR 合并按钮被禁用

## 配置分支保护规则

1. 进入 GitHub 仓库 → **Settings** → **Branches** → **Add branch protection rule**
2. **Branch name pattern**：输入 `main`
3. 勾选 **Require status checks to pass before merging**
4. 搜索并勾选 `test`（CI workflow 中的 job name）
5. 勾选 **Require branches to be up to date before merging**
6. 点击 **Create** 保存

## 本地使用

```bash
# 运行测试生成覆盖率文件
go test -short -race -coverprofile=coverage.out -covermode=atomic ./...

# 检查覆盖率阈值
bash scripts/check-coverage.sh

# 查看详细覆盖率报告
go tool cover -func=coverage.out
go tool cover -html=coverage.out  # 浏览器查看逐行覆盖
```

## 阈值调整

编辑 `scripts/check-coverage.sh`，修改第二行参数：

```bash
bash scripts/check-coverage.sh coverage.out 90.0
#                                              ^^^^ 阈值百分比
```

或在 CI 中通过环境变量覆盖（需要修改 workflow）。
