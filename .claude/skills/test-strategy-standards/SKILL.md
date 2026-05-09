---
name: test-strategy-standards
description: >
    每个 feature 的 test-strategy.md 质量标准。定义
    5 种必需的测试类型（单元、集成、E2E Gherkin、
    性能、安全）、每种类型从 SDD 产物中的推导来源、
    质量标准、期望格式和最低覆盖率。
    配合 interview-guide 和 test-strategy-example 作为质量标尺使用。
---

# 测试策略标准

## 格式规范

> 如果项目中存在 `docs/sdd/test-strategy-format.md`，读取它并将其用作格式规范 — 它替代以下默认格式，可能已为此项目定制。

## 默认格式

### 什么是 test-strategy.md

`test-strategy.md` 规范了 feature **应该存在哪些测试**，
按类型组织，并可追溯到产生它们的产物。

它不是执行计划也不是覆盖率报告 — 是在实现之前对测试的规范，
类似于代码的 `requirements.md`。

---

## 在 SDD 流程中的位置

```
constitution.md → prd.md → stories.md → scenarios.feature
→ requirements.md → nf-requirements.md → design.md
→ test-strategy.md → tasks.md
```

`test-strategy.md` 在 **设计之后**和 **tasks 之前**生成。
`tasks-agent` 读取 `test-strategy.md` 来生成测试 task。

---

## 5 种测试类型及其来源

| 类型        | 主要来源                                               | 测试内容                                                     |
| ----------- | ------------------------------------------------------------ | --------------------------------------------------------------- |
| 单元    | `design.md` → 领域方法                             | 隔离的业务规则，无外部依赖           |
| 集成  | `design.md` → repositories 和 adapters                        | 组件 + 真实依赖（数据库、Redis、外部服务） |
| E2E Gherkin | `scenarios.feature`                                          | 通过可执行 step definitions 的完整 API 流程          |
| 性能 | `nf-requirements.md` → 可测量的 NFR                      | 延迟、吞吐量、操作时间指标             |
| 安全   | `nf-requirements.md` → 安全 NFR + `prd.md` → 风险 | 速率限制、时序攻击、枚举、token 重用      |

---

## 必需结构

```markdown
# 测试策略 — <Feature 名称>

## 1. 单元测试

...

---

## 2. 集成测试

...

---

## 3. E2E Gherkin 测试

...

---

## 4. 性能测试

...

---

## 5. 安全测试

...

---

## 覆盖率摘要

...
```

---

## 各类型格式

### 1. 单元测试

**目的：** 在完全隔离中验证业务规则 — 无数据库、HTTP 或外部服务。

**格式：**

```markdown
### UT-<N>: <ComponentName>.<method>()

- **测试内容:** <预期行为>
- **覆盖案例:**
    - 正常路径: <描述>
    - <错误条件 1>: <预期行为>
    - <边缘条件>: <预期行为>
- **所需 Mock:** <列表或"无 — 纯领域">
- **可追溯性:** <REQ-N> · <NFR-N>
```

**质量检查清单：**

- [ ] `design.md` 中标识的每个领域方法一个 UT
- [ ] 每个 UT 至少覆盖：正常路径 + 方法的每种错误条件
- [ ] 没有 UT 依赖数据库、HTTP 或外部服务
- [ ] 需要时声明了 Mock
- [ ] 每个 UT 追溯到至少一个 REQ

---

### 2. 集成测试

**目的：** 验证基础设施组件与其真实依赖一起正确工作。

**格式：**

```markdown
### IT-<N>: <ComponentName> — <方法或行为>

- **测试内容:** <使用真实依赖的行为>
- **使用的真实依赖:** <测试数据库 | 测试 Redis | mock 的外部服务>
- **覆盖案例:**
    - <案例 1>: <预期行为>
    - <案例 2>: <预期行为>
- **所需 Setup:** <测试的数据库/Redis 初始状态>
- **可追溯性:** <REQ-N> · <NFR-N>
```

**质量检查清单：**

- [ ] 每个 repository 一个 IT（覆盖其主要方法）
- [ ] 每个外部服务 adapter 一个 IT
- [ ] 每个 IT 指定哪些依赖是真实的，哪些是 mock 的
- [ ] 描述了外部状态的 setup/teardown
- [ ] 依赖失败行为已覆盖（如：Redis 不可用）

---

### 3. E2E Gherkin 测试

**目的：** 通过 step definitions 将 BDD 场景作为自动化端到端测试执行。

## Task 推导的规范性规则

以下规则必须被 `tasks-agent` 在生成 `tasks.md` 时遵循。

### 规则 1 — 强制性可追溯性

`scenarios.feature` 中定义的每个 Scenario 必须产生至少一个
本文档中规范的自动化测试。

不能存在没有对应 GH 的 Scenario。

### 规则 2 — 强制性 Task 推导

本策略中定义的每个 GH-\* 项目必须在 `tasks.md` 中产生至少一个
实现 task。

该 task 必须：

- 实现必要的 step definitions
- 将 Scenario 作为自动化测试执行
- 验证 `Then` 步骤中定义的结果

### 规则 3 — 工具无关性

`test-strategy.md` 不定义特定工具。

执行工具（测试框架、runner、
BDD 库等）的选择应由项目或 `design.md` 中
定义的架构决定。

本文档仅规范：

- 被测试的行为
- 可追溯性
- 测试执行要求

### 规则 4 — 最低覆盖率

强制性最低覆盖率：

- 100% 的 Scenarios 必须有对应的 GH
- 每个 GH 必须是可自动化的
- `tasks-agent` 必须生成足够的 task 来实现本文档中定义的所有 GH

**格式：**

```markdown
### GH-<N>: Scenario "<Scenario 的确切名称>"

- **文件:** `docs/features/<slug>/scenarios.feature`
- **必要的 Step Definitions:**
    - `Given <step 的确切文本>` → <该 step 应该做什么>
    - `When <step 的确切文本>` → <该 step 应该做什么>
    - `Then <step 的确切文本>` → <该 step 应该验证什么>
- **可从其他 Scenarios 复用的 Steps:** <列表或"无">
- **必要的初始状态:** <数据库中的数据、活跃 mock>
- **可追溯性:** <REQ-N> · <NFR-N>
```

**质量检查清单：**

- [ ] `.feature` 文件中的每个 Scenario 一个 GH — 无例外
- [ ] 每个 GH 明确可自动化
- [ ] 每个 GH 必须在 tasks.md 中产生至少一个实现 task
- [ ] 每个 step definition 描述了其预期实现
- [ ] Scenarios 之间可复用的 steps 已标识（避免重复）
- [ ] 每个场景的必要初始状态已描述
- [ ] 需要操作外部状态的场景（如：强制 token 过期）描述了机制

---

### 4. 性能测试

**目的：** 验证可测量的性能 NFR 是否满足。

**格式：**

```markdown
### PT-<N>: <描述性标题>

- **测量内容:** <指标：p95 延迟、操作时间、吞吐量>
- **阈值:** <NFR 的值，如：≤ 30s, ≥ 100ms, ≤ 200ms p95>
- **测量方法:** <本地 benchmark | 负载测试 | CI 中测量>
- **执行次数:** <如：10 次执行，100 rps 持续 60s>
- **可追溯性:** <NFR-N>
```

**质量检查清单：**

- [ ] 每个有可测量标准的 NFR 一个 PT
- [ ] 阈值直接从 NFR 推导（非编造）
- [ ] 测量方法可在 CI 中执行
- [ ] 执行次数足够产生统计显著的结果

---

### 5. 安全测试

**目的：** 验证安全机制按 NFR 和 PRD 风险中的规范工作。

**格式：**

```markdown
### ST-<N>: <攻击或漏洞的描述性标题>

- **验证内容:** <预期安全行为>
- **模拟的攻击向量:** <如：账户枚举、暴力破解、token 重放>
- **覆盖案例:**
    - <案例 1>: <系统预期行为>
    - <案例 2>: <系统预期行为>
- **可追溯性:** <NFR-N> · <PRD 风险>
```

**质量检查清单：**

- [ ] 每个安全 NFR 一个 ST
- [ ] PRD 中每个有可验证技术缓解措施的风险一个 ST
- [ ] 每个 ST 描述模拟的攻击向量（不仅仅是"测试安全"）
- [ ] 预期行为是具体的（HTTP 状态码、消息、无敏感数据）

---

### 覆盖率摘要

**目的：** 将每个 REQ 和 NFR 与覆盖它们的测试交叉制表。

**格式：**

```markdown
## 覆盖率摘要

| 需求 | 单元 | 集成 | E2E Gherkin | 性能 | 安全 |
| --------- | -------- | ---------- | ----------- | ----------- | --------- |
| REQ-1     | UT-1     | IT-1, IT-2 | GH-1        | PT-1        | —         |
| REQ-2     | UT-2     | IT-3       | GH-4        | —           | —         |
| NFR-1     | —        | —          | —           | PT-1        | —         |
| NFR-4     | —        | —          | GH-7        | —           | ST-1      |
```

**质量检查清单：**

- [ ] 每个 REQ 至少有 1 个 E2E Gherkin 测试 AND 至少 1 个单元或集成测试
- [ ] 每个可测量的 NFR 至少有 1 个性能测试
- [ ] 每个安全 NFR 至少有 1 个安全测试
- [ ] 表中没有完全空白的行

---

## 最低必需覆盖率

- [ ] `design.md` 中的每个领域方法至少有 1 个 UT
- [ ] `design.md` 中的每个 repository 至少有 1 个 IT
- [ ] `design.md` 中的每个外部服务 adapter 至少有 1 个 IT
- [ ] `.feature` 中的每个 Scenario 有 1 个对应的 GH
- [ ] 每个有可测量值的 NFR 有 1 个 PT
- [ ] 每个安全 NFR 和 PRD 中每个可缓解的风险有 1 个 ST

---

## 一般格式规则

- **各类型的 ID：** UT-N, IT-N, GH-N, PT-N, ST-N — 每种类型内顺序编号
- **可追溯性始终存在：** 每个测试指向产生它的 REQ、NFR 或 Scenario
- **无代码：** `test-strategy.md` 描述测试什么，而非如何实现
- **分隔符：** 类型 section 之间使用 `---`
- **语言：** 标题和描述使用中文；组件和方法名使用英文

---

## 参考

- 产物格式（可定制）：`docs/sdd/test-strategy-format.md`
- 规范示例（可定制）：`docs/sdd/test-strategy-example.md`
- 访谈指南：`references/interview-guide.md`
