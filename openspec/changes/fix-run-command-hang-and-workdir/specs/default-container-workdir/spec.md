## ADDED Requirements

### Requirement: 工作目录确定与自动挂载

系统 SHALL 按以下优先级确定工作目录路径，并将该路径以读写方式绑定挂载到容器内相同位置（1:1），设为容器 WorkingDir：
1. `-w/--workdir` 显式指定值
2. 主机当前工作目录 `os.Getwd()`
3. `/workspace`（兜底）

`-m/--mount` 指定的多个只读挂载目录不参与工作目录推导。

#### Scenario: 未指定 -w 时使用主机当前目录并自动挂载

- **WHEN** 用户在 `/home/user/project` 执行 `agent-forge run` 且未提供 `-w` 标志
- **THEN** 主机 `/home/user/project` 以读写方式绑定挂载到容器内同路径
- **AND** 容器工作目录设置为 `/home/user/project`
- **AND** 用户进入容器后 `pwd` 输出 `/home/user/project`

#### Scenario: 显式指定 -w 时自动挂载并设为工作目录

- **WHEN** 用户执行 `agent-forge run -w /custom`
- **THEN** 主机 `/custom` 以读写方式绑定挂载到容器内同路径
- **AND** 容器工作目录设置为 `/custom`
- **AND** 忽略主机 PWD

#### Scenario: os.Getwd 失败时 fallback

- **WHEN** 系统无法获取主机当前工作目录（`os.Getwd()` 返回错误）且未提供 `-w`
- **THEN** 容器工作目录设置为 `/workspace`

#### Scenario: -m 多个只读挂载不影响工作目录

- **WHEN** 用户执行 `agent-forge run -m /host/ref1 -m /host/ref2` 且未提供 `-w`
- **THEN** `/host/ref1` 和 `/host/ref2` 分别以只读方式挂载到容器内同路径
- **AND** 容器工作目录为主机当前目录（从 `os.Getwd()` 获取），不受 `-m` 影响
