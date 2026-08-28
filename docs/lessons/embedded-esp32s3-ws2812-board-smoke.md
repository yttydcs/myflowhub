# Embedded ESP32-S3 WS2812 Board Smoke

## Summary

在 Windows 上给 `ESP32-S3` 做 `ESP-IDF` + `WS2812` 板级联调时，最容易误判的不是代码本身，而是三类环境假象：

- 板子连的是 `5 GHz` SSID，`ESP32-S3` 实际只能用 `2.4 GHz`
- `sdkconfig` 改了，但没有重新 `build`，结果刷进去的仍是旧 `bin`
- `USB Serial/JTAG (COM8)` 在设备管理里显示正常，却不适合作为第一诊断入口去跑 `idf.py monitor`

这类问题会让人误以为“C SDK 不通”或“板子没跑起来”，实际先做环境前置检查和显式 `esptool` 探针，能明显缩短排障路径。

## Lookup Hints

- Symptoms:
  - 板子刷完后一直不连本机 mock server
  - `COM8` 偶发可见但 `monitor` 行为异常，或者一开 monitor 板子就进下载态
  - 改过 `SSID/endpoint` 之后，板子仍表现得像在跑旧配置
- Trigger Conditions:
  - `ESP32-S3` 使用 `USB Serial/JTAG`
  - 主机同时存在 `5 GHz` 与 `2.4 GHz` 同名或相近 SSID
  - 依赖 `sdkconfig` 本地改值做一次性板级 smoke
- Keywords:
  - `COM8`
  - `USB Serial/JTAG`
  - `esptool chip-id`
  - `idf.py monitor`
  - `<redacted-2.4g-ssid>`
  - `WS2812`
  - `GPIO48`
  - `write-flash`
- Quick checks:
  - 先确认目标 SSID 是 `2.4 GHz`
  - 改完 `sdkconfig` 之后，先重新 `build`
  - 用 `rg -a` 检查 `bin` 里是否真的包含新的 `SSID/endpoint`
  - 用显式 `esptool --chip esp32s3 -p COMx chip-id` 代替先开 `monitor`

## Symptoms

- mock server 一直没有来自板子的连接记录
- 板子已刷写，但 LED 行为和预期配置不一致
- 串口层偶发报“设备没有发挥作用”或 monitor 导致板子 reset
- 用户以为刷进去的是新固件，实际 `bin` 里仍是旧的 `SSID/endpoint`

## Impact

- 会把环境问题误判为 SDK、Wi-Fi 或协议问题
- 反复刷写和反复开 monitor 会浪费大量时间
- 如果不检查 `bin` 本身，容易在“板子不联网”和“固件仍是旧配置”之间来回误诊

## Trigger Conditions

- `ESP32-S3` 板卡通过 `USB Serial/JTAG` 直连 Windows
- 本地 Wi-Fi 同时存在 `5 GHz` 和 `2.4 GHz`
- `sdkconfig` 作为本地 smoke 配置承载真实 `SSID/endpoint`
- 把 `idf.py monitor` 当成第一诊断手段

## Root Cause

- `ESP32-S3` 只支持 `2.4 GHz Wi-Fi`，如果主机或操作者默认记住的是 `5 GHz` SSID，会导致板子永远连不上
- `sdkconfig` 只是输入，不是产物；如果不重新 `build`，刷写的仍可能是旧 `bin`
- `USB Serial/JTAG` 的 `COM` 口即使显示 `Started`，也不代表当前最适合拿来走 `monitor` 复位链路

## Investigation Trail

1. 先看主机当前网络环境，确认真实可用的 `2.4 GHz` SSID 名称
2. 对 `build/mfh_esp32s3_basic.bin` 做字符串检索，确认新配置是否真的编进固件
3. 用 `esptool --chip esp32s3 -p COM8 chip-id` 验证串口和芯片握手，而不是先开 `idf.py monitor`
4. 用显式 `write-flash` 完成刷写，再从 mock server 看是否收到真实连接
5. 只有在需要串口日志时，才谨慎尝试 monitor，并预期它可能触发额外 reset

## Resolution

- 把 Wi-Fi 改成真实可用的 `2.4 GHz` SSID；历史环境的具体名称已脱敏
- 重新 `idf.py build`
- 用 `rg -a` 确认 `bin` 已包含新的 `SSID/endpoint`
- 用显式 `esptool chip-id` 和 `write-flash` 完成探测与刷写
- 通过本机 `19000` mock server 看到真实 `node_echo -> node_echo_resp` 闭环，确认板子已联网并运行新固件

## Prevention / Guardrails

- `ESP32-S3` Wi-Fi smoke 前先问清或确认：目标 SSID 是否为 `2.4 GHz`
- 任何通过 `sdkconfig` 改的本地参数，在刷板前都先检查一次新 `bin`
- 优先使用显式路径：
  - `C:\Espressif\tools\python\v6.0\venv\Scripts\python.exe`
  - `D:\Espressif\v6.0\esp-idf\tools\idf.py`
  - `D:\Espressif\v6.0\esp-idf\components\esptool_py\esptool\esptool.py`
- `idf.py monitor` 不作为第一诊断入口；先做 `chip-id`、`write-flash`、mock server 观察

## Related Docs

- Change:
  - `docs/change/2026-04-05_embedded-sdk-c-ws2812.md`
- Requirements:
  - `repo/MyFlowHub-EmbeddedSDK/docs/requirements/embedded-sdk-mvp.md`
- Specs:
  - `repo/MyFlowHub-EmbeddedSDK/docs/specs/esp-idf-environment.md`
  - `repo/MyFlowHub-EmbeddedSDK/docs/specs/validation-matrix.md`
