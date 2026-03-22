# 变更归档：Win 按钮 hover 手型与禁选中文本

## 变更背景 / 目标

- 当前 Win 界面里的按钮在鼠标悬停时会出现文本选择光标，且按钮文字可被选中复制，破坏了桌面按钮的基础交互一致性。
- 你要求：
  1. 所有按钮 hover 时统一显示手型光标；
  2. 按钮文字不能被复制；
  3. 不改动业务流程，只修复交互样式。

目标：
- 统一共享按钮组件与原生按钮的交互样式；
- 在不改动业务逻辑的前提下，消除按钮文本选择态。

## Requirements / Specs Impact

- Requirements impact：`none`
- Specs impact：`none`
- Related requirements：`none`
- Related specs：`none`

## 具体变更内容（新增 / 修改 / 删除）

### 新增

- 无。

### 修改

- `frontend/src/components/ui/button/Button.vue`
  - 为共享 `Button` 默认类补充 `cursor-pointer` 与 `select-none`；
  - 移除 `disabled:pointer-events-none`，避免禁用按钮 hover 时丢失按钮命中，导致光标退回文本选择态。
- `frontend/src/style.css`
  - 在 `@layer base` 中新增按钮级基础规则，覆盖：
    - `button`
    - `[role="button"]`
    - `input[type="button"]`
    - `input[type="submit"]`
    - `input[type="reset"]`
  - 统一应用手型光标与禁选中文本样式。

### 删除

- 无。

## 对应 plan / todo 任务映射

- `T1`：统一按钮交互样式。
- `T2`：验证与回归检查。
- `T3`：执行 Code Review。
- `T4`：归档变更并更新索引。

## 关键设计决策与权衡

- 采用“双层收口”而不是只改一个点：
  - 共享 `Button` 组件负责覆盖组件化按钮；
  - 全局基础样式负责覆盖仓库内大量原生 `<button>`。
- 不把修复散落到各页面：
  - 优点：改动集中、后续新增按钮默认继承；
  - 权衡：需要在共享组件里取消 `disabled:pointer-events-none`，以满足“所有按钮 hover 都是手型”的明确需求。

### 性能要点

- 纯 CSS / Tailwind 类调整，无新增 JS、I/O、轮询或重复计算。
- 仅影响 hover 与 selection 的基础样式，不增加运行时负担。

### 可扩展性要点

- 新增页面或窗口里的原生按钮会自动继承基础交互规则。
- 共享 `Button` 组件继续作为主路径入口，后续若要统一按钮视觉语义，可在同一处扩展。

## 测试与验证方式 / 结果

- 代码差异审查：`git diff -- frontend/src/components/ui/button/Button.vue frontend/src/style.css`（通过）。
- 原生按钮覆盖复核：
  - `rg -n -e '<button\\b' -e '\\[role="button"\\]' frontend/src`
  - 结论：仓内存在大量原生 `<button>`，全局基础规则命中路径正确。
- 依赖安装：`npm --prefix frontend ci`（通过）。
- 构建验证：`npm --prefix frontend run build`（失败，非本次改动引入）。
  - 失败信息：`Could not resolve "../../wailsjs/go/main/App" from "src/pages/TopicBus.vue?vue&type=script&setup=true&lang.ts"`。
  - 结论：当前 worktree 缺少 `wailsjs` 生成物，属于既有环境问题，未发现本次样式改动导致的新构建错误。

## 潜在影响与回滚方案

### 潜在影响

- 禁用态共享按钮不再使用 `pointer-events-none`，因此 hover 时会显示手型光标。
- 这与常见“禁用按钮显示不可用光标”的默认习惯不同，但与本次明确需求保持一致。

### 回滚方案

- 直接回退以下文件即可完整撤销本次变更：
  - `frontend/src/components/ui/button/Button.vue`
  - `frontend/src/style.css`

## 子Agent执行轨迹

- `T1` → 主Agent → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-button-pointer` → `frontend/src/components/ui/button/Button.vue`, `frontend/src/style.css` → 验收通过
- `T2` → 主Agent → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-button-pointer` → 无新增实现文件 → `npm ci` 通过，构建受既有环境问题阻塞
- `T3` → 主Agent → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-button-pointer` → `plan.md` → Code Review 通过
- `T4` → 主Agent → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-button-pointer` → `docs/change/2026-03-22_win-button-pointer.md`, `docs/change/README.md` → 归档通过
