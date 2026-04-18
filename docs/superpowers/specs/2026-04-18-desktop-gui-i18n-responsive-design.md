# Desktop GUI i18n And Responsive Hardening Design

## Goal

在不改动桌面 backend 契约的前提下，为当前 Wails/React 桌面前端补齐三项能力：

1. 提供一份基于现状的中文版 GUI 操作手册。
2. 为桌面前端加入中英文双语 i18n，默认中文。
3. 收紧小屏与空工作区行为：
   - 小屏时不再自动改单列或改变信息结构。
   - 当可视区域不足时允许页面自身滚动。
   - 在未加载缓存目录时，除 Landing 打开目录入口外，其余导航不可进入；点击时给出友好提示。

## Constraints

- 不新增第三方依赖。
- 不改 Go backend 绑定协议，不新增 Wails binding。
- 文案切换仅在前端完成。
- 操作手册必须描述当前真实能力，不补写不存在的功能。

## Current State

- 前端文案为纯英文硬编码，分散在 `App.tsx`、各页面组件、共享组件、store 映射与 fallback 文案中。
- 现有布局在窄宽度下会触发多处 `@media` 单列重排，与桌面端“保持操作台结构”的要求不符。
- 当前 `Sidebar` 导航在无工作区时仍可点击；`App.tsx` 会因无工作区统一回落到 `LandingPage`，但没有显式门禁反馈。
- Landing 已具备打开目录与 recent workspace 恢复入口，适合作为“无工作区唯一允许进入的页面”。

## Design Decisions

### 1. i18n Architecture

使用轻量前端字典层，不引入 `react-intl`、`i18next` 等依赖。

- 新增 `desktop/frontend/src/i18n/` 目录：
  - `messages.ts`：声明 `zh-CN` 与 `en-US` 字典。
  - `index.ts`：暴露 `LocaleKey`、`translate()`、`getMessages()` 等轻量 API。
- 在 `app-store.ts` 中加入 `locale` 状态，默认值为 `zh-CN`。
- 本轮只要求“默认中文”，但仍提供切换能力的状态接口，便于后续接设置页。
- 组件优先接收已翻译字符串；少量与 store 逻辑强耦合的提示直接在 store 内通过字典生成。

### 2. Workspace Gate

无工作区时的 GUI 行为统一如下：

- Landing 仍可显示并允许：
  - 打开目录
  - recent workspace 重开
  - invalid workspace 恢复
- `Overview / Explorer / Config / Operations` 导航按钮在视觉上表现为不可用。
- 若用户点击被禁用项，前端不跳页，而是弹出一条友好 toast，明确说明需要先打开缓存目录。
- 一旦存在有效工作区，导航恢复正常。

这条规则只适用于“未加载缓存目录”。`InvalidWorkspace` 仍回 Landing，但允许用户继续重新选择目录。

### 3. Responsive Policy

本轮明确采用“桌面布局锁定 + 容器滚动”策略：

- 删除当前会改变主布局结构的窄屏媒体查询：
  - `.frame` 不再改成单列。
  - `.explorerLayout` 不再改成单列。
  - `.operationsLayout` 不再改成单列。
  - `.fieldRow` 不再自动压成单列。
- 为主壳层与关键内容区设置桌面最小宽度：
  - 主框架最小宽度保持 sidebar + content 的双栏结构。
  - Explorer 与 Operations 内部分栏保持原结构。
- 当 viewport 小于最小宽度时：
  - `main.frame` 提供横向滚动。
  - 页面纵向内容继续正常纵向滚动。
  - 表格与配置区继续使用局部横向滚动。

目标是“看不全但不变形”，而不是“强行适配到手机布局”。

### 4. Chinese GUI Manual

新增独立中文文档，内容必须和当前 GUI 一致，覆盖：

- 启动前准备
- 在 Linux / Windows 上如何启动桌面端
- Landing / Overview / Explorer / Config / Operations 的逐步操作
- 任务执行、确认弹窗、Toast、detail pane 的含义
- 无工作区、无效工作区、stale snapshot、degraded read-only 等状态说明
- 当前已知限制

## Files In Scope

- `desktop/frontend/src/App.tsx`
- `desktop/frontend/src/state/app-store.ts`
- `desktop/frontend/src/components/*.tsx`
- `desktop/frontend/src/pages/*.tsx`
- `desktop/frontend/src/styles/shell.module.css`
- `desktop/frontend/src/components/__tests__/*.tsx`
- `desktop/frontend/src/__tests__/app-shell.test.tsx`
- `docs/desktop-gui-user-manual.zh-CN.md`
- `docs/superpowers/plans/2026-04-18-desktop-gui-i18n-responsive.md`

## Verification

- 前端单测覆盖：
  - 默认中文文案可见
  - 英文切换可用
  - 无工作区时导航被门禁并出现友好 toast
  - 窄宽度下壳层 landmark 仍可渲染，且不依赖单列布局
- `cd desktop/frontend && npm test -- --runInBand`
- `cd desktop/frontend && npm run build`

