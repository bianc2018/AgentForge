# 测试策略 — AgentForge

## 1. 单元测试

### UT-1: `DepsModule.ExpandDeps()`

- **测试内容:** 将元标签和单体依赖列表展开为完整的安装指令列表
- **覆盖案例:**
  - 正常路径（`all`）: 展开为全部 agent、runtime、tool 的完整列表
  - 正常路径（`mini`）: 展开为常用子集，数量少于 `all`
  - 单体依赖（`claude,golang@1.21,node@20`）: 精确展开为指定项
  - 混合元标签 + 单体: `all+claude` 去重后展开
  - 未知依赖名称: 保留为系统包名称
  - 空输入: 返回空列表
- **所需 Mock:** 无 — 纯领域逻辑
- **可追溯性:** REQ-1 · REQ-2

### UT-2: `DepsModule.ResolveInstallMethod()`

- **测试内容:** 根据依赖名称返回正确的安装方式（agent/runtime/tool/系统包）
- **覆盖案例:**
  - agent（claude）: 返回 npm install -g 指令
  - runtime 带版本（golang@1.21）: 返回指定版本的 go binary 下载指令
  - runtime 无版本（node）: 返回 nodesource 安装指令
  - tool（speckit）: 返回对应安装指令
  - 未知名称: 返回 yum install 系统包指令
  - 格式错误的名称（`@1.21` 无基础名）: 返回错误
- **所需 Mock:** 无 — 纯领域逻辑
- **可追溯性:** REQ-1 · REQ-2

### UT-3: `DockerfileGenerator.Generate()`

- **测试内容:** 根据依赖列表、基础镜像和参数生成合法的 Dockerfile 内容
- **覆盖案例:**
  - 正常路径: 生成包含 FROM、RUN 等指令的合法 Dockerfile
  - 指定基础镜像（`-b docker.1ms.run/centos:7`）: FROM 指令使用指定镜像
  - 国内镜像源替换: yarn/npm 源切换为 npmmirror，pip 切换为阿里云
  - 带 `--gh-proxy`: GitHub clone/下载 URL 使用代理
  - 带 `--no-cache` 标记: 无特殊影响（Dockerfile 本身无变化，由 build 参数控制）
  - 空依赖列表: 生成最小 Dockerfile（仅 FROM + 基础配置）
  - 环境变量注入: 生成的 Dockerfile 包含 ENV 指令
- **所需 Mock:** 无 — 纯字符串生成
- **可追溯性:** REQ-1 · REQ-3 · REQ-5 · REQ-6

### UT-4: `BuildEngine.CalculateBackoff()`

- **测试内容:** 指数退避等待时间的计算
- **覆盖案例:**
  - 第 1 次重试: 等待 1 秒
  - 第 2 次重试: 等待 2 秒
  - 第 3 次重试: 等待 4 秒
  - 第 N 次重试: 等待 2^(N-1) 秒（不超过上限）
  - 最大重试次数边界: 超过 `--max-retry` 后不继续重试
- **所需 Mock:** 无 — 纯数学计算
- **可追溯性:** REQ-4 · NFR-10

### UT-5: `ArgsPersistence.Save()`

- **测试内容:** 将运行参数持久化到 `.last_args` 文件
- **覆盖案例:**
  - 正常保存: 所有参数字段准确写入文件
  - 端口映射多值: `-p 3000:3000 -p 8080:8080` 正确序列化
  - 环境变量多值: `-e KEY1=VAL1 -e KEY2=VAL2` 正确序列化
  - 挂载路径多值: `-m /a -m /b` 正确序列化
  - 空值字段: 未提供的参数保存为空
  - 写入权限: 文件写入后权限为 0600（可选 mock 验证）
- **所需 Mock:** mock 文件系统（os.File）
- **可追溯性:** REQ-16 · NFR-12

### UT-6: `ArgsPersistence.Load()`

- **测试内容:** 从 `.last_args` 文件还原参数集
- **覆盖案例:**
  - 正常加载: 正确解析所有字段返回结构化参数
  - 文件不存在: 返回 ErrFileNotFound
  - 文件格式错误: 部分字段缺失时以空值填充，不崩溃
  - 空文件: 返回空参数集，不崩溃
- **所需 Mock:** mock 文件系统（os.File）
- **可追溯性:** REQ-16 · REQ-17

### UT-7: `WrapperLoader.Generate()`

- **测试内容:** 生成包含所有已安装 agent wrapper 函数的 bash 脚本
- **覆盖案例:**
  - 正常路径: 生成的脚本包含 claude/opencode/kimi/deepseek-tui 的函数定义
  - 脚本语法: 生成的 bash 内容语法正确（可通过 `bash -n` 验证）
  - 函数名冲突: 不同 agent 的函数名互不重叠
- **所需 Mock:** 无 — 纯字符串生成
- **可追溯性:** REQ-14

### UT-8: `EndpointManager.MaskKey()`

- **测试内容:** 对 API key 做掩码处理（前 8 字符 + `***` + 后 4 字符）
- **覆盖案例:**
  - 正常路径: `sk-test-key-value` → `sk-test-***alue`（前 8 + `***` + 后 4）
  - 短 key（长度 < 12）: 短于前 8 + 后 4 的 key，合理截断
  - 空 key: 返回空字符串，不崩溃
  - 恰好 12 字符: 前 8 + `***` + 后 4 无重叠
  - 特殊字符 key: 正确处理包含特殊字符的 key
- **所需 Mock:** 无 — 纯字符串处理
- **可追溯性:** REQ-21 · NFR-6

### UT-9: `EndpointManager.ParseEndpointEnv()`

- **测试内容:** 解析 `endpoint.env` 文件的 KEY=VALUE 内容
- **覆盖案例:**
  - 正常路径: 解析所有标准字段（PROVIDER, URL, KEY, MODEL 等）
  - 缺少可选字段: MODEL_OPUS 等缺失时正确设置默认值
  - 多余空白行: 跳过空行和注释行
  - 格式错误: 非 `KEY=VALUE` 行跳过不崩溃
  - 空文件: 返回空配置不崩溃
- **所需 Mock:** 无 — 纯字符串解析（或 mock io.Reader）
- **可追溯性:** REQ-22 · REQ-23

### UT-10: `ApplySyncer.FormatForAgent()`

- **测试内容:** 将端点配置格式化为各 agent 期望的配置格式
- **覆盖案例:**
  - claude: 输出 key=value 格式的 `.env` 内容
  - opencode: 输出 key=value 格式的 `.env` 内容（字段不同）
  - kimi: 输出 TOML 格式的 `[api]\nkey = "xxx"\nbase_url = "yyy"` 内容
  - deepseek-tui: 输出 key=value 格式的 `.env` 内容（字段不同）
  - 未知 agent: 返回错误
  - 配置含特殊字符: TOML 字符串中引号正确转义
- **所需 Mock:** 无 — 纯模板化字符串生成
- **可追溯性:** REQ-28 · REQ-29

### UT-11: `ProviderAgentMatrix.GetProviders()`

- **测试内容:** 返回所有受支持的 LLM 服务商列表
- **覆盖案例:**
  - 正常路径: 返回 [deepseek, openai, anthropic]
  - 返回不可变: 多次调用返回相同列表
- **所需 Mock:** 无 — 纯领域逻辑
- **可追溯性:** REQ-19

### UT-12: `ProviderAgentMatrix.GetAgentsForProvider()`

- **测试内容:** 查询指定 provider 可服务的 agent 列表
- **覆盖案例:**
  - deepseek: 返回 [claude, opencode, kimi, deepseek-tui]
  - openai: 返回 [claude, opencode]
  - anthropic: 返回 [claude]
  - 未知 provider: 返回空列表
- **所需 Mock:** 无 — 纯领域逻辑
- **可追溯性:** REQ-19 · REQ-28

### UT-13: `ConfigResolver.Resolve()`

- **测试内容:** 根据 `-c` 参数或默认值解析配置目录路径
- **覆盖案例:**
  - 默认路径: 未指定 `-c` 时返回 `$(pwd)/coding-config`
  - 自定义路径: `-c /path/to/config` 返回 `/path/to/config`
  - 相对路径: `-c ./config` 解析为绝对路径
  - 路径不存在: 返回路径但不创建目录（后续操作处理）
- **所需 Mock:** mock 当前工作目录
- **可追溯性:** REQ-2 · REQ-11 · REQ-13

### UT-14: `SelfUpdateEngine.BackupAndRollback()`

- **测试内容:** 更新前备份当前版本，更新失败时回滚
- **覆盖案例:**
  - 正常更新: 备份完成 → 新版本写入成功 → 删除备份
  - 下载失败回滚: 备份后下载失败 → 从备份恢复原始版本
  - 写入失败回滚: 新版本写入时失败 → 回滚到备份版本
  - 备份文件已存在: 覆盖旧备份
- **所需 Mock:** mock 文件系统（os.File），mock HTTP client
- **可追溯性:** REQ-36 · NFR-13

### UT-15: `VersionInfo.Format()`

- **测试内容:** 格式化版本号和 git hash 输出
- **覆盖案例:**
  - 正常路径: 输出 "agent-forge X.Y.Z (hash)"
  - 空 hash: 输出 "agent-forge X.Y.Z (unknown)"
  - 版本号格式: 语义化版本号展示
- **所需 Mock:** 无 — 纯字符串格式化
- **可追溯性:** REQ-36 · NFR-21 · NFR-22

### UT-16: `PackageManagerAdapter.Detect()`

- **测试内容:** 识别当前操作系统的包管理器
- **覆盖案例:**
  - apt-get 可用: 返回 apt-get（Debian/Ubuntu）
  - dnf 可用: 返回 dnf（Fedora/RHEL 8+）
  - yum 可用: 返回 yum（CentOS 7/RHEL 7）
  - brew 可用: 返回 brew（macOS）
  - 无可用包管理器: 返回错误
- **所需 Mock:** mock os.Exec 判断各包管理器是否存在
- **可追溯性:** REQ-32 · NFR-19

### UT-17: `RunEngine.AssembleContainerConfig()`

- **测试内容:** 根据参数组装 SDK `ContainerCreate` 配置结构
- **覆盖案例:**
  - agent 模式: Cmd 设置为 agent 命令，Tty=true, OpenStdin=true
  - bash 模式: Cmd 设置为 bash 加载 wrapper，Tty=true
  - Docker-in-Docker 模式: Privileged=true, User="root"，挂载 docker.sock
  - 后台命令模式: AutoRemove=true，Cmd 为指定命令，无 Tty
  - 端口映射: PortBindings 正确转换 `-p 3000:3000`
  - 只读挂载: Mounts 正确设置为只读（NFR-8）
  - 环境变量: Env 数组包含所有 `-e` 指定的键值对
  - 工作目录: WorkingDir 指定为 `-w` 参数值
  - 多端口/多挂载: 多个 `-p` 和 `-m` 参数全部包含
- **所需 Mock:** 无 — 纯数据结构组装
- **可追溯性:** REQ-9 · REQ-10 · REQ-11 · REQ-12 · REQ-13 · REQ-14 · REQ-15 · REQ-18 · NFR-7 · NFR-8

### UT-19: `ArgsParser.Parse()`

- **测试内容:** 解析各命令的命令行参数为结构化配置
- **覆盖案例:**
  - 正常路径（build）: `-d all --max-retry 3 --gh-proxy https://proxy.example.com` 正确解析
  - 正常路径（run）: `-a claude -p 3000:3000 -p 8080:8080 -m /data -e KEY=VAL -w /work` 正确解析
  - 短参/长参别名: `-r` 等价于 `--recall`，`-R` 等价于 `--rebuild`
  - 多值参数: 多次 `-p` 和 `-m` 全部收集
  - 缺失参数: 可选参数未指定时设为默认值
  - 无效参数名: 返回错误
  - 参数值格式错误: 返回解析错误
- **所需 Mock:** 无 — 纯字符串解析
- **可追溯性:** 所有涉及 CLI 参数的 REQ

### UT-18: `DiagnosticEngine.ClassifyIssue()`

- **测试内容:** 根据 Docker API 错误类型分类诊断问题
- **覆盖案例:**
  - socket 不可达: 标记为核心依赖缺失
  - Docker Ping 失败: 标记为运行时异常
  - 权限不足: 标记为运行时权限问题
  - BuildKit 不可用: 标记为可选工具提示
  - 所有检查通过: 返回全部通过
- **所需 Mock:** mock Docker SDK client 返回各类错误
- **可追溯性:** REQ-31 · NFR-17 · NFR-18

---

## 3. E2E Gherkin 测试

### GH-1: Scenario "构建包含全部依赖的镜像"

- **文件:** `docs/features/agent-forge/scenarios.feature`
- **必要的 Step Definitions:**
  - `Given Docker Engine 已安装并运行` → 检查 docker info 返回正常
  - `When 开发者执行 build -d all --max-retry 3` → 调用 build 命令并传入参数
  - `Then 构建过程退出码为 0` → 验证进程退出码
  - `And docker images 列表中包含新生成的镜像` → 通过 docker images 或 SDK ImageList 验证镜像存在
- **可从其他 Scenarios 复用的 Steps:** `Given Docker Engine 已安装并运行`, `Then 构建过程退出码为 0`
- **必要的初始状态:** Docker Engine 已运行，无同名镜像干扰
- **可追溯性:** REQ-1 · REQ-3 · REQ-4 · REQ-6 · NFR-1 · NFR-10

### GH-2: Scenario "构建包含指定依赖的自定义镜像"

- **文件:** `docs/features/agent-forge/scenarios.feature`
- **必要的 Step Definitions:**
  - `When 开发者执行 build -d claude,golang@1.21,node@16 -b docker.1ms.run/centos:7 -c /path/to/config` → 调用 build 命令
  - `And 容器内 go version 输出 1.21.x` → 通过 exec 或 run 容器执行 go version
  - `And 容器内 node --version 输出 16.x` → 通过 exec 或 run 容器执行 node --version
- **可从其他 Scenarios 复用的 Steps:** `Given Docker Engine 已安装并运行`, `Then 构建过程退出码为 0`
- **必要的初始状态:** Docker Engine 已运行
- **可追溯性:** REQ-1 · REQ-2 · REQ-3

### GH-3: Scenario "构建过程中网络错误时自动重试"

- **文件:** `docs/features/agent-forge/scenarios.feature`
- **必要的 Step Definitions:**
  - `Given 构建过程中首次请求 GitHub 资源超时` → 通过 mock 代理或网络限制模拟首次超时
  - `When 开发者执行 build -d claude --max-retry 3 --gh-proxy https://gh-proxy.example.com` → 调用 build 命令
  - `Then 系统按指数退避策略自动重试` → 验证重试次数和等待间隔
  - `And 在三次重试内构建成功` → 验证最终成功
- **可从其他 Scenarios 复用的 Steps:** `Given Docker Engine 已安装并运行`, `Then 构建过程退出码为 0`
- **必要的初始状态:** Docker Engine 已运行，可模拟网络错误
- **可追溯性:** REQ-4 · REQ-5 · NFR-10

### GH-4: Scenario "重建镜像成功替换旧标签"

- **文件:** `docs/features/agent-forge/scenarios.feature`
- **必要的 Step Definitions:**
  - `Given 存在一个已构建的镜像 agent-forge:latest` → 预先构建一个基础镜像
  - `When 开发者执行 build -R -d claude,golang@1.21` → 调用 rebuild 命令
  - `Then 系统自动叠加 --no-cache 强制跳过缓存` → 验证构建参数包含 NoCache
  - `And 构建成功后临时标签替换原镜像标签` → 验证 ImageTag API 调用
  - `And 旧镜像被删除` → 验证 ImageRemove API 调用
- **可从其他 Scenarios 复用的 Steps:** `Given Docker Engine 已安装并运行`, `Then 构建过程退出码为 0`
- **必要的初始状态:** 已构建的 agent-forge:latest 镜像存在于本地
- **可追溯性:** REQ-6 · REQ-7 · NFR-23

### GH-5: Scenario "重建失败时保留旧镜像"

- **文件:** `docs/features/agent-forge/scenarios.feature`
- **必要的 Step Definitions:**
  - `When 开发者执行 build -R -d invalid-package-that-fails` → 调用 rebuild 命令，使用无效依赖
  - `Then 构建失败后清理临时标签` → 验证 ImageRemove 清理临时标签
  - `And 原镜像 agent-forge:latest 保持不变` → 验证原镜像仍存在且标签不变
  - `And 构建过程退出码非零` → 验证退出码不为 0
- **可从其他 Scenarios 复用的 Steps:** `Given 存在一个已构建的镜像 agent-forge:latest`
- **必要的初始状态:** 已构建的 agent-forge:latest 镜像
- **可追溯性:** REQ-8 · NFR-11

### GH-6: Scenario "启动指定 agent 带完整配置的交互式终端"

- **文件:** `docs/features/agent-forge/scenarios.feature`
- **必要的 Step Definitions:**
  - `Given 已构建 AgentForge 镜像` → 确保镜像存在
  - `When 开发者执行 run -a claude -p 3000:3000 -m /host/data -w /workspace -e OPENAI_KEY=sk-xxx` → 调用 run 命令
  - `Then 容器启动并进入 claude 交互式终端` → 验证容器 claude 进程运行
  - `And 容器内端口 3000 可访问` → 验证端口映射
  - `And 容器内 /host/data 目录存在且挂载自宿主机` → 验证目录挂载
  - `And 容器内工作目录为 /workspace` → 验证工作目录
  - `And 容器内环境变量 OPENAI_KEY 值为 sk-xxx` → 验证环境变量
- **可从其他 Scenarios 复用的 Steps:** `Given 已构建 AgentForge 镜像`
- **必要的初始状态:** 已构建 AgentForge 镜像
- **可追溯性:** REQ-9 · REQ-10 · REQ-11 · REQ-12 · REQ-13 · NFR-3 · NFR-8

### GH-7: Scenario "不指定 agent 以 bash 模式启动容器"

- **文件:** `docs/features/agent-forge/scenarios.feature`
- **必要的 Step Definitions:**
  - `When 开发者执行 run 命令且不指定 -a 参数` → 调用 run 命令无 -a
  - `Then 容器启动并进入 bash shell` → 验证容器 bash 进程运行
  - `And bash 环境中自动加载了 claude、opencode、kimi、deepseek-tui 等 wrapper 函数` → 验证 wrapper 函数已定义
  - `And 开发者可在容器内直接通过 wrapper 函数名调用任意已安装的 AI agent` → 验证 agent 可执行
- **可从其他 Scenarios 复用的 Steps:** `Given 已构建 AgentForge 镜像`
- **必要的初始状态:** 已构建 AgentForge 镜像
- **可追溯性:** REQ-14

### GH-8: Scenario "以 Docker-in-Docker 特权模式启动容器"

- **文件:** `docs/features/agent-forge/scenarios.feature`
- **必要的 Step Definitions:**
  - `When 开发者执行 run --docker` → 调用 run 命令
  - `Then 容器以特权模式和 root 用户启动` → 验证容器 Privileged 和 User 配置
  - `And 容器内 dockerd 守护进程自动启动` → 验证 dockerd 进程运行
  - `And 容器内可正常执行 docker ps 等 docker 命令` → 在容器内执行 docker ps
- **可从其他 Scenarios 复用的 Steps:** `Given 已构建 AgentForge 镜像`
- **必要的初始状态:** 已构建 AgentForge 镜像
- **可追溯性:** REQ-15 · NFR-7

### GH-9: Scenario "通过 -r 参数恢复上次运行参数启动容器"

- **文件:** `docs/features/agent-forge/scenarios.feature`
- **必要的 Step Definitions:**
  - `Given 开发者之前执行过一次 run -a claude -p 3000:3000 -m /host/data` → 预执行 run 命令或手动创建 .last_args
  - `And .last_args 文件已自动持久化上次运行的全部参数` → 验证 .last_args 文件存在
  - `When 开发者执行 run -r` → 调用 run -r 命令
  - `Then 系统从 .last_args 文件恢复上次运行的完整参数` → 验证参数被读取
  - `And 容器以与上次运行完全相同的配置启动` → 验证容器配置一致
  - `And 容器内 claude 交互式终端可用，端口 3000 已映射，/host/data 目录已挂载` → 验证全部参数生效
- **可从其他 Scenarios 复用的 Steps:** `Given 已构建 AgentForge 镜像`, `Then 容器启动并进入 claude 交互式终端`
- **必要的初始状态:** 已构建 AgentForge 镜像，已创建 .last_args 文件
- **可追溯性:** REQ-16 · REQ-17 · NFR-12

### GH-10: Scenario "不存在历史参数时使用 -r 恢复失败"

- **文件:** `docs/features/agent-forge/scenarios.feature`
- **必要的 Step Definitions:**
  - `Given 从未执行过 run 命令或 .last_args 文件不存在` → 确保 .last_args 不存在
  - `When 开发者执行 run -r` → 调用 run -r 命令
  - `Then 系统提示无法回忆上次运行参数` → 验证输出包含提示信息
  - `And 容器不会启动` → 验证无容器创建
- **可从其他 Scenarios 复用的 Steps:** 无
- **必要的初始状态:** .last_args 文件不存在
- **可追溯性:** REQ-17

### GH-11: Scenario "后台执行命令后自动退出容器"

- **文件:** `docs/features/agent-forge/scenarios.feature`
- **必要的 Step Definitions:**
  - `When 开发者执行 run --run "npm test"` → 调用 run --run 命令
  - `Then 容器在后台启动并执行 npm test 命令` → 验证容器以非交互模式启动
  - `And 命令执行完成后容器自动退出` → 验证容器状态为 exited
  - `And 容器退出码与 npm test 的退出码一致` → 验证退出码传递
- **可从其他 Scenarios 复用的 Steps:** `Given 已构建 AgentForge 镜像`
- **必要的初始状态:** 已构建 AgentForge 镜像
- **可追溯性:** REQ-18

### GH-12: Scenario "带全部参数新增 LLM 端点"

- **文件:** `docs/features/agent-forge/scenarios.feature`
- **必要的 Step Definitions:**
  - `When 开发者执行 endpoint add my-ep --provider openai --url https://api.openai.com --key sk-test-key-value --model gpt-4 --model-opus gpt-4-32k --model-sonnet gpt-4-turbo --model-haiku gpt-3.5-turbo --model-subagent gpt-4-mini` → 调用 endpoint add 命令
  - `Then 端点 my-ep 创建成功` → 验证 endpoint.env 文件存在
  - `And endpoint list 输出表中包含 my-ep` → 验证 list 输出
  - `And endpoint show my-ep 显示 API key 为 sk-test***alue 掩码格式` → 验证 key 掩码
- **可从其他 Scenarios 复用的 Steps:** 无
- **必要的初始状态:** 配置目录存在，无同名端点
- **可追溯性:** REQ-22 · REQ-23 · NFR-9 · NFR-14

### GH-13: Scenario "缺少参数时交互式新增 LLM 端点"

- **文件:** `docs/features/agent-forge/scenarios.feature`
- **必要的 Step Definitions:**
  - `When 开发者执行 endpoint add my-ep 且未提供 --provider 和 --url 参数` → 调用 endpoint add 缺少参数
  - `Then 系统逐个提问缺失的配置项：provider、url、model` → 验证交互式提示
  - `And 开发者依次输入 deepseek、https://api.deepseek.com` → 模拟用户输入
  - `Then 端点 my-ep 创建成功` → 验证端点文件存在
  - `And endpoint list 输出表中包含 my-ep` → 验证 list 输出
- **可从其他 Scenarios 复用的 Steps:** `Then 端点 <name> 创建成功`, `And endpoint list 输出表中包含 <name>`
- **必要的初始状态:** 无同名端点
- **可追溯性:** REQ-22 · REQ-23 · NFR-14

### GH-14: Scenario "修改已有端点的配置"

- **文件:** `docs/features/agent-forge/scenarios.feature`
- **必要的 Step Definitions:**
  - `Given 存在已创建的端点 my-ep` → 预创建端点
  - `When 开发者执行 endpoint set my-ep --key sk-new-key --model gpt-5` → 调用 endpoint set 命令
  - `Then 端点 my-ep 的 API key 更新为 sk-new-key` → 验证 endpoint.env 内容
  - `And 端点 my-ep 的模型更新为 gpt-5` → 验证 MODEL 字段更新
- **可从其他 Scenarios 复用的 Steps:** `Given 存在已创建的端点 <name>`
- **必要的初始状态:** 已创建端点 my-ep
- **可追溯性:** REQ-24

### GH-15: Scenario "删除 LLM 端点"

- **文件:** `docs/features/agent-forge/scenarios.feature`
- **必要的 Step Definitions:**
  - `When 开发者执行 endpoint rm my-ep` → 调用 endpoint rm 命令
  - `Then 端点 my-ep 及其对应目录被删除` → 验证目录不存在
  - `And endpoint list 输出中不再包含 my-ep` → 验证 list 不再显示
- **可从其他 Scenarios 复用的 Steps:** `Given 存在已创建的端点 my-ep`
- **必要的初始状态:** 已创建端点 my-ep
- **可追溯性:** REQ-25

### GH-16: Scenario "查看提供商列表和端点详情"

- **文件:** `docs/features/agent-forge/scenarios.feature`
- **必要的 Step Definitions:**
  - `When 开发者执行 endpoint providers` → 调用 endpoint providers
  - `Then 输出列出所有支持的 LLM 服务商及其对应的 AI agent` → 验证 provider-agent 映射表输出
  - `When 开发者执行 endpoint list` → 调用 endpoint list
  - `Then 输出以 NAME / PROVIDER / MODEL 表格格式列出所有端点` → 验证表格格式
  - `When 开发者执行 endpoint show my-ep` → 调用 endpoint show
  - `Then 输出显示 my-ep 的详细配置` → 验证配置显示
  - `And API key 显示为前 8 字符加 *** 加后 4 字符的掩码格式` → 验证掩码格式
- **可从其他 Scenarios 复用的 Steps:** `Given 存在已创建的端点 my-ep`
- **必要的初始状态:** 已创建端点 my-ep
- **可追溯性:** REQ-19 · REQ-20 · REQ-21 · NFR-5 · NFR-6

### GH-17: Scenario "测试端点连通性成功"

- **文件:** `docs/features/agent-forge/scenarios.feature`
- **必要的 Step Definitions:**
  - `Given 存在已创建的可达端点 my-ep` → 预创建可达端点（使用 mock HTTP server）
  - `When 开发者执行 endpoint test my-ep` → 调用 endpoint test 命令
  - `Then 系统向端点发送 POST chat/completions 请求` → 验证 HTTP 请求
  - `And 输出包含请求延迟和回复摘要` → 验证输出格式
  - `And 退出码为 0` → 验证退出码
- **可从其他 Scenarios 复用的 Steps:** `Given 存在已创建的端点 <name>`
- **必要的初始状态:** 已创建端点，启动 mock HTTP server 模拟 LLM 端点
- **可追溯性:** REQ-26 · NFR-4

### GH-18: Scenario "测试端点连通性失败"

- **文件:** `docs/features/agent-forge/scenarios.feature`
- **必要的 Step Definitions:**
  - `Given 存在已创建的不可达端点 broken-ep` → 预创建端点，URL 指向不可达地址
  - `When 开发者执行 endpoint test broken-ep` → 调用 endpoint test 命令
  - `Then 系统向端点发送 POST chat/completions 请求` → 验证 HTTP 请求触发
  - `And 请求失败（连接超时、认证失败或端点不可达）` → 模拟连接失败
  - `And 输出明确的错误信息` → 验证错误信息包含原因、上下文和建议
  - `And 退出码非零` → 验证退出码
- **可从其他 Scenarios 复用的 Steps:** `Given 存在已创建的端点 <name>`
- **必要的初始状态:** 已创建端点，地址不可达（如 localhost:1）
- **可追溯性:** REQ-27 · NFR-4 · NFR-16

### GH-19: Scenario "同步端点配置到 agent"

- **文件:** `docs/features/agent-forge/scenarios.feature`
- **必要的 Step Definitions:**
  - `When 开发者执行 endpoint apply my-ep` → 调用 endpoint apply 命令
  - `Then 端点 my-ep 的配置写入 claude 的 .claude/.env 文件` → 验证文件内容
  - `And 写入 opencode 的 .opencode/.env 文件` → 验证文件内容
  - `And 写入 kimi 的 .kimi/config.toml 文件` → 验证文件内容
  - `And 写入 deepseek-tui 的 .deepseek/.env 文件` → 验证文件内容
  - `When 开发者执行 endpoint apply my-ep --agent claude,kimi` → 调用 apply 带 --agent 过滤
  - `Then 端点 my-ep 的配置仅写入 claude 和 kimi 的配置文件` → 验证 claude/kimi 已更新
  - `And opencode 和 deepseek-tui 的配置文件不受影响` → 验证 opencode/dstui 未变更
- **可从其他 Scenarios 复用的 Steps:** `Given 存在已创建的端点 my-ep`
- **必要的初始状态:** 已创建端点 my-ep，各 agent 配置目录存在
- **可追溯性:** REQ-28 · REQ-29 · NFR-9

### GH-20: Scenario "查看 agent 端点映射关系"

- **文件:** `docs/features/agent-forge/scenarios.feature`
- **必要的 Step Definitions:**
  - `When 开发者执行 endpoint status` → 调用 endpoint status 命令
  - `Then 输出表格包含每个 agent 名称和其关联的端点名称` → 验证映射表输出
- **可从其他 Scenarios 复用的 Steps:** `Given 存在已创建的端点 my-ep`
- **必要的初始状态:** 已创建端点 my-ep
- **可追溯性:** REQ-30

### GH-21: Scenario "环境诊断"

- **文件:** `docs/features/agent-forge/scenarios.feature`
- **必要的 Step Definitions:**
  - `Given Docker Engine 已安装并运行` → 验证 Docker 可用
  - `Given Docker 核心依赖已安装` → 验证 Docker 已安装
  - `When 开发者执行 doctor` → 调用 doctor 命令
  - `Then 核心依赖检查全部通过` → 验证第一层诊断结果
  - `And 运行时检查 Docker daemon 运行状态正常` → 验证第二层诊断结果
  - `And 可选工具检查 buildx 安装状态` → 验证第三层诊断结果
  - `And 所有三层诊断输出均为通过状态` → 验证最终状态
  - `Given Docker 核心依赖缺失` → 模拟缺失场景
  - `Then 系统检测到缺失的核心依赖` → 验证检测逻辑
  - `And 自动使用 apt-get / dnf / yum / brew 安装缺失组件` → 验证包管理器调用
  - `And 安装完成后重新检测` → 验证重检测机制
  - `And 修复后诊断全部通过` → 验证修复后状态
- **可从其他 Scenarios 复用的 Steps:** `Given Docker Engine 已安装并运行`
- **必要的初始状态:** Docker Engine 运行状态可切换
- **可追溯性:** REQ-31 · REQ-32 · NFR-17 · NFR-19 · NFR-18

### GH-22: Scenario "查询容器内依赖安装状态"

- **文件:** `docs/features/agent-forge/scenarios.feature`
- **必要的 Step Definitions:**
  - `When 开发者在宿主机执行 deps` → 调用 deps 命令
  - `Then 系统自动生成检测脚本` → 验证脚本生成逻辑
  - `And 通过 docker run --rm 在临时容器中执行检测` → 验证容器启动
  - `And 输出按 agent / skill / tool / runtime 分类显示安装状态和版本号` → 验证分类输出
  - `And 检测完成后临时容器自动销毁` → 验证容器清理
- **可从其他 Scenarios 复用的 Steps:** `Given 已构建 AgentForge 镜像`
- **必要的初始状态:** 已构建 AgentForge 镜像
- **可追溯性:** REQ-33

### GH-23: Scenario "导出和导入镜像实现离线分发"

- **文件:** `docs/features/agent-forge/scenarios.feature`
- **必要的 Step Definitions:**
  - `When 开发者执行 export agent-forge.tar` → 调用 export 命令
  - `Then 镜像被导出为 agent-forge.tar 文件` → 验证 tar 文件存在
  - `When 开发者在另一台机器上执行 import agent-forge.tar` → 调用 import 命令
  - `Then docker images 中显示已加载的镜像` → 验证镜像已加载
  - `And 可使用 run -a claude 正常启动容器` → 验证镜像可用
- **可从其他 Scenarios 复用的 Steps:** `Given 已构建 AgentForge 镜像`
- **必要的初始状态:** 已构建 AgentForge 镜像，可写入目标目录
- **可追溯性:** REQ-34 · REQ-35

### GH-24: Scenario "工具自更新和版本信息查看"

- **文件:** `docs/features/agent-forge/scenarios.feature`
- **必要的 Step Definitions:**
  - `Given Git remote 或 UPDATE_URL 中有新版本` → 启动 mock HTTP server 返回新版本
  - `When 开发者执行 update` → 调用 update 命令
  - `Then 系统从远端下载更新` → 验证 HTTP 请求
  - `And 嵌入新的 git hash` → 验证版本信息更新
  - `And 系统版本号更新` → 验证版本号变更
  - `When 开发者执行 version` → 调用 version 命令
  - `Then 输出格式化的版本号和当前 git hash` → 验证输出格式
  - `When 开发者执行任意命令的 --help` → 调用 `--help`
  - `Then 输出格式一致的帮助信息，包含命令用法和参数说明` → 验证帮助格式
- **可从其他 Scenarios 复用的 Steps:** 无
- **必要的初始状态:** mock HTTP server 就绪
- **可追溯性:** REQ-36 · REQ-37 · NFR-13 · NFR-15 · NFR-21 · NFR-22

---

## 2. 集成测试

### IT-1: `EndpointManager` — 端点配置完整 CRUD 及测试、状态查询

- **测试内容:** 在真实文件系统上执行端点配置的增删改查全生命周期，以及端点连通性测试和状态映射查询
- **使用的真实依赖:** 临时目录（t.TempDir()），mock HTTP server（用于 endpoint test）
- **覆盖案例:**
  - `add` 创建: 创建 `<name>/endpoint.env`，内容包含所有 8 个配置字段
  - `add` 文件权限: 创建后文件权限为 0600（NFR-9）
  - `set` 修改: 更新单个字段（如 MODEL），文件内容正确变更
  - `set` 不存在端点: 返回错误
  - `show` 查看: 正确读取并显示端点配置，KEY 为掩码格式
  - `list` 列出: 遍历目录列出所有端点
  - `rm` 删除: 删除端点目录及其内容
  - `rm` 不存在端点: 返回错误
  - `test` 成功: mock HTTP server 返回 200，输出包含延迟和回复摘要（REQ-26）
  - `test` 失败: mock HTTP server 返回超时/401，输出错误信息，退出码非零（REQ-27）
  - `status` 查询: 输出 agent-端点映射关系表（REQ-30）
- **使用的真实依赖:** 临时目录（t.TempDir()），mock 文件系统权限
- **覆盖案例:**
  - `add` 创建: 创建 `<name>/endpoint.env`，内容包含所有 8 个配置字段
  - `add` 文件权限: 创建后文件权限为 0600（NFR-9）
  - `set` 修改: 更新单个字段（如 MODEL），文件内容正确变更
  - `set` 不存在端点: 返回错误
  - `show` 查看: 正确读取并显示端点配置，KEY 为掩码格式
  - `list` 列出: 遍历目录列出所有端点
  - `rm` 删除: 删除端点目录及其内容
  - `rm` 不存在端点: 返回错误
- **所需 Setup:** 在每个测试前创建临时配置目录，测试后清理
- **可追溯性:** REQ-20 · REQ-21 · REQ-22 · REQ-24 · REQ-25 · NFR-9

### IT-2: `ApplySyncer` — 端点配置同步到 agent 配置文件

- **测试内容:** 将端点配置写入各 agent 的不同格式配置文件
- **使用的真实依赖:** 临时目录，Provider-Agent Matrix 映射
- **覆盖案例:**
  - 同步到 claude: `.claude/.env` 文件内容为 key=value 格式，包含 API_KEY 和 BASE_URL
  - 同步到 kimi: `.kimi/config.toml` 文件内容为 TOML 格式
  - 同步到 opencode: `.opencode/.env` 文件内容正确
  - 同步到 deepseek-tui: `.deepseek/.env` 文件内容正确
  - 指定 `--agent` 过滤: 仅写入指定 agent 的配置文件
  - 文件权限: 所有写入文件权限为 0600（NFR-9）
  - 不指定端点名（同步全部）: 遍历所有端点写入
- **所需 Setup:** 预创建端点配置文件和 agent 配置目录，测试后清理
- **可追溯性:** REQ-28 · REQ-29 · NFR-9

### IT-3: `ArgsPersistence` — .last_args 文件持久化与恢复

- **测试内容:** 在真实文件系统上保存和加载 `.last_args` 文件
- **使用的真实依赖:** 临时目录
- **覆盖案例:**
  - `save` 写入: 保存后文件内容与传入参数完全一致
  - `load` 读取: 读取 `.last_args` 还原为结构化参数
  - `load` 文件不存在: 返回 ErrFileNotFound
  - 保存后立即读取: 数据往返无误（同一配置 save 再 load 得到相同参数集）
  - 包含多值参数（多端口、多挂载）: 正确序列化和反序列化
- **所需 Setup:** 创建临时配置目录，测试后清理
- **可追溯性:** REQ-16 · REQ-17 · NFR-12

### IT-4: `DockerHelper` — Docker daemon 连接与基础操作

- **测试内容:** 通过 Docker SDK 与真实 Docker daemon 通信
- **使用的真实依赖:** Docker Engine >= 20.10（`/var/run/docker.sock`）
- **覆盖案例:**
  - Ping: SDK Ping 返回正常（daemon 可达）
  - Info: 获取 Docker 版本信息，版本 >= 20.10
  - ImageList: 列出本地镜像（至少包含基础镜像）
  - BuildKit 检测: 检查 DOCKER_BUILDKIT 环境变量
  - 连接失败处理: 当 socket 不可达时返回正确错误类型
- **所需 Setup:** 跳过条件 — Docker daemon 必须运行，否则测试跳过
- **可追溯性:** NFR-17 · NFR-18

### IT-5: `BuildEngine` — 完整镜像构建流程

- **测试内容:** 通过 BuildEngine 编排真实 Docker 镜像构建
- **使用的真实依赖:** Docker Engine，DockerfileGenerator，DepsModule
- **覆盖案例:**
  - 最小依赖构建: `-d claude` 构建退出码 0，镜像在 ImageList 中可见
  - 自定义基础镜像: `-b <轻量基础镜像>` 构建成功
  - `--no-cache`: 构建命令包含 `NoCache=true` 参数
  - `-R/--rebuild` 成功: 构建使用临时标签，成功后替换原标签，旧镜像被删除
  - `-R/--rebuild` 失败: 构建失败后清理临时标签，原镜像不变，退出码非零（REQ-8、NFR-11）
  - 网络错误重试逻辑: 通过 mock 网络错误验证指数退避调用次数
  - 构建失败处理: 依赖不存在时构建失败，退出码非零
- **所需 Setup:** 构建过程需测试用 Dockerfile 模板，测试完成后清理构建产生的镜像
- **可追溯性:** REQ-1 · REQ-4 · REQ-6 · REQ-7 · REQ-8 · NFR-1 · NFR-2 · NFR-10 · NFR-11 · NFR-23

### IT-6: `RunEngine` — 容器完整生命周期管理

- **测试内容:** 通过 RunEngine 创建、启动、停止容器
- **使用的真实依赖:** Docker Engine
- **覆盖案例:**
  - 容器创建: ContainerCreate 返回有效 ID
  - 容器启动: ContainerStart 成功，容器状态为 running
  - 端口映射: 容器配置包含正确的 PortBindings
  - 目录挂载（只读）: Mounts 包含 `ReadOnly=true`
  - 环境变量: Env 包含所有指定环境变量
  - 工作目录: WorkingDir 正确设置
  - 特权模式: Privileged=true 仅在 `--docker` 参数时设置
  - bash 模式: Cmd 包含 bash 命令
  - 后台命令模式: Cmd 为指定命令，AutoRemove=true
  - 容器停止后自动清理: 测试容器在测试后删除
- **所需 Setup:** 需要预构建测试镜像，测试后清理所有创建的容器
- **可追溯性:** REQ-9 · REQ-10 · REQ-11 · REQ-12 · REQ-13 · REQ-14 · REQ-15 · REQ-18 · NFR-7 · NFR-8

### IT-7: `DistributionEngine` — 镜像导出和导入

- **测试内容:** 通过 Docker SDK 导出镜像为 tar 文件并重新导入
- **使用的真实依赖:** Docker Engine，临时文件
- **覆盖案例:**
  - export: ImageSave API 调用成功，输出 tar 文件存在且非空
  - import: ImageLoad API 调用成功，镜像在 ImageList 中可见
  - export 后 import: 导入后的镜像可用 `run` 正常启动容器
  - export 不存在的镜像: 返回错误
  - import 不存在的文件: 返回错误
- **所需 Setup:** 预构建测试用镜像，测试后清理导出文件和导入的镜像
- **可追溯性:** REQ-34 · REQ-35

### IT-8: `DepsInspector` — 容器内依赖检测

- **测试内容:** 生成检测脚本并在临时容器中执行
- **使用的真实依赖:** Docker Engine
- **覆盖案例:**
  - 检测脚本生成: 输出合法的 bash 脚本，按 agent/skill/tool/runtime 分类
  - 临时容器执行: 通过 `docker run --rm` 成功执行检测脚本
  - 输出解析: 检测结果分类输出（agent: claude, runtime: golang 等）
  - 容器自动销毁: 检测完成后容器不在 `docker ps -a` 中
- **所需 Setup:** 预构建测试镜像，测试后无残留容器
- **可追溯性:** REQ-33

### IT-9: `CLIRouter` — CLI 命令路由和退出码

- **测试内容:** 验证 cobra 命令树设置和各命令退出码
- **使用的真实依赖:** 临时目录（部分命令需要文件系统）
- **覆盖案例:**
  - build 子命令: `build` 路由到 BuildEngine
  - run 子命令: `run` 作为默认命令，无参数时路由到 RunEngine
  - endpoint 子命令: 9 个子命令正确路由
  - doctor 命令: 路由到 DiagnosticEngine
  - deps 命令: 路由到 Deps Inspector
  - export/import 命令: 正确路由
  - update 命令: 路由到 SelfUpdateEngine
  - version 命令: 路由到 Version Info
  - help 命令: `--help` 输出格式一致的帮助信息
  - 无效命令: 返回错误和帮助提示
  - 退出码: 成功返回 0，参数错误返回 2
- **所需 Setup:** 构建 cobra 命令树实例，对需要 Docker 的命令使用 mock 或 skip
- **可追溯性:** REQ-37 · NFR-15 · NFR-16

### IT-10: `DiagnosticEngine` — 三层环境诊断

- **测试内容:** 执行三层诊断流程，验证检测和自动修复
- **使用的真实依赖:** Docker Engine，包管理器（mock 用于自动安装场景）
- **覆盖案例:**
  - 三层全部通过: Docker daemon 运行，Docker 已安装，buildx 可选
  - 核心依赖缺失: 检测到 Docker 未安装（mock 场景）
  - 运行时异常: Docker daemon 未运行（mock 场景）
  - 自动修复: 包管理器安装后重新检测
- **所需 Setup:** 使用 mock 模拟缺失依赖场景，真实 Docker 验证通过场景
- **可追溯性:** REQ-31 · REQ-32 · NFR-17 · NFR-19

---

## 4. 性能测试

### PT-1: `build -d all` 完整构建时间（NFR-1）

- **测量内容:** `build -d all --max-retry 3` 命令从执行到退出码返回 0 的端到端耗时
- **阈值:** ≤ 15 分钟（基础镜像已缓存 + 国内镜像源可用）
- **测量方法:** 在 CI 或本地测试环境中使用 `time` 命令包裹 build 调用，记录 wall clock 时间
- **执行次数:** 3 次执行，取最大值
- **可追溯性:** NFR-1

### PT-2: mini 镜像体积比例（NFR-2）

- **测量内容:** `build -d mini` 构建的镜像体积与 `build -d all` 构建的镜像体积之比
- **阈值:** mini 体积 < all 体积的 60%
- **测量方法:** 通过 `docker images` 或 SDK ImageList 获取镜像大小（Size 字段），计算比例
- **执行次数:** 分别构建 all 和 mini 各 1 次，验证比例
- **可追溯性:** NFR-2

### PT-3: 容器启动到交互终端就绪时间（NFR-3）

- **测量内容:** `run -a <agent>` 命令从执行到容器内交互终端提示符可接受输入的端到端时间
- **阈值:** ≤ 10 秒
- **测量方法:** 使用 `time` 包裹 run 命令，测量从执行到容器启动完成的时间；或通过 SDK 事件监听 ContainerStart 到 ContainerRunning 的时间差
- **执行次数:** 5 次执行，取 p95 值
- **可追溯性:** NFR-3

### PT-4: endpoint test 超时断开时间（NFR-4）

- **测量内容:** 当端点不可达时，`endpoint test <name>` 命令的超时断开时间
- **阈值:** ≤ 30 秒（超时断开并返回非零退出码）
- **测量方法:** 创建不可达端点（指向 localhost:1），执行 `time endpoint test` 测量超时时间
- **执行次数:** 3 次执行，取最大值
- **可追溯性:** NFR-4

### PT-5: 非构建类命令响应时间（NFR-5）

- **测量内容:** `version`、`--help`、`endpoint list`、`endpoint providers` 等非构建类命令的响应时间
- **阈值:** ≤ 1 秒
- **测量方法:** 使用 Go benchmark 或 `time` 命令测量各命令执行时间
- **执行次数:** 每个命令 5 次执行，取平均值
- **可追溯性:** NFR-5

---

## 5. 安全测试

### ST-1: API key 脱敏输出（NFR-6）

- **验证内容:** 所有可能输出 API key 的场景中，密钥以前 8 字符 + `***` + 后 4 字符格式脱敏显示
- **模拟的攻击向量:** 肩窥攻击、日志泄露、共享终端屏幕截取
- **覆盖案例:**
  - `endpoint show <name>`: KEY 字段输出为掩码格式，不泄露完整 key
  - `endpoint list`: 不输出 KEY 字段（仅 NAME/PROVIDER/MODEL）
  - `endpoint add` 回显确认: 不显示完整 key
  - 错误信息中包含 key: 错误上下文中的 key 片段做掩码处理
  - version/info 输出: 不泄露任何配置信息
- **可追溯性:** NFR-6 · PRD 风险（API key 明文存储）

### ST-2: 非特权容器默认安全模式（NFR-7）

- **验证内容:** 仅当显式传入 `--docker` 或 `--dind` 时容器以特权模式启动，否则以非特权模式运行
- **模拟的攻击向量:** 容器逃逸、宿主内核权限滥用
- **覆盖案例:**
  - 默认 run 无 `--docker`/`--dind`: 容器 `Privileged=false`，`User` 非 root
  - run `-a claude`: 非特权模式，无额外 docker.sock 挂载
  - run `--docker`: `Privileged=true`，`User="root"`，docker.sock 已挂载
  - run `--dind`: `Privileged=true`，`User="root"`，dockerd 已启动
  - 未指定 `--docker`/`--dind` 时即使指定 `-a`+`-p` 等参数也不启用特权
- **可追溯性:** NFR-7 · PRD 风险（容器特权模式安全风险）

### ST-3: 宿主机目录只读挂载（NFR-8）

- **验证内容:** 通过 `-m` 参数挂载的宿主机目录在容器内以只读权限挂载
- **模拟的攻击向量:** 容器内进程篡改宿主机文件
- **覆盖案例:**
  - `-m /host/data`: 容器内 `/host/data` 的挂载模式为只读（`:ro`）
  - 容器内尝试写入挂载目录: 返回权限拒绝错误
  - 多目录挂载: 所有 `-m` 指定的目录均为只读
  - 未指定 `-m` 时无额外挂载
- **可追溯性:** NFR-8 · PRD 风险（容器内挂载宿主机目录被意外修改）

### ST-4: 配置文件权限安全（NFR-9）

- **验证内容:** 所有端点配置文件和 agent 配置文件的权限为 `0600`（仅文件所有者可读写）
- **模拟的攻击向量:** 多用户系统上其他用户窃取 API key
- **覆盖案例:**
  - `endpoint add` 后: `endpoint.env` 文件权限为 0600
  - `endpoint set` 后: 修改后的文件权限仍为 0600
  - `endpoint apply` 后: agent 配置文件权限为 0600（claude/opencode/kimi/dstui）
  - 目录权限: endpoints/ 目录权限不为 0777（合理限制）
- **可追溯性:** NFR-9 · PRD 风险（API key 配置文件泄露）

### ST-5: 自更新失败自动回滚（NFR-13）

- **验证内容:** 更新过程中发生任何错误时，系统自动回滚到备份版本，CLI 工具保持可运行状态
- **模拟的攻击向量:** 更新过程中网络中断、磁盘写入失败、二进制完整性校验失败
- **覆盖案例:**
  - 下载过程中网络中断: 回滚到备份版本，原始二进制不变
  - 新版本写入磁盘失败: 回滚到备份版本
  - 版本号不更新: 回滚后 version 仍输出旧版本号
  - 回滚后 CLI 功能正常: 回滚后所有命令正常工作
- **可追溯性:** NFR-13 · PRD 风险（自更新失败导致 CLI 工具损坏）

---

## 覆盖率摘要

| 需求 | 单元测试 | 集成测试 | E2E Gherkin | 性能测试 | 安全测试 |
| --------- | -------- | --------- | ----------- | --------- | -------- |
| REQ-1 | UT-1, UT-2, UT-3 | IT-5 | GH-1, GH-2 | — | — |
| REQ-2 | UT-1, UT-2, UT-13 | IT-5 | GH-2 | — | — |
| REQ-3 | UT-3 | — | GH-1, GH-2 | — | — |
| REQ-4 | UT-4 | IT-5 | GH-3 | — | — |
| REQ-5 | UT-3 | — | GH-3 | — | — |
| REQ-6 | UT-3 | IT-5 | GH-4 | — | — |
| REQ-7 | — | IT-5 | GH-4 | — | — |
| REQ-8 | — | IT-5 | GH-5 | — | — |
| REQ-9 | UT-17 | IT-6 | GH-6 | — | — |
| REQ-10 | UT-17 | IT-6 | GH-6 | — | — |
| REQ-11 | UT-17 | IT-6 | GH-6 | — | ST-3 |
| REQ-12 | UT-17 | IT-6 | GH-6 | — | — |
| REQ-13 | UT-13, UT-17 | IT-6 | GH-6 | — | — |
| REQ-14 | UT-7, UT-17 | IT-6 | GH-7 | — | — |
| REQ-15 | UT-17 | IT-6 | GH-8 | — | ST-2 |
| REQ-16 | UT-5, UT-6 | IT-3 | GH-9 | — | — |
| REQ-17 | UT-6 | IT-3 | GH-10 | — | — |
| REQ-18 | UT-17 | IT-6 | GH-11 | — | — |
| REQ-19 | UT-11, UT-12 | — | GH-16 | — | — |
| REQ-20 | — | IT-1 | GH-16 | — | — |
| REQ-21 | UT-8 | IT-1 | GH-16 | — | ST-1 |
| REQ-22 | UT-9 | IT-1 | GH-12, GH-13 | — | — |
| REQ-23 | UT-9 | IT-1 | GH-13 | — | — |
| REQ-24 | — | IT-1 | GH-14 | — | — |
| REQ-25 | — | IT-1 | GH-15 | — | — |
| REQ-26 | — | IT-1 | GH-17 | — | — |
| REQ-27 | — | IT-1 | GH-18 | — | — |
| REQ-28 | UT-10 | IT-2 | GH-19 | — | — |
| REQ-29 | UT-10 | IT-2 | GH-19 | — | — |
| REQ-30 | — | IT-1 | GH-20 | — | — |
| REQ-31 | UT-18 | IT-10 | GH-21 | — | — |
| REQ-32 | UT-16 | IT-10 | GH-21 | — | — |
| REQ-33 | — | IT-8 | GH-22 | — | — |
| REQ-34 | — | IT-7 | GH-23 | — | — |
| REQ-35 | — | IT-7 | GH-23 | — | — |
| REQ-36 | UT-14, UT-15 | — | GH-24 | — | ST-5 |
| REQ-37 | — | IT-9 | GH-24 | — | — |
| NFR-1 | — | IT-5 | GH-1 | PT-1 | — |
| NFR-2 | — | — | — | PT-2 | — |
| NFR-3 | — | IT-6 | GH-6 | PT-3 | — |
| NFR-4 | — | — | GH-17, GH-18 | PT-4 | — |
| NFR-5 | — | — | GH-16 | PT-5 | — |
| NFR-6 | UT-8 | — | GH-16 | — | ST-1 |
| NFR-7 | UT-17 | IT-6 | GH-8 | — | ST-2 |
| NFR-8 | UT-17 | IT-6 | GH-6 | — | ST-3 |
| NFR-9 | — | IT-1, IT-2 | GH-12, GH-19 | — | ST-4 |
| NFR-10 | UT-4 | IT-5 | GH-1, GH-3 | — | — |
| NFR-11 | — | IT-5 | GH-5 | — | — |
| NFR-12 | UT-5 | IT-3 | GH-9 | — | — |
| NFR-13 | UT-14 | — | GH-24 | — | ST-5 |
| NFR-14 | — | — | GH-13 | — | — |
| NFR-15 | — | IT-9 | GH-24 | — | — |
| NFR-16 | — | — | GH-10, GH-18 | — | — |
| NFR-17 | UT-18 | IT-4, IT-10 | GH-21 | — | — |
| NFR-18 | UT-18 | IT-4 | GH-21 | — | — |
| NFR-19 | UT-16 | IT-10 | GH-21 | — | — |
| NFR-20 | — | — | GH-1, GH-2 | — | — |
| NFR-21 | UT-15 | — | GH-24 | — | — |
| NFR-22 | UT-15 | — | GH-24 | — | — |
| NFR-23 | — | IT-5 | GH-4, GH-5 | — | — |


