# 功能需求 — AgentForge

## 1. 镜像构建 (Image Build)

**REQ-1**: 当开发者执行 build 命令并通过 `-d` 参数指定依赖列表时，系统必须构建包含所列 AI agent、runtime 和 tool 的 Docker 镜像。

> 来源：PRD 目标 1 / Story 1 验收标准 / BDD 场景 "构建包含全部依赖的镜像"、"构建包含指定依赖的自定义镜像"

**REQ-2**: 当开发者通过 `-b` 参数指定基础镜像、通过 `-c` 参数指定配置目录时，系统必须在构建过程中使用指定的基础镜像和配置父目录。

> 来源：PRD 目标 2 / Story 1 验收标准 / BDD 场景 "构建包含指定依赖的自定义镜像"

**REQ-3**: 当系统正在构建 Docker 镜像时，系统必须自动使用国内镜像源（npm 切换为 npmmirror，pip 切换为阿里云，yum 切换为阿里云 centos vault）以加速依赖下载。

> 来源：PRD 目标 3 / Story 1 验收标准

**REQ-4**: 如果在构建过程中发生网络错误，系统必须自动使用指数退避策略重试，重试次数不超过 `--max-retry` 指定的值（默认为 3 次）。

> 来源：PRD 目标 3 / Story 1 验收标准 / BDD 场景 "构建过程中网络错误时自动重试"

**REQ-5**: 当开发者通过 `--gh-proxy` 参数指定 GitHub 代理 URL 时，系统必须在构建过程中使用该代理下载资源。

> 来源：PRD 目标 2 / Story 1 验收标准 / BDD 场景 "构建过程中网络错误时自动重试"

**REQ-6**: 当开发者向 build 命令传递 `--no-cache` 参数时，系统必须在镜像构建过程中强制跳过 Docker 缓存。

> 来源：PRD 目标 2 / BDD 场景 "重建镜像成功替换旧标签"

**REQ-7**: 当开发者执行 build 命令并附带 `-R/--rebuild` 参数时，系统必须使用临时标签构建镜像，构建成功后使用新镜像替换原标签并删除旧镜像，以退出码 0 退出。

> 来源：PRD 替代流程 "重建模式 — rebuild" / Story 2 验收标准 / BDD 场景 "重建镜像成功替换旧标签"

**REQ-8**: 如果重建过程失败，系统必须清理临时标签，保留原镜像不变，并以非零退出码退出。

> 来源：PRD 替代流程 "重建模式 — rebuild" / Story 2 验收标准 / BDD 场景 "重建失败时保留旧镜像"

## 2. 容器运行 — Agent 交互终端 (Container Run: Agent Interactive Terminal)

**REQ-9**: 当开发者执行 run 命令并通过 `-a` 参数指定 AI agent（claude、opencode、kimi 或 deepseek-tui）时，系统必须启动容器并运行对应的交互式终端。

> 来源：PRD 目标 4 / Story 3 验收标准 / BDD 场景 "启动指定 agent 带完整配置的交互式终端"

**REQ-10**: 当开发者通过 `-p` 参数指定端口映射时，系统必须将指定的主机端口发布到对应的容器端口。

> 来源：PRD 目标 5 / Story 3 验收标准 / BDD 场景 "启动指定 agent 带完整配置的交互式终端"

**REQ-11**: 当开发者通过 `-m` 参数指定目录路径时，系统必须将指定的主机目录以只读方式挂载到容器内的相同路径。

> 来源：PRD 目标 5 / Story 3 验收标准 / BDD 场景 "启动指定 agent 带完整配置的交互式终端"

**REQ-12**: 当开发者通过 `-e` 参数指定 KEY=VALUE 键值对时，系统必须将指定的环境变量注入容器。

> 来源：PRD 目标 5 / Story 3 验收标准 / BDD 场景 "启动指定 agent 带完整配置的交互式终端"

**REQ-13**: 当开发者通过 `-w` 参数指定目录路径时，系统必须将指定目录设置为容器内的工作目录（默认为当前目录）。

> 来源：PRD 目标 5 / Story 3 验收标准 / BDD 场景 "启动指定 agent 带完整配置的交互式终端"

## 3. 容器运行 — 特殊模式 (Container Run: Special Modes)

**REQ-14**: 当开发者执行 run 命令且未指定 `-a` 参数时，系统必须启动容器并进入 bash shell，加载所有已安装的 AI agent wrapper 函数。

> 来源：PRD 替代流程 "无特定 agent — bash 模式" / Story 4 验收标准 / BDD 场景 "不指定 agent 以 bash 模式启动容器"

**REQ-15**: 当开发者执行 run 命令并附带 `--docker` 或 `--dind` 参数时，系统必须以特权模式启动容器，使用 root 用户并自动启动容器内的 dockerd 守护进程。

> 来源：PRD 替代流程 "Docker-in-Docker 模式" / Story 5 验收标准 / BDD 场景 "以 Docker-in-Docker 特权模式启动容器"

**REQ-16**: 当开发者执行 run 命令并附带 `-r/--recall` 参数时，系统必须从 `.last_args` 文件中恢复全部参数集，并使用与上次运行完全相同的配置启动容器。

> 来源：PRD 替代流程 "恢复上次运行参数" / Story 6 验收标准 / BDD 场景 "通过 -r 参数恢复上次运行参数启动容器"

**REQ-17**: 如果 `.last_args` 文件不存在，系统必须通知开发者无法恢复上次运行参数，且不得启动容器。

> 来源：Story 6 验收标准 / BDD 场景 "不存在历史参数时使用 -r 恢复失败"

**REQ-18**: 当开发者执行 run 命令并附带 `--run <command>` 参数时，系统必须在后台启动容器，执行指定命令，然后以与被执行命令相同的退出码自动退出容器。

> 来源：PRD 目标 6 / 替代流程 "后台执行模式" / Story 7 验收标准 / BDD 场景 "后台执行命令后自动退出容器"

## 4. LLM 端点 — 查看和新增 (Endpoint: View & Add)

**REQ-19**: 当开发者执行 `endpoint providers` 时，系统必须列出所有受支持的 LLM provider 及其对应的 AI agent。

> 来源：PRD 目标 8 / Story 10 验收标准 / BDD 场景 "查看提供商列表和端点详情"

**REQ-20**: 当开发者执行 `endpoint list` 时，系统必须以表格形式显示所有 endpoint，包含 NAME、PROVIDER 和 MODEL 列。

> 来源：PRD 目标 8 / Story 10 验收标准 / BDD 场景 "查看提供商列表和端点详情"

**REQ-21**: 当开发者执行 `endpoint show <name>` 时，系统必须显示指定 endpoint 的详细配置，其中 API key 以前 8 位字符加 `***` 加后 4 位字符的形式脱敏显示。

> 来源：PRD 目标 8 / Story 10 验收标准 / BDD 场景 "查看提供商列表和端点详情"

**REQ-22**: 当开发者执行 `endpoint add <name>` 并提供全部必需参数（--provider、--url、--key、--model、--model-opus、--model-sonnet、--model-haiku、--model-subagent）时，系统必须创建具有指定配置的新 endpoint。

> 来源：PRD 目标 9 / Story 8 验收标准 / BDD 场景 "带全部参数新增 LLM 端点"

**REQ-23**: 当开发者执行 `endpoint add <name>` 但未提供全部必需参数时，系统必须交互式地逐一提示输入缺失的配置项。

> 来源：PRD 目标 9 / Story 8 验收标准 / BDD 场景 "缺少参数时交互式新增 LLM 端点"

## 5. LLM 端点 — 修改、删除和测试 (Endpoint: Modify, Delete & Test)

**REQ-24**: 当开发者执行 `endpoint set <name>` 并提供更新后的参数时，系统必须使用提供的值修改指定 endpoint 的配置。

> 来源：PRD 目标 10 / Story 9 验收标准 / BDD 场景 "修改已有端点的配置"

**REQ-25**: 当开发者执行 `endpoint rm <name>` 时，系统必须删除指定的 endpoint 及其对应的目录。

> 来源：PRD 目标 10 / Story 9 验收标准 / BDD 场景 "删除 LLM 端点"

**REQ-26**: 当开发者执行 `endpoint test <name>` 且 endpoint 可达时，系统必须发送 POST chat/completions 请求，输出请求延迟和响应摘要，并以退出码 0 退出。

> 来源：PRD 目标 10 / Story 11 验收标准 / BDD 场景 "测试端点连通性成功"

**REQ-27**: 如果在 `endpoint test <name>` 过程中 endpoint 不可达、超时或返回认证错误，系统必须输出清晰的错误信息并返回非零退出码。

> 来源：PRD 替代流程 "端点测试失败" / Story 11 验收标准 / BDD 场景 "测试端点连通性失败"

## 6. LLM 端点 — 同步和映射 (Endpoint: Sync & Mapping)

**REQ-28**: 当开发者执行 `endpoint apply [name]` 且未指定 `--agent` 参数时，系统必须将 endpoint 配置写入所有适用的 AI agent 配置文件（claude 写入 `.claude/.env`，opencode 写入 `.opencode/.env`，kimi 写入 `.kimi/config.toml`，deepseek-tui 写入 `.deepseek/.env`）。

> 来源：PRD 目标 11 / Story 12 验收标准 / BDD 场景 "同步端点配置到 agent"

**REQ-29**: 当开发者执行 `endpoint apply` 并通过 `--agent` 参数指定逗号分隔的 agent 列表时，系统必须仅将 endpoint 配置同步到指定 agent 的配置文件中。

> 来源：PRD 目标 11 / Story 12 验收标准 / BDD 场景 "同步端点配置到 agent"

**REQ-30**: 当开发者执行 `endpoint status` 时，系统必须以表格形式显示每个 agent 名称及其关联的 endpoint 名称。

> 来源：PRD 目标 11 / Story 12 验收标准 / BDD 场景 "查看 agent 端点映射关系"

## 7. 环境诊断和依赖查询 (Diagnosis & Dependency Query)

**REQ-31**: 当开发者执行 doctor 命令时，系统必须按顺序执行三层环境诊断：核心依赖（docker）、运行时（Docker daemon 状态、权限）和可选工具（buildx），并输出每一层的诊断结果。Go 单二进制已消除对外部 HTTP 客户端（curl）、版本控制工具（git）和 JSON 解析器（jq）的运行时依赖，全部由标准库覆盖。

> 来源：PRD 目标 13 / Story 13 验收标准 / BDD 场景 "环境诊断"

**REQ-32**: 如果 doctor 命令检测到缺少核心依赖，系统必须自动使用 apt-get、dnf、yum 或 brew 安装缺失的组件，并在安装后重新检查。

> 来源：PRD 替代流程 "缺失依赖自动修复" / Story 13 验收标准 / BDD 场景 "环境诊断"

**REQ-33**: 当开发者在主机上执行 deps 命令时，系统必须生成检测脚本，通过 `docker run --rm` 在基于目标镜像的临时容器中执行该脚本，输出分类为 agent、skill、tool 和 runtime 的每个组件的安装状态和版本，并在完成后自动销毁临时容器。

> 来源：PRD 目标 12 / Story 14 验收标准 / BDD 场景 "查询容器内依赖安装状态"

## 8. 离线分发、自更新和帮助 (Offline Distribution, Update & Help)

**REQ-34**: 当开发者执行 `export [filename]` 时，系统必须将 Docker 镜像导出为 tar 文件（默认文件名为 `dockercoding.tar`）。

> 来源：PRD 目标 14 / Story 15 验收标准 / BDD 场景 "导出和导入镜像实现离线分发"

**REQ-35**: 当开发者执行 `import <file>` 时，系统必须从指定的 tar 文件加载 Docker 镜像，使其在 `docker images` 中可见并可用于 run 命令。

> 来源：PRD 目标 14 / Story 15 验收标准 / BDD 场景 "导出和导入镜像实现离线分发"

**REQ-36**: 当开发者执行 update 命令时，系统必须从 Git 远程仓库或 UPDATE_URL 下载最新版本，嵌入新的 git hash 并更新版本号。当开发者执行 version 命令时，系统必须输出格式化的版本号和当前 git hash。

> 来源：PRD 目标 15 / Story 16 验收标准 / BDD 场景 "工具自更新和版本信息查看"

**REQ-37**: 当开发者对任一命令或子命令附加 `--help` 参数时，系统必须输出格式一致的帮助信息，包括命令用法和参数说明。

> 来源：PRD 目标 16 / Story 16 验收标准 / BDD 场景 "工具自更新和版本信息查看"
