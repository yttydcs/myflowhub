# Todo - 重构可复现性与文档治理收口

## Context

- Branch: `refactor/reproducible-closeout`
- Base: `master@078f4b75ee8316b0baaed67c4be4d6f7fbf301d9`
- Worktree: `D:/project/MyFlowHub3/worktrees/reproducible-closeout`
- Detailed plan: [plan.md](plan.md)
- Stage: `3.3 heavy validation complete; archive ready`
- Authorization: 用户明确要求完整跑完 workflow；仅授权 RC01-RC04 和默认本地 archive/merge/cleanup。

## Will Execute

- [x] RC01 — 冻结并审计选入/排除来源
- [x] RC02 — 修复 canonical HEAD 构建与测试可复现性（补充 tracked frontend generated EOL/MD5 契约）
- [x] RC03 — 导入并治理旧仓提取文档和 checkout 路由
- [x] RC04 — 运行当前 worktree 与全新 checkout 完整本地门禁

## Will Not Execute Now

- [ ] NX01 — 论文、附件和无关主 checkout dirt
- [ ] NX02 — 新 serial/USB/WebSocket Transport
- [ ] NX03 — legacy compatibility bridge
- [ ] NX04 — push/release/sign/publish/remote archive
- [ ] NX05 — 外部硬件、签名平台和商店认证

## Gate

- Blocked: no
- Enter archive
- Do not dispatch implementation sub-agents
