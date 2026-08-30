# Windows 上 Go 交叉编译测试必须使用 compile-only

## Summary

在 Windows 主机设置 `GOOS=linux` 后直接执行 `go test`，Go 会先成功交叉编译测试二进制，然后尝试在 Windows 上运行该 Linux 二进制，最终产生与产品逻辑无关的 `not a valid Win32 application`。跨平台编译门禁应显式使用 compile-only 命令。

## Lookup Hints

- 症状：`fork/exec ... .test: %1 is not a valid Win32 application`
- 关键词：`GOOS=linux`、`go test`、`not a valid Win32 application`、cross compile、compile-only、`go test -c`
- 触发条件：在 Windows shell 中为非 Windows 目标设置 `GOOS`/`GOARCH` 后运行普通 `go test`
- 快速检查：确认失败发生在测试二进制启动阶段，而不是 compile 阶段；改用 `go test -c -o NUL` 重跑

## Symptoms

- Go package 能完成编译，但测试命令在启动生成的 `.test` 文件时报 Win32 executable 错误。
- 同一 package 在本机目标测试通过，切换到 Linux target 后只出现进程格式错误。
- 错误容易被误判为 Linux-specific code 或 cgo build failure。

## Impact

- 跨平台验证被误报为产品失败。
- Agent 可能在无关代码上反复修复，或错误地跳过平台门禁。
- CI/本地脚本若混用“编译验证”和“运行测试”，会让结果难以解释。

## Trigger Conditions

- Host OS 与 `GOOS` 不同。
- 使用 `go test` 而不是 `go test -c` 或 `go build`。
- 没有真实目标平台 runner、容器、VM 或 emulator 来执行生成的测试二进制。

## Root Cause

`go test` 的语义包含“编译并执行”。Go 支持交叉编译测试二进制，但 Windows 不能直接执行 Linux ELF。环境变量改变了目标格式，却没有提供能够运行该格式的执行环境。

## Investigation Trail

1. 首次运行 `GOOS=linux GOARCH=amd64 go test`，compile 完成后出现 `not a valid Win32 application`。
2. 检查错误阶段，确认不是类型、依赖或 cgo 编译错误，而是 Windows 启动目标二进制失败。
3. 改用 `go test -c -o NUL` 对目标 package 做 compile-only 检查。
4. Desktop 非 Windows credential package 和 facade 均成功生成目标测试二进制，证明跨平台编译路径有效。

## Resolution

Windows PowerShell 中使用：

```powershell
$env:GOWORK = 'off'
$env:GOOS = 'linux'
$env:GOARCH = 'amd64'
go test -c -o NUL ./apps/desktop
go test -c -o NUL ./sdk/bindings/desktop
```

需要验证实际运行行为时，必须在 Linux runner、容器、VM 或目标设备上执行，不把 compile-only 冒充 runtime test。

## Prevention / Guardrails

- 验证计划明确区分 native test、cross compile 和 target runtime test。
- Windows 上对非 Windows `GOOS` 使用 `go test -c` 或 `go build`。
- 脚本输出中标注 `compile-only`，不要报告为“目标平台测试已运行”。
- 只有目标平台 runner 真正执行了测试二进制，才能声明 cross-platform runtime Passed。

## Related Intake / Features / Requirements / Specs / Decisions / Changes

- [Node Enrollment 原始请求](../intake/2026-08-30_node-enrollment-central-authority.md)
- [Desktop](../features/desktop.md)
- [受控准入需求](../requirements/auth-controlled-admission.md)
- [Node Enrollment 与 Admission Authority](../specs/node-enrollment-and-admission-authority.md)
- [集中式 Admission Authority](../decisions/2026-08-30_centralized-admission-authority.md)
- [Node Enrollment 与集中式 Admission Authority 变更](../change/2026-08-30_node-enrollment-admission-authority.md)
