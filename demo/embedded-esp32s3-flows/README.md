# Embedded ESP32-S3 Demo Flows

这套目录产出的是可导入到 MyFlowHub Flow 页面里的 `FlowPayload` JSON，不是直接写进 hub 的 live deployment。

## 包含内容

- `flow_params.example.json`
  - 现场参数模板。
- `generate_demo_flows.ps1`
  - 根据参数模板生成最终 flow JSON。
- `out/*.json`
  - 生成后的演示 flow payload。

## 当前口径

- 两块板都使用 `examples/esp32s3_basic`
- 当前按“一块拓展板 + 一块核心板”设计：
  - `board1` 约定为带外设的拓展板
  - `board2` 约定为没有拓展外设的核心板
- 蜂鸣器默认按 `GPIO7 / buzzer_gpio7_on`
- 现场 DHT11 wiring 是 `GPIO8`
- 当前示例默认温度变量名是 `dht11_gpio8_temperature_c`

上面最后两点不要混淆：
- `GPIO8` 是现场连线
- `dht11_gpio8_temperature_c` 是当前示例变量名口径

## 生成方式

1. 复制参数模板：

```powershell
Copy-Item .\flow_params.example.json .\flow_params.local.json
```

2. 修改 `flow_params.local.json`：
   - `executor_node`
   - `board1.node_id`
   - `board2.node_id`
   - 若现场变量名与默认值不同，再修改对应 `*_var`
   - 若高温阈值要调整，修改 `temperature_alarm.threshold_c`
   - `board1` 填完整外设变量
   - `board2` 只需要 LED 变量；按钮、DHT11、relay、buzzer 可以留空或不填
   - 若按钮板、传感器板、reset 板不是同一块，更新 `roles.*`

3. 生成 flow：

```powershell
pwsh -File .\generate_demo_flows.ps1 -ParamsPath .\flow_params.local.json
```

默认输出目录是 `out/`。

## 参数说明

- `executor_node`
  - 计划部署这些 flow 的执行节点。
  - 注意：`FlowPayload` 本身不包含这个字段，真正部署时仍要在 Win 里选择目标 executor。
- `visibility`
  - 生成的 `varstore::set` 默认写入可见性，当前建议保持 `public`。
- `roles.button_board`
  - `button12` 触发源所在板。
- `roles.sensor_board`
  - `dht11_temperature_var` 所在板。
- `roles.reset_board`
  - `button13` reset 触发源所在板。
- `roles.scene_board`
  - 按钮联动里负责远端可见反馈的那块板。
  - 在当前默认口径下，这块板是“核心板”，只承担 LED 可视反馈。
- `temperature_alarm.threshold_c`
  - 温度告警阈值；达到或超过这个整数温度时走告警路径。
- `temperature_alarm.max_c`
  - 生成字符串分支时使用的上限值。
  - 当前 runtime 里 `varstore.value` 是字符串，所以这里会展开成 `threshold_c..max_c` 的枚举分支。

## Flow 列表

### `esp32s3_button12_cross_board_scene.json`

用途：
- 监听 `button12`
- 按下时：
  - 本板拉起绿色确认态
  - 拓展板打开本地蜂鸣器和 `relay_gpio4_on`
  - 核心板只亮绿色
- 松开时：
  - 两块板回到蓝色待机
  - 拓展板蜂鸣器关闭
  - 拓展板 `relay_gpio4_on=false`

设计要点：
- `var_changed` trigger 本身不带变量值
- 所以 flow 先 `varstore::get`，再对 `/value == "true"` 做 branch

### `esp32s3_dht11_temperature_alarm.json`

用途：
- 监听 `dht11_temperature_var`
- 当温度低于 `temperature_alarm.threshold_c` 时：
  - 两块板切到绿色正常态
  - 拓展板关闭蜂鸣器
  - 拓展板清掉 `relay_gpio5_on`
- 当温度达到或超过阈值，或者温度变量被删除时：
  - 拓展板切到红色告警态
  - 拓展板打开蜂鸣器和 `relay_gpio5_on`
  - `node5` 这块核心板切到蓝灯高亮

设计要点：
- 当前 `varstore.value` 是字符串，不能直接用数值 `gt/gte`
- 这条 flow 会把 `threshold_c..max_c` 展开成字符串等值分支，例如阈值 `30` 会匹配：
  - `"30"`
  - `"31"`
  - ...
  - `"60"`

### `esp32s3_demo_reset.json`

用途：
- 监听 `button13`
- 按下时把两块板恢复到统一待机态：
  - 拓展板的 relay / buzzer 关闭
  - 两块板 LED 回到低亮度蓝色待机

## 导入与部署建议

1. 先确认两块板都已经审批通过并在线。
2. 先生成最终 JSON。
3. 在 Win Flow 页面分别导入这 3 个 payload。
4. 部署时都指向同一个 `executor_node`。
5. 建议部署顺序：
   - `ESP32 Demo Reset`
   - `ESP32 Button12 Cross-Board Scene`
   - `ESP32 DHT11 Temperature Alarm`

## 现场检查点

- `board1.node_id` / `board2.node_id` 必须填对
- 当前推荐：
  - `board1 = 拓展板`
  - `board2 = 核心板`
- `buzzer_var` 应该是 `buzzer_gpio7_on`
- 当前默认温度变量名是 `dht11_gpio8_temperature_c`
- 如果现场温度变量名不是这个，再只改 `dht11_temperature_var`
- 若 flow 已部署但板子没反应，先查：
  - 节点是否在线
  - 是否审批通过
  - flow 是否部署到了正确的 `executor_node`
