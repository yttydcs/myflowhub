# 2026-04-05 EmbeddedSDK C WS2812 Demo

## Background

`MyFlowHub-EmbeddedSDK` 的 `examples/esp32s3_basic/` 之前只覆盖最小 `ESP-IDF` one-shot request 路径，缺少板侧可视状态。
本轮 workflow 的目标是在不把示例扩成完整 live runtime 的前提下，为 `ESP32-S3` 补齐 `GPIO48` 单颗 `WS2812` 显示，并在真实板子上完成一次从 Wi-Fi 到 MyFlowHub 请求闭环的板级验证。

## Changes

- 为 `examples/esp32s3_basic` 引入 `espressif/led_strip`，新增：
  - `examples/esp32s3_basic/main/idf_component.yml`
  - `examples/esp32s3_basic/dependencies.lock`
- 在 `examples/esp32s3_basic/main/Kconfig.projbuild` 增加 `WS2812 GPIO / pixel_count / on / rgb_hex / brightness` 示例配置。
- 在 `examples/esp32s3_basic/main/main.c` 增加：
  - `WS2812` 初始化
  - `#RRGGBB` 解析与亮度缩放
  - fail-fast 配置校验
  - 启动、Wi-Fi、请求中、失败、成功后的 LED 状态切换
- 在 `examples/esp32s3_basic/main/CMakeLists.txt` 中补齐 `led_strip` 依赖。
- 更新 `examples/esp32s3_basic/README.md` 与 `examples/README.md`，明确：
  - 接线
  - `menuconfig`
  - `build/flash`
  - LED 状态语义
  - 该示例仍是 one-shot demo，不是 `MicroPython` live subscribe runtime

## Related Plan

- `plan.md`
  - worktree: `D:\project\MyFlowHub3\worktrees\feat-embedded-sdk-c-ws2812\plan.md`
- `docs/plan/plan_archive_2026-04-05_embedded-sdk-c-ws2812.md`

## Related Requirements

- `repo/MyFlowHub-EmbeddedSDK/docs/requirements/embedded-sdk-mvp.md`

## Related Specs

- `repo/MyFlowHub-EmbeddedSDK/docs/specs/repo-architecture.md`
- `repo/MyFlowHub-EmbeddedSDK/docs/specs/esp-idf-environment.md`
- `repo/MyFlowHub-EmbeddedSDK/docs/specs/esp-idf-component.md`
- `repo/MyFlowHub-EmbeddedSDK/docs/specs/c-typed-helpers.md`
- `repo/MyFlowHub-EmbeddedSDK/docs/specs/validation-matrix.md`

## Lessons Impact

`updated`

## Related Lessons

- `docs/lessons/embedded-esp32s3-ws2812-board-smoke.md`
- `repo/MyFlowHub-EmbeddedSDK/docs/lessons/windows-validation-preflight.md`
- `repo/MyFlowHub-EmbeddedSDK/docs/lessons/micropython-live-demo-preflight.md`
- `repo/MyFlowHub-EmbeddedSDK/docs/lessons/esp-idf-system-headers-pedantic-compat.md`

## Searchable Lessons Summary

- `ESP32-S3` 板侧 `WS2812` 联调如果一直不连本机 mock server，先确认不是接到了 `5 GHz` SSID。
- `sdkconfig` 改完之后必须重新 `build`；不要假设旧 `bin` 已经带上新的 `SSID/endpoint`。
- `USB Serial/JTAG (COM8)` 在 Windows 上即使显示 `Started`，也可能暂时不可用；优先用显式 `esptool chip-id` 和 `write-flash` 判活，不要先依赖 `idf.py monitor`。
- `idf.py monitor` 在这块板上可能触发 reset 并把板子带进 ROM/下载态，不适合作为第一诊断入口。

## Requirements Impact

`none`

## Specs Impact

`none`

## 对应 Plan 任务映射

- `CWS2812-1`
  - 完成 `led_strip` 依赖、Kconfig 和本地 LED 驱动辅助逻辑
- `CWS2812-2`
  - 完成 LED 与示例主流程集成，补齐状态颜色语义
- `CWS2812-3`
  - 完成 README 与 examples 索引更新
- `CWS2812-4`
  - 完成真实 `ESP-IDF v6.0 + COM8` build、flash 和板级请求闭环验证
- `CWS2812-5`
  - 未进入；用户明确选择方案 A，不扩到 `MicroPython` live parity

## 关键设计决策与权衡

- 采用“示例层最小硬件增强”，不改 `C` SDK 公共契约。
- LED 语义对齐 `MicroPython` 的 `on/rgb_hex/brightness`，但不承诺 live subscribe。
- 保留 `dependencies.lock` 作为示例依赖真相；不提交 `managed_components/` 下载产物。
- 真实板级验证使用显式 `esptool` 路径与 `COM8`，避免继续依赖不稳定的 `idf.py monitor` 入口。

## Validation

- Build:
  - `eim.exe run "Set-Location 'D:\project\MyFlowHub3\worktrees\feat-embedded-sdk-c-ws2812\examples\esp32s3_basic'; python 'D:\Espressif\v6.0\esp-idf\tools\idf.py' build" v6.0`
  - 结果：通过；构建日志确认使用 `sdkconfig` 中已脱敏的 `2.4 GHz` SSID、`tcp://192.168.8.100:19000`
- Firmware sanity:
  - 对 `build/mfh_esp32s3_basic.bin` 做字符串检索
  - 结果：确认固件包含已脱敏的 Wi-Fi SSID/密码和 `tcp://192.168.8.100:19000`；具体凭据未保留在归档中
- Port / chip check:
  - `esptool --chip esp32s3 -p COM8 chip-id`
  - 结果：能识别 `ESP32-S3`，`USB mode=USB-Serial/JTAG`
- Flash:
  - `python esptool.py --chip esp32s3 -p COM8 -b 460800 --before default-reset --after hard-reset write-flash --flash-mode dio --flash-freq 80m --flash-size 2MB 0x0 bootloader/bootloader.bin 0x8000 partition_table/partition-table.bin 0x10000 mfh_esp32s3_basic.bin`
  - 结果：通过
- Real board smoke:
  - 本机 `19000` mock server 收到来自 `192.168.8.105` 的真实连接
  - 请求：`node_echo`
  - 数据：`{"message":"hello from c ws2812 demo"}`
  - 响应：`node_echo_resp`
  - 结果：`request handled`
- Board observation:
  - 用户确认 `GPIO48` 上单颗 `WS2812` 在启动后先偏黄，再能切到红色，随后在闭环验证中已能按当前 one-shot 语义运行

## Rollback

- 代码回滚：
  - 回退 `examples/README.md`
  - 回退 `examples/esp32s3_basic/README.md`
  - 回退 `examples/esp32s3_basic/main/CMakeLists.txt`
  - 回退 `examples/esp32s3_basic/main/Kconfig.projbuild`
  - 回退 `examples/esp32s3_basic/main/main.c`
  - 删除 `examples/esp32s3_basic/main/idf_component.yml`
  - 删除 `examples/esp32s3_basic/dependencies.lock`
- 运行态回滚：
  - 若需恢复板上旧程序，重新刷回之前固件
- 清理：
  - 不保留 `managed_components/`、mock server、串口抓取和 build 内临时日志

## 潜在影响

- `examples/esp32s3_basic/` 现在依赖 `ESP-IDF` component manager 首次拉取 `led_strip`
- 示例的最终目标仍是“最小可见 one-shot demo”，不是 live runtime；若后续要做变量驱动实时变色，必须重新开 workflow 扩范围

## 子Agent执行轨迹

- 本轮未派发子 Agent
