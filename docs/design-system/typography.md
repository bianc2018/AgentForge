# 排版

## 当前状态

AgentForge 是一个纯后端 CLI 项目，当前**没有视觉 UI 层**。没有 CSS、字体配置或组件库需要定义排版层级。

## 字体系列

> 当前未定义。当项目引入前端界面时，建议使用以下配置：

| 系列 | CSS 变量 / 类 | 用途 |
|---------|----------------------|-----|
| Inter / system-ui | `font-sans` / `var(--font-sans)` | 通用文本 |
| JetBrains Mono / monospace | `font-mono` / `var(--font-mono)` | 代码、技术数据 |

## 排版层级

> 当前未定义。引入前端界面时建议的层级：

| 层级 | Tailwind 类 | 大小 | 粗细 | 行高 | 用途 |
|-------|----------------|---------|------|-------------|-----|
| Display | `text-4xl font-bold` | 36px | 700 | 1.2 | Hero 页面标题 |
| H1 | `text-3xl font-bold` | 30px | 700 | 1.25 | 页面主标题 |
| H2 | `text-2xl font-semibold` | 24px | 600 | 1.3 | 主要 section |
| H3 | `text-xl font-semibold` | 20px | 600 | 1.4 | 子 section |
| Body | `text-base` | 16px | 400 | 1.5 | 正文 |
| Small | `text-sm` | 14px | 400 | 1.5 | 标签、元数据 |
| Caption | `text-xs` | 12px | 400 | 1.4 | 图例、tooltip |

## 使用规则

- 当前无需使用排版层级
- 引入前端时：绝不使用层级之外的任意大小
- 引入前端时：每个层级对应明确的 Tailwind 类，禁止混用
