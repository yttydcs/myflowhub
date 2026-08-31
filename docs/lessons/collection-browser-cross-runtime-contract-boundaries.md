# Collection browser 的跨运行时契约边界

## 适用场景

当 Go provider、JSON/Wails binding、JavaScript UI 与真实 packaged Desktop 共同实现 Collection 分页、member detail
和内容 renderer 时，组件测试通过并不代表跨运行时契约正确。尤其要同时区分 UI 原子状态、JSON 安全数字、opaque
cursor、wire envelope 与解码后的 member content。

## 复现症状与根因

### 返回或关闭后操作状态再次出现

- 症状：jsdom 测试通过，但真实 Wails 中 Return/Close 看起来无效，之后切换 View 或 selection 又恢复旧 draft。
- 根因：selection 与 focused action 分散在多个 state 更新中，旧 closure 可以重新提交半套状态；同时 packaged
  进程可能仍嵌入上一次 build 的 `dist`。
- 防错：把 selection/action 作为一个原子状态更新；真实 GUI 验证前重新 build、重新 package、关闭旧进程并确认
  asset hash 变化。

### 下一页报 revision 不是安全正整数

- 症状：第一页正常，翻页后 JavaScript 报 `Collection page revision 必须是安全范围内的正整数`，或 cursor 被判 stale。
- 根因：Go `uint64` hash 被直接放进 JSON number，超过 `2^53-1` 后 JavaScript 舍入；若 cursor 仍绑定完整 hash，
  页面 revision 与 cursor 就不再代表同一快照。
- 防错：wire revision 限制在 `1..2^53-1` 且耗尽时明确失败；cursor 继续使用 opaque、完整 fingerprint，并同时绑定
  parent/revision/fingerprint。不能让客户端解析 cursor，也不能用缩短数字代替快照指纹。

### JSON member 显示 wrapper 字段或大量占位符

- 症状：读取 JSON 文件后 renderer 显示 `Version/Encoding/Data` 等 envelope 字段，真正的 `fixture/items` 不见了。
- 根因：filesystem read response schema 被直接套到已解码的 inner JSON 上，混淆了 transport envelope 与内容 schema。
- 防错：先严格校验 envelope 和 encoding，再依据 `content_type`/member schema 选择 inner renderer；unknown binary、
  HTML 与 SVG 必须走不可执行 fallback，不能因为扩展名或 Resource name 绕过 content gate。

## 最小检查顺序

1. 确认测试使用的是当前源码生成的 `dist` 与当前 packaged executable，而不是陈旧嵌入物。
2. 比较 Go response、Wails JSON 与浏览器实际值，特别检查 revision 是否为 `Number.isSafeInteger`。
3. 分别记录 envelope schema、member descriptor schema 与 decoded content schema；不要让一个 schema 承担三层职责。
4. 用真实 Hub、真实 permit/policy、真实 provider 做分页、Forbidden、Return/Close 与二次重启 smoke。
5. 检查 View persistence 中不存在 member key、content、draft、result、credential、permit 或 permission decision。

## 关键词

`focusedAction`、`generation token`、`Number.isSafeInteger`、`2^53-1`、`opaque cursor`、`full fingerprint`、
`content_type`、`filesystem read envelope`、`stale bundle`、`Wails go:embed`。

## 当前技术真相

- [Resource Collections and Actions](../specs/resource-collections-and-actions.md)
- [Desktop](../features/desktop.md)
- [Flow](../features/flow.md)
- [Resource Platform v2](../specs/resource-platform-v2.md)

## 变更证据

- [Resource Collections、Capability Actions 与 Desktop 交互](../change/2026-08-31_resource-collections-and-capability-actions.md)
