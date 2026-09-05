# Android、嵌入式退役与 attached bindings 收敛

## 结果

按用户确认范围，移除 Android 通用应用、Metrics/Clipboard Android 应用及其移动绑定；移除 C、ESP32、MicroPython 实现与专属测试、夹具、工具链和 CI。

Windows Clipboard 迁至 NodeHost：身份、Node、父连接与关闭由 Host 管理，资源在网络启动前注册，同步引擎只使用 attached SDK 和只读 ConnectionStatus。原身份、配置和历史继续复用。

Go SDK 不再导出 NewClient、Close、Connect、ConnectManaged 或 owning Connection。通用与 Desktop bindings 只保留 attached facade，Close 只清理 facade 订阅。独立 EnrollmentBootstrap 继续承担入网握手与凭据持久化；Legacy Join 保留。

生成契约的 methods/desktop_methods 与实际 portable/Desktop binding API 对齐；旧生命周期及不属于 binding 的文件选择入口不再列入 manifest。Wails App 的文件选择能力保持原有实现。

Windows Desktop、Metrics、Clipboard、通用网络核心、QUIC 与 Windows/Linux RFCOMM 保留。Clipboard Web 仍是明确标注的 UI 预览。没有引入新的 Flow、语言或移动运行时方案。

## 规模

相对本轮开始时的工作区内容，删除 106 个文件，其中包含二进制资源。非 Markdown 文本净减少 **7,260 行**：实现、生成契约与构建配置净减 6,363 行，测试与测试夹具净减 897 行。统计包含本轮新增的回归测试，不包含文档和二进制字节变化，也不包含用户原有修改。

## 验证

| 验证 | 结果与覆盖 |
| --- | --- |
| `go test ./... -count=1`、`go vet ./...`、`go build ./...` | 通过，包含 Desktop/Metrics、资源操作、授权、入网与产品集成 |
| SDK、portable/Desktop bindings、NodeHost、Clipboard 的 `go test -race` | 通过，覆盖订阅取消、Host ownership、退出与并发路径 |
| Clipboard Go 回归 | 双节点同步、真实 TCP 进程、既有身份/配置/历史保留、重复关闭和准备失败释放状态目录通过 |
| `generate/generated` 与生成契约 freshness | 通过；Wails 生成内容无行为变更，API manifest 与实际 attached facade 对齐 |
| `test/clipboard` | Flutter analyze 与 widget test 通过 |
| `build/clipboard` | Windows release、同目录 Go bridge 与 Web release 构建通过 |
| RFCOMM 编译 | Windows 全量构建通过；另验证 Linux 实现、Darwin stub 及 Android unsupported stub；不代表恢复 Android 产品支持 |
| 文档与范围检查 | 修改文档的相对链接有效，retired build targets 不再被脚本/CI/模块依赖要求 |

未部署或操作正在运行的用户节点。没有新增蓝牙硬件 smoke、Android/嵌入式设备测试或 Linux GUI 构建；已退役平台不再是当前门禁。

## 文档影响与索引

| 类别 | 当前事实与追溯入口 | 索引 |
| --- | --- | --- |
| Intake / Decision | [原始请求](../intake/2026-09-06_android-embedded-retirement-and-runtime-convergence.md)、[旧方案与采用方案对比](../decisions/2026-09-06_retire-mobile-embedded-and-owning-bindings.md) | intake、decisions 已更新 |
| Requirements | [Android/嵌入式重新设计](../requirements/mobile-embedded-redesign.md)、[统一运行时](../requirements/unified-node-runtime.md)、[资源平台](../requirements/extensible-resource-platform.md)、[准入](../requirements/auth-controlled-admission.md)、[待办](../../todo.md) | requirements 已更新 |
| Features | [Clipboard](../features/clipboard-node.md)、[Metrics](../features/metrics-node.md)、[Android 历史](../features/android.md)、[Embedded 历史](../features/embedded.md) | [features 索引](../features/README.md)已将退役产品单列 |
| Specs | [NodeHost](../specs/node-host-runtime.md)、[构建/CI](../specs/build-and-ci.md)、[模块边界](../specs/repository-and-module-boundaries.md)、[协议映射](../specs/protocol_map.md)、[准入契约](../specs/node-enrollment-and-admission-authority.md)、[Collection 调用方](../specs/resource-collections-and-actions.md) | specs 已更新 |
| 产品入口 | [根 README](../../README.md)、[仓库边界](../../repos.md)、[Clipboard 源码说明](../../apps/nodes/clipboard/README.md)、[Flutter 壳](../../apps/nodes/clipboard/app/README.md)、[Metrics 源码说明](../../apps/nodes/metrics/README.md) | 入口与当前平台范围一致 |

先更新稳定需求、产品和契约，再记录本归档。历史 ADR、plan/change、migration 与排障证据保持原文；Android/Embedded feature 顶部明确标记退役。

## 兼容与回退

SDK 生命周期接口是有意的源码兼容性变更；已有调用方应迁到 NodeHost。文件格式与 wire 不因本次清理改变。本机 SDK、已安装应用与运行状态不清除。

本轮工作在 sibling worktree 完成验证后合入 canonical checkout 的未提交修改。只同步相对本轮基线发生变化的文件，保留原有未提交文档、guide 和设计草稿；未 commit、push 或发布。退役前源码可从 Git 的 `476c6ea85901980fdf9fefadb6fba0986bde0303` 追溯，回退需连同 SDK/Clipboard、依赖、生成契约和构建入口一并评估，不能只恢复单个应用目录。
