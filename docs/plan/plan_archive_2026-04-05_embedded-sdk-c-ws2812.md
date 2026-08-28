# Plan Archive - EmbeddedSDK C WS2812 Demo

## Workflow Information
- Repo: `MyFlowHub-EmbeddedSDK`
- Branch: `feat/embedded-sdk-c-ws2812`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-embedded-sdk-c-ws2812`
- Current Stage: `Closed (Archived in control-plane; code merged to repo main during workflow end)`

## Goal

把 `examples/esp32s3_basic/` 做成一个能在 `ESP32-S3` 上通过 `GPIO48` 驱动单颗 `WS2812` 的最小 `ESP-IDF` 示例，并保留现有 one-shot MyFlowHub 请求闭环。

## Current State At Closeout

- `examples/esp32s3_basic/` 已新增 `WS2812` 驱动、Kconfig 配置和 LED 状态语义。
- README 已补齐接线、配置、构建、烧录与边界说明。
- 真实 `ESP-IDF v6.0` build 已通过。
- 实板 `COM8` 刷写已通过，并完成到本机 `tcp://192.168.8.100:19000` 的真实请求闭环验证。
- 用户选择方案 A，本轮不扩展到 `MicroPython` live subscribe parity。

## Related Requirements

- `repo/MyFlowHub-EmbeddedSDK/docs/requirements/embedded-sdk-mvp.md`

## Related Specs

- `repo/MyFlowHub-EmbeddedSDK/docs/specs/repo-architecture.md`
- `repo/MyFlowHub-EmbeddedSDK/docs/specs/esp-idf-environment.md`
- `repo/MyFlowHub-EmbeddedSDK/docs/specs/esp-idf-component.md`
- `repo/MyFlowHub-EmbeddedSDK/docs/specs/c-typed-helpers.md`
- `repo/MyFlowHub-EmbeddedSDK/docs/specs/validation-matrix.md`

## Requirements Impact

`none`

## Specs Impact

`none`

## Lessons Impact

`updated`

## Tasks

- `CWS2812-1`
  - 为 `esp32s3_basic` 增加 `WS2812` 组件依赖、配置项和本地 LED 驱动
  - 状态：完成
- `CWS2812-2`
  - 把 LED 驱动接入当前示例主流程，并明确请求前后的 LED 行为
  - 状态：完成
- `CWS2812-3`
  - 更新 README，写清接线、构建、烧录和与 `MicroPython` 示例的边界
  - 状态：完成
- `CWS2812-4`
  - 在本机 `ESP-IDF v6.0 + COM8` 条件下完成 build，并尽量完成真实 flash / observe
  - 状态：完成
- `CWS2812-5`
  - 若用户改选 parity 路线，则暂停当前实现，回到 `Stage 1/2` 扩 scope
  - 状态：未触发

## Acceptance

- `examples/esp32s3_basic/` 在 `ESP-IDF v6.0` 下可 build
- 示例烧录到真实板子后，可在 `GPIO48` 上看到单颗 `WS2812` 被驱动
- README 能让接手者复现 `menuconfig/build/flash`
- 与 `MicroPython` live demo 的差异被明确写出，不误导成已支持 live subscribe

## Tests

- `eim.exe run "Set-Location 'D:\project\MyFlowHub3\worktrees\feat-embedded-sdk-c-ws2812\examples\esp32s3_basic'; python 'D:\Espressif\v6.0\esp-idf\tools\idf.py' build" v6.0`
- `rg -a "<redacted-2.4g-ssid>|tcp://192.168.8.100:19000" build/mfh_esp32s3_basic.bin`
- `python esptool.py --chip esp32s3 -p COM8 chip-id`
- `python esptool.py --chip esp32s3 -p COM8 ... write-flash ...`
- mock server log:
  - `accepted from ('192.168.8.105', ...)`
  - `request_action=node_echo`
  - `response_action=node_echo_resp`

## Review Summary

- 需求覆盖：通过
  - 已覆盖 `WS2812` 驱动、示例闭环、文档说明和实板验证
- 架构合理性：通过
  - 改动限制在示例层，不扩展 `C` SDK 公共契约
- 性能风险：通过
  - 无新增热路径；仅增加一次性 LED 初始化和示例状态切换
- 可读性与一致性：通过
  - LED 语义和 `MicroPython` 示例保持同名概念，README 明确边界
- 可扩展性与配置化：通过
  - `GPIO/pixel_count/rgb_hex/brightness` 全部配置化
- 稳定性与安全：通过
  - 非法配置 fail-fast，不把真实 Wi-Fi/endpoint 写回仓库
- 测试覆盖情况：通过
  - 覆盖 build、flash、串口/芯片探测和真实请求闭环
- 子Agent治理与审计：通过
  - 本轮未使用子 Agent

## Rollback

- 回退示例代码与 README 改动
- 删除 `examples/esp32s3_basic/main/idf_component.yml`
- 删除 `examples/esp32s3_basic/dependencies.lock`
- 如需恢复板子旧程序，重新刷回之前固件

## Notes

- `managed_components/` 属于下载产物，不纳入提交
- 本轮关闭 workflow 时，worktree `plan.md` 不回并到产品仓库主线；控制面归档见本文件和 `docs/change/2026-04-05_embedded-sdk-c-ws2812.md`
