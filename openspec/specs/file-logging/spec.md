## Purpose

定义 AgentForge CLI 的文件日志系统——按日期轮转、级别过滤、JSON 格式输出。

## Requirements

### Requirement: 日志写入配置目录下的日期文件

系统 SHALL 将日志写入 `<config-dir>/log/<YYYY-MM-DD>.log`，自动按天轮转。

#### Scenario: 同一天日志追加到同一文件

- **WHEN** 同一天内多次执行 CLI 命令
- **THEN** 日志 SHALL 追加到当天的 `<YYYY-MM-DD>.log` 文件末尾
- **AND** 文件内容为 JSON 格式，每行一条日志记录

#### Scenario: 跨天自动创建新文件

- **WHEN** 日期发生变化后执行命令
- **THEN** 系统 SHALL 创建新日期的 `.log` 文件
- **AND** 旧文件保持不变

#### Scenario: 日志目录自动创建

- **WHEN** `<config-dir>/log/` 目录不存在
- **THEN** 系统 SHALL 自动创建该目录

### Requirement: 支持日志级别配置

系统 SHALL 支持通过环境变量 `AGENTFORGE_LOG_LEVEL` 配置日志级别，值为 `debug`/`info`/`warn`/`error`，不区分大小写。

#### Scenario: 默认日志级别为 info

- **WHEN** 环境变量 `AGENTFORGE_LOG_LEVEL` 未设置
- **THEN** 系统 SHALL 使用 `info` 级别，记录 info/warn/error 日志，忽略 debug 日志

#### Scenario: 设置为 debug 级别记录所有日志

- **WHEN** `AGENTFORGE_LOG_LEVEL=debug`
- **THEN** 系统 SHALL 记录所有级别的日志（debug/info/warn/error）

#### Scenario: 非法级别值回退到默认

- **WHEN** `AGENTFORGE_LOG_LEVEL` 设置为非法值（如 `verbose`）
- **THEN** 系统 SHALL 回退到默认 `info` 级别

### Requirement: 日志初始化失败不阻塞命令

日志系统初始化失败时 SHALL NOT 导致命令执行失败，仅静默跳过日志功能。

#### Scenario: 日志目录无写入权限时降级

- **WHEN** `<config-dir>/log/` 目录无法创建或写入
- **THEN** 系统 SHALL 静默跳过日志初始化
- **AND** 命令正常执行，无日志文件产生

### Requirement: 日志格式为 JSON

每条日志 SHALL 为一行 JSON，包含 `time`（ISO 8601）、`level`（DEBUG/INFO/WARN/ERROR）、`msg`（消息内容）及可选的键值对。

#### Scenario: 日志行格式正确

- **WHEN** 系统记录一条 `info` 级别日志 "容器启动成功" 附带 `container_id=abc123`
- **THEN** 日志行 SHALL 包含 `"level":"INFO"`、`"msg":"容器启动成功"`、`"container_id":"abc123"`
