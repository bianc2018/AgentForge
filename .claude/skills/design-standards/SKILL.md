---
name: design-standards
description: >
    本项目技术设计文档的质量标准。
    定义 6 个必需 section、各 section 的质量标准、
    如何从所有先前 SDD 产物（PRD、Stories、BDD Scenarios、
    Requirements、NF-Requirements）推导决策、期望格式
    和一般规则。配合 interview-guide 进行访谈，
    以 design-example 作为质量标尺。
---

# 技术设计标准

## 格式规范

> 如果项目中存在 `docs/sdd/design-format.md`，读取它并将其用作格式规范 — 它替代以下默认格式，可能已为此项目定制。

## 默认格式

### 必需结构（6 个 section，按此顺序）

1. 技术概述
2. 组件架构
3. 数据模型
4. API/契约
5. 执行流程
6. 技术决策

**各产物的输入来源：**

| 产物 | 在设计中的角色 |
|----------|----------------|
| `requirements.md` | 系统**必须**做什么 |
| `nf-requirements.md` | 性能、安全、可观测性约束 |
| `scenarios.feature` | 执行流程和 API 契约的骨架 |
| `stories.md` | 验收标准和输入验证 |
| `prd.md` | 外部依赖、风险、不在范围内 |
| `docs/constitution.md` | 不可协商的架构约束 |

---

## 各 section 检查清单

### 1. 技术概述
_格式：连续文本，2-4 句话。_
- [ ] 描述主要技术方案（而非 feature 对用户做什么）
- [ ] 提及关键技术（如：OTP、bcrypt、Redis、邮件服务）
- [ ] 引用架构层或模式（如：hexagonal）
- [ ] 与 `docs/constitution.md` 的约束一致

### 2. 组件架构
_格式：`flowchart TD` 图，后跟每个组件的 `### ComponentName` 块，包含层、职责和依赖。_
- [ ] 包含 `flowchart TD` 图，显示所有组件之间的依赖关系（infra → application → domain）
- [ ] 每个组件有单一且清晰的职责
- [ ] 分层遵循 `docs/constitution.md` 的架构
- [ ] 依赖指向内部（infra → application → domain）
- [ ] 所有功能需求至少被一个组件处理
- [ ] 外部服务隔离在基础设施 adapter 中
- [ ] 新组件显式标注

### 3. 数据模型
_格式：`erDiagram` 图，后跟每个实体的 `字段 | 类型 | 描述` 表格，包含显式关系。_
- [ ] 包含 `erDiagram` 图，显示所有实体及其关系
- [ ] 实体可从需求或 BDD 场景推导 — 无凭空编造
- [ ] 当 NFR 要求时，过期和一次性使用字段必须存在
- [ ] 如果项目模式要求，审计字段已声明
- [ ] 无 DDL 或 migration（仅逻辑模型）

### 4. API/契约
_格式：`### METHOD /path`，包含认证、请求体、200 响应和错误表。_
- [ ] 每个 endpoint 对应至少一个需求或 BDD 场景
- [ ] 输入载荷定义了字段和类型
- [ ] 所有错误场景都有对应的 HTTP 状态码
- [ ] 速率限制（NFR）的锁具有 429 状态码
- [ ] 每个 endpoint 的认证已声明

### 5. 执行流程
_格式：`### 流程：<Scenario 名称>`，带负责组件的编号步骤，后跟 Mermaid `sequenceDiagram`。_
- [ ] `.feature` 中每个 Scenario 一个流程
- [ ] 从请求到响应的逐步正常路径
- [ ] 每个步骤标明负责的组件
- [ ] 每个流程包含 `sequenceDiagram`，参与者和消息对应于编号步骤
- [ ] 替代流程覆盖所有 BDD 错误场景
- [ ] 没有模糊步骤（"系统处理"但未指定内容）

### 6. 技术决策
_格式：`### DT-N：<标题>`，包含问题、替代方案、决策、理由和相关需求。_
- [ ] 仅包含有真正权衡的决策（非显而易见的）
- [ ] 每个决策至少两个替代方案
- [ ] 理由提及权衡
- [ ] 每个决策追溯到 REQ 或 NFR
- [ ] 没有决策与 `docs/constitution.md` 矛盾

---

## 格式规则

- **语言：** 中文。组件名、字段和 endpoints 保持英文。
- **可追溯性：** 始终引用促使决策的 REQ-N 或 NFR-N。
- **无代码：** 无源代码；仅在执行流程中可接受伪代码。
- **不重复：** 如果信息在 PRD 或需求中已存在，引用 — 不复制。
- **分隔符：** 在 section 之间使用 `---`。

---

## 参考

- 产物格式（可定制）：`docs/sdd/design-format.md`
- 规范示例（可定制）：`docs/sdd/design-example.md`
- 问题库：`references/interview-guide.md`
