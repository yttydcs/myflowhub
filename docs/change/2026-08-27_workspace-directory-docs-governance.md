# 2026-08-27 工作区目录清理与文档治理

## Background

vNext 已完成单仓迁移和旧仓退役，但控制目录仍包含旧多仓 `go.work`、可再生成构建产物、本机临时配置，以及被旧版本覆盖的文档索引。需要把文件系统状态和 governed docs 同步到当前 monorepo 事实。

## Changes

- 清理约 183 MiB 的构建、测试依赖、运行日志、临时验证副本、旧仓代码副本和空目录。
- 删除指向已退役 `repo/MyFlowHub-*` 的 `go.work/go.work.sum` 及过期本机配置。
- 将 `附件/代码` 中 21 个旧 Go 模块移入回收站；保留 `_android-sdk`、附件中的非代码资料、`论文/` 与 `附件.zip`。
- 恢复 canonical `scripts/run-dev.ps1` 与 vNext `docs/specs/protocol_map.md`。
- 恢复被忽略规则误伤的 `build/toolchain.json`，修正单仓校验对根 `.git` 的误判，并消除链路测试管道并发关闭的竞态。
- 固定生成契约为 LF，保证 Windows 与 CI 对生成结果执行相同的字节级校验。
- 修复根文档入口、feature/spec/plan/change/lesson/intake 索引及稳定文档断链。

## Related Plan

- 无独立执行计划；本次为归档后的目录与文档治理收尾。

## Related Intake

- [工作区目录清理与文档治理](../intake/2026-08-27_workspace-directory-docs-governance.md)

## Related Features

- [功能索引](../features/README.md)

## Related Requirements

- [统一节点运行时](../requirements/unified-node-runtime.md)

## Related Specs

- [仓库与模块边界](../specs/repository-and-module-boundaries.md)
- [构建与 CI](../specs/build-and-ci.md)
- [协议映射](../specs/protocol_map.md)

## Related Decisions

- [使用单一 Canonical Monorepo](../decisions/2026-08-27_single-canonical-monorepo.md)

## Lessons Impact

- updated：合并 vNext 当前约束与旧多仓排障细节，移除指向已删除仓库和脚本的链接。

## Related Lessons

- [Authority 管理与节点树](../lessons/authority-local-admin-actions.md)
- [生成绑定与协议漂移](../lessons/wails-binding-proto-drift.md)
- [前端与 PowerShell 预检](../lessons/frontend-and-powershell-preflight.md)

## Searchable Lessons Summary

- 旧仓路径、根 `go.work` 和旧启动脚本不能继续作为单仓运行时的隐式回退。
- 关键词：`repo/MyFlowHub-*`、`go.work`、`protocol_map`、`toolchain.json`、`.gitattributes`、docs index、broken link。

## Intake Impact

- updated：新增本次整理请求记录。

## Feature Impact

- none：仅修复索引，不改变产品行为。

## Requirements Impact

- none：沿用统一节点运行时和单仓约束。

## Specs Impact

- updated：恢复 vNext 协议映射和工具链清单，固定生成契约换行，并修复规范索引。

## Decision Impact

- none：执行既有单仓决策。

## Validation

- 删除目标均经过根目录边界、Git 跟踪状态和进程占用检查。
- 分类索引覆盖所有直接叶子文档。
- 稳定文档相对链接检查无断链。
- `go test ./internal/archtest ./internal/migrationtest ./sdk/bindings` 通过。
- `go test ./runtime/link -count=20` 通过。
- `go test ./...` 与 `go vet ./...` 在无旧 `go.work` 的根模块下通过。

## Rollback

- 除 `tmp/` 的测试依赖与临时预览外，其余清理目标（包括 `附件/代码`）已进入 Windows 回收站。
- 被替换的旧脚本和旧协议映射仍保存在 stash `514135dd02428220825c07c211a47c725a4d98ad`。
- `tmp/` 仅包含可重新安装的测试依赖和临时预览，未保留恢复副本。

## Follow-up

- 2026-08-28 根据工作区规范，将完整 canonical checkout 迁入 `repo/MyFlowHub`；参见 [Canonical checkout 迁入 repo/MyFlowHub](2026-08-28_canonical-checkout-relocation.md)。
