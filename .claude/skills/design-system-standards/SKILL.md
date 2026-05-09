---
name: design-system-standards
description: >
    docs/design-system/ 中 design system 文件的质量标准。
    定义每个文件的必需结构（colors.md, typography.md,
    spacing.md, components.md, themes.md）、各文件的质量标准、
    如何从现有配置中提取决策（适用于任何技术栈：Tailwind、
    CSS Modules、Styled Components 等）以及一般规则。
    配合 interview-guide 和 design-system-example 使用。
---

# Design System 标准

## 什么是 design system

Design system 定义项目的**全局视觉决策** — 确保无论正在实现哪个 feature 或由谁实现，都保持一致的视觉标识。

它是 `constitution.md` 的视觉等价物：正如 constitution 确保每个领域组件遵循相同的架构规则，design system 确保每个主要按钮在所有屏幕上都是相同的。

---

## 在项目中的位置

```
docs/
  constitution.md          ← 全局架构规则
  design-system/
    colors.md              ← 调色板和颜色的语义使用
    typography.md          ← 排版层级和使用
    spacing.md             ← 间距 token
    components.md          ← 基础和组合组件规范
    themes.md              ← 主题（light/dark）和 CSS 变量
  features/
    <slug>/
      ui-spec.md           ← 使用 design system，指定 feature 特定的 UI
```

---

## 格式规范

> 如果项目中存在 `docs/sdd/design-system-format.md`，读取它并将其用作格式规范 — 它替代以下默认格式，可能已为此项目定制。

## 默认格式

### 必需创建顺序

```
colors.md → typography.md → spacing.md → components.md → themes.md
```

每个文件引用之前的文件。不按顺序创建会导致不一致。

---

## 各文件的结构和标准

### colors.md

**目的：** 使用语义名称和使用规则定义完整的调色板。

**必需结构：**
```markdown
# 颜色

## 语义调色板
具有功能含义的颜色 — 通过 Tailwind 类或 CSS 变量使用。

| Token | 值 | Tailwind 类 | 用途 |
|-------|-------|----------------|-----|
| primary | #... / hsl(...) | `bg-primary` / `text-primary` | 主要操作、CTA |
| primary-foreground | ... | `text-primary-foreground` | primary 背景上的文本 |
| secondary | ... | `bg-secondary` | 次要操作 |
| destructive | ... | `bg-destructive` | 破坏性操作、错误 |
| muted | ... | `bg-muted` | 微妙背景、禁用 |
| accent | ... | `bg-accent` | 突出显示、悬停状态 |
| background | ... | `bg-background` | 页面主背景 |
| foreground | ... | `text-foreground` | 主文本 |
| border | ... | `border-border` | 组件边框 |
| ring | ... | `ring` | 可见焦点（可访问性） |

## 辅助调色板
项目的附加颜色（如有）— 状态、分类等。

## 使用规则
- 绝不在代码中直接使用 hex 值 — 始终通过语义 token
- <项目特定的规则>

## 可访问性
- primary 在 background 上：对比度 <N>:1（WCAG AA 最低要求：正常文本 4.5:1）
- destructive 在 background 上：对比度 <N>:1
```

**质量检查清单：**
- [ ] 语义 token 至少覆盖：主要操作、破坏性操作、背景、主文本和边框
- [ ] 每个 token 有值、使用参考（类、CSS 变量或 prop）和使用描述
- [ ] 声明了主要组合的最低 WCAG AA 对比度
- [ ] 明确规则：不在代码中直接使用颜色值 — 始终通过语义 token
- [ ] 调色板从项目中找到的配置推导 — 非编造

---

### typography.md

**目的：** 定义排版层级、字体系列和使用规则。

**必需结构：**
```markdown
# 排版

## 字体系列
| 系列 | CSS 变量 / 类 | 用途 |
|---------|----------------------|-----|
| <名称> | `font-sans` / `var(--font-sans)` | 通用文本 |
| <名称> | `font-mono` | 代码、技术数据 |

## 排版层级
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
- <何时使用每个层级>
- 绝不使用层级之外的任意大小
```

**质量检查清单：**
- [ ] 至少声明一个字体系列，带有使用参考（类、CSS 变量或 prop）
- [ ] 层级至少 5 个级别，覆盖从标题到 caption
- [ ] 每个级别有使用参考、px 大小、粗细和明确用途
- [ ] 当项目有自定义字体时，从项目配置中推导

---

### spacing.md

**目的：** 定义间距层级及如何应用。

**必需结构：**
```markdown
# 间距

## 基础层级
项目使用 <描述层级 — 如：Tailwind 默认 4px 基础层级、通过 CSS 自定义属性的自定义层级等>：

| Token | 值 | 使用参考 | 典型用途 |
|-------|-------|------------------|------------|
| xs / 1 | 4px | `p-1` / `padding: var(--space-xs)` | 最小内部间距 |
| sm / 2 | 8px | `p-2` / `padding: var(--space-sm)` | 小组件内部间距 |
| md / 3 | 12px | `p-3` / `padding: var(--space-md)` | 输入框和 badges 的 padding |
| lg / 4 | 16px | `p-4` / `padding: var(--space-lg)` | 卡片和 section 的标准 padding |
| xl / 6 | 24px | `p-6` / `padding: var(--space-xl)` | 主容器的 padding |
| 2xl / 8 | 32px | `p-8` / `padding: var(--space-2xl)` | section 之间的间距 |
| 3xl / 12 | 48px | `py-12` / `padding: var(--space-3xl)` | 大块间距 |
| 4xl / 16 | 64px | `py-16` / `padding: var(--space-4xl)` | Hero section 间距 |

> **注意：** "使用参考"列应反映项目的实际约定 — Tailwind 类、CSS 自定义属性、组件 props 或其他约定。

## 使用规则
- 组件内部 padding：使用 xs–lg 层级
- 列表元素之间的 gap：使用 xs–sm 层级
- 容器/页面 padding：使用 xl–2xl 层级
- 绝不使用任意值 — 调整到最近的 token

## 网格和布局
- 最大容器：<项目的值和约定>
- 列：使用项目默认 gap 的 grid
```

**质量检查清单：**
- [ ] 基础层级已声明 — 从项目配置推导（非假设）
- [ ] 至少 8 个 token，包含 px 值、使用参考和典型用途
- [ ] 明确规则：何时使用层级的每个范围
- [ ] 明确规则：反对任意值
- [ ] 项目的容器和网格信息

---

### components.md

**目的：** 规范所有基础组件（来自检测到的库 — shadcn/ui、MUI、Radix 等）和可复用组合组件 — 变体、状态、何时使用每个。

**必需结构：**
```markdown
# 组件

## 基础组件

### Button
> **注意：** 根据检测到的库记录变体和大小。以下示例使用 props 约定（shadcn/ui）；适配为项目的 CSS 类、token 或 props。

**变体：**
| 变体 | 参考（prop / 类 / token） | 何时使用 |
|----------|-----------------------------------|-------------|
| primary | 如：`variant="default"` / `.btn-primary` | 屏幕的主要操作 — 每个 section 最多 1 个 |
| secondary | 如：`variant="secondary"` / `.btn-secondary` | 次要操作，主要操作的替代 |
| destructive | 如：`variant="destructive"` / `.btn-danger` | 不可逆操作（删除、取消） |
| outline | 如：`variant="outline"` / `.btn-outline` | 三级操作，较低视觉权重 |
| ghost | 如：`variant="ghost"` / `.btn-ghost` | 工具栏、菜单、密集区域中的操作 |
| link | 如：`variant="link"` / `.btn-link` | 文本中的内联导航 |

**大小：** 根据库记录（如：`size="sm"` | `size="default"` | `size="lg"` 或等效类）

**状态：**
- Loading：使用 `disabled` + 内部 spinner — 绝不在无视觉反馈的情况下禁用
- Disabled：`disabled` prop — 不明显时通过 tooltip 解释原因

**规则：**
- 每个视觉 section 绝不超过一个 `default` 按钮
- 破坏性按钮始终要求确认（modal 或 popover）

---

### Input
**必须实现的必需状态：**
- Default, Focus（可见 ring）, Error（destructive 边框 + 下方消息）, Disabled, Loading（readonly + skeleton）

**规则：**
- 每个 input 都有通过 `htmlFor` 关联的 `<label>` — 绝不用 placeholder 替代 label
- 错误消息在 input 下方，绝不在上方或作为 tooltip
- 使用 `aria-describedby` 指向错误消息

---

### <其他已安装的 shadcn 组件>
...

---

## 可复用组合组件

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
```

**质量检查清单：**
- [ ] 检测到的库中的每个组件在文件中都有一条目
- [ ] Button（或等效组件）记录了所有变体及何时使用
- [ ] Input 的所有状态已声明（default、focus、error、disabled）
- [ ] Label 的可访问性规则明确
- [ ] 组合组件有接口（props）、用途和规则声明
- [ ] 基础组件和可复用组合组件之间有明确区分

---

### themes.md

**目的：** 定义项目的主题（light/dark）以及 CSS 变量如何映射到每个主题。

**必需结构：**
```markdown
# 主题

## 可用主题
| 主题 | CSS 类 | 何时激活 |
|------|-----------|--------------|
| Light | `:root`（默认） | 系统偏好或手动选择 |
| Dark | `.dark` | 系统偏好或手动选择 |

## 各主题的 CSS 变量

| 变量 | Light | Dark | 语义 Token |
|----------|-------|------|----------------|
| `--background` | 0 0% 100% | 224 71% 4% | `background` |
| `--foreground` | 224 71% 4% | 213 31% 91% | `foreground` |
| `--primary` | ... | ... | `primary` |
| `--primary-foreground` | ... | ... | `primary-foreground` |
| ... | | | |

## 如何添加新主题
1. 创建 CSS 选择器或激活机制（如：`.theme-brand`、`[data-theme="brand"]`）
2. 重新定义相关的 CSS 变量或 token
3. 根据项目策略将选择器/机制应用于根元素

## 项目如何激活/切换主题
<描述项目中找到的实际策略：next-themes、纯 CSS、React context、data-theme 属性等>
```

**质量检查清单：**
- [ ] 所有可用主题已声明
- [ ] CSS 变量（或等效）表覆盖 `colors.md` 中的所有 token
- [ ] 如何添加新主题的说明
- [ ] 记录了项目如何激活/切换主题（找到的实际策略）
- [ ] 从找到的样式文件中推导 — 非编造

---

## 一般格式规则

- **语言：** 标题和描述使用中文；类名、token 和 props 使用英文
- **无编造值：** 所有内容从项目中找到的配置和代码推导
- **交叉引用：** `typography.md` 引用 `colors.md` 中的颜色；`components.md` 引用两者
- **代码示例：** 对 Tailwind 类和组件 props 使用代码块
- **语气：** 规定性 — "对 Y 使用 X"、"绝不 Z"

---

## 参考

- 产物格式（可定制）：`docs/sdd/design-system-format.md`
- 访谈指南：`references/interview-guide.md`
