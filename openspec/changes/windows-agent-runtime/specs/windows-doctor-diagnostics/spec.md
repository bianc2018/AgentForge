## ADDED Requirements

### Requirement: doctor 诊断新增第四层——平台兼容性

系统 SHALL 在 `diagnosticengine` 中新增第四层诊断（平台兼容性），检查项包括：
- 基础镜像平台与 Docker daemon OSType 是否匹配
- 若检测到 Windows 镜像但 daemon 为 Linux → 报告不兼容并给出建议
- Windows daemon 版本检查（最低要求 Docker Engine 20.10+，支持 Windows 容器）

#### Scenario: 全通过——Linux daemon + Linux 镜像

- **WHEN** 用户执行 `agent-forge doctor` 且 daemon OSType 为 `linux`、默认镜像为 Linux
- **THEN** 第四层诊断为通过
- **AND** 输出 `第四层 - 平台兼容性: 通过`

#### Scenario: 不兼容——Linux daemon + Windows 镜像

- **WHEN** Docker daemon OSType 为 `linux` 但环境中有 Windows 镜像缓存
- **THEN** 第四层诊断为未通过
- **AND** 输出建议 `当前 Docker daemon 不支持 Windows 容器`

### Requirement: doctor 平台诊断不影响前三层

系统 SHALL 保持现有三层诊断（核心依赖、运行时、可选工具）的逻辑不变，第四层失败不阻止前三层执行。

#### Scenario: 前三层通过但第四层失败

- **WHEN** Docker 正常运行但平台兼容性不佳
- **THEN** 前三层仍输出 `通过`
- **AND** 第四层输出 `未通过` 及建议
- **AND** CLI 以退出码 1 退出
