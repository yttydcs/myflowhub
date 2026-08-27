# Embedded Toolchain And Board Preflight

## Summary

Embedded 验证必须先区分源码、host toolchain、board toolchain、网络/接线和真实设备证据。缺少 `idf.py`、2.4GHz 网络或真板不能记为代码失败，也不能用 host test 冒充设备通过。

## Lookup Hints

- 症状：`python` 无输出、`idf.py` not found、ESP-IDF 系统头在 `-Wpedantic` 下失败、开发板连 `127.0.0.1`、MIP 安装成功但 import 失败。
- 关键词：WindowsApps python stub、ESP-IDF 6、`-Werror`、2.4GHz SSID、package-only import、uasyncio awaitable。
- 快速检查：记录实际 executable/version；扫描 SSID；核对 LAN 地址/GPIO；在只有安装产物的环境执行 eager import。

## Symptoms

- CI/本机把环境缺失误报为源码编译错误。
- host C 零 warning，但 ESP-IDF 因外部系统头 pedantic 失败。
- 板卡网络正常却连接 loopback，或 5GHz-only SSID 根本不可见。
- MicroPython 包 metadata 正常，实际导入 eager module 时缺文件或 coroutine 识别不同。

## Impact

浪费在错误层级排查时间，制造假失败/假通过，或把现场凭据和板卡参数硬编码进仓库。

## Trigger Conditions

- PATH 中命中 WindowsApps stub，未记录实际工具位置。
- 对 host、ESP-IDF、MicroPython 共用一套未经验证的 warning/runtime 假设。
- 未完成接线、网络、permit 和 endpoint 预检就调协议。
- 只测试源码树 import，不测试安装包表面。

## Root Cause

Embedded 的可用性由多层独立证据组成：源码正确、工具链可用、package 完整、板卡外围正确、网络可达、准入/授权成功。任一层不能替代其他层。

## Investigation Trail

1. 记录 `python`、compiler、CMake、`idf.py` 的绝对路径和版本。
2. host C 保持严格 warning；ESP-IDF 单独识别外部系统头兼容性。
3. 核对 GPIO、电压、2.4GHz SSID、设备可达的 LAN endpoint 和 parent permit。
4. 用 package-only 环境测试 MicroPython eager imports，并在真板检查 uasyncio 行为。

## Resolution

- 环境不可用记为 Unavailable，源码失败保留原始错误，两者不混写。
- host 使用 `-Wall -Wextra -Wpedantic -Werror`；ESP-IDF 可去掉 `-Wpedantic`，但保留其他 warning-as-error。
- 配置通过 example/runtime 注入，不提交 SSID、密码、endpoint 或 permit。
- scripted transport 等待排队写入并覆盖 partial frame、断线和重连。

## Prevention / Guardrails

- CI 固定 ESP-IDF 版本并提供 source-build gate；真实 flash/runtime 另列设备证据。
- MIP/package metadata、package-only import 和真板 async smoke 分别记录。
- 调协议前按“工具链→接线→网络→admission→resource authorization”顺序预检。

## Related Docs

- [Embedded feature](../features/embedded.md)
- [Build and CI](../specs/build-and-ci.md)
- [Operational lifecycle](../specs/operational-lifecycle.md)
- [vNext full migration](../change/2026-08-27_vnext-full-migration.md)
