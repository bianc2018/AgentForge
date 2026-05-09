---
name: reqs-standards
description: >
    本项目 EARS 格式功能需求的质量标准。
    定义每条需求的必需结构、四种 EARS 模式、
    质量标准、如何从 PRD、User Stories 和 BDD 场景推导需求、
    期望格式和一般规则。配合 interview-guide 进行访谈，
    以 reqs-example 作为质量标尺。
---

# 功能需求标准（EARS）

## 格式规范

> 如果项目中存在 `docs/sdd/reqs-format.md`，读取它并将其用作格式规范 — 它替代以下默认格式，可能已为此项目定制。

## 默认格式

### 四种 EARS 模式

| 模式 | 何时使用 | 语法 |
|--------|-------------|---------|
| **普适** | 始终必需的行为 | `The system shall <behavior>.` |
| **事件** | 外部事件触发行为 | `When <event>, the system shall <behavior>.` |
| **状态** | 系统处于特定状态 | `While <state>, the system shall <behavior>.` |
| **不期望** | 失败或错误条件 | `If <condition>, the system shall <behavior>.` |

> 规则：错误和失败使用 `If`，绝不使用 `When`。

---

## 文件结构

```markdown
# 功能需求 — <Feature 名称>

## <组 1，如：主流程>

**REQ-1**: When <event>, the system shall <behavior>.

> 来源：PRD 主流程 / BDD 场景 "场景标题"

## <组 2，如：输入验证>

**REQ-3**: If <condition>, the system shall <behavior>.

> 来源：PRD 替代流程 Y / 故事 Z，标准 N
```

---

## 每条需求检查清单

**清晰性：**
- [ ] 描述单个行为
- [ ] 使用 `shall` — 绝不使用 "should"、"may"、"can" 或 "must"
- [ ] 主语始终是 "the system"

**可测试性：**
- [ ] 可以编写证明满足或违反的测试
- [ ] 无模糊术语："正确地"、"适当的"、"必要时"、"快速"
- [ ] 行为可从外部观察

**技术无关性：**
- [ ] 无语言、框架、库或数据库
- [ ] 无 JWT、bcrypt、MySQL、Redis、Next.js、React
- [ ] 业务领域词汇

**可追溯性：**
- [ ] `> 来源：` 字段标识来源（PRD、Story 或 BDD 场景）
- [ ] 每个 BDD 场景至少有一条对应的需求

---

## 需求集的质量标准

| 标准 | 验证 |
|---|---|
| 覆盖率 | 每个 BDD 场景至少有 1 条需求 |
| 完整性 | 所有 Stories 的验收标准均已覆盖 |
| 一致性 | 与 PRD 词汇统一 |
| 非冗余 | 两条需求不描述相同行为 |
| 范围 | 没有需求覆盖 PRD 中"不在范围内"的项目 |

---

## 如何从产物中推导

| 来源 | 产生的 EARS 模式 |
|---|---|
| PRD → 带有"允许"的目标 | `When <操作>, the system shall <行为>` |
| PRD → 带有"拒绝"/"验证"的目标 | `If <条件>, the system shall <行为>` |
| PRD → 主流程（系统步骤） | `When <用户事件>, the system shall <响应>` |
| PRD → 替代流程 | `If <失败条件>, the system shall <行为>` |
| Stories → 正常路径验收标准 | `When <操作>, the system shall <行为>` |
| Stories → 验证标准 | `If <条件>, the system shall <行为>` |
| BDD → 每个 Scenario 的 When + Then | 转换为 EARS 义务；Then 中的 `And` → 考虑第 2 条需求 |

---

## 格式规则

- **文件：** `docs/features/<slug>/requirements.md`
- **需求语言：** 英文（EARS 标准）
- **评论和注释：** 中文（`> 来源：`、`##` 标题）
- **编号：** 全局顺序 — REQ-1, REQ-2...（不按组重新开始）
- **分组：** 按类别使用 `##` 标题（主流程、验证、安全...）

---

## 参考

- 产物格式（可定制）：`docs/sdd/reqs-format.md`
- 规范示例（可定制）：`docs/sdd/reqs-example.md`
- 问题库：`references/interview-guide.md`
