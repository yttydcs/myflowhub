# 2026-03-21 Win：对齐 Proto exec cap_query 基线以恢复构建

## 变更背景 / 目标
- 背景：`MyFlowHub-Win` 的 Flow 能力查询已接入 `ExecCapQuerySimple`，Go 侧 `internal/services/flow/service.go` 依赖 `protocolexec.CapQueryReq/Resp` 与 `ActionCapQuery*`。
- 问题：当前 `go.mod` 仍锁定 `github.com/yttydcs/myflowhub-proto v0.1.1`，该版本不包含上述协议类型，导致 `wails build` 在生成 bindings 阶段直接编译失败。
- 目标：在不回退 Win 现有能力查询功能的前提下，对齐到已存在的 Proto 基线，恢复 Wails 构建链路。

## 具体变更内容（新增 / 修改 / 删除）
### 修改
- `repo/MyFlowHub-Win/go.mod`
  - 将 `github.com/yttydcs/myflowhub-proto` 从 `v0.1.1` 升级到 `v0.1.2-0.20260318063708-7eef50dcc471`。
- `repo/MyFlowHub-Win/go.sum`
  - 更新对应 pseudo-version 的校验和。

### 新增
- 本文档
  - 记录本次依赖基线对齐、验证方式与回滚方案。

### 删除
- 无。

## 对应 plan.md 任务映射
- `WIN-MOD-ALIGN`
  - 完成 `go.mod/go.sum` 对齐到 Proto pseudo-version。
- `WIN-BUILD-VERIFY`
  - 完成 `go test ./...`、`npm run build`、`wails build -debug -skipembedcreate -nopackage`。
- `WIN-ARCHIVE`
  - 完成本归档文档。

## 关键设计决策与权衡
- 决策：不采用本地 `replace`，改为依赖可解析的 Proto pseudo-version `v0.1.2-0.20260318063708-7eef50dcc471`。
  - 原因：在恢复构建的同时，保持依赖来源可审计、可复现，避免把本地工作区目录结构耦入 Win 构建链路。
  - 权衡：该版本不是正式 tag，而是指向明确 commit 的基线；比本地 `replace` 更稳定，但仍需要后续正式发版来收敛。

## 测试与验证方式 / 结果
- 执行：`go list -m -json github.com/yttydcs/myflowhub-proto@7eef50d`
  - 结果：解析得到 `v0.1.2-0.20260318063708-7eef50dcc471`。
- 执行：`$env:GOWORK='off'; go test ./... -count=1 -p 1`
  - 结果：通过。
- 执行：`cd frontend && npm run build`
  - 结果：通过。
- 执行：`$env:GOWORK='off'; wails build -debug -skipembedcreate -nopackage`
  - 结果：通过，成功生成 `build/bin/myflowhub-win.exe`。

## 3.3 Code Review 结论
- 需求覆盖：通过
  - 已消除当前构建阻塞，且未删除 `ExecCapQuerySimple` 功能路径。
- 架构合理性：通过
  - 问题根因是协议依赖版本不匹配，修复点收敛在依赖基线，不扩散到前端或 Flow 逻辑。
- 性能风险：通过
  - 仅依赖版本升级，无新增运行时 I/O、循环或请求。
- 可读性与一致性：通过
  - 变更集中在 `go.mod/go.sum`，与问题根因直接对应。
- 可扩展性与配置化：通过
  - 后续可以直接切换到正式 tag，无需再改 Win 业务代码。
- 稳定性与安全：通过
  - 未放宽权限、未绕过协议校验、未引入本地 `replace`。
- 测试覆盖情况：通过
  - 已覆盖 Go 编译、前端构建与 Wails 整体构建链路。
- 子Agent治理与审计：通过
  - 本轮未使用子Agent；原因是关键路径高度耦合、无安全可拆分写集。

## 潜在影响与回滚方案
- 潜在影响：
  - 依赖从正式 tag 升级到 pseudo-version，后续若 Proto 发布正式版本，建议再做一次版本收敛。
  - `npm run build` 仍有前端 chunk size 警告，但不影响当前构建成功。
- 回滚方案：
  1. 将 `repo/MyFlowHub-Win/go.mod` 中 `myflowhub-proto` 恢复为 `v0.1.1`。
  2. 回退 `repo/MyFlowHub-Win/go.sum` 对应校验和。
  3. 接受构建重新回到当前已知失败状态，或改走本地 `replace` 临时方案。

## 子Agent执行轨迹
- 本轮未使用子Agent。
