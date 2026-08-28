# 2026-04-04 Win：修复空 `node_modules` 目录导致的前端构建误判

## 变更背景 / 目标
- 用户在 `repo\MyFlowHub-Win` 遇到 Wails 前端构建失败：
  - `Installing frontend dependencies: Done.`
  - 随后 `npm run build` 报 `Cannot find module '...frontend\\node_modules\\vite\\bin\\vite.js'`
- 复现确认后，当前仓内存在一个关键坏态：
  - `frontend/node_modules` 目录存在，但内容为空
  - 在这个状态下，Wails 仍会继续进入 `npm run build`
- 本次目标是让仓内前端构建入口在遇到空或残缺的 `node_modules` 时自动自愈，而不是继续误报“安装已完成”。

## 具体变更内容
- `frontend/package.json`
  - `dev` / `build` / `preview` 入口不再直连 `node_modules/vite/bin/vite.js`
  - 统一改为 `node ./scripts/run-vite.mjs <mode>`
- `frontend/scripts/run-vite.mjs`
  - 新增仓内前端构建守卫脚本
  - 在启动 Vite 前检查关键依赖文件：
    - `node_modules/vite/bin/vite.js`
    - `node_modules/@vitejs/plugin-vue/package.json`
    - `node_modules/@babel/parser/package.json`
  - 若缺失则执行一次 `npm install`
  - 自愈后仍缺文件时显式失败
  - `build` 成功后补回 `frontend/dist/placeholder.txt`
- `frontend/scripts/run-vite.test.mjs`
  - 新增脚本级测试
  - 覆盖缺失检测、自愈路径、失败显式报错和 placeholder 回写

## Requirements impact
`none`

## Specs impact
`none`

## Lessons impact
`updated`

## Related requirements
- `none`

## Related specs
- `none`

## Related lessons
- `docs/lessons/frontend-build-empty-node-modules.md`
- `repo/MyFlowHub-Win/docs/lessons/frontend-build-babel-parser-missing.md`
- `docs/lessons/wails-embed-dist-placeholder.md`

## 对应 plan.md 任务映射
- `WIN-BUILD-GUARD-1`
  - `frontend/package.json`
  - `frontend/scripts/run-vite.mjs`
  - `frontend/scripts/run-vite.test.mjs`
- `WIN-BUILD-GUARD-2`
  - `npm install`
  - `npm run test -- scripts/run-vite.test.mjs`
  - `npm run build`
  - 空 `frontend/node_modules` 目录下的 `wails build -debug -skipembedcreate -nopackage`
- `WIN-BUILD-GUARD-3`
  - 本归档
  - `docs/lessons/frontend-build-empty-node-modules.md`
  - `docs/lessons/README.md`
  - `docs/change/README.md`

## 经验 / 教训摘要
- 对当前 Win 前端构建链而言，`node_modules` “目录存在”不等于“构建依赖可用”。
- 这个问题与 `@babel/parser` 缺失、`frontend/wailsjs/**` 缺失、`dist/placeholder.txt` 丢失属于不同层级；不能混成一个排查分支。
- 最小安全修复不是让每次构建都 `npm ci`，而是在仓内入口只对缺失状态做一次自愈。

## 可复用排查线索
- 症状
  - `Installing frontend dependencies: Done.`
  - `Cannot find module '...frontend\\node_modules\\vite\\bin\\vite.js'`
  - 失败点出现在 `npm run build`
- 触发条件
  - `frontend/node_modules` 目录存在，但内容为空或缺少关键构建依赖
- 关键词
  - `vite/bin/vite.js`
  - `Installing frontend dependencies: Done.`
  - `empty node_modules`
  - `package.json.md5`
  - `wails build`
- 快速检查
  - 检查 `frontend/node_modules` 是否只是空目录
  - 检查 `frontend/node_modules/vite/bin/vite.js` 是否存在
  - 检查 `frontend/node_modules/@vitejs/plugin-vue/package.json` 是否存在
  - 检查 `frontend/node_modules/@babel/parser/package.json` 是否存在

## 关键设计决策与权衡
- 采用仓内 `run-vite.mjs` 守卫脚本，而不是继续把所有责任留给 Wails 安装阶段
  - 原因：当前问题稳定发生在“Wails 进入 `npm run build` 之前未修复空目录”，仓内入口更可控
- 不把 `wails.json` 改成 `npm ci`
  - 原因：`npm ci` 会把每次构建放大成全量 reinstall，I/O 成本过高
- 只在缺失时 install
  - 原因：正常构建路径应该保持轻量，不引入不必要的重复安装

## 测试与验证方式 / 结果
- `npm install`
  - 结果：通过
- `npm run test -- scripts/run-vite.test.mjs`
  - 结果：通过
  - 说明：`5` 个脚本级测试全部通过
- `npm run build`
  - 结果：通过
- 手工构造空 `frontend/node_modules` 目录后执行 `$env:GOWORK='off'; wails build -debug -skipembedcreate -nopackage`
  - 结果：通过
  - 说明：用户原始 `vite.js` 缺失崩溃不再复现

## 潜在影响
- `dev` / `build` / `preview` 现在都依赖仓内守卫脚本；后续若升级 Vite 或 Vue 编译链，需要同步维护脚本中的关键依赖清单。
- 当前修复只覆盖“空或残缺的依赖目录”这一层问题；若是 `wailsjs` 绑定缺失、embed placeholder 丢失或其它业务模块解析错误，仍需按对应 lessons 排查。

## 回滚方案
- 回退以下文件即可撤销本轮修复：
  - `frontend/package.json`
  - `frontend/scripts/run-vite.mjs`
  - `frontend/scripts/run-vite.test.mjs`
  - `docs/change/2026-04-04_win-frontend-empty-node-modules-guard.md`
  - `docs/lessons/frontend-build-empty-node-modules.md`
  - `docs/change/README.md`
  - `docs/lessons/README.md`

## 子Agent执行轨迹
- 本轮未使用子 Agent。
