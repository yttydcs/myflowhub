# Win 中文补齐与设置卡片保存 Workflow Plan

## 项目目标与当前状态

- 目标：
  - 修复 `MyFlowHub-Win` 中文界面中 `VarPool`、`TopicBus` 及相关界面的漏翻和英文回退问题
  - 将 `TopicBus` 统一翻译为“主题总线”，将 `VarPool` 统一翻译为“变量池”
  - 调整设置页保存交互，使每个可编辑卡片独立保存自己的配置
- 当前状态：
  - 独占分支已创建：`fix/win-translation-settings`
  - 独占 worktree 已创建：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-translation-settings`
  - 已完成需求分析与架构设计
  - 已完成静态走查，确认存在“翻译 key 缺失”和“后定义英文词条覆盖前定义中文词条”两类问题
  - 已确认交互决策：
    - `TopicBus` 统一为“主题总线”
    - 设置页采用“每个可编辑卡片独立保存各自内容”

## Workflow 信息

- 仓库：`MyFlowHub-Win`
- 分支：`fix/win-translation-settings`
- Base：`main`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-translation-settings`
- 当前阶段：`4 归档变更（已完成，等待用户确认是否结束 workflow）`
- 计划文档：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-translation-settings\plan.md`

## 文档治理与影响检查

- 使用 `$docs-governor` 的结论：
  - 当前仓 docs 树已存在且受治理，无需补齐结构
  - 当前 workflow 的执行计划落点为 worktree 根 `plan.md`
  - 当前 workflow 的完成归档落点为 `docs/change/2026-03-22_win-translation-settings.md`
  - `docs/design/`、`docs/plan_archive/` 为历史遗留目录，不作为本轮新增文档落点
- Requirements impact：`none`
- Specs impact：`none`
- Related requirements：`none（当前仓无与“Win 中文术语与设置交互细节”对应的受治理 requirements 叶子文档）`
- Related specs：`none（本轮不修改接口、协议、数据结构与长期技术契约）`
- 变更归档路径：`docs/change/2026-03-22_win-translation-settings.md`
- Lessons 路径：`none（阶段 4 已复核：不新增 lessons 文档）`

## 阶段 1 需求分析摘要

- 目标：
  - 中文界面下不再出现 `VarPool` 未翻译为“变量池”的情况
  - 中文界面下 `TopicBus` 统一显示为“主题总线”
  - 静态走查并补齐当前实际使用但未翻译的 key
  - 设置页每个可编辑卡片都有独立保存入口，且仅保存各自负责的配置
- 范围：
  - 必须做：
    - 修正 `frontend/src/i18n/messages/**` 中的缺失和覆盖词条
    - 走查 `frontend/src/pages/**`、`frontend/src/windows/**`、`frontend/src/components/**` 中的 `t("...")` 调用
    - 修正 `frontend/src/pages/Settings.vue` 中应用设置、语言设置、主题总线设置的保存交互
  - 可选：
    - 对已存在但风格不统一的 `VarPool` / `TopicBus` 复合短语做同轮统一
  - 不做：
    - 后端接口、协议字段、服务名、Go 绑定名重命名
    - 超出设置页的通用表单保存框架重构
- 使用场景：
  - 用户在中文环境浏览导航、变量池、主题总线、主题总线窗口和设置页时，看到稳定中文文案
  - 用户在设置页修改连接默认值、界面偏好、语言、主题总线配置时，能在对应卡片内完成保存
- 输入输出：
  - 输入：前端 i18n 词条、设置页草稿状态、TopicBus/VarPool store 的现有保存 API
  - 输出：补齐后的中文文案、独立卡片保存按钮和对应保存逻辑
- 边界异常：
  - 同一 key 在不同 message 文件中重复定义，后定义值会覆盖前定义值
  - 部分 key 只在窗口文件或组件文件中使用，肉眼不易覆盖完全
  - 设置页语言属于全局偏好，应用设置属于 profile 级，主题总线偏好属于功能级，不能混存
- 验收标准：
  - `zh-CN` 下不再看到 `VarPool` 英文作为用户界面文案
  - `zh-CN` 下 `TopicBus` 统一显示为“主题总线”
  - 本轮走查到的漏翻 key 全部补齐
  - 设置页三个可编辑区域具备独立保存入口并只保存各自配置
  - 静态检查无新增未翻译 key 回退
- 风险：
  - 重复 key 如果只改一个文件，可能继续被其它 message 文件覆盖
  - 设置页拆分保存时，如果错误复用原总保存状态，可能导致按钮禁用条件或回滚行为异常

## 阶段 2 架构设计摘要

- 总体方案：
  - 继续沿用现有 `t(key)` + 分模块 message 文件的 i18n 结构，不引入新的翻译框架
  - 在词条层面做两类修复：
    - 补齐缺失 key
    - 对重复 key 统一最终落值，消除英文覆盖
  - 在设置页交互层面按数据归属拆分保存动作，而不是维持一个跨卡片总保存
- 选型理由：
  - 当前 i18n 机制已覆盖大多数页面，问题集中在词条治理，不需要额外抽象层
  - 设置页现有 store 已按应用设置、语言、主题总线偏好分离，独立保存可以复用现有 API，改动最小且边界清晰
- 备选方案与不采用原因：
  - 方案 A：新增 i18n key 冲突检测器并重构 message 聚合顺序
    - 不采用原因：本轮目标是修复用户可见问题，先统一词条值即可，避免扩大范围
  - 方案 B：设置页保留总保存按钮，只在每张卡片复制一个“总保存”
    - 不采用原因：与“独立保存各自内容”的确认要求不一致，且会让语言/应用/主题总线互相耦合
- 模块职责：
  - `frontend/src/i18n/messages/**`
    - 维护最终展示词条
    - 负责确保重复 key 的最终中文落值正确
  - `frontend/src/pages/Settings.vue`
    - 负责各卡片草稿、脏状态、按钮启用条件与保存动作编排
  - `frontend/src/stores/appSettings.ts`
    - 保存 profile 级应用设置
  - `frontend/src/stores/language.ts`
    - 保存全局语言偏好
  - `frontend/src/stores/topicbus.ts`
    - 保存主题总线偏好并管理缓存事件
- 数据 / 调用流：
  - 应用设置卡片：
    - UI draft -> `appSettings.save()` -> store state -> `syncDraft()`
  - 语言卡片：
    - `languageDraft` -> `languageStore.save()` -> locale state -> `syncDraft()`
  - 主题总线卡片：
    - 输入框草稿 -> `topicbus.savePrefs()` -> store state -> `syncTopicBusDraft()`
  - 文案解析：
    - `t(key)` -> `messages.zh-CN[key]` -> 若缺失则回退 `en` 或原 key
- 接口草案：
  - 不新增后端接口
  - 前端页面新增三个独立动作：
    - `saveAppSettingsCard()`
    - `saveLanguageCard()`
    - `applyTopicBusSettings()`（保留并改为主题总线卡片专属）
- 错误与安全：
  - 保存前保持现有输入校验：
    - 主题总线目标节点 ID、最大事件数必须为正数
  - 保存失败后重新从 store / 后端加载对应域数据，避免草稿与真实状态漂移
  - 不改变任何权限、认证与后端调用边界
- 性能与测试策略：
  - 词条走查使用静态扫描，避免手工逐页查找遗漏
  - 保存逻辑按域拆分，只刷新对应 store，避免一次保存触发不必要 I/O
  - 验证包括：
    - 缺失 key 扫描
    - 前端构建 / SFC 解析
    - 设置页交互静态检查
- 可扩展性设计点：
  - 设置页卡片级保存逻辑按配置域拆开，后续新增卡片时可复用相同模式
  - 文案统一集中在 message 文件，后续若引入术语表可直接迁移

## 可执行任务清单

- [x] `T1` 修复中文词条与术语统一
- [x] `T2` 补齐静态走查发现的漏翻 key
- [x] `T3` 重构设置页为卡片级独立保存
- [x] `T4` 验证与补充走查
- [x] `T5` 强制 Code Review
- [x] `T6` 归档变更并更新 docs/change 索引

## 任务详情

### `T1` 修复中文词条与术语统一

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-translation-settings`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-translation-settings\plan.md`
- 目标：
  - 将 `VarPool` 最终统一为“变量池”
  - 将 `TopicBus` 最终统一为“主题总线”
  - 修复重复 key 覆盖导致的英文回退
- 涉及模块 / 文件：
  - `frontend/src/i18n/messages/shell.ts`
  - `frontend/src/i18n/messages/operations.ts`
  - `frontend/src/i18n/messages/showcase.ts`
  - `frontend/src/i18n/messages/signals.ts`
  - 其他命中的 message 文件
- Write set：
  - `frontend/src/i18n/messages/**`
- 关键上下文引用：
  - `frontend/src/i18n/messages/index.ts`
  - `frontend/src/layout/AppShell.vue`
  - `frontend/src/router/index.ts`
- 依赖：无
- 验收条件：
  - 重复 key 最终落值为中文
  - 用户可见术语风格一致
- 测试点：
  - 扫描重复 key
  - 检查 `zh-CN` 词条中 `VarPool` / `TopicBus` 的最终定义
- 回滚点：
  - 回退对应 message 文件改动
- 风险与注意事项：
  - 不能误改协议名、方法名等本应保留英文的技术标识

### `T2` 补齐静态走查发现的漏翻 key

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-translation-settings`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-translation-settings\plan.md`
- 目标：
  - 补齐本轮静态走查发现的缺失 key
  - 复跑扫描，确认无新增确定性漏翻
- 涉及模块 / 文件：
  - `frontend/src/i18n/messages/signals.ts`
  - `frontend/src/i18n/messages/file.ts`
  - 其他新增命中文件
- Write set：
  - `frontend/src/i18n/messages/**`
- 关键上下文引用：
  - `frontend/src/windows/TopicBusWindow.vue`
  - `frontend/src/components/file/OfferNodeTreePicker.vue`
  - `frontend/src/pages/Settings.vue`
- 依赖：
  - `T1`
- 验收条件：
  - `Cache`、`Events`、`Target`、`root`、`topic.status` 等 key 具备中文值
  - `TopicBus cached events cleared.` 等设置页提示语具备中文值
- 测试点：
  - 缺失 key 扫描脚本复跑结果为 0 或仅剩技术常量
- 回滚点：
  - 回退新增 message 词条
- 风险与注意事项：
  - 扫描脚本只覆盖字面量 `t("...")`，模板字符串或动态 key 不纳入本轮保证

### `T3` 重构设置页为卡片级独立保存

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-translation-settings`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-translation-settings\plan.md`
- 目标：
  - 应用设置卡片独立保存 profile 级设置
  - 语言卡片独立保存全局语言偏好
  - 主题总线卡片独立保存主题总线偏好
- 涉及模块 / 文件：
  - `frontend/src/pages/Settings.vue`
  - 如有必要：`frontend/src/stores/appSettings.ts`
  - 如有必要：`frontend/src/stores/language.ts`
  - 如有必要：`frontend/src/stores/topicbus.ts`
- Write set：
  - `frontend/src/pages/Settings.vue`
  - 必要时允许修改上述三个 store 文件
- 关键上下文引用：
  - `frontend/src/pages/Settings.vue`
  - `frontend/src/stores/appSettings.ts`
  - `frontend/src/stores/language.ts`
  - `frontend/src/stores/topicbus.ts`
- 依赖：
  - `T1`
  - `T2`
- 验收条件：
  - 每个可编辑卡片有自己的保存按钮
  - 各自按钮只触发对应配置域保存
  - 保存失败时只回滚/重载对应配置域
  - about 卡片保持只读，不新增保存按钮
- 测试点：
  - 独立脏状态计算
  - 按钮禁用条件正确
  - 保存函数不交叉调用其它域
- 回滚点：
  - 回退 `Settings.vue` 及相关 store 改动
- 风险与注意事项：
  - 语言卡片保存后会立即切换语言，按钮文案需保持稳定
  - 不能因为拆分保存而丢失原“恢复默认值”的能力边界

### `T4` 验证与补充走查

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-translation-settings`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-translation-settings\plan.md`
- 目标：
  - 执行静态扫描、前端构建或至少 SFC 解析验证
  - 走查修改后的术语是否仍有遗漏
- 涉及模块 / 文件：
  - `frontend/src/**`
- Write set：
  - 无新增实现写集；如发现漏项，可回到 `T1` / `T2` / `T3`
- 依赖：
  - `T1`
  - `T2`
  - `T3`
- 验收条件：
  - 缺失 key 扫描结果符合预期
  - 改动文件可通过 Vue SFC 解析
  - 若构建失败，能明确区分是本轮问题还是既有环境问题
- 测试点：
  - 扫描脚本
  - `npm run build`
  - 必要时 `@vue/compiler-sfc` 解析
- 回滚点：
  - 按问题回退对应任务改动
- 风险与注意事项：
  - 既有 `wailsjs` 生成物问题如果出现，需要在 Review 和 Change 中单独记录

### `T5` 强制 Code Review

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-translation-settings`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-translation-settings\plan.md`
- 目标：
  - 按流程输出需求覆盖、架构、性能、可读性、扩展性、安全、测试和子Agent治理结论
- 涉及模块 / 文件：
  - 本轮全部改动文件
- Write set：
  - 无实现写集；如 Review 不通过，返回 `T1` / `T2` / `T3`
- 依赖：
  - `T4`
- 验收条件：
  - Review 全项给出通过/不通过
  - 若不通过，明确返回任务和修正点
- 测试点：
  - Review 记录完整
- 回滚点：
  - 不适用
- 风险与注意事项：
  - 本轮若不使用子Agent，也必须记录“不使用原因”

### `T6` 归档变更并更新 docs/change 索引

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-translation-settings`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-translation-settings\plan.md`
- 目标：
  - 完成 `docs/change/2026-03-22_win-translation-settings.md`
  - 按 `$docs-governor` 规则复核 requirement/spec impact、lessons 和索引更新
- 涉及模块 / 文件：
  - `docs/change/2026-03-22_win-translation-settings.md`
  - `docs/change/README.md`
  - 必要时：`docs/README.md`
- Write set：
  - 上述 docs 文件
- 依赖：
  - `T5`
- 验收条件：
  - change 文档完整记录任务映射、设计权衡、测试结果、回滚方案
  - 索引可导航到本次归档
- 测试点：
  - 人工核对文档结构与链接
- 回滚点：
  - 删除本次 change 文档并回退索引改动
- 风险与注意事项：
  - 阶段 4 前必须再次显式使用 `$docs-governor`

## 并行性评估

- 结论：当前计划默认不使用子Agent
- 原因：
  - `T1`、`T2`、`T3` 都会集中写入 `frontend/src/i18n/messages/**` 与 `frontend/src/pages/Settings.vue`，写集高度重叠
  - 本轮任务规模较小，主Agent直接完成更稳妥
  - 进入 `3.2` 时会再执行一次并行性评估并记录最终结论

## 回归与验证步骤

1. 运行静态扫描脚本，确认缺失翻译 key 列表
2. 完成改动后复跑扫描脚本，检查剩余漏翻
3. 在 `frontend/` 目录执行构建验证
4. 必要时使用 `@vue/compiler-sfc` 解析改动过的 Vue 文件
5. 完成强制 Code Review 后进入阶段 `4`

## 执行结果

- `T1`
  - 已统一 `VarPool` / `TopicBus` 的用户可见中文术语，并消除了被后定义英文值覆盖的实际命中项
- `T2`
  - 已补齐字面量 `t("...")` 缺失 key 与 route meta 动态标题 / 副标题缺失翻译
  - 同 key 同 value 的剩余英文项仅保留技术常量、协议名和示例值
- `T3`
  - `Settings.vue` 已拆分为连接设置、界面设置、语言设置、主题总线设置四个独立保存域
  - “恢复默认值”已移动到页面摘要区，避免附着在单一卡片下造成语义混淆
- `T4`
  - 字面量缺失 key 扫描：`TOTAL_MISSING=0`
  - route meta 扫描：`TOTAL_ROUTE_MISSING=0`
  - `npm ci`：通过
  - `npm run build`：失败，原因是既有 `wailsjs/go/session/SessionService` 生成绑定缺失，与本轮改动文件无直接关联
  - 局部语法解析：通过
- `T5`
  - Code Review 已通过，无阻塞性问题
- `T6`
  - 已新增 `docs/change/2026-03-22_win-translation-settings.md`
  - 已更新 `docs/change/README.md`

## Code Review 结论

- 需求覆盖：通过
- 架构合理性：通过
- 性能风险：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖情况：通过
- 子Agent治理与审计：通过（本轮未使用子Agent）
- 结论：
  - Review 通过，无需返回 `3.2`
  - 残余风险：完整前端构建仍受既有 `wailsjs` 生成绑定缺失阻塞；i18n 重复 key 仍缺少自动化冲突检测
