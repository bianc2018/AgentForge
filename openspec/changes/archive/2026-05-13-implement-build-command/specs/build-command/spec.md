## ADDED Requirements

### Requirement: Build command executes Docker image build

`agent-forge build` 命令 SHALL 调用构建引擎执行实际的 Docker 镜像构建，而非输出占位文本。

#### Scenario: Build with all deps and custom base image

- **WHEN** 用户执行 `agent-forge build -b docker.1ms.run/ubuntu:22.04 -d all`
- **THEN** 系统 SHALL 解析参数，生成 Dockerfile，创建构建上下文，调用 Docker Engine API 构建镜像，并输出构建日志

#### Scenario: Build with default parameters

- **WHEN** 用户执行 `agent-forge build`（无任何参数）
- **THEN** 系统 SHALL 使用默认基础镜像 `docker.1ms.run/centos:7` 和空依赖列表执行构建

#### Scenario: Build with rebuild mode

- **WHEN** 用户执行 `agent-forge build -R`
- **THEN** 系统 SHALL 使用临时标签构建，成功后原子替换 `agent-forge:latest` 标签

#### Scenario: Build with no-cache flag

- **WHEN** 用户执行 `agent-forge build --no-cache`
- **THEN** 系统 SHALL 在 ImageBuild API 调用中设置 `NoCache: true`，强制跳过 Docker 构建缓存

#### Scenario: Build with GitHub proxy

- **WHEN** 用户执行 `agent-forge build --gh-proxy https://ghproxy.net`
- **THEN** 系统 SHALL 将代理 URL 传递给 Dockerfile 生成器，用代理包装 GitHub 下载 URL

#### Scenario: Build fails when Docker daemon is unreachable

- **WHEN** Docker daemon 未运行，用户执行 `agent-forge build`
- **THEN** 系统 SHALL 返回错误并以非零退出码退出

### Requirement: Build command supports all declared flags

build 命令 SHALL 接受并正确处理所有在 `init()` 中已声明的 flag，与 `buildengine.BuildParams` 字段一一对应。

#### Scenario: All flags are parsed into BuildParams

- **WHEN** 用户执行 `agent-forge build -b ubuntu:22.04 -d claude,golang --no-cache -R --max-retry 5 --gh-proxy https://example.com -c /path/to/config`
- **THEN** 系统 SHALL 将每个 flag 值正确映射到 `BuildParams` 对应字段，并传递给 `engine.Build()`
