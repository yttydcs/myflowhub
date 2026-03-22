# Win Showcase 窗口标题占位精简

## 项目目标与当前状态
- 目标：进一步压缩 Showcase 独立 Viewer / Editor 窗口顶部占位，移除 `Showcase Viewer` 与 `Showcase Editor` 这两个额外标题。
- 当前状态：Showcase 已有独立编辑窗口与独立 Viewer，编辑窗口近期已完成 layout 弹窗化和 full-bleed 收敛，但两个窗口顶部仍保留模块标题文字。

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`refactor/showcase-window-title-trim`
- Base：`main`
- Worktree：`D:\\project\\MyFlowHub3\\worktrees\\MyFlowHub-Win-showcase-window-title-trim`
- 当前阶段：`4 归档变更（待用户确认是否结束 workflow）`

## 需求分析结论（阶段 1）
- 目标：移除 `Showcase Editor` 与 `Showcase Viewer` 顶部标题，进一步压缩窗口头部占位。
- 必须范围：
  - `frontend/src/pages/Showcase.vue` 移除编辑窗口头部标题。
  - `frontend/src/windows/ShowcaseWindow.vue` 移除 viewer 窗口头部标题。
  - 收掉标题删除后残留的顶部间距。
- 可选范围：无。
- 不做：
  - 不调整 screen name、连接状态、layout badge、时间或其它元信息。
  - 不修改 store、路由、窗口打开逻辑、数据结构和保存逻辑。
- 使用场景：用户打开 Showcase Editor / Viewer 时，界面首屏更聚焦当前 screen 与内容区，而不是重复模块名。
- 功能需求：两个窗口不再渲染模块标题；其他头部信息保持可见。
- 非功能需求：变更最小化、不增加滚动、不破坏现有交互与布局自适应。
- 输入输出：
  - 输入：现有 Showcase Editor / Viewer Vue 模板。
  - 输出：更紧凑的头部结构。
- 边界异常：加载中、screen 缺失、断连等分支保持现状。
- 验收标准：
  - 代码中不再存在这两个标题的渲染节点。
  - 标题相关额外占位消失。
  - 其它头部元素与主功能不回归。
- 风险：
  - 若仅删文本不处理相邻 `mt-*`，会留下空白占位。
- 问题清单：无。
- 阻塞：否。

## 架构设计结论（阶段 2）
- 总体方案：仅在模板层删除静态标题节点，并同步压缩相邻标题间距，不调整脚本逻辑。
- 选型理由：目标文本为纯静态文案，没有状态依赖；模板层改动最小、最稳妥、回滚成本最低。
- 模块职责：
  - `Showcase.vue` 继续负责独立编辑窗口 UI。
  - `ShowcaseWindow.vue` 继续负责独立运行展示 UI。
- 数据 / 调用流：保持不变，仍由现有 `showcase` store 提供 screen 与连接状态。
- 接口草案：无新增接口。
- 错误与安全：无新增错误路径；现有加载失败和缺失态保持不变。
- 性能与测试策略：
  - 运行时零新增开销。
  - 通过文本搜索与静态构建校验确认未引入模板错误。
- 可扩展性设计点：改动局部化，后续若继续压缩头部，可在同一区域继续演进而不触及业务逻辑。
- 备选方案对比：
  - CSS 隐藏：实现快，但保留无意义节点与样式耦合，不采用。
  - 直接删除模板节点：结构最干净，采用。
- 问题清单：无。
- 阻塞：否。

## 可执行任务清单
- [x] T1 明确本轮标题收口需求边界
  - 目标：确认要移除的标题文本、保留项与不做项。
  - 涉及模块 / 文件：`plan.md`、`frontend/src/pages/Showcase.vue`、`frontend/src/windows/ShowcaseWindow.vue`
  - 验收条件：需求、边界、验收标准明确，无阻塞问题。
  - 测试点：N/A
  - 回滚点：修改 `plan.md`
  - 依赖关系：无

- [x] T2 完成架构设计与实现方案
  - 目标：确定仅通过模板层移除标题占位，尽量不影响现有按钮、状态和布局密度。
  - 涉及模块 / 文件：`plan.md`、`frontend/src/pages/Showcase.vue`、`frontend/src/windows/ShowcaseWindow.vue`
  - 验收条件：模块职责、布局影响、验证策略明确。
  - 测试点：N/A
  - 回滚点：修改 `plan.md`
  - 依赖关系：T1

- [x] T3 实现窗口标题占位精简
  - 目标：移除 Editor / Viewer 窗口顶部 `Showcase Editor` / `Showcase Viewer` 模块标题文字。
  - 涉及模块 / 文件：`frontend/src/pages/Showcase.vue`、`frontend/src/windows/ShowcaseWindow.vue`
  - 验收条件：目标文字不再渲染，窗口顶部信息更紧凑，现有控件与状态仍可见。
  - 测试点：Vue SFC 语法解析、关键文本搜索。
  - 回滚点：回退上述两个文件。
  - 依赖关系：T2

- [x] T4 进行 Code Review 与归档
  - 目标：输出 review 结论并归档本轮变更。
  - 涉及模块 / 文件：`plan.md`、`docs/change/*`
  - 验收条件：review 通过，生成归档文档。
  - 测试点：实现与文档一致性。
  - 回滚点：删除新增文档并回退代码。
  - 依赖关系：T3

## 并行与所有权
- 当前判断：本轮仅有一个极小的模板收口任务，验收条件统一、改动文件极少，拆分子 Agent 不会带来有效收益，继续由主 Agent 串行完成。

## Code Review（阶段 3.3）
- 需求覆盖：通过。
  - `frontend/src/pages/Showcase.vue` 已移除 `Showcase Editor` 标题并同步去除标题后的顶部间距。
  - `frontend/src/windows/ShowcaseWindow.vue` 已移除 `Showcase Viewer` 标题并同步去除标题后的顶部间距。
- 架构合理性：通过。
  - 改动仅落在模板层，没有影响 store、路由、数据流与窗口打开逻辑。
- 性能风险：通过。
  - 仅减少静态 DOM，无新增计算、I/O、订阅、渲染负担。
- 可读性与一致性：通过。
  - 头部结构更直接，没有保留无意义的隐藏标题或样式残留。
- 可扩展性与配置化：通过。
  - 头部继续保持局部化结构，后续若继续精简，可在同一区域演进。
- 稳定性与安全：通过。
  - 未改输入校验、保存逻辑、运行时权限或外部调用。
- 测试覆盖情况：有条件通过。
  - `rg -n "Showcase Editor|Showcase Viewer" frontend/src/pages/Showcase.vue frontend/src/windows/ShowcaseWindow.vue` 无命中，确认目标模板文案已清除。
  - `npm ci` 成功，验证环境依赖已补齐。
  - `npm run build` 未通过，但失败原因是既有的 `src/pages/Home.vue` 解析 `../../wailsjs/go/session/SessionService` 失败，与本次改动文件无关。
- 子Agent治理与审计：通过。
  - 本轮未使用子Agent；原因是任务单一、写集极小、无需并行拆分。
