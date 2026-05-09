# 注释示例 — 功能需求（EARS）

本文件展示了**配送员登录** feature 的功能需求所应用的
预期格式和质量标准。
在审查草稿时用作质量标尺。

---

# 功能需求 — 配送员登录

## 主流程

**REQ-1**: When a delivery driver submits their credentials, the system shall validate the provided email and password.

> 来源：PRD 主流程（步骤 2）/ BDD 场景 "使用有效凭据登录"

<!-- ✅ 好：事件清晰（凭据提交），行为可验证（验证），
         领域词汇，无技术，用户操作使用正确的 `When` 模式 -->

**REQ-2**: When valid credentials are submitted, the system shall authenticate the delivery driver.

> 来源：PRD 主流程（步骤 3）/ BDD 场景 "使用有效凭据登录" / 故事 1，标准 1

<!-- ✅ 好：验证（REQ-1）和认证（REQ-2）之间有清晰的区分，
         "valid credentials" 连接了 BDD 场景的 `Given` -->

**REQ-3**: After successful authentication, the system shall create an active session for the delivery driver.

> 来源：PRD 成功标准 "会话已启动" / 故事 1，标准 2

<!-- ✅ 好：认证后行为分离为独立需求，
         不提及 JWT、cookie 或 Redis — 设计决策，不是需求 -->

**REQ-4**: After successful authentication, the system shall redirect the delivery driver to the main dashboard.

> 来源：PRD 主流程（步骤 4）/ 故事 1，标准 3

<!-- ✅ 好：可观察结果（重定向），不提及具体路由或框架 -->

---

## 输入验证

**REQ-5**: If the submitted email does not match any registered account, the system shall reject the authentication request.

> 来源：PRD 替代流程 "邮箱未注册" / BDD 场景 "使用未注册邮箱登录"

<!-- ✅ 好：失败条件使用正确的 `If` 模式，
         "reject" 是无歧义的可验证行为 -->

**REQ-6**: If an invalid password is submitted, the system shall reject the authentication request.

> 来源：PRD 替代流程 "密码错误" / BDD 场景 "使用错误密码登录"

<!-- ✅ 好：与 REQ-5 分离 — 无效邮箱和无效密码是不同的条件 -->

**REQ-7**: If authentication fails, the system shall display an error message to the delivery driver without revealing which credential is incorrect.

> 来源：PRD 目标 "拒绝无效凭据但不揭示哪个字段错误" / 故事 2，标准 2

<!-- ✅ 好："without revealing which credential is incorrect" 以
         非技术且可验证的方式捕捉了安全需求 -->

---

## 安全

**REQ-8**: The system shall log all authentication attempts, whether successful or failed.

> 来源：PRD 审计要求 / 故事 3，标准 1

<!-- ✅ 好：正确的普适模式（始终发生），日志行为可通过审计界面观察 -->

**REQ-9**: If a delivery driver submits five consecutive failed authentication attempts, the system shall temporarily block further attempts from that account.

> 来源：PRD 风险 "暴力破解攻击" / BDD 场景 "多次失败尝试后锁定"

<!-- ✅ 好：精确的条件（"连续五次失败尝试"），清晰的行为（"暂时阻止"），
         "that account" 在不提及实现的情况下限制了范围 -->

---

## 有问题的需求示例（不要使用）

**❌ 技术暴露：**
```
The system shall store the JWT token in the Redis cache with a TTL of 3600 seconds.
```
*问题：提到了 JWT、Redis 和 TTL — 实现决策，不是需求。*

**❌ 用户行为，而非系统行为：**
```
The delivery driver shall enter their email and password in the login form.
```
*问题：描述了用户操作。需求的主语应始终是 "The system"。*

**❌ 对失败条件使用 `When`：**
```
When invalid credentials are submitted, the system shall display an error.
```
*问题：失败条件必须使用 `If`，而非 `When`。应使用：`If invalid credentials are submitted, the system shall...`*

**❌ 模糊行为：**
```
The system shall handle authentication errors appropriately.
```
*问题："appropriately" 是主观的且不可验证。系统具体必须做什么？*

**❌ 一个需求中有两个行为：**
```
When valid credentials are submitted, the system shall authenticate the driver and redirect them to the dashboard.
```
*问题：认证和重定向是不同的行为 — 应分离为 REQ-2 和 REQ-4。*
