# constitution.md

### 1. Purpose

本文档定义了约束 AgentForge 项目所有实现的不可协商技术规则。任何歧义或冲突必须在推进之前显式澄清——绝不假设。

---

### 2. Must Do

1. 所有业务逻辑必须位于 Service 层，CLI 层、Provider 层和 Infrastructure 层不得包含业务逻辑。

2. 所有跨层调用必须遵循单向依赖原则——每层只能依赖其直接下层，禁止 CLI 层直接调用 Provider 层或 Infrastructure 层。

3. 所有外部系统（Docker、LLM API、Git、包管理器、文件系统）调用必须通过对应的 Provider 封装，Service 层禁止直接执行外部命令或调用外部 SDK。

4. 所有错误必须使用标准化结构传播，必须包含 error code、message 和操作上下文（operationId、timestamp），禁止在中间层吞没错误或抛出无上下文异常。

5. 所有外部输入必须在 CLI 层完成格式与类型校验，非法数据必须在进入 Service 层之前被拒绝。

6. 所有状态变更操作必须以结构化日志记录，日志必须包含操作类型、操作标识和结果状态。

7. 所有实现必须可追溯到一个具体功能域（Agent 管理、Endpoint 管理、构建编排、运行编排、诊断、镜像传输）及对应的 functional requirement。

---

### 3. Ask Before Proceeding

8. 如果 functional requirement 存在歧义、缺失关键细节或包含未定义的术语，必须在继续实现之前向架构师或产品负责人寻求明确解释——禁止自行假设。

9. 如果一个实现存在多个有效的技术方案（如同步 vs 异步、新 Provider vs 复用现有 Provider、不同的 Provider 接口设计），必须先论证方案优劣并获得确认，然后才能开始编码。

10. 如果变更涉及公共 API 签名、Provider 接口契约、数据模型或跨层调用链路的修改，必须暂停实现并评估影响范围，获得批准后方可继续。

11. 如果实现过程中发现与 constitution.md 中任何规则存在冲突或矛盾，必须立即停止并上报以解决冲突——不得通过绕过或妥协的方式继续。

---

### 4. Never Do

12. 绝不绕过 Provider 层在 Service 层中直接调用外部系统（Docker、LLM API、Git、包管理器、文件系统）或外部 SDK。

13. 绝不在 CLI 层、Provider 层或 Infrastructure 层中包含业务逻辑。

14. 绝不跨层调用——禁止 CLI 层直接调用 Provider 层或 Infrastructure 层，必须经 Service 层中转。

15. 绝不静默吞没错误或在中间层返回不带 operationId 与 error code 的通用错误响应。

16. 绝不假设未指定的需求或自行推断业务规则。

17. 绝不将 Service 层耦合到具体框架、库或基础设施实现细节。

18. 绝不允许未在 CLI 层完成格式与类型校验的输入进入 Service 层。

---

### 5. Enforcement

19. 任何实现计划必须显式描述其对本 constitution 中各项规则的合规情况，未通过合规审查不得开始编码。

20. 任何违反本 constitution 中规则的行为将使该实现无效，必须在 merge 之前修正所有违规。

21. 任何未解决的歧义或待定决策必须阻止实现推进，直至问题获得明确解答。

