# 注释示例 — constitution.md

此文件展示了一个质量足够的 constitution.md 示例，附有解释每个部分为什么有效的注释。

---

## constitution.md

### Purpose

<!-- ✅ 两句直接的话。声明了范围（"this system"）、规则类型（"non-negotiable technical rules"）和使用说明（"Any ambiguity must be resolved explicitly"）。 -->

This document defines the non-negotiable technical rules that govern all implementations in this system.
Any ambiguity must be resolved explicitly — never assumed.

---

### Must Do

<!-- ✅ 5 条规则覆盖：分层分离、错误传播、输入验证、日志记录和可追溯性。每条以 "All [主体] must [动词]" 开头，可机械验证。 -->

1. All business logic must reside in the service layer and remain independent from transport and persistence layers.

   <!-- ✅ 指定了层（service layer）。隐含禁止与 transport 和 persistence 混合。可验证：审查 PR 时可以检查 service 之外是否有业务逻辑。 -->

2. All errors must be propagated using a standardized structure and include contextual metadata (requestId, timestamp, error code).

   <!-- ✅ 定义了预期格式（标准化结构）和必需字段（requestId, timestamp, error code）。可验证：错误对象可被检查。 -->

3. All external inputs must be validated at system boundaries before reaching domain logic.

   <!-- ✅ 定义了验证发生的位置（system boundaries）和不变条件（在领域之前）。可验证：跟踪请求流程，验证必须在 service 之前。 -->

4. All state-changing operations must be logged using structured logging.

   <!-- ✅ 定义了范围（state-changing operations）和格式（structured logging）。可验证：检查写入操作时，每个必须有对应的日志。 -->

5. All changes must be traceable back to a functional requirement or BDD scenario.

   <!-- ✅ 将可追溯性定义为强制性的，并指定了有效目标（功能需求或 BDD）。可验证：commits 或 PRs 必须引用需求或场景。 -->

---

### Ask Before Proceeding

<!-- ✅ 4 条规则涵盖四个最常见的批准门槛：需求不明确、架构决策、契约变更和与 constitution 本身的冲突。每条都有条件（If X）和操作（must Y）。 -->

6. If a requirement is ambiguous or incomplete, the implementation must stop and request clarification before proceeding.

   <!-- ✅ 条件清晰（不明确或不完整）。操作明确（停止并要求澄清）。不给假设留空间。 -->

7. If multiple architectural approaches are possible (e.g., sync vs async, cache vs direct query), a justification must be provided and validated.

   <!-- ✅ 条件有具体示例（sync/async、cache/query）。操作要求论证 AND 验证 — 不仅仅是记录选择。 -->

8. If a change impacts data model, contracts, or public APIs, confirmation must be obtained before implementation.

   <!-- ✅ 定义了风险范围（数据模型、契约、公共 API）。操作要求在实施前确认 — 非事后。 -->

9. If a requirement appears to conflict with an existing rule in this constitution, the conflict must be explicitly raised.

   <!-- ✅ 关闭了自洽性循环：constitution 也可以被质疑。防止开发人员压制真实的冲突。 -->

---

### Never Do

<!-- ✅ 5 条绝对禁止涵盖：直接访问数据库、领域外的业务逻辑、需求假设、静默错误和框架耦合。每条以 "Never" 开头并描述了具体的反模式。 -->

10. Never access the database directly from controllers or UI layers.

    <!-- ✅ 明确指出了被禁止的层（controllers、UI）。可验证：controller 中的 ORM/查询导入或调用是直接违规。 -->

11. Never embed business logic inside controllers, repositories, or external adapters.

    <!-- ✅ 明确列出了被禁止的层（controllers、repositories、adapters）。从负面补充了规则 1。 -->

12. Never assume missing requirements or infer business rules without explicit specification.

    <!-- ✅ 禁止假设和推断。从负面补充了规则 6。 -->

13. Never swallow errors silently or return generic error responses without context.

    <!-- ✅ 一条规则中两个反模式：静默错误和无上下文地返回通用响应。补充了规则 2。 -->

14. Never couple domain logic to specific frameworks, libraries, or infrastructure details.

    <!-- ✅ 将耦合推广到任何框架或库 — 不仅仅是数据库。保持领域可移植。 -->

---

### Enforcement

<!-- ✅ 3 条执行规则涵盖：实施前门槛、违规后果和缺乏清晰度时的阻止。后果是具体的（无效、阻止、修正）。 -->

15. Any implementation plan must explicitly describe how it complies with this constitution.

    <!-- ✅ 实施前门槛：计划必须在开始前论证合规性。适用于代码审查或设计文档。 -->

16. Any violation of these rules invalidates the implementation and must be corrected before merge.

    <!-- ✅ 清晰具体的后果（无效 + 阻止 merge）。无文档化的例外或 workarounds。 -->

17. Any missing clarification must block implementation until resolved.

    <!-- ✅ 以后果强化了规则 6-9：缺乏清晰度 = 完全阻止。防止"我先做了看看会发生什么"。 -->

---

## 此示例为何质量高

1. **全局编号**：规则从 1 到 17 连续编号 — 便于引用（"违反规则 13"）
2. **互补性**：Must Do 和 Never Do 从正反两面覆盖相同主题（如：规则 1 + 规则 11 = 分层分离）
3. **无重复**：每条规则只在一个 section 中存在
4. **无模糊性**："structured logging"、"requestId, timestamp, error code" — 没有"最佳实践"或"适当地"
5. **具体执行**："invalidates the implementation"、"must be corrected before merge" — 不是"建议审查"
