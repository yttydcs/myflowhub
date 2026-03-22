# 2026-03-22_win-translation-settings

## 变更背景 / 目标
- 背景：
  - Win 端中文界面中，`VarPool` / `TopicBus` 相关页面、窗口和提示语存在英文回退。
  - 当前 i18n 消息表存在重复 key，后定义英文值会覆盖前定义中文值。
  - 设置页此前只有跨卡片总保存按钮，导致部分可编辑卡片缺少独立保存入口。
- 目标：
  - 将 `VarPool` 统一显示为“变量池”，将 `TopicBus` 统一显示为“主题总线”。
  - 走查并补齐当前实际使用但未翻译的 key，包括动态 route meta 标题 / 副标题。
  - 将设置页调整为卡片级独立保存，确保每个可编辑卡片只保存自己的配置域。

## 文档治理结论
- Requirements impact：`none`
- Specs impact：`none`
- Related requirements：`none`
- Related specs：`none`
- Lessons impact：`none`
  - 结论理由：本轮属于既有 Win 前端的常规 i18n 与交互治理，未形成新的事故复盘或团队级故障排查规则；重复 key 风险已在本变更中记录。
- Index update：
  - 已更新 `docs/change/README.md`
  - 已更新 `docs/plan/README.md`
  - 已更新根级 `plan.md`
  - 无需更新 `docs/README.md`

## 具体变更内容（新增 / 修改 / 删除）

### 新增
- `docs/change/2026-03-22_win-translation-settings.md`
  - 记录本轮中文补齐、设置页独立保存与验证结论
- `docs/plan/plan_archive_2026-03-22_win-translation-settings.md`
  - 归档本轮 workflow 的完整计划正文、任务拆分与 Review 结果

### 修改
- `frontend/src/i18n/messages/shell.ts`
  - 补齐 route meta 标题 / 副标题缺失翻译
  - 统一 `VarPool` 相关错误提示为“变量池”
- `frontend/src/i18n/messages/operations.ts`
  - 修正 `TopicBus` / `VarPool` 用户可见术语
  - 补齐 `Flow ID` / `Run ID` / `string` 等低风险界面词条
- `frontend/src/i18n/messages/settings.ts`
  - 新增设置页独立保存按钮、toast 与说明文案
  - 补齐“主题总线缓存事件已清空”等设置页缺失 key
- `frontend/src/i18n/messages/signals.ts`
  - 统一主题总线 / 变量池术语
  - 补齐 `Target` / `Events` / `Cache` / `topic.status` 等窗口与状态 key
  - 将 `ready` 翻译为“就绪”
- `frontend/src/i18n/messages/stores.ts`
  - 统一主题总线窗口、变量池、topic 输入提示等用户可见文案
- `frontend/src/i18n/messages/showcase.ts`
  - 统一 `TopicBus` / `VarPool` 相关词条
- `frontend/src/i18n/messages/file.ts`
  - 补齐 `root` 的中文显示
- `frontend/src/pages/Settings.vue`
  - 将应用设置草稿拆分为“连接与身份”和“界面节奏”两个独立配置域
  - 保留语言和主题总线各自独立草稿
  - 为四张可编辑卡片增加独立保存按钮
  - 将全局“恢复默认值”上移到页面摘要区，避免挂在单一卡片下造成语义混淆
- `plan.md`
  - 更新当前 workflow 的执行状态、验证结果和 Code Review 结论

### 删除
- 无功能删除

## 对应 `plan.md` 任务映射
- `T1`
  - `frontend/src/i18n/messages/shell.ts`
  - `frontend/src/i18n/messages/operations.ts`
  - `frontend/src/i18n/messages/showcase.ts`
  - `frontend/src/i18n/messages/signals.ts`
  - `frontend/src/i18n/messages/stores.ts`
- `T2`
  - `frontend/src/i18n/messages/settings.ts`
  - `frontend/src/i18n/messages/file.ts`
  - route meta 动态标题 / 副标题补齐
- `T3`
  - `frontend/src/pages/Settings.vue`
- `T4`
  - 缺失 key 扫描
  - route meta 翻译扫描
  - `npm ci`
  - `npm run build`
  - `@vue/compiler-sfc` + TypeScript 局部语法解析
- `T5`
  - `3.3 Code Review`
- `T6`
  - `docs/change/2026-03-22_win-translation-settings.md`
  - `docs/change/README.md`
  - `docs/plan/plan_archive_2026-03-22_win-translation-settings.md`
  - `docs/plan/README.md`
  - 根级 `plan.md`

## 关键设计决策与权衡（尤其性能 / 扩展性）
- 决策：保留现有 message 聚合结构，只修正最终落值与缺失 key
  - 原因：本轮目标是快速收敛用户可见英文残留，不扩展到 i18n 框架或合并顺序重构。
  - 权衡：重复 key 的结构性风险仍在，但这次已经把实际覆盖到的用户可见值统一为中文。

- 决策：设置页按配置域拆分草稿和保存动作，而不是继续使用共享 `draft`
  - 原因：如果继续共用一个草稿对象，保存某张卡片时会顺带提交另一张卡片的未保存改动，不满足“卡片独立保存各自内容”。
  - 权衡：页面本地状态更细，但保存边界清晰、回滚范围更小，也避免不必要的后端 I/O。

- 决策：保留协议标识与示例值的英文原文
  - 原因：`topic.status`、`status.flag`、`flow_id`、`ConfigGetSimple` 等属于实际 topic、字段或方法标识，翻译会误导输入与排障。
  - 权衡：中文包里仍保留少量技术英文，但它们是有意保留的技术常量，不是界面漏翻。

## 测试与验证方式 / 结果
- 字面量 `t("...")` 缺失 key 扫描
  - 结果：通过
  - 结论：`TOTAL_MISSING=0`
- route meta 标题 / 副标题翻译扫描
  - 结果：通过
  - 结论：`TOTAL_ROUTE_MISSING=0`
- 同 key 同 value 英文词条扫描
  - 结果：通过
  - 结论：剩余 30 项均为协议标识、方法名、示例 topic 或技术常量，按设计保留英文
- `frontend/npm ci`
  - 结果：通过
- `frontend/npm run build`
  - 结果：失败
  - 失败原因：`src/pages/Home.vue` 依赖的 `../../wailsjs/go/session/SessionService` 无法解析，属于既有 `wailsjs` 生成绑定缺失，与本轮改动文件无直接关联
- 局部语法解析
  - 范围：全部改动过的 `frontend/src/i18n/messages/*.ts` 与 `frontend/src/pages/Settings.vue`
  - 结果：通过
  - 方法：`typescript.transpileModule` + `@vue/compiler-sfc`
- `git diff --check`
  - 结果：通过
  - 备注：仅有 LF/CRLF 提示，无语法或空白错误

## 3.3 Code Review 结论
- 需求覆盖：通过
  - `VarPool` / `TopicBus` 术语统一、漏翻补齐、设置页卡片独立保存均已覆盖
- 架构合理性：通过
  - i18n 仍按域拆分，设置页按配置域拆分本地草稿，依赖边界清晰
- 性能风险：通过
  - 卡片保存仅触发对应 store 保存；失败回滚仅刷新对应域，避免额外 I/O
- 可读性与一致性：通过
  - 主题总线 / 变量池术语在导航、页面、窗口、toast 中已统一
- 可扩展性与配置化：通过
  - 后续新增设置卡片可复用“独立草稿 + 独立 dirty + 独立保存”的模式
- 稳定性与安全：通过
  - 主题总线配置保存前仍保留正数校验；设置保存失败时会重新加载对应域状态
- 测试覆盖情况：通过
  - 静态翻译扫描和局部语法解析通过；完整前端构建受既有 `wailsjs` 生成绑定缺失阻塞
- 子Agent治理与审计：通过
  - 本轮未使用子Agent；原因是写集集中在同一套 i18n 消息表和单个 `Settings.vue` 页面，且未获得显式委派授权
- 结论：
  - Review 通过，无需回退到 `3.2`
  - 残余风险：若后续再新增重复 key，仍可能发生中文被覆盖；当前仓尚缺自动化冲突检测

## 潜在影响与回滚方案

### 潜在影响
- 中文环境下，变量池 / 主题总线相关页面、窗口与提示语会统一显示中文术语。
- 设置页保存边界从“跨卡片总保存”变为“按卡片独立保存”，用户原有未保存草稿不会再被其它卡片的保存动作连带提交。
- route meta 标题 / 副标题已补齐中文，因此页面头部和窗口标题的中文覆盖范围扩大。

### 回滚方案
- 以任务维度回滚：
  - 回退 `T1` / `T2`：回退 `frontend/src/i18n/messages/**` 对应改动
  - 回退 `T3`：回退 `frontend/src/pages/Settings.vue`
  - 回退 `T6`：删除本次 change 文档、plan 归档并回退各级索引
- 若仅需恢复旧设置交互，可只回退 `frontend/src/pages/Settings.vue`
- 若仅需恢复原英文术语，可只回退受影响的 message 文件

## 子Agent执行轨迹（Task ID → Agent → Worktree → 文件 → 验收结果）
- `T1` → `主Agent` → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-translation-settings` → `frontend/src/i18n/messages/shell.ts`、`operations.ts`、`showcase.ts`、`signals.ts`、`stores.ts` → 通过
- `T2` → `主Agent` → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-translation-settings` → `frontend/src/i18n/messages/settings.ts`、`file.ts`、动态 route meta 文案 → 通过
- `T3` → `主Agent` → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-translation-settings` → `frontend/src/pages/Settings.vue` → 通过
- `T4` → `主Agent` → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-translation-settings` → 静态扫描、`npm ci`、`npm run build`、局部语法解析 → 通过（完整构建受既有 `wailsjs` 绑定缺失阻塞）
- `T5` → `主Agent` → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-translation-settings` → 本轮全部改动文件 → 通过
- `T6` → `主Agent` → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-translation-settings` → `docs/change/2026-03-22_win-translation-settings.md`、`docs/change/README.md`、`docs/plan/plan_archive_2026-03-22_win-translation-settings.md`、`docs/plan/README.md`、根级 `plan.md` → 通过
