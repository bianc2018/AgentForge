# 间距

## 当前状态

AgentForge 是一个纯后端 CLI 项目，当前**没有视觉 UI 层**。没有 CSS、设计 Token 或组件库需要定义间距层级。

## 基础层级

> 当前未定义。引入前端界面时建议使用 Tailwind CSS 默认间距层级（基于 4px 单位）：

| Token | 值 | 使用参考 | 典型用途 |
|-------|-------|------------------|------------|
| xs / 1 | 4px | `p-1` / `gap-1` | 最小内部间距 |
| sm / 2 | 8px | `p-2` / `gap-2` | 小组件内部间距 |
| md / 3 | 12px | `p-3` / `gap-3` | 输入框和 badges 的 padding |
| lg / 4 | 16px | `p-4` / `gap-4` | 卡片和 section 的标准 padding |
| xl / 6 | 24px | `p-6` / `gap-6` | 主容器的 padding |
| 2xl / 8 | 32px | `p-8` / `gap-8` | section 之间的间距 |
| 3xl / 12 | 48px | `py-12` / `gap-12` | 大块间距 |
| 4xl / 16 | 64px | `py-16` / `gap-16` | Hero section 间距 |

## 使用规则

- 当前无需使用间距层级
- 引入前端时：组件内部 padding 使用 xs–lg 层级
- 引入前端时：列表元素之间的 gap 使用 xs–sm 层级
- 引入前端时：容器/页面 padding 使用 xl–2xl 层级
- 引入前端时：绝不使用任意值，调整到最近的 token

## 网格和布局

- 当前未定义
- 引入前端时建议最大容器宽度：1280px (max-w-7xl)
- 列布局使用项目默认 gap 的 grid
