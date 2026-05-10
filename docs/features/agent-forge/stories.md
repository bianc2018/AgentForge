# 用户故事 — AgentForge

## 故事 1 – 构建包含指定依赖的 Docker 镜像

作为已安装 Docker Engine 的开发者
我想要通过 build 命令选择要安装的 AI agent、运行时和工具，构建自定义 Docker 镜像
以便获得一个开箱即用的容器化开发环境，无需手动逐个安装和配置依赖

_验收标准_：

- 系统必须接受通过 `-d` 参数指定依赖列表，支持使用 `+` 分隔的元标签（如 `all`、`mini`）和单体依赖名称（如 `claude`、`kimi`、`golang@1.21`）
- 系统必须接受通过 `-b` 参数指定基础镜像（默认 `docker.1ms.run/centos:7`），以及通过 `-c` 指定自定义配置父目录
- 系统必须在构建过程中自动使用国内镜像源加速（npm 映射 npmmirror、pip 映射 aliyun、yum 映射 aliyun centos vault）
- 系统必须在网络错误时按指数退避策略自动重试，重试次数可通过 `--max-retry` 控制（默认 3 次）
- 系统必须支持 `--gh-proxy` 指定 GitHub 代理 URL 以加速资源下载
- 构建成功后镜像必须在 `docker images` 中可见，退出码为 0

_优先级_：P1

## 故事 2 – 重建镜像并替换标签

作为已安装 Docker Engine 的开发者
我想要使用 rebuild 模式重建镜像，让系统自动用新镜像替换旧镜像标签
以便在更新依赖或配置后获得最新镜像，同时避免手动管理多个镜像版本

_验收标准_：

- 系统必须支持通过 `-R/--rebuild` 参数触发重建模式，并自动叠加 `--no-cache` 强制跳过缓存
- 系统必须使用临时标签进行构建，构建成功后替换原镜像标签并删除旧镜像
- 构建失败时，系统必须清理临时标签，保留原镜像不变
- 系统必须在重建成功时退出码为 0，失败时退出码非零

_优先级_：P2

## 故事 3 – 启动 AI agent 交互式终端

作为已安装 Docker Engine 的开发者
我想要通过 run 命令指定 AI agent（claude / opencode / kimi / deepseek-tui），配置端口映射、目录挂载、环境变量和工作目录后启动交互式终端
以便在容器化环境中直接使用 AI coding agent 进行编码工作

_验收标准_：

- 系统必须支持通过 `-a` 参数指定 AI agent（可选值：claude、opencode、kimi、deepseek-tui），并启动对应的交互式终端
- 系统必须支持通过 `-p` 参数多次指定端口映射（格式 `宿主机端口:容器端口`），映射后在容器内可访问
- 系统必须支持通过 `-m` 参数多次指定只读目录挂载，容器内挂载到相同路径
- 系统必须支持通过 `-e` 参数多次注入 `KEY=VALUE` 格式的环境变量
- 系统必须支持通过 `-w` 参数指定容器内工作目录（默认当前目录）
- 容器启动后指定 agent 的交互终端可用，所有配置的端口、挂载、环境变量和工作目录均生效

_优先级_：P1

## 故事 4 – 以 bash 模式启动容器

作为已安装 Docker Engine 的开发者
我想要执行 run 命令时不指定 AI agent，直接进入容器内的 bash 环境并加载所有已安装 agent 的 wrapper 函数
以便根据需要在同一容器中灵活切换使用不同的 AI coding agent

_验收标准_：

- 系统在不指定 `-a` 参数执行 `run` 命令时，必须启动容器并进入 bash shell
- 系统必须在 bash 环境中自动加载所有已安装 agent 的 wrapper 函数（claude、opencode、kimi、deepseek-tui 等）
- 开发者必须在容器内可直接通过 wrapper 函数名调用任意已安装的 AI agent

_优先级_：P2

## 故事 5 – 以 Docker-in-Docker 模式启动容器

作为已安装 Docker Engine 的开发者
我想要使用 --docker 或 --dind 参数以特权模式启动容器并自动启动 dockerd
以便在容器内部执行 docker 命令构建和运行其他容器

_验收标准_：

- 系统必须支持通过 `--docker` 或 `--dind` 参数触发特权模式，容器以特权模式和 root 用户启动
- 容器启动后系统必须自动启动 dockerd 守护进程
- 开发者必须在容器内可以正常执行 `docker ps` 等 docker 命令

_优先级_：P2

## 故事 6 – 恢复上次运行参数启动容器

作为已安装 Docker Engine 的开发者
我想要通过 -r 或 --recall 参数自动恢复上一次 run 命令的全部参数
以便无需重复输入端口映射、目录挂载、环境变量等配置即可快速启动相同容器

_验收标准_：

- 系统必须在每次 `run` 命令执行后自动将全部参数持久化到 `.last_args` 文件
- 系统必须支持通过 `-r/--recall` 参数从 `.last_args` 文件恢复上次运行的完整参数
- 使用 `-r` 启动的容器必须与上次运行具有完全相同的配置（agent、端口、挂载、环境变量等）
- 如果不存在 `.last_args` 文件，系统必须提示无法回忆上次运行参数

_优先级_：P2

## 故事 7 – 后台执行命令并自动退出容器

作为已安装 Docker Engine 的开发者
我想要使用 --run 参数在容器后台执行指定的命令
以便在容器化环境运行自动化测试、批处理脚本等无需交互的任务

_验收标准_：

- 系统必须支持 `--run <命令>` 参数，在后台启动容器并执行指定的命令
- 命令执行完成后容器必须自动退出
- 容器退出码必须与所执行命令的退出码一致

_优先级_：P2

## 故事 8 – 新增 LLM 端点

作为已安装 Docker Engine 的开发者
我想要通过 endpoint add 命令为 AI agent 配置 LLM 服务商的端点信息（provider、URL、API key、模型等）
以便 agent 能够连接到指定的 LLM 服务进行 AI 辅助编码

_验收标准_：

- 系统必须支持通过 `endpoint add <名称>` 新增端点，并提供 `--provider`（deepseek/openai/anthropic）、`--url`、`--key`、`--model`、`--model-opus`、`--model-sonnet`、`--model-haiku`、`--model-subagent` 八个配置选项
- 缺少必要参数时，系统必须以交互式方式逐个提问收集缺失的配置项
- 新增成功后，`endpoint list` 输出表中必须包含新创建的端点

_优先级_：P1

## 故事 9 – 修改和删除 LLM 端点

作为已安装 Docker Engine 的开发者
我想要通过 endpoint set 修改已有端点的配置，或通过 endpoint rm 删除不再需要的端点
以便灵活管理 LLM 服务配置，及时更新过期的 API key 或移除停用的服务商

_验收标准_：

- 系统必须支持通过 `endpoint set <名称>` 修改指定端点的配置参数
- 系统必须支持通过 `endpoint rm <名称>` 删除指定端点和其对应的目录
- 删除后，`endpoint list` 输出中不再显示被删除的端点

_优先级_：P2

## 故事 10 – 查看端点列表和详情

作为已安装 Docker Engine 的开发者
我想要通过 endpoint 命令查看支持的服务商列表、所有端点的概览表格，以及单个端点的详细配置
以便快速了解当前 LLM 端点的配置状态，并在排查问题时查看 API key 的完整性

_验收标准_：

- 系统必须支持 `endpoint providers` 列出所有支持的 LLM 服务商及其对应的 AI agent
- 系统必须支持 `endpoint list` 以 NAME / PROVIDER / MODEL 表格格式列出所有端点
- 系统必须支持 `endpoint show <名称>` 查看指定端点的详细配置，API key 必须掩码显示（前 8 个字符 + `***` + 后 4 个字符）

_优先级_：P2

## 故事 11 – 测试端点连通性

作为已安装 Docker Engine 的开发者
我想要通过 endpoint test 命令验证 LLM 端点是否可达且正常工作
以便在配置完成后或连接异常时快速确认端点状态，定位问题原因

_验收标准_：

- 系统必须支持通过 `endpoint test <名称>` 向指定端点发送 POST chat/completions 请求
- 测试成功时，系统必须输出请求延迟和回复摘要
- 测试失败时（端点不可达、超时、认证失败等），系统必须返回非零退出码并输出明确的错误信息
- 失败后开发者可以修正配置后重新测试

_优先级_：P1

## 故事 12 – 同步端点配置到 AI agent

作为已安装 Docker Engine 的开发者
我想要通过 endpoint apply 命令将端点配置同步到各 AI agent 的配置文件中，并通过 endpoint status 查看映射关系
以便 agent 在容器内启动时能够读取到正确的 LLM 连接信息，无需手动编辑不同格式的配置文件

_验收标准_：

- 系统必须支持通过 `endpoint apply [端点名称]` 将端点配置写入各 agent 的配置文件中（claude 写入 `.claude/.env`、opencode 写入 `.opencode/.env`、kimi 写入 `.kimi/config.toml`、deepseek-tui 写入 `.deepseek/.env`）
- 系统必须支持通过 `--agent a,b,c` 参数以逗号分隔筛选要同步的目标 agent
- 不指定端点名称时，系统必须同步全部端点配置
- 系统必须支持 `endpoint status [端点名称]` 查看各 agent 与端点的映射关系，输出表格包含 agent 名称和关联的端点

_优先级_：P1

## 故事 13 – 诊断环境并自动修复缺失依赖

作为已安装 Docker Engine 的开发者
我想要执行 doctor 命令自动检测开发环境中的核心依赖、运行时和可选工具，并自动安装缺失的组件
以便在使用 AgentForge 之前确保所有前置条件满足，避免因环境问题导致命令执行失败

_验收标准_：

- 系统必须按三层顺序执行诊断：核心依赖（docker）→ 运行时（Docker daemon 运行状态、权限）→ 可选工具（jq、buildx）
- 核心依赖缺失时，系统必须使用 apt-get / dnf / yum / brew 自动安装缺失的依赖
- 自动安装后，系统必须重新检测以确保依赖已被正确安装
- 系统必须输出每一层的诊断结果，清晰标注每个组件的安装状态

_优先级_：P2

## 故事 14 – 查询容器内依赖安装状态

作为已安装 Docker Engine 的开发者
我想要通过 deps 命令自动生成检测脚本，在临时容器中分类检测 agent、skill、tool、runtime 的安装状态和版本号
以便在构建后或不启动交互容器的情况下快速确认镜像中各组件的可用性

_验收标准_：

- 系统必须支持 `deps` 命令在宿主机执行，自动生成检测脚本并通过 `docker run --rm` 在目标镜像的临时容器中运行
- 系统必须按 agent / skill / tool / runtime 分类回显各组件的安装状态和版本号
- 检测完成后临时容器必须自动销毁

_优先级_：P2

## 故事 15 – 导出和导入 Docker 镜像实现离线分发

作为已安装 Docker Engine 的开发者
我想要通过 export 命令将构建好的镜像导出为 tar 文件，并在另一台机器上通过 import 命令加载
以便在无网络或受限网络环境中分发和复用容器化开发环境

_验收标准_：

- 系统必须支持 `export [文件名]`（默认 `dockercoding.tar`）将 Docker 镜像导出为 tar 文件
- 系统必须支持 `import <文件>` 从 tar 文件加载 Docker 镜像
- 导入成功后，`docker images` 中必须能看到加载的镜像
- 导入的镜像必须可以正常通过 `run` 启动容器使用 AI agent

_优先级_：P2

## 故事 16 – 管理工具自身信息和帮助

作为已安装 Docker Engine 的开发者
我想要通过 update 更新工具版本、通过 version 查看版本信息，以及通过 help 获取各命令的帮助说明
以便随时保持工具处于最新状态，并快速了解各命令的用法和参数

_验收标准_：

- 系统必须支持 `update` 命令从 Git remote 或 UPDATE_URL 下载更新，并嵌入 git hash 标识版本
- 更新成功后，`version` 输出的版本号和 git hash 必须反映新版本
- 系统必须支持 `version` 命令输出格式化的版本号和 git hash
- 每个命令和子命令都必须支持 `--help` 或 `help` 参数，输出格式一致的帮助信息

_优先级_：P3

