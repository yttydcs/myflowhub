# 2026-02-21 · Workspace · 一键启动 Server + Win（本地验证脚本）

## 变更背景 / 目标

为了方便在本机快速做“启动 + 冒烟验证”，新增 workspace 级别脚本，一条命令即可分别打开两个窗口并启动：
- `MyFlowHub-Server`：`go run ./cmd/hub_server`
- `MyFlowHub-Win`：`wails dev`

该脚本 **不进入任何 repo 的 git**，仅用于当前 `d:\project\MyFlowHub3` workspace 的开发/验证效率提升。

## 具体变更内容

### 新增
- `scripts/run-dev.ps1`：启动脚本（支持参数化、可选等待端口就绪、输出冒烟步骤）。
- `scripts/plan.md`：该脚本的执行计划与验收标准（可交接/可审计）。

## plan.md 任务映射
- S1：定义脚本接口与默认行为（已完成）
- S2：实现脚本 run-dev.ps1（已完成）
- S3：归档变更（本文件）（已完成）

## 关键设计决策与权衡

1) **两个独立窗口输出日志（用户选择 3A）**  
   采用 `Start-Process pwsh -NoExit` 分别启动 Server 与 Win 的命令行窗口，避免一个窗口里混杂日志、也便于定位问题。

2) **Server 配置通过环境变量注入**  
   `hub_server` 支持 `HUB_ADDR/HUB_NODE_ID/HUB_PARENT_ADDR` 等环境变量；脚本按此注入，避免拼接复杂 flag。

3) **默认不强制 `GOWORK=off`**  
   开发联调通常希望使用 `go.work`；脚本提供 `-GoWorkOff` 作为可选开关，用于审计/复现时走纯 module 依赖。

4) **统一 `GOTMPDIR`**  
   默认设置为 `d:\project\MyFlowHub3\.tmp\gotmp`，减少临时目录权限/性能波动导致的构建失败概率。

## 使用方式（含冒烟验证）

### 启动
```powershell
cd d:\project\MyFlowHub3
.\scripts\run-dev.ps1
```

常用参数：
```powershell
.\scripts\run-dev.ps1 -ServerAddr ':9001' -ServerNodeId 2 -WaitServer
```

### 冒烟验证步骤
1) Win 首页 Address 填：`127.0.0.1:9000`（或你设置的端口）
2) Device ID 任意非空（例如 `dev-1`），点击 Connect
3) Presets → Node Echo → Send，期望提示成功，Logs 无明显错误

## 测试与验证方式 / 结果
- 脚本可运行（跳过实际拉起进程验证）：`pwsh -File scripts/run-dev.ps1 -SkipServer -SkipWin` ✅
- 帮助信息可用：`Get-Help .\scripts\run-dev.ps1 -Detailed` ✅

## 潜在影响与回滚方案

潜在影响：
- 无（仅新增 workspace 脚本，不影响任何仓库代码与运行逻辑）。

回滚：
- 删除 `scripts/run-dev.ps1` 与 `scripts/plan.md` 即可。

