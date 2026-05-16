# 颜色

## 当前状态

AgentForge 是一个纯后端 CLI 项目，当前**没有视觉 UI 层**。没有 CSS、设计 Token 或组件库需要定义颜色。

## 语义调色板

> 当前未定义。当项目引入前端界面（Web UI 或 TUI）时，应在此定义以下语义 Token：

| Token | 值 | 使用参考 | 用途 |
|-------|-------|----------------|-----|
| primary | TBD | TBD | 主要操作、CTA |
| primary-foreground | TBD | TBD | primary 背景上的文本 |
| secondary | TBD | TBD | 次要操作 |
| destructive | TBD | TBD | 破坏性操作、错误 |
| muted | TBD | TBD | 微妙背景、禁用 |
| accent | TBD | TBD | 突出显示、悬停状态 |
| background | TBD | TBD | 页面主背景 |
| foreground | TBD | TBD | 主文本 |
| border | TBD | TBD | 组件边框 |
| ring | TBD | TBD | 可见焦点（可访问性） |

## 使用规则

- 当前无需使用颜色值
- 引入前端时：绝不在代码中直接使用 hex 值，始终通过语义 Token
- 引入前端时：遵循 WCAG AA 标准（正常文本对比度 >= 4.5:1）

## 未来考虑

若项目未来增加 Web 前端或 TUI 界面，建议采用 Tailwind CSS 的默认调色板作为起点，并根据品牌需求定制。
