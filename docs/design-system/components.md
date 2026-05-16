# 组件

## 当前状态

AgentForge 是一个纯后端 CLI 项目，当前**没有视觉 UI 层**。没有安装任何 UI 组件库（如 shadcn/ui、MUI、Radix 等），也没有组件目录。

## 基础组件

> 当前未安装任何 UI 组件库。基础组件列表为空。

当项目引入前端界面时，建议安装 shadcn/ui 作为基础组件库，包含但不限于：

### Button
**变体（建议）：**
| 变体 | 参考 | 何时使用 |
|----------|---------------------|-------------|
| default | `variant="default"` | 屏幕的主要操作 — 每个 section 最多 1 个 |
| secondary | `variant="secondary"` | 次要操作，主要操作的替代 |
| destructive | `variant="destructive"` | 不可逆操作（删除、取消） |
| outline | `variant="outline"` | 三级操作，较低视觉权重 |
| ghost | `variant="ghost"` | 工具栏、菜单、密集区域中的操作 |
| link | `variant="link"` | 文本中的内联导航 |

**大小：** `size="sm"` | `size="default"` | `size="lg"`

**状态：**
- Loading：使用 `disabled` + 内部 spinner — 绝不在无视觉反馈的情况下禁用
- Disabled：`disabled` prop — 不明显时通过 tooltip 解释原因

**规则：**
- 每个视觉 section 绝不超过一个 `default` 按钮
- 破坏性按钮始终要求确认（modal 或 popover）

### Input
**必须实现的必需状态：**
- Default, Focus（可见 ring）, Error（destructive 边框 + 下方消息）, Disabled, Loading（readonly + skeleton）

**规则：**
- 每个 input 都有通过 `htmlFor` 关联的 `<label>` — 绝不用 placeholder 替代 label
- 错误消息在 input 下方，绝不在上方或作为 tooltip
- 使用 `aria-describedby` 指向错误消息

## 可复用组合组件

> 当前未定义。

当项目引入前端界面时，建议优先定义以下跨 feature 复用组件：

### PageHeader
使用于：所有主页面
```tsx
<PageHeader
  title="string"
  description="string (可选)"
  action={<Button>...</Button>} // 可选
/>
```
**规则：** 始终在主内容顶部、导航下方。

### EmptyState
使用于：列表或搜索结果为空时
```tsx
<EmptyState
  icon={<Icon />}
  title="string"
  description="string"
  action={<Button>...</Button>} // 可选
/>
```

### LoadingSpinner / Skeleton
- `LoadingSpinner`：整页加载或按钮操作
- `Skeleton`：即将出现的内容加载（保留布局）
