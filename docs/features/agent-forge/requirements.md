# 功能需求 — AgentForge

## 1. 镜像构建 (Image Build)

**REQ-1**: When the developer executes the build command with the `-d` parameter specifying dependencies, the system shall build a Docker image containing the listed AI agents, runtimes, and tools.

> 来源：PRD 目标 1 / Story 1 验收标准 / BDD 场景 "构建包含全部依赖的镜像"、"构建包含指定依赖的自定义镜像"

**REQ-2**: When the developer specifies the `-b` parameter for base image and the `-c` parameter for configuration directory, the system shall use the specified base image and configuration parent directory during the build process.

> 来源：PRD 目标 2 / Story 1 验收标准 / BDD 场景 "构建包含指定依赖的自定义镜像"

**REQ-3**: While building the Docker image, the system shall automatically use domestic mirror sources (npm to npmmirror, pip to aliyun, yum to aliyun centos vault) to accelerate dependency downloads.

> 来源：PRD 目标 3 / Story 1 验收标准

**REQ-4**: If a network error occurs during the build process, the system shall automatically retry with exponential backoff strategy up to the number of times specified by `--max-retry` (default 3).

> 来源：PRD 目标 3 / Story 1 验收标准 / BDD 场景 "构建过程中网络错误时自动重试"

**REQ-5**: When the developer specifies `--gh-proxy` with a URL, the system shall use the specified GitHub proxy for resource downloads during the build process.

> 来源：PRD 目标 2 / Story 1 验收标准 / BDD 场景 "构建过程中网络错误时自动重试"

**REQ-6**: When the developer passes the `--no-cache` flag to the build command, the system shall force skip the Docker cache during image building.

> 来源：PRD 目标 2 / BDD 场景 "重建镜像成功替换旧标签"

**REQ-7**: When the developer executes the build command with the `-R/--rebuild` flag, the system shall build with a temporary tag, and upon success replace the original image tag with the new image and delete the old image, exiting with code 0.

> 来源：PRD 替代流程 "重建模式 — rebuild" / Story 2 验收标准 / BDD 场景 "重建镜像成功替换旧标签"

**REQ-8**: If the rebuild process fails, the system shall clean up the temporary tag, preserve the original image unchanged, and exit with a non-zero code.

> 来源：PRD 替代流程 "重建模式 — rebuild" / Story 2 验收标准 / BDD 场景 "重建失败时保留旧镜像"

## 2. 容器运行 — Agent 交互终端 (Container Run: Agent Interactive Terminal)

**REQ-9**: When the developer executes the run command with the `-a` parameter specifying an AI agent (claude, opencode, kimi, or deepseek-tui), the system shall start a container and launch the corresponding interactive terminal.

> 来源：PRD 目标 4 / Story 3 验收标准 / BDD 场景 "启动指定 agent 带完整配置的交互式终端"

**REQ-10**: When the developer specifies the `-p` parameter with port mappings, the system shall publish the specified host ports to the corresponding container ports.

> 来源：PRD 目标 5 / Story 3 验收标准 / BDD 场景 "启动指定 agent 带完整配置的交互式终端"

**REQ-11**: When the developer specifies the `-m` parameter with directory paths, the system shall mount the specified host directories as read-only at the same paths inside the container.

> 来源：PRD 目标 5 / Story 3 验收标准 / BDD 场景 "启动指定 agent 带完整配置的交互式终端"

**REQ-12**: When the developer specifies the `-e` parameter with KEY=VALUE pairs, the system shall inject the specified environment variables into the container.

> 来源：PRD 目标 5 / Story 3 验收标准 / BDD 场景 "启动指定 agent 带完整配置的交互式终端"

**REQ-13**: When the developer specifies the `-w` parameter with a directory path, the system shall set the specified directory as the working directory inside the container (defaulting to the current directory).

> 来源：PRD 目标 5 / Story 3 验收标准 / BDD 场景 "启动指定 agent 带完整配置的交互式终端"

## 3. 容器运行 — 特殊模式 (Container Run: Special Modes)

**REQ-14**: When the developer executes the run command without the `-a` parameter, the system shall start the container and enter bash shell with all installed AI agent wrapper functions loaded.

> 来源：PRD 替代流程 "无特定 agent — bash 模式" / Story 4 验收标准 / BDD 场景 "不指定 agent 以 bash 模式启动容器"

**REQ-15**: When the developer executes the run command with the `--docker` or `--dind` parameter, the system shall start the container in privileged mode with root user and automatically start the dockerd daemon inside the container.

> 来源：PRD 替代流程 "Docker-in-Docker 模式" / Story 5 验收标准 / BDD 场景 "以 Docker-in-Docker 特权模式启动容器"

**REQ-16**: When the developer executes the run command with the `-r/--recall` parameter, the system shall restore the full set of parameters from the `.last_args` file and start the container with the identical configuration as the previous run.

> 来源：PRD 替代流程 "恢复上次运行参数" / Story 6 验收标准 / BDD 场景 "通过 -r 参数恢复上次运行参数启动容器"

**REQ-17**: If the `.last_args` file does not exist, the system shall notify the developer that the previous run parameters cannot be recalled and shall not start the container.

> 来源：Story 6 验收标准 / BDD 场景 "不存在历史参数时使用 -r 恢复失败"

**REQ-18**: When the developer executes the run command with `--run <command>`, the system shall start the container in the background, execute the specified command, and automatically exit the container with the same exit code as the executed command.

> 来源：PRD 目标 6 / 替代流程 "后台执行模式" / Story 7 验收标准 / BDD 场景 "后台执行命令后自动退出容器"

## 4. LLM 端点 — 查看和新增 (Endpoint: View & Add)

**REQ-19**: When the developer executes `endpoint providers`, the system shall list all supported LLM providers and their corresponding AI agents.

> 来源：PRD 目标 8 / Story 10 验收标准 / BDD 场景 "查看提供商列表和端点详情"

**REQ-20**: When the developer executes `endpoint list`, the system shall display all endpoints in a table with columns NAME, PROVIDER, and MODEL.

> 来源：PRD 目标 8 / Story 10 验收标准 / BDD 场景 "查看提供商列表和端点详情"

**REQ-21**: When the developer executes `endpoint show <name>`, the system shall display the detailed configuration of the specified endpoint with the API key masked as the first 8 characters plus `***` plus the last 4 characters.

> 来源：PRD 目标 8 / Story 10 验收标准 / BDD 场景 "查看提供商列表和端点详情"

**REQ-22**: When the developer executes `endpoint add <name>` with all required parameters (--provider, --url, --key, --model, --model-opus, --model-sonnet, --model-haiku, --model-subagent), the system shall create a new endpoint with the specified configuration.

> 来源：PRD 目标 9 / Story 8 验收标准 / BDD 场景 "带全部参数新增 LLM 端点"

**REQ-23**: When the developer executes `endpoint add <name>` without providing all required parameters, the system shall interactively prompt for each missing configuration item.

> 来源：PRD 目标 9 / Story 8 验收标准 / BDD 场景 "缺少参数时交互式新增 LLM 端点"

## 5. LLM 端点 — 修改、删除和测试 (Endpoint: Modify, Delete & Test)

**REQ-24**: When the developer executes `endpoint set <name>` with updated parameters, the system shall modify the specified endpoint's configuration with the provided values.

> 来源：PRD 目标 10 / Story 9 验收标准 / BDD 场景 "修改已有端点的配置"

**REQ-25**: When the developer executes `endpoint rm <name>`, the system shall delete the specified endpoint and its corresponding directory.

> 来源：PRD 目标 10 / Story 9 验收标准 / BDD 场景 "删除 LLM 端点"

**REQ-26**: When the developer executes `endpoint test <name>` and the endpoint is reachable, the system shall send a POST chat/completions request and output the request latency and response summary with exit code 0.

> 来源：PRD 目标 10 / Story 11 验收标准 / BDD 场景 "测试端点连通性成功"

**REQ-27**: If the endpoint is unreachable, times out, or returns an authentication error during `endpoint test <name>`, the system shall output a clear error message and return a non-zero exit code.

> 来源：PRD 替代流程 "端点测试失败" / Story 11 验收标准 / BDD 场景 "测试端点连通性失败"

## 6. LLM 端点 — 同步和映射 (Endpoint: Sync & Mapping)

**REQ-28**: When the developer executes `endpoint apply [name]` without specifying `--agent`, the system shall write the endpoint configuration into the configuration files of all applicable AI agents (claude to `.claude/.env`, opencode to `.opencode/.env`, kimi to `.kimi/config.toml`, deepseek-tui to `.deepseek/.env`).

> 来源：PRD 目标 11 / Story 12 验收标准 / BDD 场景 "同步端点配置到 agent"

**REQ-29**: When the developer executes `endpoint apply` with the `--agent` parameter specifying a comma-separated list of agents, the system shall only sync the endpoint configuration to the specified agents' configuration files.

> 来源：PRD 目标 11 / Story 12 验收标准 / BDD 场景 "同步端点配置到 agent"

**REQ-30**: When the developer executes `endpoint status`, the system shall display a table showing each agent name and its associated endpoint name.

> 来源：PRD 目标 11 / Story 12 验收标准 / BDD 场景 "查看 agent 端点映射关系"

## 7. 环境诊断和依赖查询 (Diagnosis & Dependency Query)

**REQ-31**: When the developer executes the doctor command, the system shall perform three-layer environment diagnosis in sequential order: core dependencies (docker, curl, git), runtime (Docker daemon status, permissions), and optional tools (jq, buildx), and output the diagnosis result for each layer.

> 来源：PRD 目标 13 / Story 13 验收标准 / BDD 场景 "环境诊断"

**REQ-32**: If the doctor command detects missing core dependencies, the system shall automatically install the missing components using apt-get, dnf, yum, or brew, and re-check after installation.

> 来源：PRD 替代流程 "缺失依赖自动修复" / Story 13 验收标准 / BDD 场景 "环境诊断"

**REQ-33**: When the developer executes the deps command on the host, the system shall generate a detection script, execute it via `docker run --rm` in a temporary container from the target image, output the installation status and version of each component classified as agent, skill, tool, and runtime, and automatically destroy the temporary container after completion.

> 来源：PRD 目标 12 / Story 14 验收标准 / BDD 场景 "查询容器内依赖安装状态"

## 8. 离线分发、自更新和帮助 (Offline Distribution, Update & Help)

**REQ-34**: When the developer executes `export [filename]`, the system shall export the Docker image as a tar file (defaulting to `dockercoding.tar`).

> 来源：PRD 目标 14 / Story 15 验收标准 / BDD 场景 "导出和导入镜像实现离线分发"

**REQ-35**: When the developer executes `import <file>`, the system shall load the Docker image from the specified tar file, making it visible in `docker images` and usable with the run command.

> 来源：PRD 目标 14 / Story 15 验收标准 / BDD 场景 "导出和导入镜像实现离线分发"

**REQ-36**: When the developer executes the update command, the system shall download the latest version from Git remote or UPDATE_URL, embed the new git hash, and update the version number. When the developer executes the version command, the system shall output the formatted version number and current git hash.

> 来源：PRD 目标 15 / Story 16 验收标准 / BDD 场景 "工具自更新和版本信息查看"

**REQ-37**: When the developer appends `--help` to any command or subcommand, the system shall output consistently formatted help information including command usage and parameter descriptions.

> 来源：PRD 目标 16 / Story 16 验收标准 / BDD 场景 "工具自更新和版本信息查看"
