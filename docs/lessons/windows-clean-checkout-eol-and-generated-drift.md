# Windows Clean Checkout EOL And Generated Drift

## Summary

Windows 上“现有 worktree 验证通过”不代表 fresh checkout 可复现。`core.autocrlf`、tracked generator output、最后换行、Gradle TEMP 和 JDK 选择可能只在新 clone 中同时暴露。最终门禁应从 commit 创建干净 checkout，运行 generator/build/startup 后再检查 tracked tree 是否保持干净。

## Lookup Hints

- Symptoms: `gofmt -l` 列出全仓、生成后 `git status` 变脏、相同 MD5 显示 diff、Gradle loopback/JavaVersion 失败。
- Keywords: `i/lf w/crlf`, `No newline at end of file`, `Unable to establish loopback connection`, `JavaVersion.parse`, `25.0.4.1`。
- Triggers: Windows `core.autocrlf=true`、Wails/Vite 重生成、长 sandbox TEMP、系统 Java 版本高于项目 Gradle/Kotlin 支持范围。
- Quick checks:
  - `git ls-files --eol -- <path>`
  - `gofmt -l <file>`
  - `git diff --ignore-space-at-eol`
  - generator/build 前后 `git diff --exit-code`
  - `java -version`、`$env:TEMP`、`$env:MFH_JAVA_HOME`、`$env:MFH_SHORT_TEMP`

## Symptoms

- fresh clone 的 core check 报几乎所有 Go 文件未格式化，但 `go test` 正常。
- `wails generate module`、Vite build 或 `run-dev.ps1` 成功后，`dist/`、`wailsjs/`、`package.json.md5` 显示 modified。
- MD5 文本相同，diff 只显示最后换行变化。
- Gradle 在启动 single-use daemon 前报 `Unable to establish loopback connection`。
- 使用过新的系统 JDK 时，Kotlin/Gradle 在 `JavaVersion.parse` 报类似 `25.0.4.1`。

## Impact

- CI 与本机结论不一致，开发者可能错误地全仓 gofmt 或提交大量 line-ending noise。
- generated freshness 守卫失真，run-dev 每次都污染 tracked tree。
- Android 门禁被误判为代码回归，或因 `AllowUnavailable` 使用不当被错误跳过。

## Trigger Conditions

- Git for Windows 配置 `core.autocrlf=true`，而仓库没有为 canonical source 定义 EOL。
- generator 输出被版本控制，但 source/output/fingerprint 的 EOL 或末尾换行合同不一致。
- Gradle 使用由宿主虚拟化出的过长 TEMP 路径。
- PATH 首个 Java 高于 Gradle/Kotlin 已验证版本。

## Root Cause

1. Go blob 是 LF，但 checkout filter 写成 CRLF；`gofmt` 比较 canonical bytes，因此正确报告差异。
2. Vite/Wails 从 CRLF input 生成混合 EOL，Git text heuristic 与 generator bytes 不一致。
3. fingerprint generator 对末尾换行敏感，文本值相同不等于 blob 相同。
4. Windows Gradle daemon 的 AF_UNIX/loopback 初始化对 TEMP 路径敏感；Gradle 8.7/Kotlin 组合不支持 Java 25 的版本串。

## Investigation Trail

1. 在原 worktree 运行单测只能证明语义，不足以证明 checkout filter。
2. 从 commit 创建独立 clone，并确认 `.git/` 为真实目录。
3. 用 `git ls-files --eol` 观察 `i/lf w/crlf`，再对单个文件运行 `gofmt -l`。
4. 修复 Go EOL 后，运行全产品门禁，分别处理短 TEMP 和 JDK 21。
5. generation/build/startup 后检查 tracked status，用 `--ignore-space-at-eol` 区分语义与 byte drift。
6. 为精确 generated trees 固定 LF，并让 generator 决定 fingerprint 的无尾换行 blob；从新 commit 再建 fresh worktree复验。

## Resolution

- `.gitattributes`：`*.go text eol=lf`。
- binding contract 与 Desktop/Metrics tracked `index.html`、`dist/**`、`wailsjs/**` 固定 LF。
- `package.json.md5` 作为 generator-owned raw fingerprint，不由编辑器补尾换行。
- Gradle 验证使用短的绝对 `MFH_SHORT_TEMP` 和兼容 JDK 17/21 的 `MFH_JAVA_HOME`，只作用于当前进程。
- 最终要求 generator、frontend build 和 run-dev 结束后 `git status` 仍为空。

## Prevention / Guardrails

- 新增受版本控制 generator output 时，同时定义 byte/EOL ownership 和 clean-generation gate。
- 任何“全仓未格式化”先检查 EOL，不要立即全仓 rewrite。
- 构建文档记录可选工具的显式 `UNAVAILABLE` 语义；缺工具与真实执行失败必须区分。
- 不修改全局 TEMP/JAVA_HOME 解决单项目问题，使用项目已定义的 per-process override。
- 高风险归档前至少一次从 commit 创建 fresh checkout，并在完整操作后验证 tracked clean。

## Related Docs

- [Build and CI](../specs/build-and-ci.md)
- [Repository boundaries](../specs/repository-and-module-boundaries.md)
- [本轮 change](../change/2026-08-28_reproducible-refactor-closeout.md)
- [Wails cross-project bindings](wails-bindings-cross-project.md)
- [Wails binding/proto drift](wails-binding-proto-drift.md)
