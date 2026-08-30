# Desktop Schema-driven Resource Widgets

## 变更背景 / 目标

Desktop 原先主要以 JSON textarea/pre 展示 Variable、Command、Stream、File 与第一方结构化资源，虽然通用，但无法表达数值范围、步长、字段类型、数组结构和资源语义。本次变更建立“提供方决定数据类型与约束、Desktop 决定兼容显示组件、用户选择具体显示方式”的生产级边界，让资源可以通过安全、可切换、可持久化的控件直接使用。

本 workflow 不包含品牌图标工作，没有设计、替换、生成或归档任何项目图标。

## 文档治理影响

- Docs root: `docs/`
- Intake impact: updated
- Feature impact: updated
- Requirements impact: updated
- Specs impact: added `desktop-schema-rendering.md`
- Decision impact: added provider schema / Desktop renderer ownership ADR
- Lessons impact: none; existing generated-contract、Wails binding 与 frontend preflight lessons 已覆盖本次可复用风险
- Related intake: [Desktop schema-driven Resource widgets](../intake/2026-08-30_desktop-schema-driven-resource-widgets.md)
- Related feature: [Desktop](../features/desktop.md)
- Related requirements: [Desktop resource workspace](../requirements/desktop-resource-workspace.md)
- Related spec: [Desktop schema rendering](../specs/desktop-schema-rendering.md)
- Related decision: [Provider schema and Desktop renderer ownership](../decisions/2026-08-30_provider-schema-desktop-renderer-ownership.md)
- Related lessons: [Observable side effects and generated contracts](../lessons/observable-side-effects-and-generated-contracts.md)、[Wails binding proto drift](../lessons/wails-binding-proto-drift.md)、[Frontend and PowerShell preflight](../lessons/frontend-and-powershell-preflight.md)

## 具体变更内容

- 在 `protocol` 增加 provider-owned、受限且确定性的第一方 data schema 定义；现有 Go payload `Validate` 仍是运行时最终权威。
- 扩展 bindings 生成链，输出 Desktop 使用的 schema artifact，并加入覆盖、排序、边界、确定性与 freshness 检查。
- 新增纯前端 `SchemaResolver` 与 `RendererRegistry`，按 schema ID、数据形状、能力和尺寸筛选/排序显示方式；未知或不兼容 schema 明确退回结构化/Raw JSON。
- Variable 支持 boolean、enum、number/integer、string、object、array 等结构化显示与编辑；数值控件遵守 provider 的 min/max/step，写入采用显式草稿与 Apply/Reset。
- Command、Topic publish 与通用 operation 由 schema 生成带标签的数字、文本、对象和可增删数组表单；只在显式 Execute 时调用 API，provider 错误保留并显示。
- Stream/Topic 使用 100 条有界、逐帧批处理缓冲，支持暂停、清空、筛选、自动滚动及 gap/expired/error 状态。
- File 上传接入 Wails 原生文件选择器与 in-flight/error 状态；当前 blocking `UploadFile` binding 不返回 transfer handle，因此没有伪造逐传输取消或字节进度，真实进度继续来自 `file/progress` / `file/transfers`。
- 第一方 catalog、topology、health、config、flow、audit、notification 与 file schema 获得 schema-ID-selected 结构化适配器。
- Workspace 增加键盘可访问的显示方式选择器；renderer ID 与 allowlisted presentation settings 按 View widget 保存。View schema 保持 v3，嵌套 split 结构和拖拽比例继续使用既有布局模型。

## 任务映射

| Task | 结果 |
| --- | --- |
| DOC01 | 完成 provider/display/user 所有权、feature、requirement、spec、ADR 与索引。 |
| SCHEMA01 | 完成第一方 declarative schema、生成 artifact、覆盖/parity/freshness gates。 |
| RENDER01 | 完成 resolver、registry、compatibility/ranking、legacy alias 和 fallback。 |
| VALUE01 | 完成常见标量/对象/数组显示与安全 staged Variable 控件。 |
| OP01 | 完成 schema-generated operation form、显式 Execute、结果/错误与 Raw JSON。 |
| LIVE01 | 完成有界 event UX、原生文件选择和第一方结构化适配器。 |
| VIEW01 | 完成显示方式选择、View 持久化、响应式密度和可访问性整合。 |
| QA01 | 完成全仓、generated、frontend、Wails production 和真实 GUI 验收。 |

## 关键设计决策与边界

1. Provider schema 只描述数据语义、结构和约束；Desktop renderer registry 只提供兼容显示；用户选择不能改变权限或有效值范围。
2. Phase 1 使用由 canonical protocol schema 生成的第一方 provider，没有修改 `SchemaDescriptorV2` wire，也没有引入 owner-served schema discovery。
3. Renderer 是本地可信实现，远程不可下发可执行 UI；未知数据只进入安全 fallback。
4. View 只保存 renderer ID 与显示设置，不保存资源值、草稿、事件、Permit、凭据、文件内容或操作 payload。
5. Presentation settings 使用显式 allowlist；本阶段 allowlist 为空，避免 renderer 配置成为数据泄漏通道。
6. 文件上传的 per-transfer cancel/byte progress 留在未来 transfer-handle binding，不通过前端假状态绕过协议边界。

## 测试与验证

- `GOWORK=off go test ./... -count=1`: 全仓 package 与 integration tests 通过。
- `GOWORK=off .\scripts\mfh.ps1 -Action check -Target generated`: generated output fresh，工作树无漂移。
- `npm test`: 12 个文件、75 个测试通过；包含 numeric bounds/step、registry/fallback、operation form、10k Resource tree 性能预算、View/layout 与 WAI-ARIA 行为。
- `npm run build`: TypeScript 与 Vite production build 通过。
- `wails build -clean -trimpath -platform windows/amd64 -o mfh-desktop.exe`: Windows production build 通过，产物约 11.9 MB。
- 真实 Wails：隔离 default-deny Hub `127.0.0.1:7443` 完成 one-use admission，加载 2 Nodes / 24 root resources；验证健康状态专用显示、Raw JSON 往返切换、`flow/create` 结构化表单、数组动态项、显式 invoke/错误、比例拖拽、View 保存、重启恢复及完整浅/深主题。
- View v3 实际保存 `mfh.structured.health.v1`、`mfh.command` 与 split weights `0.6484375 / 0.3515625`；重启恢复两组件与比例，未持久化操作草稿/错误。
- 安全检查：settings/View/state JSON 中没有 Permit、signature 或 private key；身份文件保持 Windows DPAPI 保护。
- 截图证据：[浅色主题与重启恢复](verification/2026-08-30_desktop-schema-widgets-light.png)、[深色主题与嵌套 operation 表单](verification/2026-08-30_desktop-schema-widgets-dark.png)。

## 性能、稳定性与安全复核

- Schema 解析与 registry 排序是纯函数，定义和 render tree 有显式深度、字段、数组与 payload 限制。
- Resource tree 的 10k catalog 用例在测试预算内；event buffer 固定 100，并通过 animation-frame batching 避免高频逐事件渲染。
- resize/docking 只更新本地布局与密度，不触发逐像素网络请求；真实拖拽后比例平滑写入 View。
- read-only 资源不出现写控件；敏感 write-only 字段使用 password input 且不回显；所有 runtime permission/revision/validator 仍为最终门禁。
- Forbidden、schema unsupported、provider validation error、disconnect 等状态不会静默降级为成功。

## 迁移边界与回滚

- 现有 Profile、身份、View v3 和布局树无需迁移；legacy renderer ID 由 registry alias 安全解析。
- 旧 View 中不兼容或已移除的 renderer 不被静默覆写，UI 明确 fallback，并保留原始 preference 供未来恢复。
- 回滚可按 DOC01 → SCHEMA01 → RENDER01 → VALUE01/OP01/LIVE01 → VIEW01 的 checkpoint 逆序进行；wire、Hub policy 与 CredentialStore 格式未改变。
- 隔离 GUI 测试 Hub、Profile、Permit 与临时身份只存在于项目 `.tmp` 测试目录，已停止，不属于产品或归档内容。
- 未推送、发布或创建 remote。

## 后续分期

- `REMOTE01`: owner-served schema discovery、签名/信任、cache、版本兼容和 invalidation，必须另开 protocol workflow。
- `PLUGIN01`: 第三方可执行 renderer 仍需 sandbox/trust model，不由 declarative schema 自动开启。
- File transfer handle/cancel/byte progress 需要后端 binding 契约扩展。
- 完整 dashboard query/history/chart engine 不在本次范围。

## 子 Agent 执行轨迹

- 无。schema、registry、renderer、Workspace、styles 与 generated artifact 共享同一契约面，且用户没有请求 `$m-go` 或委派执行；本 workflow 由主 agent 顺序完成。
