# powershell-utf8-nobom-parse

## Summary
Windows PowerShell 5.1 可能会把 UTF-8 无 BOM 的 `.ps1` 脚本解析错，尤其是脚本里带中文注释时，外部启动会直接报语法错误而不是正常执行。

## Lookup Hints
- `Unexpected token '}'`
- `powershell.exe -File`
- `UTF-8 no BOM`
- `PowerShell 5.1`
- `Chinese comment`
- `start-myflowhub-mcp.ps1`
- `install-codex-myflowhub-mcp.ps1`

## Symptoms
- 脚本在 Windows PowerShell 5.1 下启动失败。
- 报错常见为 `Unexpected token '}'` 或类似解析错误。
- 同一脚本在 PowerShell 7 下可能正常。

## Impact
- 启动脚本不可用，`myflowhub-mcp` 无法被稳定拉起。
- 安装脚本无法正确生成或预览 Codex 配置。

## Trigger Conditions
- 使用 `powershell.exe -NoProfile -ExecutionPolicy Bypass -File ...` 执行脚本。
- `.ps1` 以 UTF-8 无 BOM 保存。
- 脚本中包含中文注释或其他非 ASCII 内容。

## Root Cause
Windows PowerShell 5.1 对 UTF-8 无 BOM 脚本的解析兼容性差，遇到非 ASCII 内容时可能误判 token 边界，最终在看似无关的 `}` 位置抛出语法错误。

## Investigation Trail
- 先在 Windows PowerShell 5.1 下复现。
- 对比 PowerShell 7 的执行结果。
- 检查脚本编码和注释内容，确认是否为 UTF-8 无 BOM。
- 将脚本注释改为 ASCII 后，解析错误消失。

## Resolution
- 保持这些外部入口脚本尽量 ASCII 注释。
- 必要时改成 UTF-8 with BOM 或其他兼容编码。
- 保留 PowerShell 5.1 冒烟检查，避免回归。

## Prevention / Guardrails
- 新增或修改 `.ps1` 脚本时，优先避免中文注释。
- 为 Windows 入口脚本保留 PowerShell 5.1 smoke test。
- 如果必须保留非 ASCII 内容，显式确认脚本编码。

## Related Docs
- [docs/change/2026-05-28_win-mcp-shared-http-server.md](../change/2026-05-28_win-mcp-shared-http-server.md)
- [frontend-and-powershell-preflight.md](frontend-and-powershell-preflight.md)
