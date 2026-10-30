# Coco Server 设计基线（草案 v0.1 · 待确认）

> 依据：`docs/ui/style-audit.md` + 现有代码静态提取。分支 `knowledge-hub`。
> 原则：只记录现状、复用现有变量、不引入新设计语言；**本文件为草案，未实施任何 Token/主题改动**，§10 为拟修改文件清单，等待确认。
> 所有依赖实际渲染的结论标「待验证」。

---

## 1. Token 分层架构（现状）

项目实际存在 **5 层变量体系**，自上而下派生，目前唯一权威源头是 antd 6 的 CSS 变量：

```
L1 源头配置    src/theme/settings.ts          主题色板 + 明暗 container/layout 文本色
               └→ src/store/slice/theme/shared.ts  运行时注入 <style id="theme-vars">
L2 antd 6      cssVar: true 自动生成 --ant-*   组件样式的唯一事实源
L3 桥接层      src/theme/vars.ts               uno theme ← → --ant-* / --*-color
L4 语义别名层  src/styles/css/wiki.css :root   --wiki-*（派生自 --ant-*，含 .dark 覆盖）
L5 内容/隔离域 DocumentDrawer/markdown.scss    --color-fg-default 等（GitHub Primer 风格，文档渲染专用）
               widgets/searchbox variables.css --infini__searchbox-*（embed 隔离域，独立构建）
```

**判定**：L4/L5 不是冲突，而是合理的域隔离（审计报告 E 类）。真正的问题是 L3 桥接层产出的部分语义类**零使用**（§8）。

## 2. 颜色 Token（名称 · 实际值 · 语义）

### 2.1 源头色板（settings.ts `otherColor` / themeColor）

| Token | 值 | 语义 | 用途 |
|---|---|---|---|
| `themeColor` | `#0087FF` | 品牌主色 | primary 全链源头 |
| `otherColor.success` | `#52c41a` | 成功/正向 | antd colorSuccess |
| `otherColor.warning` | `#faad14` | 警告 | antd colorWarning |
| `otherColor.error` | `#f5222d` | 错误/危险 | antd colorError |
| `otherColor.info` | `#2080f0` | 信息 | antd colorInfo |

⚠️ **色值冲突（已确认）**：涨跌/状态场景存在第三套值——内联 `#ff4d4f`（红）与 `#52c41a`（绿）共 15 处，`#ff4d4f` 与 `error: #f5222d` 数值漂移且暗色模式不跟随 token。→ 待确认语义后统一（§10-P4）。

### 2.2 中性色（明/暗两套，settings.ts tokens）

| 语义 | Light | Dark |
|---|---|---|
| 正文 `base-text` | `rgb(31,31,31)` | `rgb(224,224,224)` |
| 容器 `container` | `rgb(255,255,255)` | `rgb(28,28,28)` |
| 布局底 `layout` | `rgba(238,240,243,1)` | `rgb(18,18,18)` |
| 反色块 `inverted` | `rgb(0,20,40)` | （未单独定义，沿用 light） |

### 2.3 antd ↔ 项目 Token 映射（theme/vars.ts 桥接，实际生效路径）

| 项目侧名称 | antd 源 | 实际使用度 |
|---|---|---|
| `--primary-color` 系（50–950 色阶） | 由 `@sa/color` 从 themeColor 生成 | scss 中 `var(--ant-color-primary)` 21 处 ✅ |
| uno 类 `text-text-1/2/3` | `--ant-color-text` / `-secondary` / `-tertiary` | **0 处使用** ❌（scss 直接写 `var(--ant-color-text)` 9 处） |
| uno 类 `bg-container` / `bg-layout` | `--container-bg-color` / `--layout-bg-color` | 6 / 5 处 ✅ |
| uno 类 `border` / `border-secondary` | `--ant-color-border(-secondary)` | 低（scss 常直接用 antd 变量） |
| `--wiki-*` 20 个语义变量 | 全部派生自 `--ant-*`（accent/panel/border/text/shadow/grad/glow） | wiki 域 100+ 处 ✅ |
| markdown `--color-fg-default` 等 | 独立 Primer 调色板（light `#24292f` / dark `#c9d1d9`） | 仅文档渲染内容 |

**关键映射结论**：页面上实际消费的主路径是 **`var(--ant-*)` 原生变量**，而非 uno 语义类。治理时应顺势把 `--ant-*` 定为文档化标准，而不是推广未使用的别名。

## 3. 字体 / 字号 / 行高

| 项 | 实际值 | 来源 |
|---|---|---|
| 根字号 | `16px`（html, body, #root） | `styles/css/global.css` |
| 正文字号 | antd 默认 `14px`（`var(--ant-font-size)`，scss 中 8 处引用） | antd cssVar |
| 行高 | antd 默认（`var(--ant-line-height)`，8 处引用） | antd cssVar |
| 字体栈 | 系统无衬线（Tailwind 风格 reset：`ui-sans-serif, system-ui, ...`） | `styles/css/reset.css` |
| 等宽栈 | `ui-monospace, 'SF Mono', Menlo, Consolas, monospace` | reset.css / wiki.css:439 |
| 图标字号 | uno theme：icon-xs 0.875 / icon 1.125 / icon-large 1.5 / icon-xl 2rem | uno.config.ts |
| 页头标题 | `16px / font-500`；描述 `12px / --ant-color-text-tertiary` | `PageHeaderTitle.tsx`（共享组件） |

无自定义 fontFamily 引入（无外部字体文件）✅。

## 4. 间距 / 圆角 / 边框 / 阴影

| 类别 | 实际值 | 来源 / 用途 |
|---|---|---|
| 卡片内边距 | `12px 16px` | ConfigProvider `Card.styles.body`（全局统一） |
| 页面内容区 | `12px 16px`（`.settings-tabs-content`）；tabs 式页 `0 24px` | 页面 scss（两值并存，成文于 §7） |
| Collapse 内容 | `16px 16px 24px 16px` | getAntdTheme 组件 token |
| 卡片圆角 | `8px`（`card-wrapper` shortcut，39 处） | uno.config.ts |
| 组件圆角 | antd 默认 `6px`（tree/list 覆盖处亦用 6/8px） | wiki.css |
| 卡片阴影 | `0 1px 2px rgba(0,0,0,.04), 0 8px 24px rgba(0,0,0,.05)` | `--wiki-shadow-card` |
| 悬浮阴影 | `0 2px 4px rgba(0,0,0,.05), 0 16px 40px primary 16%` | `--wiki-shadow-lift` |
| 布局阴影 | header/tab `0 1px 2px rgba(0,21,41,.08)`；sider `2px 0 8px rgba(29,35,41,.05)` | settings.ts tokens（uno 侧 `shadow-header` 等类 **0 使用**，仅定义） |
| 边框 | 统一走 `--ant-color-border(-secondary)`；wiki 域另有 55% 透明软化 `--wiki-border-soft` | |
| 通用布局 shortcut | `flex-center / flex-col-center / ellipsis-text / h-calc()` 等 | uno.config + @sa/uno-preset |

## 5. 布局规范

| 项 | 值 |
|---|---|
| Header 高 | 56px（fixedHeaderAndTab: true） |
| 多标签栏 | 44px，**默认不可见**（tab.visible: false） |
| Sider | 宽 258 / 收起 64 / mix 90 / mix 收起 64；菜单 itemMarginInline 8px |
| Footer | 48px，默认不可见 |
| 内容最小高 | `calc(100vh - 176px)`（页面 scss 模式） |
| 暗色切换 | `html.dark` 类 + antd darkAlgorithm + CSS 变量双写（App.tsx 含合成层重绘 hack，勿删） |

## 6. 组件常规尺寸 / 状态 / 允许变体（现状归纳，非新约定）

| 组件 | 常规（默认） | 变体与用量 | 备注 |
|---|---|---|---|
| Button | antd 默认 32px；small=28px（全局 token controlHeightSM） | `size="small"` 48 处；`large` 4 处 | 密度混排，目标约定**待确认**（审计 #6） |
| Table | antd 默认（**0 处 size 定制**，密度天然一致 ✅） | 无 | 最好的一致性样本 |
| Tabs（页面级） | nav 高 40px、ink bar 4px、tab 内边距 8px、间隔 38px | settings 模式被 4 页 scss 复制（各 10 处 `.ant-tabs-*`） | **应上移组件 token**（§10-P1） |
| Card | body `flex:1; padding:12px 16px`（全局） | 圆角 8px（card-wrapper） | |
| Menu | itemSelectedBg = primary 10% 透明 | dark 透明底 | getAntdTheme |
| Modal | antd 默认（27 处未发现自定义宽高约定） | — | 变体清单待验证 |
| 页头 | `PageHeaderTitle`（icon + 16px/500 标题 + 12px 描述） | description 可选 | 唯一页头组件，✅ 复用 |
| 空态/加载 | `src/common/src/DropdownList`（List/Empty/Error/Loading 模块化） | — | 复用而非重写 |
| 图标 | 三套并存（antd icons 5.6.1+6.2.3 / iconify / lucide） | — | 收敛方案见审计 §6，暂缓 |

## 7. 页面模式基准（代码级，视觉待验证）

| 类型 | 基准页面 | 关键模式 |
|---|---|---|
| 列表页 | `pages/pipeline/list`、`pages/skill/list` | `card-wrapper` 容器 + ATable 默认密度 + PageHeaderTitle |
| 详情页 | `pages/data-source/detail` | 专属 `detail/index.scss`（10 处 .ant-* 待 token 化） |
| 表单页 | `pages/settings`（tabs 式）| 40px tabs + `12px 16px` 内容区，模式被复制 4 份 |
| 特殊风格域 | `pages/wiki/*` | `--wiki-*` 语义层 + Yuque 式密度（**例外，保留**） |

## 8. Token 重复 / 冲突 / 未使用清单

| 类型 | 项 | 证据 | 处置建议 |
|---|---|---|---|
| 未使用 | uno 类 `text-text-1/2/3` 及派生 shortcut `text-color-*` | 全仓 0 处 | 保留定义（无害）但文档标注"非标准路径"；或待确认后移除 |
| 未使用 | uno 类 `shadow-header/sider/tab` | 类形式 0 处（值仅被 css 内部使用） | 同上 |
| 冲突 | error 三值：`#f5222d`(config) vs `#ff4d4f`(内联) vs antd 默认 | 审计 D1 | P4 统一（需语义确认） |
| 重复 | `.ant-tabs` 页面级覆盖 4 份 × 10 处 | settings/security/integration/data-source scss | P1 上移 `Tabs` 组件 token |
| 命名误导 | `darkMode.css` 内容为 sider 动画 | 审计 A2 | P5 改名 |
| 死文件 | `styles/scss/global.scss`、`scrollbar.scss` | 全仓 0 引用 | P5 删除（待确认） |

## 9. 已确认的业务例外（**保留，不统一**）

1. wiki 域整体视觉语言（`--wiki-*` + 30px 密度树 + 渐变标题）——有明确设计注释。
2. `markdown.scss` 的 Primer 调色板与 `!important`——服务第三方渲染内容。
3. `widgets/searchbox` 的 `--infini__searchbox-*` 隔离域——embed 场景。
4. 设置页 vs 列表页的信息密度差异。
5. 涨跌/状态色若业务确认需要独立红绿色板（区别于语义 error/success），则作为例外保留但**须收编为命名变量**（如 `--color-up/--color-down`），不允许继续内联。

## 10. 最小变量文件改造方案（**仅清单，未实施**）

| 优先级 | 改动 | 涉及文件 | 视觉影响 |
|---|---|---|---|
| P1 | 4 页 `.ant-tabs` 覆盖上移 `getAntdTheme().components.Tabs` | `src/store/slice/theme/shared.ts`；删 4 个页面 scss 对应块 | 低（逐页回归） |
| P2 | `--wiki-*` 别名层从 wiki.css 抽到共享文件（若 wiki 外复用） | `src/styles/css/wiki.css` → `src/styles/css/semantic-vars.css` | 零 |
| P3 | 内联硬编码色 → `var(--ant-color-*)`（或 P5 确认后的 `--color-up/down`） | 约 8 个 tsx 文件 15 处 | 零～低（暗色改善） |
| P4 | error/涨跌色语义统一 | `theme/settings.ts` 或内联处 | 待业务确认 |
| P5 | 清理：删死文件、darkMode.css 改名 | `styles/scss/*`、`styles/css/darkMode.css` | 零 |

**不做**：不新建变量文件、不改 uno.config 主题值、不动 L4/L5 域隔离、不批量改业务页面样式。

## 11. 尚待确认事项

1. Button 密度目标：列表页是否统一 `small`？（影响 §6 成文）
2. 涨跌/状态色语义（P4）：走语义 error/success 还是独立 up/down 色板？
3. 未使用的 uno 语义类（text-text-*）删除还是保留？
4. `--wiki-*` 是否会被其他模块复用（决定 P2 是否执行）？
5. Modal 宽度/圆角是否已有隐性约定（需浏览器抽查，待验证）。
6. 各基准页实际渲染截图比对（§7 全部"待验证"项的闭环）。

---
*草案 v0.1 —— 确认 §10 优先级与 §11 事项后，再实施 Token/主题配置。*
