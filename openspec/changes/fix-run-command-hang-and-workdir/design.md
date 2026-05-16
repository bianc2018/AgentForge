## Context

`agent-forge run` 在交互模式下通过 Docker Engine API 完成以下流程：

1. `AssembleContainerConfig` 构建容器配置（Cmd、Tty、WorkingDir、Env、Mounts 等）
2. `ContainerCreate` 创建容器
3. `ContainerStart` 启动容器
4. `ContainerAttach` 建立 attach 连接
5. 终端设为原始模式（raw mode）
6. 启动两个 goroutine 做双向流复制（stdin→attach, attach→stdout）
7. `select` 阻塞等待流结束或信号

当前存在两个问题：

- **问题 1（I/O 挂起假象）**：容器启动后，Docker daemon 分配的 PTY 尺寸与用户终端不匹配，且 bash 在无 TERM 环境变量时可能不输出提示符。附加后无初始输出触发，用户看到空白屏幕误以为"卡住"。按 Ctrl+C 时 `0x03` 字节传入容器，bash 收到 SIGINT 打印 `^C` 和新提示符，才产生可见输出。

- **问题 2（工作目录错误）**：`params.Workdir` 零值为 `""`，传入容器配置后 Docker 回退到镜像默认 WORKDIR。当前 Dockerfile 无 WORKDIR 指令，centos:7 镜像默认 `/`，且无自动从挂载路径推导工作目录的逻辑。

## Goals / Non-Goals

**Goals:**
- 附加到容器后立即同步终端尺寸，确保 bash 能正确渲染提示符
- 容器启动后短时间内用户即能看到 bash 提示符
- 提供合理的默认工作目录：`-w` 指定时自动挂载其路径（rw）并设为 WorkingDir，否则挂载主机 `$PWD`（rw）并设为 WorkingDir，`os.Getwd()` 失败时 fallback `/workspace`
- 注入 `TERM` 环境变量以保障终端兼容性

**Non-Goals:**
- 不修改 `--run` 后台命令模式的行为
- 不修改 Dockerfile 生成逻辑（保持无 WORKDIR 指令）
- 不增加新的 CLI 标志
- 不改动 `--docker` 或 `--recall` 模式

## Decisions

### Decision 1: 在 ContainerAttach 之后调用 ContainerResize 同步终端尺寸

**选择**：在 `Run()` 方法中，`ContainerAttach` 成功后、进入原始模式之前，调用 `ContainerResize` 将当前终端尺寸传递给 Docker PTY。

**备选方案**：
- *方案 A（未选）*：在流复制期间监听 SIGWINCH 动态调整。这是最佳实践，但本次修复范围仅限于解决初始假死，动态调整留作后续增强。
- *方案 B（未选）*：启动容器后 sleep 固定时间等待 bash 就绪。不可靠且引入不必要的延迟。

**理由**：ContainerResize 能立即让 Docker PTY 知晓正确的行列数，bash 在 `TIOCGWINSZ` 后能正确渲染提示符。一次同步即可解决核心问题。

### Decision 2: 注入 TERM 环境变量（自动检测 + 兜底）

**选择**：在 `AssembleContainerConfig` 中，从主机环境变量 `os.Getenv("TERM")` 读取终端类型。若主机 TERM 非空则直接使用，若为空则 fallback 到 `xterm-256color`。用户通过 `-e TERM=...` 显式指定时始终覆盖自动检测值。

**备选方案**：
- *方案 A（未选）*：写死 `xterm-256color`。简单但忽略了用户实际终端类型（如 tmux 的 `screen-256color`、kitty 的 `xterm-kitty`），可能导致转义序列不匹配。
- *方案 B（未选）*：在容器中安装 ncurses。大幅增加镜像体积。

**理由**：现代终端模拟器均会导出 `$TERM`，直接透传能保证容器内 terminfo 与用户实际终端一致。`xterm-256color` 仅作为极端情况（TERM 未设置）的兜底值。

### Decision 3: 默认工作目录策略（-w 优先，兜底主机 PWD）

**选择**：`-m` 是只读参考挂载（多个），`-w` 是读写工作目录（单个），两者语义不同不混淆。`-w` 和默认 PWD 均需自动绑定挂载到容器内同路径（1:1 读写），否则容器内该路径不存在或无意义。逻辑：
1. 确定工作目录路径：`-w` 值 > `os.Getwd()` > `/workspace`
2. 将该路径以读写方式绑定挂载到容器内相同路径（1:1）
3. 设为容器 WorkingDir

**备选方案**：
- *方案 A（未选）*：默认使用 `/root`。无语义，用户困惑。
- *方案 B（未选）*：从 `-m` 挂载目标推导工作目录。`-m` 挂载是只读参考目录，不应作为读写工作目录。

**理由**：对齐 `docker run -v $(pwd):$(pwd) -w $(pwd)` 的标准行为。用户在没有 `-w` 时自然期望进入当前目录，而非手动 cd。`/workspace` 仅作为极端兜底。

### Decision 4: 在 AssembleContainerConfig 中计算默认工作目录

**选择**：工作目录的默认值计算逻辑放在 `AssembleContainerConfig` 中，而非 `cmd/run.go` 的参数解析阶段。

**理由**：
- 挂载信息在 `AssembleContainerConfig` 中已可用（`params.Mounts` 直接访问）
- 保持参数解析层简单，不以复杂的默认值逻辑污染
- 工作目录默认值与容器配置紧密耦合，放在同一处便于理解和维护

## Risks / Trade-offs

- **ContainerResize 可能在极老旧 Docker 版本上不支持** → 降级处理：API 调用失败时记录日志继续执行，不阻塞主流程
- **用户自行设置了 TERM 环境变量时不应覆盖** → 环境变量合并逻辑中，用户指定的 Envs 后合并，优先级更高（`params.Envs` 在 `containerConfig.Env` 之后被添加到构建函数中）
- **`/workspace` 目录可能在镜像中不存在** → Docker 在容器创建时若 WorkingDir 不存在会自动创建，无需额外处理
