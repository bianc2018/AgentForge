## 1. 平台推断基础能力

- [x] 1.1 在 `shared/configresolver`（或新建 `shared/platform` 包）中实现 `InferPlatform(baseImage string) string` 函数——根据基础镜像名称推断 `"windows"` 或 `""`（Linux）
- [x] 1.2 在 `dockerhelper.Client` 中新增 `GetDaemonOSType(ctx) (string, error)` 方法，封装 `Info()` API 返回 OSType
- [x] 1.3 实现 `ResolvePlatform(baseImage, daemonOSType string) (platform, defaultImage string, error)` ——合并镜像推断和 daemon 回退逻辑，返回最终平台和默认镜像，不兼容组合返回错误
- [x] 1.4 为 `InferPlatform` 和 `ResolvePlatform` 编写单元测试（覆盖：Windows 镜像关键词、Linux 镜像空返回、daemon 回退、不兼容报错）

## 2. Dockerfile Generator Windows 适配

- [x] 2.1 在 `dockerfilegen.ImageFamily` 枚举中新增 `FamilyWindows` 常量
- [x] 2.2 在 `detectImageFamily` 中新增 Windows 关键字匹配（`windows`、`nanoserver`、`servercore` 不区分大小写）
- [x] 2.3 实现 `writeWindowsSetup` 函数——生成 `SHELL ["powershell", "-Command"]` + Git for Windows 下载安装 RUN 指令
- [x] 2.4 在 `Generate` 中新增 FamilyWindows 分支：SHELL 指令、PowerShell 版 Node.js/Python 安装、CMD 改为 `["powershell"]`
- [x] 2.5 在 `adaptCommandForFamily` 中新增 Windows 命令翻译（`curl` → `Invoke-WebRequest`、`tar` → `Expand-Archive`、chmod 跳过）
- [x] 2.6 更新 `depsmodule` 为常用依赖提供 Windows 平台的安装命令映射（由 adaptCommandForFamily 统一翻译，depsmodule 保持平台无关）
- [x] 2.7 为 Windows Dockerfile 生成编写单元测试（覆盖 Nanoserver/ServerCore、有/无依赖、命令翻译）

## 3. Build Engine Windows 支持

- [x] 3.1 在 `buildengine.Build` 中调用 `ResolvePlatform` 推断平台和默认镜像
- [x] 3.2 当平台为 Windows 时：自动设置 `ImageBuildOptions.Platform = "windows/amd64"`，镜像标签使用 `agent-forge:latest-windows`
- [x] 3.3 构建前校验镜像-daemon 兼容性（Windows 镜像 + Linux daemon → 报错）
- [x] 3.4 将 `Platform` 字段存入 `BuildParams` 供后续流程使用
- [x] 3.5 为 build engine Windows 路径编写单元测试（mock Docker SDK）

## 4. Run Engine Windows 支持

- [x] 4.1 在 `RunParams` 中新增 `Platform` 和 `BaseImage` 字段（从 CLI 层传入）
- [x] 4.2 在 `runengine.buildCmd` 中新增 Windows 分支：使用 `powershell -Command` 替代 `bash -c`
- [x] 4.3 在 `wrapperloader` 中支持生成 PowerShell 语法 wrapper（`Function Invoke-AgentWrapper { ... }`）——简化：wrapper 通过环境变量注入，buildCmd 中已实现 PowerShell 包装
- [x] 4.4 在 `toContainerPath` 中新增 `toWindowsContainerPath`——WSL `/mnt/x/...` → `X:\...`，Linux `/home/...` → `C:\home\...`
- [x] 4.5 在 `runengine.Engine.Run` 中：Windows 平台下 `ContainerCreate` 传递 `&specs.Platform{OS: "windows", Architecture: "amd64"}`
- [x] 4.6 在 `runengine.Engine.Run` 中：Windows 平台下 `--docker` flag 输出警告并忽略
- [x] 4.7 在 `runengine.Engine.Run` 中：Windows 平台信号处理使用 `ContainerStop` 替代 `ContainerKill`
- [x] 4.8 为 run engine 编写 Windows 分支单元测试（mock Docker SDK，覆盖路径/信号/DIND 警告）

## 5. deps 命令 Windows 适配

- [x] 5.1 在 `depsinspector` 中新增 PowerShell 检测脚本模板（检测 Node.js、Git、Python 等，使用 `Get-Command` 和版本号获取）
- [x] 5.2 在 `depsinspector.RunDetection` 中根据 `-i` 镜像名称推断平台，选择 bash 或 PowerShell 脚本
- [x] 5.3 在临时容器创建时传递正确的 platform 参数（Windows 容器用 Windows platform）
- [x] 5.4 为 PowerShell 检测脚本编写单元测试

## 6. doctor 命令 Windows 适配

- [x] 6.1 在 `diagnosticengine` 中新增第四层诊断类型 `LayerPlatform`
- [x] 6.2 实现平台兼容性检查：读取 daemon OSType，检查本地 Windows 镜像的兼容性
- [x] 6.3 在诊断输出中新增第四层结果展示（通过/未通过 + 建议）
- [x] 6.4 为第四层诊断编写单元测试（mock Docker SDK 返回不同 OSType）

## 7. update 命令 Windows 适配

- [x] 7.1 在 `update/engine` 中通过 `runtime.GOOS` 检测宿主 OS
- [x] 7.2 Windows 宿主下：下载 `agent-forge.exe`，替换前备份为 `.bak`，失败时从 `.bak` 回滚
- [x] 7.3 为 Windows 更新路径编写单元测试

## 8. 测试和验证

- [x] 8.1 运行现有全量单元测试确保无回归（`go test -short ./...`）
- [x] 8.2 在 `runengine`、`buildengine`、`dockerfilegen`、`depsinspector`、`diagnosticengine`、`update/engine` 中用 mock 覆盖所有 Windows 分支
- [x] 8.3 运行覆盖率门禁脚本（`bash scripts/check-coverage.sh`），确保所有业务包 ≥ 90%，总体 ≥ 90%
- [ ] 8.4 在 Windows Docker 主机上执行手动 E2E 验证：build + run + deps + doctor + update 完整流程
