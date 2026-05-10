Feature: AgentForge
  为开发者提供统一的 CLI 工具，通过 Docker 容器化技术一键构建、运行和管理多种 AI coding agent 及其运行时环境。

  Scenario: 构建包含全部依赖的镜像
    Given Docker Engine 已安装并运行
    When 开发者执行 build -d all --max-retry 3
    Then 构建过程退出码为 0
    And docker images 列表中包含新生成的镜像

  Scenario: 构建包含指定依赖的自定义镜像
    Given Docker Engine 已安装并运行
    When 开发者执行 build -d claude,golang@1.21,node@20 -b docker.1ms.run/centos:7 -c /path/to/config
    Then 构建过程退出码为 0
    And 容器内 go version 输出 1.21.x
    And 容器内 node --version 输出 20.x

  Scenario: 构建过程中网络错误时自动重试
    Given Docker Engine 已安装并运行
    And 构建过程中首次请求 GitHub 资源超时
    When 开发者执行 build -d claude --max-retry 3 --gh-proxy https://gh-proxy.example.com
    Then 系统按指数退避策略自动重试
    And 在三次重试内构建成功
    And 构建过程退出码为 0

  Scenario: 重建镜像成功替换旧标签
    Given 存在一个已构建的镜像 agent-forge:latest
    When 开发者执行 build -R -d claude,golang@1.21
    Then 系统自动叠加 --no-cache 强制跳过缓存
    And 构建成功后临时标签替换原镜像标签
    And 旧镜像被删除
    And 构建过程退出码为 0

  Scenario: 重建失败时保留旧镜像
    Given 存在一个已构建的镜像 agent-forge:latest
    When 开发者执行 build -R -d invalid-package-that-fails
    Then 系统自动叠加 --no-cache 强制跳过缓存
    And 构建失败后清理临时标签
    And 原镜像 agent-forge:latest 保持不变
    And 构建过程退出码非零

  Scenario: 启动指定 agent 带完整配置的交互式终端
    Given 已构建 AgentForge 镜像
    When 开发者执行 run -a claude -p 3000:3000 -m /host/data -w /workspace -e OPENAI_KEY=sk-xxx
    Then 容器启动并进入 claude 交互式终端
    And 容器内端口 3000 可访问
    And 容器内 /host/data 目录存在且挂载自宿主机
    And 容器内工作目录为 /workspace
    And 容器内环境变量 OPENAI_KEY 值为 sk-xxx

  Scenario: 不指定 agent 以 bash 模式启动容器
    Given 已构建 AgentForge 镜像
    When 开发者执行 run 命令且不指定 -a 参数
    Then 容器启动并进入 bash shell
    And bash 环境中自动加载了 claude、opencode、kimi、deepseek-tui 等 wrapper 函数
    And 开发者可在容器内直接通过 wrapper 函数名调用任意已安装的 AI agent

  Scenario: 以 Docker-in-Docker 特权模式启动容器
    Given 已构建 AgentForge 镜像
    When 开发者执行 run --docker
    Then 容器以特权模式和 root 用户启动
    And 容器内 dockerd 守护进程自动启动
    And 容器内可正常执行 docker ps 等 docker 命令

  Scenario: 通过 -r 参数恢复上次运行参数启动容器
    Given 开发者之前执行过一次 run -a claude -p 3000:3000 -m /host/data
    And .last_args 文件已自动持久化上次运行的全部参数
    When 开发者执行 run -r
    Then 系统从 .last_args 文件恢复上次运行的完整参数
    And 容器以与上次运行完全相同的配置启动
    And 容器内 claude 交互式终端可用，端口 3000 已映射，/host/data 目录已挂载

  Scenario: 不存在历史参数时使用 -r 恢复失败
    Given 从未执行过 run 命令或 .last_args 文件不存在
    When 开发者执行 run -r
    Then 系统提示无法回忆上次运行参数
    And 容器不会启动

  Scenario: 后台执行命令后自动退出容器
    Given 已构建 AgentForge 镜像
    When 开发者执行 run --run "npm test"
    Then 容器在后台启动并执行 npm test 命令
    And 命令执行完成后容器自动退出
    And 容器退出码与 npm test 的退出码一致

  Scenario: 带全部参数新增 LLM 端点
    When 开发者执行 endpoint add my-ep --provider openai --url https://api.openai.com --key sk-test-key-value --model gpt-4 --model-opus gpt-4-32k --model-sonnet gpt-4-turbo --model-haiku gpt-3.5-turbo --model-subagent gpt-4-mini
    Then 端点 my-ep 创建成功
    And endpoint list 输出表中包含 my-ep
    And endpoint show my-ep 显示 API key 为 sk-test***alue 掩码格式

  Scenario: 缺少参数时交互式新增 LLM 端点
    When 开发者执行 endpoint add my-ep 且未提供 --provider 和 --url 参数
    Then 系统逐个提问缺失的配置项：provider、url、model
    And 开发者依次输入 deepseek、https://api.deepseek.com
    Then 端点 my-ep 创建成功
    And endpoint list 输出表中包含 my-ep

  Scenario: 修改已有端点的配置
    Given 存在已创建的端点 my-ep
    When 开发者执行 endpoint set my-ep --key sk-new-key --model gpt-5
    Then 端点 my-ep 的 API key 更新为 sk-new-key
    And 端点 my-ep 的模型更新为 gpt-5

  Scenario: 删除 LLM 端点
    Given 存在已创建的端点 my-ep
    When 开发者执行 endpoint rm my-ep
    Then 端点 my-ep 及其对应目录被删除
    And endpoint list 输出中不再包含 my-ep

  Scenario: 查看提供商列表和端点详情
    Given 存在已创建的端点 my-ep
    When 开发者执行 endpoint providers
    Then 输出列出所有支持的 LLM 服务商及其对应的 AI agent
    When 开发者执行 endpoint list
    Then 输出以 NAME / PROVIDER / MODEL 表格格式列出所有端点
    When 开发者执行 endpoint show my-ep
    Then 输出显示 my-ep 的详细配置
    And API key 显示为前 8 字符加 *** 加后 4 字符的掩码格式

  Scenario: 测试端点连通性成功
    Given 存在已创建的可达端点 my-ep
    When 开发者执行 endpoint test my-ep
    Then 系统向端点发送 POST chat/completions 请求
    And 输出包含请求延迟和回复摘要
    And 退出码为 0

  Scenario: 测试端点连通性失败
    Given 存在已创建的不可达端点 broken-ep
    When 开发者执行 endpoint test broken-ep
    Then 系统向端点发送 POST chat/completions 请求
    And 请求失败（连接超时、认证失败或端点不可达）
    And 输出明确的错误信息
    And 退出码非零

  Scenario: 同步端点配置到 agent
    Given 存在已创建的端点 my-ep
    When 开发者执行 endpoint apply my-ep
    Then 端点 my-ep 的配置写入 claude 的 .claude/.env 文件
    And 写入 opencode 的 .opencode/.env 文件
    And 写入 kimi 的 .kimi/config.toml 文件
    And 写入 deepseek-tui 的 .deepseek/.env 文件
    When 开发者执行 endpoint apply my-ep --agent claude,kimi
    Then 端点 my-ep 的配置仅写入 claude 和 kimi 的配置文件
    And opencode 和 deepseek-tui 的配置文件不受影响

  Scenario: 查看 agent 端点映射关系
    Given 存在已创建的端点 my-ep
    When 开发者执行 endpoint status
    Then 输出表格包含每个 agent 名称和其关联的端点名称

  Scenario: 环境诊断
    Given Docker Engine 已安装并运行
    And curl、git 等核心依赖已安装
    When 开发者执行 doctor
    Then 核心依赖检查全部通过
    And 运行时检查 Docker daemon 运行状态正常
    And 可选工具检查 jq、buildx 安装状态
    And 所有三层诊断输出均为通过状态
    Given Docker Engine 已安装
    And curl 或 git 等核心依赖缺失
    When 开发者执行 doctor
    Then 系统检测到缺失的核心依赖
    And 自动使用 apt-get / dnf / yum / brew 安装缺失组件
    And 安装完成后重新检测
    And 修复后诊断全部通过

  Scenario: 查询容器内依赖安装状态
    Given 已构建 AgentForge 镜像
    When 开发者在宿主机执行 deps
    Then 系统自动生成检测脚本
    And 通过 docker run --rm 在临时容器中执行检测
    And 输出按 agent / skill / tool / runtime 分类显示安装状态和版本号
    And 检测完成后临时容器自动销毁

  Scenario: 导出和导入镜像实现离线分发
    Given 已构建 AgentForge 镜像
    When 开发者执行 export agent-forge.tar
    Then 镜像被导出为 agent-forge.tar 文件
    When 开发者在另一台机器上执行 import agent-forge.tar
    Then docker images 中显示已加载的镜像
    And 可使用 run -a claude 正常启动容器

  Scenario: 工具自更新和版本信息查看
    Given Git remote 或 UPDATE_URL 中有新版本
    When 开发者执行 update
    Then 系统从远端下载更新
    And 嵌入新的 git hash
    And 系统版本号更新
    When 开发者执行 version
    Then 输出格式化的版本号和当前 git hash
    When 开发者执行任意命令的 --help
    Then 输出格式一致的帮助信息，包含命令用法和参数说明

