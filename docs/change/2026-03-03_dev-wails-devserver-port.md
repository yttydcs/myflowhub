# 2026-03-03 · Workspace · 修复 Win 与 MetricsNode 同时 wails dev 的 DevServer 端口冲突（避免 Open Window 串台）

## 变更背景 / 目标

在本地使用 `scripts/run-dev.ps1` 同时启动：
- `repo/MyFlowHub-Win`（`wails dev`）
- `repo/MyFlowHub-MetricsNode/windows`（`wails dev`）

时，两端会尝试使用相同的 Wails DevServer 端口（例如 `127.0.0.1:34115`），导致其中一端启动报错：
- `listen tcp 127.0.0.1:34115: bind: Only one usage of each socket address ...`

进一步地，当 Win 侧 DevServer 未成功启动时，Win 内的 `Open Window`（例如打开 `wails.localhost:34115/#/...`）会实际命中 MetricsNode 的 DevServer，从而出现“打开的是 MetricsNode 界面”的串台现象。

目标：
- Win 与 MetricsNode 的 `wails dev` DevServer 端口不再冲突；
- 端口被占用时自动避让并提示最终端口；
- 避免 `Open Window` 串台到错误的 DevServer。

## 具体变更内容

### 修改

- `scripts/run-dev.ps1`
  - 新增参数：
    - `-WinDevServerPortStart`（默认 `34115`）
    - `-MetricsDevServerPortStart`（默认 `34116`）
    - `-DevServerPortSearchMax`（默认 `50`）
  - 启动 Win / MetricsNode 前，先探测并分配可用端口（占用则递增寻找下一个可用端口），并确保两端端口不重复。
  - `wails dev` 显式传入 `-devserver 127.0.0.1:<port>`，避免 Wails 自动选端口撞车。

## plan.md 任务映射

- `scripts/plan.md`
  - S8：`run-dev.ps1` 为 Win / MetricsNode 分配不同 DevServer 端口（自动避让）
  - S9：更新归档（本文件）

## 关键设计决策与权衡

1) **在脚本侧解决端口冲突（根因修复）**  
相比在前端 `window.open` 做绕行，DevServer 冲突才是导致“串台/绑定失败”的根因，优先在启动脚本中一次性解决。

2) **采用“起始端口 + 递增避让”的策略**  
默认保持端口稳定（Win `34115`、MetricsNode `34116`），当端口被占用时自动寻找下一个可用端口，减少人工干预；同时脚本会提示最终端口，便于排查。

3) **绑定地址固定为 `127.0.0.1`**  
DevServer 仅供本机开发调试使用，不对外暴露，保持安全默认。

## 测试与验证方式 / 结果

已验证：脚本可正常执行（不启动任何端时可正常输出帮助/冒烟步骤），新增参数可在 `Get-Help` 中看到。

待验证：端到端仍需在本机实际运行 `wails dev` 后确认两端 DevServer 端口不同、且 Win 的 `Open Window` 不再串台。

手工验证（端到端，建议执行）：

1) 运行：
   ```powershell
   cd d:\project\MyFlowHub3
   .\scripts\run-dev.ps1
   ```
2) 观察 Win 与 MetricsNode 两个窗口日志中 `Using DevServer URL`：
   - 预期两端端口不同；
   - 不再出现 `bind` 报错。
3) 在 Win 中触发 `Open Window`（例如 Showcase Viewer）：
   - 预期不再打开成 MetricsNode UI。

端口避让验证（可选）：
- 先手动占用 `34115`（或让其他进程占用），再运行脚本，预期 Win 会提示端口已避让并使用更高端口。

## 潜在影响与回滚方案

潜在影响：
- DevServer 端口在被占用时会发生变化；脚本会输出最终端口用于定位。

回滚：
- 回滚 `scripts/run-dev.ps1` 到本变更前版本（移除 `-devserver` 端口分配逻辑）即可恢复旧行为。
