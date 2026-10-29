# A 阶段 Windows x64 部署与验证

当前为本地候选包，正式验收状态以同目录 manifest 和实施 review 为准。包名含 worktree 时，其完整源文件指纹见 source-files.json，base_commit 只是起点，不能将其当成已冻结的 N 版本或 N−1。未包含任何准入 Permit、设备私钥或可直接使用的业务凭据。

## 设备与文件

正式环境为两台 Windows 11 x64、同一局域网 TCP：A 启动 Hub 和 Desktop，B 启动 Metrics 及独立负载示例。可以先在同一台机器用不同目录、不同身份、独立进程验证功能；本机结果不能代替两台物理设备、设备音频输出或 2 小时正式容量验收。

分别解压 hub、desktop、metrics、variable-example ZIP。Hub 包含 mfh-hub.exe 和 mfh-admin.exe；Desktop 包含 mfh-desktop.exe；Metrics 包含界面 mfh-metrics.exe 和可选的 mfh-metrics-console.exe，二者使用同一产品状态时只能运行其中一个；示例包含 mfh-variable-example.exe 与提供者源码。完整同版本可构建源码位于 source-*.zip，构建入口是 scripts/mfh.ps1。

运行不需要 Go、Node 或 Wails。两个图形界面依赖 Microsoft Edge WebView2 Runtime x64；缺少时从[微软下载页](https://developer.microsoft.com/en-us/microsoft-edge/webview2/)选择 Evergreen Standalone Installer 的 x64 并安装，再启动应用；运行时检测和部署方法见[微软分发说明](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/distribution)。Hub、管理工具和示例是独立 Go EXE。音频验收需要 B 存在可用的 Windows 默认音频输出端点，普通用户必须有对应设备操作权限。

为四个身份分别选用绝对状态目录，例如 A 的 Hub、Desktop，B 的 Metrics、Example；目录不得相同、不得放进安装目录或可公开同步的资料夹。Desktop 默认用户配置根为 `%APPDATA%\MyFlowHub\desktop-resource-workspace`，可在启动进程前设置 `MFH_DESKTOP_CONFIG_DIR` 指向隔离绝对目录。Metrics 界面“连接与身份”里显式选择自己的状态目录；示例通过 -state 指定。Hub 仅在 A 的 LAN 地址/端口监听，并按实际环境开放该 TCP 端口。

Metrics 图形版另在 `%APPDATA%\MyFlowHub\metrics` 保存公开连接参数和上次连接意图；可用 `MFH_METRICS_CONFIG_DIR` 指向独立绝对目录。成功连接后关闭窗口，下次打开自动使用原状态连接；主动点击“停止节点”后重开仍保持停止。Permit 不保存到此配置。身份目录丢失或配置损坏会明确报错，应恢复原状态或检查参数，不能靠自动生成新身份绕过。备份、升级及回滚时同时保留此 GUI 配置根和实际节点状态根。`b1fcd1a` 的 Metrics GUI 不读取该连接配置；手工重填参数不满足 A 已确认的保存状态自动恢复标准，不能作为无损回滚通过的依据。

## 首次引导和准入

1. 保持 Hub 停止，运行 `mfh-hub.exe -state <Hub绝对状态目录> -id 1 -identity`，记录公钥。需要改变 ID 时在首次创建前决定；后续保持原身份目录和 ID。
2. 首个管理员运行 `mfh-admin.exe -state <Admin绝对状态目录> -op identity`，取得 device_public_key。在停止的 Hub 上执行 `mfh-hub.exe -state <Hub状态目录> -id 1 -issue-public-key <Admin设备公钥> -issue-target-id 1`，将返回 JSON 安全保存为管理员 Permit。
3. 启动 `mfh-hub.exe -state <Hub状态目录> -id 1 -listen <A的LAN地址:7331>`。管理员执行 `mfh-admin.exe -state <Admin状态目录> -op enroll -endpoint <A地址:7331> -parent-id 1 -parent-key <Hub公钥> -authority-key <Hub公钥> -permit <AdminPermit路径>`，从 Grant 记录已分配 Node ID。
4. 停止 Hub，执行 `mfh-hub.exe -state <Hub状态目录> -id 1 -policy bind -subject <AdminNodeID> -definition superadmin -policy-scope authority-domain` 建立首个管理员，再以原状态启动 Hub。这是显式引导，仅管理员拥有该 Binding，业务 Desktop 不使用它。
5. Desktop 首次连接中准备 Profile/设备公钥，填写 Hub endpoint，选择 Permit 准入或 Pending 审批。Permit 方式按管理员相同步骤为 Desktop 的设备公钥签发新的 Enrollment Permit；连接界面提交 Permit 后记录分配的 Desktop Node ID，核对并固定父/Authority 指纹。
6. Metrics 界面准备独立身份、公钥及配置，使用独立未占用 Node ID。显式 ID 产品的兼容准入：在停止的 Hub 上执行 `mfh-hub.exe -state <Hub状态目录> -id 1 -issue-node-id <MetricsNodeID> -issue-public-key <Metrics公钥>`，把返回的 Legacy Permit 提供给 Metrics，恢复 Hub 后连接。此 Permit 类型与步骤2的 Enrollment Permit 不可互换。
7. 示例同样先用 `mfh-variable-example.exe -state <Example状态目录> -id <ExampleNodeID> -identity` 取得公钥，按步骤6签发 Legacy Permit，然后在 B 运行 `mfh-variable-example.exe -state <Example状态目录> -id <ExampleNodeID> -parent-id 1 -parent-key <Hub公钥> -endpoint <A地址:7331> -permit <ExamplePermit路径> -count 1998`。示例不操作实际音频设备。

准入只证明身份和网络成员资格；此时 Desktop 读取业务资源仍应明确被拒绝，必须再授予业务权限。Permit 有有效期和使用边界，状态与实际拒绝以服务器响应为准，不能修改身份或清空目录绕过。

## 明确授权

少量精确权限可在 Hub 停止时使用 `-policy grant -subject <DesktopNodeID> -action <read/subscribe/write/list> -resource-node <ownerID> -resource <资源名>`；`-policy revoke` 收回相同元组。正常在线管理使用已授予 superadmin 的独立管理员通过管理资源操作，不并发直接编辑服务器文件。

2000个变量不能逐个堆叠超过4096条的 exact Grant。应创建窄 Definition，再给指定 owner 的 Desktop 建立 Binding；不要给 Desktop superadmin 或全域 all 权限。示例 Definition 请求可保存为 definition.json：

```json
{"version":1,"id":"phase-a.example","label":"A示例变量","rules":[{"resource":{"kind":"exact","value":"system/catalog"},"capability":{"kind":"exact","values":["list","read"]}},{"resource":{"kind":"prefix","value":"example/foundation"},"capability":{"kind":"exact","values":["read","subscribe","write"]}}]}
```

由管理员在线调用：

```powershell
./mfh-admin.exe -state <Admin目录> -endpoint <A地址:7331> `
  -op operate -owner-id 1 -resource system/policy/definitions -capability create `
  -schema mfh.policy.definition-put.v1 -request-file <definition.json路径>
```

定义的能力列表需排序，服务器仍会按每个实际资源声明的能力拒绝不支持的写入。创建 Binding 请求使用 `mfh.policy.binding-create.v1`，目标 `system/policy/bindings` 的 create。例如填写已取得的 ID 和本地文件绝对路径后运行：

```powershell
$desktopNodeID = '<Desktop的NodeID>'
$exampleNodeID = '<Example的NodeID>'
$bindingFile = '<binding.json绝对路径>'
$bindingRequest = @{ version=1; binding_id=[Guid]::NewGuid().ToString('N'); subject=$desktopNodeID; definition_id='phase-a.example'; scope=@{kind='owner';node_id=$exampleNodeID} }
[IO.File]::WriteAllText($bindingFile, ($bindingRequest | ConvertTo-Json -Depth 5), [Text.UTF8Encoding]::new($false))
./mfh-admin.exe -state <Admin目录> -endpoint <A地址:7331> `
  -op operate -owner-id 1 -resource system/policy/bindings -capability create `
  -schema mfh.policy.binding-create.v1 -request-file $bindingFile
```

保留返回的 Binding ID。撤销时向同一资源的 revoke capability 提交 `mfh.policy.binding-revoke.v1` 的 `{version:1,binding_id:<返回ID>}`；撤销后再检查是否存在其他重叠 Grant/Binding。

Metrics 使用另一个 owner Binding，规则仅包括其 system/catalog、`metrics/volume_percent` 和 `metrics/volume_muted` 的 read/subscribe/write；先在 Metrics 的采集策略中启用这两个资源、允许其 write 并设为1000ms。Hub 拓扑发现单独授予 `system/topology` 的 children，确需完整子树时再明确授予 subtree。不要把目录中存在某个能力当作已经获权。

## 日常界面与失败处理

Desktop 从已授权拓扑找到 Metrics/示例，加载目录，在资源总览订阅所选2000个业务变量；滚动和筛选保持屏幕外订阅。检查器或 View 用标准控件编辑并提交。音量20%、40%及恢复初值允许误差1个百分点，静音精确切换并恢复；正常提交至实际确认、成功界面和独立实际回读不超过2秒，本地改变到远端显示不超过3秒。

5秒没有可靠结果时显示“不确定”，保留草稿并先刷新确认；收到可靠成功但回读失败单独提示。禁止自动写重试、断网补发和重启重放。撤 write 后服务端拒绝新写但保留有效读取；再撤 read/subscribe 后已有订阅/界面3秒内失效，旧值不可继续标为实时。重新授权后刷新/重订阅，由用户决定新的提交。

Desktop 为 5 秒可见反馈上限预留界面处理时间：原生写前读取与发送共用 4 秒预算，前端整次等待（含独立回读）最多 4.5 秒。若原生端确认写前读取失败、尚未发送写入，显示已拒绝；无法确定是否已发送/执行时显示不确定。迟到回复不覆盖已结束的提交，不会重新启动计时或自动重发。

正式网络检查分别中断 Desktop—Hub 和 Metrics—Hub，主动断开与静默丢包各5轮；5秒内识别，实际双向可达起10秒内恢复最新状态。单独正常退出/强杀/手动重启三个产品，同一身份及配置启动起15秒内恢复；正常退出未保存内容应提示并可取消。

## 升级、回滚和移除

先停止目标产品，备份其完整状态及用户配置，并保留 Windows ACL、加密保护方式和同一 Windows 用户上下文。DPAPI 保护的 Desktop 身份不应当作可移植明文凭据。不要公开上传状态、私钥、Permit 或 token；公开证据只包含版本、哈希、计时、计数和脱敏结果。

升级时在新目录解压 N，保留原产品状态路径，手动启动新 EXE。仅在目标版本已经通过对应产品的保存状态恢复验证后执行回滚，并继续读取**最新已提交状态和最新撤权**；不能恢复旧策略备份来让已经撤销的权限复活。兼容测试基线为 b1fcd1ac0319ebb349c8289cc6f61ed2d7302dc8，但它不是全产品已通过的回滚版本：2026-10-29 原生验证发现其 Metrics GUI 启动后未恢复已有连接，因此目前禁止将该 GUI 作为 A 的受支持回滚目标。保留当前 GUI、原 GUI 配置根及节点状态；若已替换，恢复当前 GUI 程序即可重新读取原状态，不恢复过期策略备份。此限制是未通过项，不降低验收要求。正式 N 仍需另行锁定干净候选 commit、生成物和包哈希；当前 working-tree 预览不充当该版本。

2026-10-29 后续本机验证采用真实提交 `6cd785a7b179fa899a9fd70c7d48188161d1fb58` 作为当前预览的回滚基线，包含 Metrics GUI 连接恢复。该基线与本轮修复版 Desktop 的独立切换已验证保存身份、配置、View 和最新撤权保留，六项替换均在 15 秒内恢复。Hub、Metrics 在这两个源码快照中的二进制相同，其结果仅证明独立替换恢复，不证明不同二进制间的新协议/数据迁移兼容。旧 `b1fcd1a` 的 GUI 失败仍保留；历史后端兼容结果不扩大上述支持窗口。正式版本冻结、F01 未通过历史项与 F04 双机/干净环境仍单列，详见活动验收记录。

默认卸载仅停止进程并移除该产品解压目录，保留状态；再次部署指向原状态可恢复。显式清除时先核对绝对路径只属于该产品，再由用户删除其对应状态/Profile；不删除其他产品目录。删除本地身份不等于撤销网络准入，还需管理员收回对应准入/Binding。

Hub/Desktop/Metrics 的 N/N−1 八组合须分别记录连接、准入、读取、订阅及一次授权控制结果；只有 N/N/N 承担2000订阅的完整容量标准。配置写入或模拟值测试不能冒充 Windows 实际音量/静音验收。

链路心跳兼容：b1fcd1a产品不发送周期心跳。当前产品每秒发送心跳，子连接在认证完成后立即发送首个心跳，对端首次收到时回应一次，避免刚连接即静默中断时漏开检测。收到已认证同一对端、当前拓扑epoch的心跳后，才启用该链路3秒空闲检测，启用后不会因沉默降回兼容状态。未发送心跳的旧版对端仍可连接、读取、订阅和控制，但不承诺该混合链路的新版静默故障识别时限。5秒识别、10秒恢复与完整故障矩阵必须在全新版候选上验证；不能将旧版兼容路径用于放行该门槛。底层显式配置IdleTimeout的非产品用户仍保持原语义，只有产品选择此兼容行为。

## 测试与证据

本机：从完整源码运行 `go test ./... -count=1 -timeout=180s`、`go vet ./...`、Desktop 前端 `npm test`；本机 TCP 测试覆盖容量、撤权、20轮并发/实例替换和每路径每故障5轮恢复。图形预览使用模拟数据，只验证布局、筛选、后台订阅及输入交互。

正式：10分钟预热后健康运行2小时，各变量1Hz，分散至少100次合法写入。按产品及专属子进程记录 private committed bytes，前后10分钟中位数增长不得超过 max(25%,64MiB)，并记录连接/兴趣/待处理请求回落。实际 B 音频状态由独立观察者回读，不能仅凭客户端显示判定。记录系统构建号、候选与包SHA-256、逐次单调计时和原始日志，最终恢复原音量/静音及本次创建的测试网络规则。


本机长一些的参考流量可复用同一用例（PowerShell）：

```powershell
$env:MFH_PHASE_A_LOCAL_LOAD_SECONDS='60'
go test ./tests/integration -run '^TestPhaseAReferenceTraffic$' -count=1 -v -timeout=180s
Remove-Item Env:MFH_PHASE_A_LOCAL_LOAD_SECONDS
```

该用例是单测试进程内多节点真实TCP和生产Desktop SDK通路，不包含原生界面或实际音频。可将秒数设为7200并相应把Go timeout设为7500s作本机长时诊断；这仍不自动满足正式两设备、10分钟预热和产品进程私有提交中位数验收。版本后端矩阵通过 `MFH_PHASE_A_BASELINE_ROOT` 指向真实b1fcd1a导出源码后运行 `scripts/mfh.ps1 -Action test -Target compat`，报告明确标注Desktop绑定适配器和配置控制范围。
