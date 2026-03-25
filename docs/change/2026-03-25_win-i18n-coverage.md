# 2026-03-25_win-i18n-coverage

## 变更背景 / 目标
- 背景：
  - `2026-03-22_win-settings-i18n` 已完成 Win 前端首批 i18n 基础接入，但最近新增的 `Flow Editor`、`Showcase` 和 `Permissions` 相关功能仍有漏接词条与 raw 英文错误。
  - 本轮 quote-aware 静态扫描前，实际缺失 key 集中在 8 个文件，缺口主要位于 `FlowNodeInspector`、`FlowFieldBindingDialog`、`flow.ts`、`Showcase` 新模式和 `Permissions` 校验路径。
  - 少量 store 仍直接抛出英文 `Error.message`，会通过 `toast.errorOf(...)` 原样进入 UI。
- 目标：
  - 补齐 Win 前端剩余高频路径 i18n 缺口。
  - 把会直接进入 UI 的 store 错误统一接入 `t(...)`。
  - 保持现有行为和技术契约不变。

## 具体变更内容

### 修改
- `frontend/src/i18n/messages/automation.ts`
  - 补齐 `Flow Editor / FlowNode / flow.ts / flow_json_pointer.ts` 相关缺失词条，共覆盖字段绑定、普通模式、compose 节点、JSON Pointer 校验和 DAG/advanced spec 校验文案。
- `frontend/src/i18n/messages/showcase.ts`
  - 补齐 `metric / badge / progress` 模式、变量来源提示、数值区间错误与空态提示词条。
- `frontend/src/i18n/messages/operations.ts`
  - 补齐 `Permissions` store 和页面校验的缺失词条，包括 authority override、save stage fallback 和 Permission binding 文案。
- `frontend/src/stores/permissions.ts`
  - 直接抛给 UI 的错误改为 `t(...)`，避免 toast 出现英文 fallback。
  - `SavePolicy` 的 fallback stage 错误改为 i18n key，而不是内联英文模板字符串。
- `frontend/src/stores/flow_json_pointer.ts`
  - JSON Pointer 解析错误改为 `t(...)`。
- `frontend/src/stores/appSettings.ts`
  - `App` binding 缺失错误改为 `t(...)`。
- `frontend/src/stores/devices.ts`
  - `Management` binding 缺失错误改为 `t(...)`。
- `frontend/src/stores/flowProjects.ts`
  - `App / Flow` binding 缺失错误改为 `t(...)`。
- `frontend/src/stores/language.ts`
  - `App` binding 缺失错误改为 `t(...)`。

### 删除
- 无功能删除。

## Requirements impact
- none

## Specs impact
- none

## Lessons impact
- none
- 理由：
  - 本轮属于既有 Win 前端的 i18n 收口和错误文案治理，没有形成需要长期检索的事故级规则。
  - 扫描脚本对带引号 key 的解析注意事项已记录在本变更的排查线索中，无需单独抽取 lesson。

## Related requirements
- `docs/requirements/flow-editor-visual-form.md`
- `docs/requirements/showcase-display-widgets.md`

## Related specs
- `docs/specs/flow-editor-visual-form.md`
- `docs/specs/showcase-display-widgets.md`

## Related lessons
- none

## 对应 plan.md 任务映射
- `T1`
  - `frontend/src/i18n/messages/automation.ts`
  - `frontend/src/stores/flow_json_pointer.ts`
- `T2`
  - `frontend/src/i18n/messages/showcase.ts`
- `T3`
  - `frontend/src/i18n/messages/operations.ts`
  - `frontend/src/stores/permissions.ts`
  - `frontend/src/stores/appSettings.ts`
  - `frontend/src/stores/devices.ts`
  - `frontend/src/stores/flowProjects.ts`
  - `frontend/src/stores/language.ts`
- `T4`
  - quote-aware 缺失 key 扫描
  - `git diff --check`
  - `frontend/npm run build`
  - 变更文件 TypeScript 局部转译
  - `3.3 Code Review`
- `T5`
  - `docs/change/2026-03-25_win-i18n-coverage.md`
  - `docs/change/README.md`
  - `plan.md`

## 经验 / 教训摘要
- 新功能即使页面层已经使用 `t(...)`，如果消息表没有同步扩展，仍会在中文环境里直接暴露英文或 key。
- store 里抛出的 raw 英文 `Error.message` 不能靠页面层兜底；只要 `toast.errorOf(err, ...)` 使用 `err.message`，就必须在抛错点本地化。
- 缺失 key 静态扫描如果使用不带引用回溯的正则，会对包含引号的 key 产生假阳性。

## 可复用排查线索
- 症状：
  - 中文界面下 `Flow Editor / Showcase / Permissions` 出现英文按钮、空态提示、校验错误，或直接显示原始 key。
- 触发条件：
  - 在首批 i18n 改造后新增 UI/校验逻辑，但没有同步补 `frontend/src/i18n/messages/**`。
  - store 继续抛 raw 英文错误字符串。
- 关键词：
  - `TOTAL_MISSING`
  - `Permission binding`
  - `JSON Pointer contains an invalid escape sequence`
  - `Visual form unavailable`
  - `No numeric value yet`
- 快速检查：
  - 用 quote-aware 正则扫描 `t(...)` key 是否全部命中消息表。
  - grep `frontend/src/stores/**` 中的 raw `throw new Error(...)` 英文模板。
  - 核对新功能所在域的 message 文件是否同步扩展。

## 关键设计决策与权衡
- 决策：继续沿用现有轻量 i18n 架构，只补消息表和必要调用点。
  - 原因：问题是覆盖缺口，不是框架能力不足。
  - 权衡：不引入新抽象，改动面最小，但后续新增功能仍需保持 message 文件同步维护。

- 决策：把 store 错误在抛错点接到 `t(...)`，而不是留给页面层统一翻译。
  - 原因：页面层通常只能拿到 `err.message`，无法可靠推导原始语义。
  - 权衡：store 会多依赖一个 i18n 运行时入口，但可保证 toast / dialog 路径一致本地化。

- 决策：保留纯技术示例和协议常量原值。
  - 原因：如 JSON Pointer 示例、`flow_id / run_id`、高级参数示例属于实际输入格式，强行翻译会误导用户输入。
  - 权衡：这些文本不一定全部走 `t(...)`，但它们是有意保留的技术字面量，不属于自然语言漏翻。

## 测试与验证方式 / 结果
- quote-aware 缺失 key 扫描
  - 结果：通过
  - 结论：`TOTAL_MISSING=0`
- `git diff --check`
  - 结果：通过
  - 备注：仅有 LF/CRLF warning，无空白或补丁格式问题
- 变更文件 TypeScript 局部转译
  - 结果：通过
  - 结论：`TS_TRANSPILE=PASS`
- `frontend/npm run build`
  - 结果：失败
  - 失败原因：仓库既有 `wailsjs` 生成绑定缺失，`src/windows/TopicBusWindow.vue` 无法解析 `../../wailsjs/go/main/App`
  - 判断：失败点与本轮改动文件无直接关联

## 潜在影响
- 中文环境下，`Flow Editor / Showcase / Permissions` 的界面文案、空态和校验错误将更完整，英文 fallback 显著减少。
- `App / Management / Flow / Permission` 等 binding 缺失时，toast 或异常提示将走本地化文案。
- 技术示例和协议常量仍保留原值，不会影响现有输入格式。

## 回滚方案
- 回退消息表：
  - `frontend/src/i18n/messages/automation.ts`
  - `frontend/src/i18n/messages/showcase.ts`
  - `frontend/src/i18n/messages/operations.ts`
- 回退错误接线：
  - `frontend/src/stores/permissions.ts`
  - `frontend/src/stores/flow_json_pointer.ts`
  - `frontend/src/stores/appSettings.ts`
  - `frontend/src/stores/devices.ts`
  - `frontend/src/stores/flowProjects.ts`
  - `frontend/src/stores/language.ts`
- 文档回滚：
  - 删除 `docs/change/2026-03-25_win-i18n-coverage.md`
  - 回退 `docs/change/README.md`
  - 回退 `plan.md`

## 子Agent执行轨迹
- 本轮未使用子Agent。
