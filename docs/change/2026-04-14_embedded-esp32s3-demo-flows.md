# 2026-04-14_embedded-esp32s3-demo-flows

## 变更背景 / 目标

当前现场演示路线已经收敛到“两块 ESP32-S3 + 云端 Hub + Win Flow 部署”，但 live 环境仍存在三个不稳定因素：

- 板子的 `node_id` 可能漂移
- 当前硬件其实是“1 块拓展板 + 1 块核心板”，第二块板不带按钮 / DHT11 / relay / buzzer
- `DHT11` 现场 wiring 是 `GPIO8`，且当前 live 示例变量名已经收敛到 `dht11_gpio8_temperature_c`
- Flow 真正部署时还需要人工选择正确的 `executor_node`

因此本轮不直接把 flow 写进 live hub，而是先产出一套可参数化生成、可导入、可交接的演示 flow 包。

## 具体变更内容

- 新增 `demo/embedded-esp32s3-flows/flow_params.example.json`
  - 集中承载 `executor_node`、双板 `node_id`、角色分配和变量名口径
  - 默认约定 `board1=拓展板`、`board2=核心板`
- 新增 `demo/embedded-esp32s3-flows/generate_demo_flows.ps1`
  - 根据参数文件生成最终 `FlowPayload` JSON
  - 内置参数校验、角色映射和 3 条 flow 的 payload builder
  - 允许 `board2` 缺省按钮 / DHT11 / relay / buzzer 字段
- 新增 `demo/embedded-esp32s3-flows/README.md`
  - 说明参数替换、生成命令、导入/部署顺序和现场检查点
- 新增 `demo/embedded-esp32s3-flows/out/*.json`
  - `esp32s3_button12_cross_board_scene.json`
  - `esp32s3_dht11_temperature_alarm.json`
  - `esp32s3_demo_reset.json`

本轮 3 条 flow 的设计边界：

- 按钮和 reset 都使用 `var_changed -> varstore::get -> branch`
- DHT11 告警 flow 使用温度字符串枚举阈值，不直接做 `gt/gte` 数值比较
- DHT11 `deleted` 事件会直接走告警路径
- 拓展板承担本地 relay / buzzer / 传感器输入，核心板只承担 LED 可视反馈，且高温时切到蓝灯

## Requirements impact

- `none`

## Specs impact

- `none`

## Lessons impact

- `none`

## Related requirements

- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`

## Related specs

- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Proto\docs\flow_contract.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\specs\flow-editor-visual-form.md`

## Related lessons

- `D:\project\MyFlowHub3\repo\MyFlowHub-EmbeddedSDK\docs\lessons\esp32s3-live-demo-preflight.md`

## 对应 plan.md 任务映射

- `DEMOFLOW-1`
  - `demo/embedded-esp32s3-flows/README.md`
  - `demo/embedded-esp32s3-flows/flow_params.example.json`
- `DEMOFLOW-2`
  - `demo/embedded-esp32s3-flows/generate_demo_flows.ps1`
- `DEMOFLOW-3`
  - `demo/embedded-esp32s3-flows/out/esp32s3_button12_cross_board_scene.json`
  - `demo/embedded-esp32s3-flows/out/esp32s3_dht11_temperature_alarm.json`
  - `demo/embedded-esp32s3-flows/out/esp32s3_demo_reset.json`
- `DEMOFLOW-4`
  - 本文档
  - `demo/embedded-esp32s3-flows/README.md`

## 经验 / 教训摘要

- 当前 MyFlowHub flow 设计里，`var_changed` 只告诉你“哪个变量变了”，不会直接携带变量当前值。
- 对 ESP32 板侧布尔量做场景判断时，最稳的是先 `varstore::get`，再对 `/value` 做字符串分支。
- 当前 `varstore.value` 仍是字符串，因此温度阈值 flow 只能用 `"30".."60"` 这类字符串枚举分支表达。
- 现场 wiring 和变量名可以脱钩，但本轮 live 口径已经对齐到 `DHT11 GPIO8 -> dht11_gpio8_temperature_c`。
- 如果现场只有 1 块拓展板，生成器必须允许另一块核心板只提供 LED 变量，不能再把双板都当成全功能外设板。

## 可复用排查线索

- 症状
  - flow 导入成功，但现场没有板侧反馈
  - DHT11 guard 没走正常/告警预期路径
  - 按钮 flow 总是走同一条分支
- 触发条件
  - `node_id` 填错
  - 变量名按 wiring 猜测被改错
  - flow 部署到错误的 `executor_node`
- 关键词
  - `var_changed`
  - `varstore::get`
  - `buzzer_gpio7_on`
  - `dht11_gpio8_temperature_c`
  - `executor_node`
- 快速检查
  - 确认 `flow_params.local.json` 里的 `board1.node_id` / `board2.node_id`
  - 确认 `board1` 仍是拓展板、`board2` 仍是核心板
  - 确认 `buzzer_var` 是否仍是 `buzzer_gpio7_on`
  - 确认 DHT11 温度变量名是否仍为 `dht11_gpio8_temperature_c`
  - 确认 Win 部署时选择的目标节点是否与 `executor_node` 一致

## 关键设计决策与权衡

- 选择“本地产物 + 参数化生成”，而不是直接 live 写入
  - 原因：当前现场节点和变量口径仍可能漂移，先产出演示包更稳
- 选择“字符串枚举阈值告警”，而不是 `gt/gte` 数值比较
  - 原因：当前 `varstore.value` 是字符串，运行时数值比较会直接失败
- 选择生成最终可导入 JSON，而不是只写 README
  - 原因：用户要求的是可直接用于演示的 flow，而不是概念说明

## 测试与验证方式 / 结果

- `pwsh -File .\demo\embedded-esp32s3-flows\generate_demo_flows.ps1`
  - 结果：通过，生成 3 个 flow payload
- `ConvertFrom-Json`
  - 结果：3 个输出 JSON 均解析成功
- 轻量图一致性检查
  - 检查项：
    - node id 不重复
    - edge `from/to` 均命中存在节点
  - 结果：3 个输出 JSON 均通过
- 单拓展板语义检查
  - 检查项：
    - `board2` 是否仍可缺省按钮 / DHT11 / relay / buzzer 字段
    - 高温告警时 `board2` 是否只写 LED 且蓝色分量被拉高
  - 结果：通过，`board2` 未被引用任何非 LED 变量，且高温态写入蓝灯

## 潜在影响

- `out/*.json` 是基于示例参数生成的结构化产物，不保证与现场 live `node_id` 完全一致。
- 真正部署前仍应先复制 `flow_params.example.json` 为本地参数文件并替换现场值。
- 如果现场把拓展板 / 核心板角色对调，必须同步修改 `roles.*` 和相关变量名，否则触发与动作会落到错误板子上。

## 回滚方案

1. 删除 `demo/embedded-esp32s3-flows/` 目录
2. 删除本归档和 `docs/change/README.md` 中的索引项

## 子Agent执行轨迹

- 本轮未使用子Agent。
