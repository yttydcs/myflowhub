# 2026-03-27_win-project-meta-uuid

## 变更背景 / 目标
- 背景:
  - `repo/MyFlowHub-Server/docs/specs/flow.md` 已明确要求 `flow_id` 为 UUID。
  - Win 端项目中心在 `frontend/src/stores/flowProjects.ts` 中默认生成的是 `fl_` 前缀随机串，导致“本地可创建，部署时才因 `flow_id` 非 UUID 被拒绝”。
- 目标:
  - 让 Win 端默认生成的 `flowId` 与 Server 稳定契约对齐。
  - 将非法 `flowId` 的失败前移到本地写入/部署前，而不是等远端报错。

## 具体变更内容
### 修改
- `frontend/src/stores/flowProjects.ts`
  - 新增 `uuidPattern`、`isUUIDLike(...)`、`normalizeFlowID(...)`、`makeUUID(...)`、`ensureFlowIDFormat(...)`。
  - `makeFlowID(...)` 改为生成 UUID，不再生成 `fl_` 前缀随机串。
  - `ensureUniqueFlowID(...)` 升级为统一处理“非空 + UUID 格式 + 本地唯一”。
  - `createProject(...)`、`updateProjectMeta(...)`、`saveProjectPayload(...)`、`deployProject(...)` 全部复用统一校验。
  - 部署前的 overwrite 检查改为用规范化后的 `flowId` 比较，避免大小写差异导致误判。
- `frontend/src/stores/flowProjects.test.ts`
  - 新增 store 级回归测试，覆盖 UUID 自动生成、非法元数据编辑、非法 payload 回写、历史坏数据部署前拦截。

### 删除
- 删除旧的随机 token 生成路径，不再使用 `fl_` 形式默认 `flowId`。

## Requirements impact
- none

## Specs impact
- none

## Lessons impact
- none

## Related requirements
- `repo/MyFlowHub-Server/docs/requirements/flow_data_dag.md`

## Related specs
- `repo/MyFlowHub-Server/docs/specs/flow.md`

## Related lessons
- none

## 对应 plan.md 任务映射
- `UUID-1`
  - `frontend/src/stores/flowProjects.ts`
  - 收敛 UUID 生成与格式校验 helper
- `UUID-2`
  - `frontend/src/stores/flowProjects.ts`
  - 接入 create/update/payload/deploy 四条路径
- `TEST-1`
  - `frontend/src/stores/flowProjects.test.ts`
  - `npm test -- flowProjects`
- `REVIEW-1`
  - stage 3.3 checklist 复核
- `ARCHIVE-1`
  - 本文档
  - `docs/change/README.md`

## 经验 / 教训摘要
- 客户端默认生成的标识符不能只追求“看起来友好”，必须与服务端稳定契约一致，否则会形成延迟失败。
- 对历史坏数据更安全的策略通常不是“加载时自动改写”，而是“加载兼容、写入和部署前显式阻断”。

## 可复用排查线索
- Symptoms:
  - 新建本地 Flow 项目后，部署时报 `flow_id` 非 UUID 或本地提示 `Flow ID must be a UUID.`
- Trigger Conditions:
  - 项目由旧版 Win 自动创建，默认 `flowId` 仍是 `fl_...`
  - 用户手工在 Meta 中输入了非 UUID
  - 编辑器 payload 回写了非 UUID 的 `flow_id`
- Keywords:
  - `flow_id`
  - `uuid`
  - `fl_`
  - `Flow ID must be a UUID`
  - `flowProjects makeFlowID`
- Quick Checks:
  - 检查本地项目记录中的 `flowId` 是否仍为 `fl_` 前缀
  - 搜索 `frontend/src/stores/flowProjects.ts` 中的 `makeFlowID`、`ensureUniqueFlowID`
  - 复现路径：Create Project -> Deploy

## 关键设计决策与权衡
1. 不自动迁移历史非 UUID `flowId`
   - `flowId` 是部署身份，自动改写会改变后续覆盖/删除目标，风险高于收益。
2. 只在写入与部署前阻断，不在加载时丢弃历史项目
   - 保持本地数据可见、可修复，避免升级后用户项目直接消失。
3. 对通过校验的 `flowId` 统一规范化为小写
   - 便于本地唯一性判断和 overwrite 检查保持一致。

## 测试与验证方式 / 结果
- `frontend`
  - 执行：`npm test -- flowProjects`
  - 结果：通过（1 个测试文件，4 个测试用例）
- `frontend`
  - 执行：`npm run build`
  - 结果：失败
  - 原因：仓库基线缺失 `../../wailsjs/go/main/App`，错误出现在 `src/windows/TopicBusWindow.vue`，与本次 `flowProjects` 修复无关
- worktree
  - 执行：`git diff --check`
  - 结果：通过（仅有 Git 行尾转换 warning，无 whitespace/blocking issue）

## 潜在影响
- 历史非 UUID 项目现在会更早在本地报错，用户需要先修正 `flowId` 再部署。
- 手工输入的大写 UUID 会被规范化存储为小写。

## 回滚方案
- 回退以下文件即可恢复旧行为：
  - `frontend/src/stores/flowProjects.ts`
  - `frontend/src/stores/flowProjects.test.ts`
  - `docs/change/2026-03-27_win-project-meta-uuid.md`
  - `docs/change/README.md`

## 子Agent执行轨迹
- 本次未使用子Agent。
