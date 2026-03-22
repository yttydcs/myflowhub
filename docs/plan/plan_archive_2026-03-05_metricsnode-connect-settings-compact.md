# Todo - MetricsNode Connect 权限修复 + Settings 紧凑化（Android）

- Worktree: `d:\project\MyFlowHub3\worktrees\fix-metricsnode-connect-permission-ui`
- Branch: `fix/metricsnode-connect-permission-ui`
- Date: 2026-03-04
- 状态: 已完成（待用户确认是否结束 workflow）

## 项目目标与当前状态

### 目标
1. 修复 Connect 点击后出现 `dial tcp 127.0.0.1:9000: socket: operation not permitted` 的阻塞问题。
2. 优化 Settings 页面组件过大问题，在不改变功能的前提下实现紧凑布局。

### 当前状态
- 已定位 Android 连接异常根因候选：`AndroidManifest.xml` 缺少 `android.permission.INTERNET`。
- Settings 页面位于 `MainActivity.kt`，当前使用默认 `OutlinedTextField` + `Switch` 组合，控件高度与视觉占用偏大。

## 可执行任务清单（Checklist）

- [x] T1 修复 Connect 网络权限
- [x] T2 Settings 页面紧凑化
- [x] T3 本地构建验证
- [x] T4 Code Review（3.3）
- [x] T5 归档 docs/change（4）

## 任务详情

### T1 修复 Connect 网络权限
- 目标:
  - 让 Android 具备基础 TCP 联网权限，消除 `socket: operation not permitted`。
- 涉及模块/文件:
  - `android/app/src/main/AndroidManifest.xml`
- 验收条件:
  - Manifest 包含 `android.permission.INTERNET`。
  - Connect 操作不再因权限缺失直接返回 `operation not permitted`。
- 测试点:
  - 编译通过。
  - 运行后点击 Connect，错误类型从权限错误转为真实连接结果（连接成功或业务侧连接失败提示）。
- 回滚点:
  - 移除新增权限声明（单文件回滚）。

### T2 Settings 页面紧凑化
- 目标:
  - 缩小 Settings 关键组件视觉体积（输入框、开关、行间距、字号），保留现有交互与数据流。
- 涉及模块/文件:
  - `android/app/src/main/java/com/myflowhub/metricsnode/MainActivity.kt`
- 验收条件:
  - 页面结构仍为 `Metric / Var Name / Value / Enabled / Writable`。
  - 可读性不下降，编辑/切换/保存逻辑保持一致。
  - 组件视觉尺寸较当前明显收敛。
- 测试点:
  - 构建通过。
  - 手动检查：输入 var_name、切换 enabled/writable、保存状态与错误提示正常。
- 回滚点:
  - 回退 `MainActivity.kt` 中 Settings 相关样式变更。

### T3 本地构建验证
- 目标:
  - 确认代码变更不破坏 Android 构建。
- 涉及模块/文件:
  - Android Gradle 构建系统
- 验收条件:
  - `:app:assembleDebug` 成功。
- 测试点:
  - 记录构建命令与结果。
- 执行记录:
  - `ANDROID_HOME=d:\\project\\MyFlowHub3\\_android-sdk`
  - `android\\gradlew.bat :app:assembleDebug`
  - 结果：`BUILD SUCCESSFUL`
- 回滚点:
  - 按任务粒度回退至最近可编译版本。

### T4 Code Review（阶段 3.3）
- 目标:
  - 对需求覆盖、架构、性能、可读性、扩展性、稳定性/安全、测试覆盖进行逐项评审并出结论。
- 涉及模块/文件:
  - 本次所有改动文件
- 验收条件:
  - 各项结论清晰；若不通过必须回到 3.2 修正。
- 回滚点:
  - 根据 review 结论回退对应提交/文件。

### T5 归档 docs/change（阶段 4）
- 目标:
  - 形成可审计变更文档，支持后续交接。
- 涉及模块/文件:
  - `docs/change/2026-03-04_metricsnode-connect-settings-compact.md`
- 验收条件:
  - 文档包含背景/目标、改动内容、任务映射、设计权衡、测试结果、影响与回滚方案。
- 回滚点:
  - 删除或重写归档文档。

## 依赖关系
- T1、T2 可并行，但为降低风险先完成 T1（阻塞问题）再做 T2（体验问题）。
- T3 依赖 T1/T2。
- T4 依赖 T3。
- T5 依赖 T4 通过。

## 风险与注意事项
1. 增加 `INTERNET` 仅放开联网能力，不改变业务访问控制；需继续依赖后端认证。
2. Settings 紧凑化若过度，可能影响点击命中和可读性，需要在紧凑与可用性间平衡。
3. 若目标地址仍为不可达地址（例如设备上 `127.0.0.1` 无服务），仍会连接失败，这是业务连接结果而非权限错误。
