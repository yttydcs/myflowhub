# 六类型标准变量提供者

此示例只新增提供者：依赖现有 NodeHost、标准 Variable v2 / value v1 / write v3，不新增核心 schema、Desktop renderer 或 Metrics 专用适配。所有 `example/foundation/*` 数据均明确标记为模拟负载，不能作为真实设备读回证据。

默认 6 个变量；`-count 1998` 提供 bool、number、enum、text、object、array 各 333 个。布尔与数字支持标准 CAS 写入，其他类型只读。每秒采样一次，布尔/数字采样保留最近确认的写入值；只读值按计数确定变化。短文本不超过 128 字节，对象/数组不超过 256 字节，数组不超过 8 项。

## 构建

从随候选交付的同版本完整仓库根目录运行，工具链见 [build/toolchain.json](../../build/toolchain.json)：Go 1.24.x；构建此 CLI 无需 Node、Wails 或额外业务插件。

```powershell
./scripts/mfh.ps1 -Action build -Target core
./out/bin/mfh-variable-example.exe -help
```

源码为 [provider.go](provider.go) 与 [CLI](../../cmd/mfh-variable-example/main.go)。Win11 x64 可运行包包含已构建的 EXE；运行时不需要 Go。

## 首次接入

以下使用仍受支持的显式 ID 准入兼容流程；为示例选取未使用的 ID 和独立状态目录。不要与 Hub、Desktop 或 Metrics 共用身份目录。Hub 的新 Enrollment 管理流程见 [Hub 文档](../../docs/features/hub.md#首次引导)。

1. 在设备 B 运行 `mfh-variable-example.exe -state <独立绝对路径> -id 20 -identity`，取得持久公钥。
2. Hub 停止时运行 `mfh-hub.exe -state <Hub状态绝对路径> -id 1 -issue-node-id 20 -issue-public-key <示例公钥>`，将返回 JSON 作为本次示例的 Permit 安全交给 B；不要纳入公开证据。
3. Hub 用原状态目录启动。在 B 运行：

```powershell
./mfh-variable-example.exe -state <示例状态绝对路径> -id 20 `
  -parent-id 1 -parent-key <Hub公钥> -endpoint <设备A的LAN地址:7331> `
  -permit <本地Permit路径> -count 1998
```

4. 由管理员给 Desktop 主体明确授予 owner 20 的 `system/catalog` 的 read；目录超过完整值预算时还需 list。对目标示例资源分别授予 read、subscribe；仅 bool/number 按需授予 write。准入不自动授予业务权限，不能以示例身份代替 Desktop。
5. Desktop 展开示例 owner，进入“变量总览”，订阅筛选结果；查看六类型标准定义和实时值，选中布尔/数字，在检查器显式写入。示例 namespace 不需要安装专用 renderer。

重启保留 identity/trust；进程停止后离线值不继续发布，不缓存远程写入以供补发。示例变量当前值不持久化，进程重启产生新 instance，旧草稿必须按实例边界重新确认。它不替代 Metrics 的已保存采集配置。

## 检查

```powershell
$env:GOWORK='off'
go test ./examples/variables ./tests/integration -run 'TestReferenceLoad|TestPhaseA' -count=1 -timeout=180s
```

本机测试与正式两台设备验收分别记录，后者的 2 小时运行、真实音量/静音和进程内存归属仍按 [A-AC-v1](../../docs/requirements/phase-a-acceptance.md) 执行。
