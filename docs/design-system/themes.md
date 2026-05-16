# 主题

## 当前状态

AgentForge 是一个纯后端 CLI 项目，当前**没有视觉 UI 层**。没有 CSS、设计 Token、或主题切换机制。

## 可用主题

> 当前未定义。引入前端界面时建议支持 light/dark 双主题：

| 主题 | CSS 类 | 何时激活 |
|------|-----------|--------------|
| Light | `:root`（默认） | 系统偏好或手动选择 |
| Dark | `.dark` | 系统偏好或手动选择 |

## 各主题的 CSS 变量

> 当前未定义。引入前端界面时建议的变量映射（基于 Tailwind CSS + shadcn/ui 的常见约定）：

| 变量 | Light | Dark | 语义 Token |
|----------|-------|------|----------------|
| `--background` | 0 0% 100% | 224 71% 4% | background |
| `--foreground` | 224 71% 4% | 213 31% 91% | foreground |
| `--primary` | TBD | TBD | primary |
| `--primary-foreground` | TBD | TBD | primary-foreground |
| `--secondary` | TBD | TBD | secondary |
| `--secondary-foreground` | TBD | TBD | secondary-foreground |
| `--destructive` | TBD | TBD | destructive |
| `--destructive-foreground` | TBD | TBD | destructive-foreground |
| `--muted` | TBD | TBD | muted |
| `--muted-foreground` | TBD | TBD | muted-foreground |
| `--accent` | TBD | TBD | accent |
| `--accent-foreground` | TBD | TBD | accent-foreground |
| `--border` | TBD | TBD | border |
| `--ring` | TBD | TBD | ring |

## 如何添加新主题

1. 创建 CSS 选择器（如：`.theme-brand` 或 `[data-theme="brand"]`）
2. 在对应选择器内重新定义上述 CSS 变量
3. 将选择器应用于根元素 `<html>` 或 `<body>`

## 项目如何激活/切换主题

> 当前未定义。引入前端后建议使用 `next-themes`（Next.js）或手动 `class` 切换策略：
> - 默认跟随系统偏好（`prefers-color-scheme`）
> - 提供手动切换按钮覆盖系统偏好
