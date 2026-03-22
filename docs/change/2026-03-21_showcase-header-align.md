# Showcase 头部对齐 Flow

## 变更背景 / 目标
- 背景：Showcase Editor 头部仍是上下两层，内容区还有额外的 `Editing Surface / Layout` 说明块；Viewer 头部也还是两层结构，窗口骨架与 Flow / Showcase Editor 不一致。
- 目标：将 Showcase Editor / Viewer 都收敛到更接近 Flow 的单行头部结构，上方承载名称、状态和按钮，下方直接进入编辑或展示内容。

## 具体变更内容
### 修改
- `frontend/src/pages/Showcase.vue`
  - 将编辑窗口头部改为单行布局，左侧聚合 `screen name / connected / unsaved / layout / widget count / updated time`。
  - 将原本第二行的按钮区移动到 header 右侧，形成单行按钮排。
  - 删除编辑内容区顶部的 `Editing Surface / Layout` 说明块，让预览区直接开始。
  - 移除不再使用的 `currentLayoutSummary` 计算属性。
- `frontend/src/windows/ShowcaseWindow.vue`
  - 将 viewer 窗口改为与 editor / flow 更一致的 `section + header + content` 骨架。
  - 头部改为单行布局，聚合 `screen name / connected / screenId / self / hub`。
  - 下方直接进入 loading / missing / 展示内容区，不再保留额外头部间隔结构。

### 未修改
- 未新增 viewer 控制按钮。
- 未修改 Showcase store、路由、保存逻辑、widget 交互和布局模式逻辑。

## 对应 plan.md 任务映射
- `T1` 明确本轮 Showcase 头部对齐目标：完成。
- `T2` 完成实现方案设计：完成。
- `T3` 实现 Showcase 头部对齐与内容区收口：完成。
- `T4` 进行 Code Review 与归档：完成。

## 关键设计决策与权衡
- 采用“单行 header + 直接内容区”的结构，而不是继续保留双层 header 只压缩间距。
  - 原因：用户明确要求上方尽量只有一行名称、状态和按钮，下面直接进入编辑或展示内容。
  - 收益：与 Flow 更接近，首屏结构更直接。
- Viewer 不额外发明按钮，只对齐结构。
  - 原因：viewer 当前没有对应控制逻辑，硬加按钮只会扩大本轮范围。
  - 收益：保持最小变更，同时满足“没有的部分可以忽略”的要求。
- 删除 editor 内容区说明块，而不是保留提示文字。
  - 原因：这些说明不属于实际编辑内容，占用了垂直空间。
  - 收益：编辑区域更纯粹，窗口层级更清晰。

## 测试与验证方式 / 结果
- 静态文本检查：
  - 命令：`rg -n "currentLayoutSummary|Editing Surface|Showcase Viewer|Showcase Editor" frontend/src/pages/Showcase.vue frontend/src/windows/ShowcaseWindow.vue`
  - 结果：无命中，说明旧说明文案和已废弃计算项引用已移除。
- Vue SFC 解析：
  - 命令：使用 `@vue/compiler-sfc` 解析 `src/pages/Showcase.vue` 与 `src/windows/ShowcaseWindow.vue`
  - 结果：通过。
- 依赖准备：
  - 命令：`npm ci`
  - 结果：通过。
- 前端构建：
  - 命令：`npm run build`
  - 结果：失败。
  - 失败原因：现有 `src/pages/TopicBus.vue` 无法解析 `../../wailsjs/go/main/App`，属于当前 Wails 生成绑定缺失问题，与本次 Showcase 头部对齐改动无关。

## 潜在影响与回滚方案
- 潜在影响：
  - 头部信息被压到同一行后，在较窄窗口下会通过 `flex-wrap` 自动换行，信息密度更高。
  - Viewer 的整体视觉骨架会更接近 editor / flow，但运行交互不变。
- 回滚方案：
  - 回退 `frontend/src/pages/Showcase.vue`
  - 回退 `frontend/src/windows/ShowcaseWindow.vue`

## 子Agent执行轨迹
- 本轮未使用子Agent。
