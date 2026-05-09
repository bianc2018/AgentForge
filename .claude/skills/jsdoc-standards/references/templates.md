# 各元素类型的 JSDoc 模板

## 简单函数

```ts
/**
 * 根据客户档案计算适用折扣
 *
 * @param customer - 客户数据，包括套餐和历史
 * @param amount - 毛额（以分为单位）
 * @returns 折扣金额（以分为单位，如无折扣则为 0）
 */
function calculateDiscount(customer: Customer, amount: number): number;
```

---

## 异步函数

````ts
/**
 * 获取用户的完整档案，包括权限和偏好
 *
 * @param userId - 用户的 UUID v4
 * @param options - 搜索选项
 * @param options.includeDeleted - 如果为 true，也返回已停用的用户
 * @returns 用户的完整档案
 *
 * @throws {NotFoundError} 当用户不存在或已被永久删除时
 * @throws {DatabaseError} 当数据库连接失败时
 *
 * @example
 * ```ts
 * const profile = await fetchUserProfile('abc-123', { includeDeleted: false });
 * console.log(profile.permissions); // ['read', 'write']
 * ```
 */
async function fetchUserProfile(
    userId: string,
    options: FetchProfileOptions = {},
): Promise<UserProfile>;
````

---

## 完整类

````ts
/**
 * 管理通知处理队列，支持重试
 *
 * 通知按 FIFO 顺序处理。如果失败，
 * 通知将以指数退避重新入队。
 *
 * @example
 * ```ts
 * const queue = new NotificationQueue({ maxRetries: 3 });
 * queue.push({ type: 'email', to: 'user@example.com' });
 * await queue.process();
 * ```
 */
class NotificationQueue {
    /**
     * 移入错误队列前的最大尝试次数
     */
    readonly maxRetries: number;

    /**
     * @param config - 队列初始配置
     * @param config.maxRetries - 每条通知的最大尝试次数（默认：5）
     * @param config.backoffMs - 尝试之间的基础等待时间（毫秒）（默认：1000）
     */
    constructor(config: QueueConfig = {}) {}

    /**
     * 将通知添加到队列末尾
     *
     * @param notification - 要入队的通知
     * @returns 通知在队列中的唯一 ID
     */
    push(notification: Notification): string {}

    /**
     * 处理所有待处理的通知
     *
     * @returns 处理摘要，包含成功和失败计数
     * @throws {QueueLockedError} 如果另一个实例已在处理
     */
    async process(): Promise<ProcessingResult> {}
}
````

---

## 接口和类型

```ts
/**
 * 列表查询的分页配置
 *
 * 所有列表查询都接受此对象来控制
 * 返回页面的大小和位置。
 */
interface PaginationOptions {
    /** 页码，从 1 开始 */
    page: number;

    /** 每页项目数（最大：100） */
    limit: number;

    /**
     * 基于游标的分页游标。
     * 当提供时，`page` 被忽略。
     */
    cursor?: string;
}

/**
 * 列表查询返回的通用分页结果
 *
 * @template T - 返回项目的类型
 */
type PaginatedResult<T> = {
    /** 当前页的项目列表 */
    data: T[];

    /** 可用项目总数（所有页面） */
    total: number;

    /** 指示是否有下一页可用 */
    hasNextPage: boolean;

    /** 获取下一页的游标（如果没有则为 undefined） */
    nextCursor?: string;
};
```

---

## 模块/文件（顶部注释）

```ts
/**
 * @module auth/token
 *
 * JWT token 生成和验证的实用工具。
 *
 * 预期流程：
 * 1. `generateToken` — 在登录时创建 token
 * 2. `validateToken` — 在每个认证请求中验证
 * 3. `refreshToken` — 在过期前续期
 *
 * @see {@link https://datatracker.ietf.org/doc/html/rfc7519} JWT 规范
 */
```

---

## React Hook

````ts
/**
 * 管理当前用户的认证状态
 *
 * 自动同步 localStorage 中的更改和
 * 其他浏览器标签页的登录/登出事件。
 *
 * @returns 认证状态和会话控制函数
 *
 * @example
 * ```tsx
 * function Header() {
 *   const { user, logout, isLoading } = useAuth();
 *   if (isLoading) return <Spinner />;
 *   return user ? <UserMenu onLogout={logout} /> : <LoginButton />;
 * }
 * ```
 */
function useAuth(): AuthState;
````

---

## Express/Fastify 中间件

````ts
/**
 * JWT Bearer token 认证中间件
 *
 * 验证 `Authorization` header 中的 token，解码 payload
 * 并注入 `req.user` 包含已认证用户的数据。
 *
 * @throws 如果 token 缺失或无效则响应 401
 * @throws 如果 token 已过期则响应 403
 *
 * @example
 * ```ts
 * router.get('/profile', authenticate, (req, res) => {
 *   res.json(req.user);
 * });
 * ```
 */
const authenticate: RequestHandler;
````

---

## 枚举

```ts
/**
 * 订单在其生命周期中的可能状态
 *
 * 状态之间的转换遵循以下流程：
 * PENDING → CONFIRMED → SHIPPED → DELIVERED
 * 任何状态都可以转到 CANCELLED（DELIVERED 除外）。
 */
enum OrderStatus {
    /** 订单已创建，等待付款确认 */
    PENDING = "pending",

    /** 付款已确认，订单正在分拣 */
    CONFIRMED = "confirmed",

    /** 订单已发货，等待配送 */
    SHIPPED = "shipped",

    /** 客户或承运商确认配送 */
    DELIVERED = "delivered",

    /** 订单已取消 — 查看 `cancellationReason` 了解原因 */
    CANCELLED = "cancelled",
}
```
