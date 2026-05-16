## Purpose

定义 Docker 镜像构建过程中 `docker` 依赖的跨发行版安装方式，以及系统包管理命令的发行版适配规则。

## Requirements

### Requirement: Docker CLI 跨发行版安装

系统 SHALL 在所有支持的基础镜像（CentOS 7/8、RHEL、Fedora、Ubuntu、Debian）上成功安装 Docker CLI，不依赖系统包管理器（yum/apt）。

#### Scenario: CentOS 7 基础镜像安装 Docker CLI

- **WHEN** 基础镜像为 `centos:7` 且依赖列表包含 `docker`
- **THEN** Docker CLI 通过官方静态二进制下载安装成功
- **AND** `docker --version` 可正常执行

#### Scenario: Ubuntu 基础镜像安装 Docker CLI

- **WHEN** 基础镜像为 `ubuntu:22.04` 且依赖列表包含 `docker`
- **THEN** Docker CLI 通过官方静态二进制下载安装成功
- **AND** 未调用 `apt-get install -y docker`

#### Scenario: Debian 基础镜像安装 Docker CLI

- **WHEN** 基础镜像为 `debian:12` 且依赖列表包含 `docker`
- **THEN** Docker CLI 通过官方静态二进制下载安装成功
- **AND** 未调用 `apt-get install -y docker`

### Requirement: 系统包名跨发行版映射

`adaptCommandForFamily` 函数 SHALL 将常见 RHEL 系包名映射为 Debian 系等效包名，确保 `DepSystemPkg` 类型的未知依赖在 Debian 系基础镜像上使用正确的包名。

#### Scenario: 未知系统包名自动映射

- **WHEN** 依赖列表包含未知名称（归为 `DepSystemPkg`）且基础镜像为 Debian 系
- **THEN** `yum install -y <pkg>` 被翻译为 `apt-get install -y <mapped-pkg>`
- **AND** 已知映射表中 `docker` → `docker.io` 生效

#### Scenario: RHEL 系基础镜像不触发包名映射

- **WHEN** 基础镜像为 CentOS/RHEL/Fedora 系
- **THEN** 原始 `yum` 命令保持不变，不进行包名转换
