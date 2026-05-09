---
name: jsdoc-standards
description: >
    对任何 TypeScript 或 JavaScript 文档任务使用此 skill：
    为函数、类、接口、类型、方法、模块或文件编写 JSDoc。
    每当用户要求记录文档、注释、添加 JSDoc、 
    通过注释解释代码或审查 TS/JS 中现有文档时调用。
    在创建需要从一开始就有文档的新 TS/JS 文件时也使用。
---

# TypeScript/JavaScript JSDoc 标准

## 基本原则

1. **记录意图，而非实现** — 解释_为什么_和_做什么_，而非_怎么做_
2. **避免冗余** — 不要重复函数名已经清楚说明的内容
3. **具体明确** — 像"处理数据"这样的模糊术语对任何人都没有帮助
4. **TypeScript 优先** — 利用现有类型；不要重复代码中已声明的类型

---

## 什么需要记录（必需）

| 元素                                                      | 记录？ |
| ------------------------------------------------------------- | ----------- |
| 导出/公共的函数和方法                         | ✅ 总是   |
| 导出的类                                            | ✅ 总是   |
| 复杂的公共接口和类型 | ✅ 总是   |
| 非显而易见的导出常量                              | ✅ 总是   |
| 复杂的内部函数（>10 行或非平凡逻辑） | ✅ 是      |
| 具有非显而易见行为的参数                        | ✅ 总是   |
| 琐碎的 getters/setters                                      | ❌ 否      |
| 名称自解释且逻辑简单的函数            | ❌ 否      |
| 重新导出的原始类型                                | ❌ 否      |

---

## 基本标签及何时使用

### `@param`

对每个非显而易见的参数使用。在 TypeScript 中，如果类型已在签名中，则省略类型。

```ts
// ✅ 正确 — 类型已省略，因为已在 TS 签名中
/**
 * @param userId - 数据库中用户的唯一 ID
 * @param options - 可选的搜索配置
 */
async function findUser(userId: string, options?: FindOptions): Promise<User>;

// ❌ 避免 — 类型与 TypeScript 签名冗余
/**
 * @param {string} userId - 用户的唯一 ID
 */
```

### `@returns`

描述返回什么，特别是在非显而易见的情况下。如果返回 `void` 或显而易见则省略。

```ts
/**
 * @returns 签名的 JWT token，有效期为 24 小时
 */
function generateToken(payload: JwtPayload): string;
```

### `@throws`

记录调用者需要处理的所有异常。

```ts
/**
 * @throws {NotFoundError} 当用户不存在时
 * @throws {ValidationError} 当邮箱格式无效时
 */
```

### `@example`

对于复杂的公共函数或具有非直观行为的函数是必需的。使用有效的 TypeScript 代码块。

````ts
/**
 * @example
 * ```ts
 * const result = await paginate(query, { page: 1, limit: 20 });
 * console.log(result.data); // [{ id: 1, ... }, ...]
 * ```
 */
````

### `@template`（泛型）

当名称不具自解释性时记录类型参数。

```ts
/**
 * @template T - 列表项的类型
 * @template K - 用于分组的对象键
 */
function groupBy<T, K extends keyof T>(items: T[], key: K): Map<T[K], T[]>;
```

### `@deprecated`

始终包含替代方案，并在可能时包含将被删除的版本。

```ts
/**
 * @deprecated 使用 `findUserById` 代替此函数。将在 v3.0 中删除。
 */
```

### `@see`

引用相关函数、外部文档或相关 issues。

```ts
/**
 * @see {@link https://docs.example.com/auth} 认证文档
 * @see {@link validateToken} 用于验证生成的 token
 */
```

---

## 各元素类型的模板

对于每种类型的详细模板，请参阅：`references/templates.md`

模板中涵盖的类型：

- 简单函数
- 异步函数
- 完整类（构造函数、方法、属性）
- 接口和类型
- 模块/文件（顶部注释）
- React Hook
- Express/Fastify 中间件
- 枚举

---

## TypeScript 特定规则

1. **不要重复**签名中已有的类型 — TypeScript 已经记录了它们
2. **记录类型收窄**当返回类型以非显而易见的方式依赖于输入时
3. **解释 `as unknown as T`** 和其他强制类型转换 — 始终说明理由
4. **短名称泛型**（`T`、`K`、`V`）需要 `@template`；描述性名称（`TUser`、`TResponse`）通常不需要

---

## 应避免的

```ts
// ❌ 冗余 — 重复函数名
/**
 * 获取用户。
 * @param id - 该 id
 * @returns 该用户
 */
function getUser(id: string): User;

// ✅ 正确 — 增加了真实上下文
/**
 * 从主数据库中通过 ID 查找用户。
 * 如果用户不存在则返回 null（不抛出异常）。
 *
 * @param id - 用户的 UUID v4
 * @returns 找到的用户，如果不存在则为 null
 */
function getUser(id: string): User | null;

// ❌ 实现注释（非 JSDoc）
/**
 * 循环遍历数组并过滤项目
 */

// ✅ 记录契约
/**
 * 按状态过滤交易，排除部分取消。
 * 保持原始顺序。
 */
```

---

## 格式和风格

- **第一行**：一句话摘要，无句号
- **空行**：描述和标签之间的分隔
- **时态**：使用不定式 — 不用"此函数获取..."
- **语言**：业务描述使用中文；成熟的技术术语使用英文（`token`、`payload`、`middleware`）
- **长度**：主描述最多 2 行；超过则拆分为更小的函数
