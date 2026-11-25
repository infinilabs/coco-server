# Coco Server 前端样式审计报告

> 分支：`knowledge-hub`（跟踪 `origin/feat/knowledge-hub`，本地另有若干 Go 文件改动，与前端无关）
> 审计日期：2026-10-16 ｜ 审计范围：`web/` 目录 ｜ 性质：只读审计，未修改任何业务代码
> 视觉验证说明：本次审计基于代码与配置的静态分析，**未启动浏览器截图比对**；所有依赖实际渲染的视觉结论均标注为「待验证」。

---

## 1. 实际技术栈及依赖版本（基于实际配置核查，非文件名推测）

### 1.1 核心结论：**不是 Next.js 项目**

全仓库（主应用 + 3 个 widgets 子包）无 `next` 依赖、无 `next.config.*`。实际为 **Vite 5 SPA**：

| 项 | 实际情况 |
|---|---|
| 构建 | Vite 5.4.10 + @vitejs/plugin-react 4.3.3 |
| 框架 | React 18.3.1 + TypeScript 5.6.3 |
| 路由 | `@sa/simple-router`（workspace 包）+ react-router-dom 6.27.0 + `@ohh-889/react-auto-route` 文件约定路由（`src/router/elegant/`），history 模式。**无 App Router / Pages Router 概念** |
| 包管理 | pnpm 9.12.3 workspace（`packages/` 9 个内部包 `@sa/*`，`widgets/` 3 个独立构建子应用） |
| 模板来源 | SoybeanAdmin（react-soy），package.json name 仍为 `react-soy` |
| 微前端 | wujie（`window.__POWERED_BY_WUJIE__` → `src/components/micro/`），主应用可被 wujie 宿主加载 |
| CSS-in-JS | 主应用**不直接使用** cssinjs；`widgets/ui-search` 显式依赖 `@ant-design/cssinjs ^2.0.1`；lockfile 中 `@ant-design/cssinjs` 1.24.0 与 2.1.2 两个大版本并存（1.x 来自 pro-components/entity-ui 依赖链） |
| SSR 样式注册 | 无 SSR；antd 6 开启 `cssVar: true`，样式由运行时 CSS 变量承载，无 SSR 注册需求 |

### 1.2 关键依赖版本

| 依赖 | 版本 | 备注 |
|---|---|---|
| antd | **6.1.3**（唯一版本，主应用与 ui-search 一致 ✅） | |
| @ant-design/icons | direct **5.6.1**；lock 中另有 **6.2.3**（antd 6 内部依赖） | 双大版本并存，图标实例类型/树摇行为可能不一致 |
| @ant-design/pro-components | ^2.8.10 | 仅 1 处文件使用 |
| @infinilabs/entity-ui | ^0.0.5 | 仅 1 处文件使用 |
| UnoCSS | 0.63.6（主应用）；widgets/searchbox 混用 `unocss ^66.2.3` + `@unocss/vite 0.63.6` | ⚠️ searchbox 内双 major 混装 |
| 样式预处理器 | sass 1.80.6 + less 4.6.4 **同时启用** | |

### 1.3 目录与样式方案

- 页面：`src/pages/`（31 个模块）；组件：`src/components/` + `src/common/src/`（内部共享组件，如 DropdownList）；布局：`src/layouts/`；主题：`src/theme/`（settings.ts / vars.ts）+ `src/store/slice/theme/`
- 样式技术栈为**四套并存**：
  1. **UnoCSS**（uno.config.ts：presetUno `dark: 'class'` + `@sa/uno-preset`，`preflight: false`）— 原子类为主力
  2. **CSS Modules**：14 个 `.module.less`（common、License、Resource、layouts）
  3. **页面级 SCSS/普通 CSS**：15 个 scss + 12 个 css（多为非 module 的局部文件，依赖人工前缀隔离）
  4. **内联样式**：238 处 `style={{}}`
- 无 Tailwind。图标体系三套并存：`@ant-design/icons`、`@iconify/react`（含离线集）、`lucide-react`

### 1.4 全局样式加载链（顺序即优先级关系）

```
index.html
└─ src/main.tsx
   ├─ ./plugins/assets
   │   ├─ 'uno.css'                        ← UnoCSS 产物（无 preflight）
   │   ├─ 'virtual:svg-icons-register'
   │   └─ src/styles/css/global.css
   │       ├─ @layer base, properties
   │       ├─ reset.css (layer base)
   │       ├─ nprogress.css
   │       ├─ wiki.css                     ← ⚠️ 全局加载的"页面级"样式
   │       └─ 自身还含 .ant-* / .wiki-toc-* 规则
   └─ App.tsx
       ├─ <AConfigProvider theme={getAntdTheme(...)}>   ← antd 6 cssVar + darkAlgorithm
       ├─ setupThemeVarsToHtml → 注入 <style id="theme-vars">（:root 与 html.dark 两套 CSS 变量）
       └─ toggleCssDarkMode → html.dark 类（UnoCSS dark:class 与之联动）
```

主题单一入口确认：`src/theme/settings.ts`（themeSettings，primary `#0087FF`）→ `src/store/slice/theme/shared.ts`（getAntdTheme / setupThemeVarsToHtml / toggleCssDarkMode）→ `src/theme/vars.ts`（UnoCSS theme 与 `--ant-*`、`--*-color` CSS 变量桥接，如 `text-1: var(--ant-color-text)`）。**未发现第二套主题配置**；但存在**多条暗色实现路径**（见 §3-A2）。

---

## 2. 页面清单（按 `src/pages/` 实际路由归纳）

| 类别 | 模块 |
|---|---|
| 列表页 | `wiki/list`、`integration`、`connector`、`api-token`、`user`、`role`、`webhook`、`skill`、`search` |
| 详情页 | `data-source/detail`（含 doc-detail.scss）、`wiki/article/[id]`、`pipeline`、`ontology`、`mcp-server` |
| 创建/编辑页 | `data-source/new/*`（含 `modules/FieldMapping.tsx`）、`connector/new`、`integration/modules/EditForm`、`integration/modules/Preview` |
| 设置页 | `settings`（含 `modules/ChatStartPage.scss`）、`search-settings`、`connector-settings`、`wiki-settings`、`memory-settings`、`model-provider`、`security`、`data-security` |
| 仪表盘/数据展示 | `home`（home.scss）、`search-ops`、`search-studio`、`engine-ai`、`content-dedup`、`guide`、`preview` |
| 弹窗/抽屉 | AModal 27 处；`DocumentDrawer`（markdown.scss + highlight.css）、`IntegratedStoreModal`、`Resource/EditShares`、`License/ApplyTrial`、widgets `FullscreenModal` |
| 认证/独立 | `login`、`auth`、`_builtin` |

**重复模式识别**：页面头部有共享组件 `components/stateless/common/PageHeaderTitle`；卡片有 UnoCSS shortcut `card-wrapper`（rd-8px shadow-sm，39 处使用）；但列表页/tab 表单页各自维护 scss 覆盖（见 §3-B1）。

---

## 3. 主要问题及分类（A–F）

### A. 全局设计规则缺失或冲突

**A1. `.wiki-toc-tree` 规则在 global.css 与 wiki.css 双份维护** ⚠️ 高优先
- `src/styles/css/global.css` L49–95 与 `src/styles/css/wiki.css` L510–545 各有一套 `.wiki-toc-tree.ant-tree ...` 规则：global.css 管密度/布局（line-height 30px、flex wrapper、4px indent），wiki.css 管背景/hover/圆角。同一选择器分散两文件、加载顺序上 wiki.css 先于 global.css 自身规则，**后者依赖全局文件间的隐式层叠**。global.css 注释里说"sidebar panel chrome lives in wiki.css"，但密度规则仍留在 global.css，分界已失守。
- 风险：改 wiki 侧栏时只改一处会导致"改了没生效/两处打架"。修复风险低（纯合并迁移）。

**A2. 暗色模式三机制并存**
1. antd `darkAlgorithm`（token 层）；
2. `html.dark` + UnoCSS `dark:` 变体 + `<style id="theme-vars">` 双套 CSS 变量；
3. `App.tsx` useTheme 中**双 rAF 后 `document.body.style.display` 卸载重绘 hack**（注释说明是修复 watermark/keepalive 合成层残影）。
- 三者各自正确，但相互耦合；任何新的页面级暗色覆盖如果不知道机制 3，会误判"闪一下"是自己的 bug。`src/styles/css/darkMode.css` **名不副实**——内容是 sider 展开动画的 `--uno` 指令，与暗色无关（待确认后建议改名）。

**A3. 死样式文件**
- `src/styles/scss/global.scss`、`src/styles/scss/scrollbar.scss` 全仓库（src + packages + index.html）**零引用**。属模板残留，建议删除（需人工确认，见 §8）。

### B. Ant Design 主题配置不一致

**B1. 组件级覆盖"该上 token 的在写 CSS"**
`getAntdTheme`（`src/store/slice/theme/shared.ts`）只配置了 3 个组件 token（Button.controlHeightSM、Collapse、Menu），而同类诉求散落在页面 scss 中：
- `pages/settings/index.scss`：`.ant-tabs-*` 10 处、`.ant-form-item` 结构覆盖
- `pages/security/index.scss`、`pages/integration/list/index.scss`、`pages/data-source/detail/index.scss`：各 10 处 `.ant-*`
- `styles/css/wiki.css`：15 处 `.ant-*`（ant-tree/ant-list/ant-tabs 密度与圆角）
- 合计 **21 个文件、约 110 处 `.ant-*` 选择器覆盖**，其中相当比例（tabs 密度、tree 行高、list 行距）是可上移为 `components: { Tabs: {...}, Tree: {...}, List: {...} }` token 的。这些页面级覆盖**没有父级作用域包裹**（非 module 文件），存在全局污染风险——例如两个页面同时自定义 `.ant-tabs-tab` 时按加载顺序胜出。
- 修复风险：中。token 化后视觉效果需逐页回归（待验证项）。

**B2. `!important` 32 处**，集中在 `components/DocumentDrawer/markdown.scss`（第三方渲染内容对抗 antd 默认样式，有一定合理性）、`pages/settings/index.scss`、`pages/role/modules/Permissions.module.less`、`pages/integration/modules/Preview.css` 等。markdown.scss 内的属 E 类可保留，页面级的应随 B1 一并收敛。

### C. 公共组件重复实现 / 依赖不一致

**C1. 图标库三套**：`@ant-design/icons` 5.6.1（直接依赖）与 antd 6 内部 6.2.3 并存；`@iconify/react` + `@iconify/json` 离线集；`lucide-react`。视觉上三套图标线条粗细/填充风格不一致（待验证），且双 major 的 @ant-design/icons 有类型与包体成本。建议：升级 direct 依赖到 6.2.3 对齐；中期收敛到 antd icons + iconify 两套。

**C2. 卡片样式双轨**：UnoCSS shortcut `card-wrapper`（39 处）与各处手写 `rd-8px`/`shadow-sm` 等价类、`License/index.module.less` 自绘卡片并存；`App.tsx` 的 ConfigProvider 还给 Card 统一注入了 body padding（12px 16px），三处来源叠加后各页卡片内边距一致性依赖运行时（待验证）。

**C3. cssinjs 双运行时**：主链 antd 6 用 cssinjs 2.x，pro-components/entity-ui 依赖链拉入 1.x。antd 6 + pro-components 2.8 的兼容矩阵官方已声明支持，实际运行影响待验证（若控制台无重复样式告警可降级为记录项）。

### D. 页面局部样式偏差（代码级证据）

**D1. 内联硬编码颜色 ~15 处**（`style={{}}` 内 hex/rgb），代表：
- `src/pages/...`（涨跌/状态色）：`color: up ? '#52c41a' : '#ff4d4f'` —— 与 `themeSettings.otherColor`（success `#52c41a`、error `#f5222d`）**数值漂移**（error 一个用 `#ff4d4f` 一个用 `#f5222d`，antd 默认 red 也再一个值），暗色模式下这些不会跟随 token。
- `backgroundColor: '#f5f5f5'`（浅色假设写死，暗色模式会突兀，待验证）
- `pages/data-source/modules/FieldMapping.tsx` 等多处。
- 修复风险：低。替换为 `var(--ant-color-success)` / themeVars 桥接变量即可，但需要逐处确认语义。

**D2. 按钮密度不统一**：`size="small"` 48 处 vs 默认 0 显式 vs `size="large"` 4 处。叠加 `Button.controlHeightSM=28` 的全局 token，列表页按钮实际密度由"默认 32 / small 28"混排决定（各页取舍无记录，待验证视觉）。

**D3. 样式文件组织三轨制**：同一层级的页面，有的用 `.scss`、有的用 `.module.less`、有的纯 UnoCSS 类、有的内联。无命名约定时，非 module 的 scss（如 `pages/security/index.scss`）只能靠顶层 class 手动隔离——检查发现这些文件均有页面根 class 包裹（如 `.security-page`），目前无实际冲突案例，但约定未成文，属规则缺失（A 类根因）。

### E. 合理的业务差异（**不建议"修复"**）

- `wiki.css` 的 Yuque 风格目录树密度（30px 行高、14px 字号、20px 缩进，代码注释明确说明设计意图）——有意设计。
- `widgets/`（searchbox/ui-search/fullscreen）独立样式体系：searchbox 用 `--infini__searchbox-*` 命名空间 + 纯 CSS，服务于 embed 到外部站点的场景，**与主应用隔离是正确的**，不应并入主应用主题体系。ui-search 与主应用 antd 版本一致（6.1.3）也已对齐。
- `DocumentDrawer/markdown.scss` 对渲染内容（vditor/mermaid/katex 输出）的覆盖与 `!important`：对抗第三方产物，可接受。
- 设置页 vs 列表页的密度差异本身是信息密度的合理业务差异。

### F. 证据不足，需人工确认 / 待验证

1. 各页按钮/表格实际渲染密度、卡片内边距一致性（需浏览器截图比对，见 §7 视觉基线）。
2. 暗色模式下内联 `#f5f5f5`、涨跌色等是否产生可见突兀。
3. cssinjs 双运行时是否产生样式注入顺序/闪变问题。
4. `darkMode.css` 是否有计划中的暗色用途（当前内容与暗色无关）。
5. `global.scss` / `scrollbar.scss` 是否为保留文件（当前零引用）。
6. searchbox 内 unocss 66 与 0.63 双 major 是否导致产物类名差异（该包独立构建，影响面限于 widget）。

---

## 4. 问题优先级、影响范围与修复风险汇总

| # | 问题 | 类 | 优先级 | 影响范围 | 修复风险 |
|---|---|---|---|---|---|
| 1 | wiki-toc 规则双份维护（global.css ↔ wiki.css） | A1 | 高 | wiki 全模块 | 低（纯迁移合并） |
| 2 | 页面级 `.ant-*` 覆盖未 token 化、无作用域约定 | B1/D3 | 高 | settings/security/integration/data-source/wiki 等约 10 页 | 中（需逐页视觉回归，待验证） |
| 3 | 内联硬编码颜色（含与主题色值漂移的涨跌色） | D1 | 中 | home/data-source/wiki 等约 8 文件 | 低 |
| 4 | `@ant-design/icons` 5.x/6.x 双版本 | C1 | 中 | 全局包体 + 图标一致性（待验证） | 低（升 direct 版本） |
| 5 | 死文件 global.scss / scrollbar.scss、darkMode.css 命名误导 | A3/A2 | 低 | 仓库卫生 | 极低（删/改名前人工确认） |
| 6 | 按钮密度无约定（48 small vs 默认） | D2 | 中 | 全部列表/表单页 | 中（先定约定再收敛，待验证） |
| 7 | searchbox unocss 双 major | C2 | 低 | 仅 widgets/searchbox | 低 |
| 8 | body 重绘 hack / cssinjs 双运行时 | A2/C3 | 记录 | 全局 | 仅记录，不动 |

## 5. 可复用的现有变量与公共组件（治理时应复用而非新建）

- **CSS 变量桥**：`src/theme/vars.ts` 已暴露 `--primary-color`、`--container-bg-color`、`--ant-color-text/-secondary/-tertiary`、`--ant-color-border` 等 —— D1 的硬编码色应替换为此处变量。
- **UnoCSS shortcut**：`card-wrapper`、`text-color-*`、`h-calc()` —— 卡片/文本层级统一入口。
- **组件 token 入口**：`src/store/slice/theme/shared.ts → getAntdTheme().components` —— B1 的 tabs/tree/list 覆盖应上移至此。
- **共享组件**：`PageHeaderTitle`（页面头部）、`src/common/src/DropdownList`（下拉列表态：List/Empty/Error/Loading 已模块化）、`AppProvider`（antd App message/modal 上下文）。
- **widgets 命名空间范式**：`--infini__searchbox-*` 自定义属性方案可作为"可 embed 样式"的既定范式引用。

## 6. 建议治理顺序与文件级改造计划（渐进式，保持现有设计风格）

1. **第 1 步（零视觉变化）**：合并 A1 —— 将 global.css 中 `.wiki-toc-*` 密度规则迁入 wiki.css 对应区块，global.css 只留真全局（reset/view-transition/grayscale）；删除 `global.scss`、`scrollbar.scss`（确认后）；`darkMode.css` 改名 `sider-transition.css`。
2. **第 2 步（低风险）**：D1 硬编码色替换为 themeVars 变量；`@ant-design/icons` direct 依赖升至 6.2.3。
3. **第 3 步（中风险，逐页做）**：B1 token 化 —— 按 settings → security → integration → data-source 顺序，把 `.ant-tabs/.ant-form/.ant-table` 覆盖上移到 `getAntdTheme().components`，每页迁移后截图回归（对照基线）。
4. **第 4 步（约定成文）**：在 docs/ui/ 增加样式约定（页面样式默认 UnoCSS；需要选择器时用 `.module.less`；禁止无作用域的页面级 `.ant-*`；新增主题色一律走 themeVars）。成文后补 lint（可基于已有 `@unocss/eslint-config` + 自定义规则）。
5. **暂不做**：组件库替换、UnoCSS 版本升级、wujie/微前端结构调整、widgets 样式合并。

## 7. 推荐的视觉基线页面

- **wiki 模块**（`pages/wiki/list` + `pages/wiki/article/[id]`）：样式体系最完整、注释最充分、密度设计有明确意图，适合作为布局/密度基线。
- **settings**（`pages/settings`）：表单/tab 最密集，作为表单类基线。
- **home**：仪表盘基线（含 D1 涨跌色，修正前后对比最佳）。
- 建议基线截图采集时机：第 1、2 步完成后各一轮，第 3 步每页迁移后与基线 diff。

## 8. 需要人工确认的事项

1. `global.scss`、`scrollbar.scss` 可否删除（当前零引用）。
2. `darkMode.css` 改名是否影响其他分支合并（仅本地确认即可）。
3. 按钮密度的目标约定：列表页统一 `small` 还是跟随默认？（影响第 4 步约定内容）
4. 涨跌/状态色语义：`#ff4d4f` 是否应统一为 `otherColor.error = #f5222d`，还是涨跌场景需要独立的红绿色板（中国行情惯例红涨绿跌时当前 `#52c41a` 做涨色本身是个反直觉用法，需业务确认）。
5. widgets（searchbox/ui-search）是否需要跟随主应用暗色模式联动。
6. 浏览器视觉回归验证由谁执行（本报告所有"待验证"项依赖此步闭环）。

---

*报告完。本次为只读审计：除本文件外未新增、修改、删除任何项目文件。*
