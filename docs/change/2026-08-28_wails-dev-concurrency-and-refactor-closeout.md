# 2026-08-28 Wails 开发启动并发修复与重构收口

## 变更背景 / 目标

vNext 全量迁移已经完成，但根级 `scripts/run-dev.ps1` 同时启动 Desktop 和 Metrics Windows 时，两个 `wails dev` 会并发生成 bindings 或同步共享根 `go.mod`。Metrics 前端因此可能收到 Desktop 的 `main.App` 导出面并在 TypeScript 构建阶段失败。本次修复该竞态，运行完整本地门禁，并完成根级控制文档收口。

## 具体变更内容

- `scripts/run-dev.ps1` 在启动后台进程前，按 Desktop、Metrics 顺序同步执行 `wails generate module`。
- 两个并行 `wails dev` 使用 `-skipbindings -nosyncgomod -m`，不再竞争产品 bindings 或共享 module 状态。
- 脚本明确提示：修改 exported bound Go method 后需要重启根级开发入口刷新 TypeScript bindings。
- architecture test 固定串行生成顺序和两个受保护的 dev 启动参数。
- 更新跨产品 Wails bindings lesson，记录并发触发条件、识别方式和预防规则。
- 将误占根入口的 Clipboard body history 计划归档到 `docs/plan`，根 `plan.md` 恢复为 canonical 当前状态入口。

## Docs root

- `D:/project/MyFlowHub3/repo/MyFlowHub/docs`
- 文档与代码位于同一个 canonical Git 仓库；本次只进行本地收口，没有 push 或 publication。

## Intake impact

- none：没有新增或改变原始需求证据。

## Feature impact

- none：没有改变 Desktop、Metrics 或 Hub 的用户可见产品能力。

## Requirements impact

- none：没有改变统一节点运行时或产品迁移范围。

## Specs impact

- none：修复实现恢复既有 canonical build/generation 约束，没有改变协议、API、schema 或长期架构契约。

## Decision impact

- none：沿用单一 canonical monorepo 和产品 facade 隔离决策，无需新 ADR。

## Lessons impact

- updated：[Wails Bindings Across Product Facades](../lessons/wails-bindings-cross-project.md)。

## Related intake

- [节点树、订阅与指令重构诉求](../intake/2026-08-27_node-tree-subscription-command-redesign.md)
- [vNext 全量迁移与最终切换](../intake/2026-08-27_vnext-full-migration.md)

## Related features

- [Desktop](../features/desktop.md)
- [MetricsNode](../features/metrics-node.md)

## Related requirements

- [统一节点运行时需求](../requirements/unified-node-runtime.md)

## Related specs

- [Build and CI contract](../specs/build-and-ci.md)
- [Repository and module boundaries](../specs/repository-and-module-boundaries.md)

## Related decisions

- [使用单一 Canonical Monorepo](../decisions/2026-08-27_single-canonical-monorepo.md)

## Related lessons

- [Wails Bindings Across Product Facades](../lessons/wails-bindings-cross-project.md)
- [Generated Binding And Protocol Drift](../lessons/wails-binding-proto-drift.md)

## 对应 plan.md 任务映射

- CL01 / FM14：隔离两个产品的 Wails 生成阶段和共享 `go.mod` 写入面。
- CL02 / FM14：增加根级开发入口 architecture guard 和可复用 lesson。
- CL03 / FM15：运行完整 Go、vet、前端构建与真实多产品启动门禁。
- CL04 / FM16：恢复 canonical 根文档入口并归档旧 Clipboard 计划。
- CL05 / FM16：核对本地 merge/worktree 状态并完成 control-plane 收口。

## 经验 / 教训摘要

- 多个 Wails app 即使位于不同目录，只要共享一个根 Go module，就不能假设并发 `wails dev` 的生成和 module 同步完全隔离。
- “两个应用单独启动都成功”不能证明根级并发启动安全；必须验证组合入口。
- 对生成期共享状态，最简单可靠的边界是串行生成、并行只读运行。

## 可复用排查线索

- 症状：Metrics TypeScript 报 `Configuration`、`Definitions`、`Start`、`Status` 等 Go 中存在的方法未导出。
- 触发条件：Desktop 与 Metrics 在同一根 Go module 下并发执行 `wails dev`。
- 关键词：`has no exported member`、`wailsjs/go/main/App`、`skipbindings`、`nosyncgomod`、cross-project bindings。
- 快速检查：分别执行两个应用的 `wails generate module` 并比较 `App.d.ts`；若单独生成正确而并发启动错配，检查根启动脚本是否允许 Wails 进程同时生成或同步 module。

## 关键设计决策与权衡

- 选择每次根级启动前串行生成 bindings，而不是依赖已有生成文件，以保证 facade 与当前 Go 源一致。
- 并行 dev 阶段关闭 bindings、go.mod sync 和 mod tidy 写入，换取确定性。
- 代价是 exported bound Go method 变化后需要重启根脚本；脚本已输出显式提示，避免静默使用旧绑定。

## 测试与验证方式 / 结果

- PowerShell parser：`scripts/run-dev.ps1` 无语法错误。
- `go test ./internal/archtest -count=1`：通过。
- `go test ./... -count=1`：通过，覆盖 runtime、transport、features、Hub、SDK、产品和 integration。
- `go vet ./...`：通过。
- Desktop：Vitest 2/2、`tsc --noEmit`、Vite production build 通过。
- Metrics Windows：Vitest 2/2、`tsc --noEmit`、Vite production build 通过。
- 真实根级 smoke：Hub `127.0.0.1:7431`、Desktop dev server `35115`、Metrics dev server `35116` 同时监听；`MyFlowHub` 与 `MyFlowHub Metrics` 窗口均处于 responding 状态；两个 Wails 日志均显示前端和应用编译完成，无 binding/TypeScript 错误。
- `git diff --check`：通过。
- smoke 进程、端口、临时状态和可再生输出均已清理。
- 用户曾以 Escape 停止界面自动操作，因此最终组合回归使用进程、窗口响应、端口和日志证据，不声明额外的交互截图验收。

## 潜在影响

- 根级启动首次进入 dev 前会多执行两次串行 binding generation，启动时间略有增加。
- 运行期间修改 exported bound Go method 不会自动刷新 TypeScript binding，需要重启根脚本。
- DX01 新 Transport、远端发布和真实硬件/签名认证仍属于单独授权或外部环境工作，不影响本次本地重构完成结论。

## 回滚方案

- 回退 `scripts/run-dev.ps1`、对应 architecture test 和 Wails lesson 即可撤销本修复。
- 回滚后会重新暴露两个 Wails dev 进程并发改写 bindings/module 状态的风险，不建议仅为了恢复自动 binding 刷新而回退。
- 文档入口可通过删除本变更归档、将 Clipboard 计划移回根 `plan.md` 恢复，但会再次让根入口指向已退役仓库，不建议执行。

## 子Agent执行轨迹

- none：用户没有要求子 Agent；本次诊断、实现、验证和归档均由主 Agent直接完成。
