<#
.SYNOPSIS
  一键启动 MyFlowHub-Server(hub_server) + MyFlowHub-Win(wails dev) + MyFlowHub-MetricsNode(wails dev)，用于本地冒烟验证。

.DESCRIPTION
  默认会打开三个新 PowerShell 窗口：
    1) Server：go run ./cmd/hub_server
    2) Win：wails dev
    3) MetricsNode：wails dev

  说明：
  - 本脚本放在 workspace 根目录（d:\project\MyFlowHub3\scripts\），不进入任何 repo 的 git。
  - Server 参数优先通过环境变量注入（与 cmd/hub_server/main.go 的 defaultOptions 对齐）。

.EXAMPLE
  .\scripts\run-dev.ps1

.EXAMPLE
  .\scripts\run-dev.ps1 -ServerAddr ':9001' -ServerNodeId 2 -WaitServer

.EXAMPLE
  .\scripts\run-dev.ps1 -SkipWin

.EXAMPLE
  .\scripts\run-dev.ps1 -SkipMetricsNode

.EXAMPLE
  # 指定 Win / MetricsNode 的 DevServer 起始端口（端口被占用时会自动递增避让）
  .\scripts\run-dev.ps1 -WinDevServerPortStart 34115 -MetricsDevServerPortStart 34116 -DevServerPortSearchMax 50
#>

[CmdletBinding()]
param(
  [Parameter()]
  [string]$ServerAddr = ":9000",

  [Parameter()]
  [int]$ServerNodeId = 1,

  [Parameter()]
  [string]$ParentAddr = "",

  [Parameter()]
  [int]$WinDevServerPortStart = 34115,

  [Parameter()]
  [int]$MetricsDevServerPortStart = 34116,

  [Parameter()]
  [int]$DevServerPortSearchMax = 50,

  [Parameter()]
  [switch]$GoWorkOff,

  [Parameter()]
  [string]$GoTmpDir = "",

  [Parameter()]
  [switch]$WaitServer,

  [Parameter()]
  [int]$WaitTimeoutSec = 12,

  [Parameter()]
  [switch]$SkipServer,

  [Parameter()]
  [switch]$SkipWin,

  [Parameter()]
  [switch]$SkipMetricsNode
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Assert-Command([string]$Name) {
  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    throw "缺少命令：$Name。请先安装并确保在 PATH 中可用。"
  }
}

function Resolve-WorkspaceRoot() {
  $root = Resolve-Path (Join-Path $PSScriptRoot "..")
  return $root.Path
}

function Ensure-Dir([string]$Path) {
  if (-not (Test-Path -LiteralPath $Path)) {
    New-Item -ItemType Directory -Force -Path $Path | Out-Null
  }
}

function Parse-ListenAddr([string]$Addr) {
  $raw = ([string]$Addr).Trim()
  if ($raw -eq "") {
    return @{ Host = "127.0.0.1"; Port = 9000 }
  }

  if ($raw.StartsWith(":")) {
    $portText = $raw.Substring(1)
    $port = [int]::Parse($portText)
    return @{ Host = "127.0.0.1"; Port = $port }
  }

  if ($raw -match "^(?<host>.+):(?<port>\\d+)$") {
    return @{ Host = $Matches["host"]; Port = [int]::Parse($Matches["port"]) }
  }

  throw "无法解析 ServerAddr：'$Addr'。期望格式如 ':9000' 或 '127.0.0.1:9000'。"
}

function Wait-TcpReady([string]$Host, [int]$Port, [int]$TimeoutSec) {
  $deadline = [DateTimeOffset]::UtcNow.AddSeconds([Math]::Max(1, $TimeoutSec))
  while ([DateTimeOffset]::UtcNow -lt $deadline) {
    try {
      $client = New-Object System.Net.Sockets.TcpClient
      $async = $client.BeginConnect($Host, $Port, $null, $null)
      if ($async.AsyncWaitHandle.WaitOne(500)) {
        $client.EndConnect($async)
        $client.Close()
        return $true
      }
      $client.Close()
    } catch {
      # ignore
    }
    Start-Sleep -Milliseconds 250
  }
  return $false
}

function Assert-TcpPortInRange([string]$Name, [int]$Port) {
  if ($Port -lt 1 -or $Port -gt 65535) {
    throw "$Name 端口非法：$Port。期望范围：1-65535。"
  }
}

function Test-LocalTcpPortAvailable([int]$Port) {
  $listener = $null
  try {
    $listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, $Port)
    $listener.Start()
    return $true
  } catch {
    return $false
  } finally {
    if ($null -ne $listener) {
      try { $listener.Stop() } catch { }
    }
  }
}

function Find-NextAvailableTcpPort([int]$StartPort, [int]$MaxTry, [int[]]$ExcludePorts = @()) {
  Assert-TcpPortInRange -Name "StartPort" -Port $StartPort
  if ($MaxTry -lt 0) { throw "MaxTry 不能为负数：$MaxTry" }

  $exclude = New-Object 'System.Collections.Generic.HashSet[int]'
  foreach ($p in $ExcludePorts) {
    if ($p -ge 1 -and $p -le 65535) { [void]$exclude.Add([int]$p) }
  }

  for ($i = 0; $i -le $MaxTry; $i++) {
    $port = $StartPort + $i
    if ($port -gt 65535) { break }
    if ($exclude.Contains($port)) { continue }
    if (Test-LocalTcpPortAvailable -Port $port) { return $port }
  }

  throw "无法从 $StartPort 起在 $MaxTry 次探测内找到可用端口（排除：$($ExcludePorts -join ',')）。"
}

$root = Resolve-WorkspaceRoot
$serverDir = Join-Path $root "repo\\MyFlowHub-Server"
$winDir = Join-Path $root "repo\\MyFlowHub-Win"
$metricsNodeDir = Join-Path $root "repo\\MyFlowHub-MetricsNode\\windows"

$startServer = -not $SkipServer
$startWin = -not $SkipWin
$startMetricsNode = -not $SkipMetricsNode

if ($startServer -and -not (Test-Path -LiteralPath $serverDir)) { throw "目录不存在：$serverDir" }
if ($startWin -and -not (Test-Path -LiteralPath $winDir)) { throw "目录不存在：$winDir" }
if ($startMetricsNode -and -not (Test-Path -LiteralPath $metricsNodeDir)) { throw "目录不存在：$metricsNodeDir" }

$needGo = $startServer -or $startWin -or $startMetricsNode
$needWails = $startWin -or $startMetricsNode

if ($needGo) { Assert-Command "go" }
if ($needWails) { Assert-Command "wails" }

$devServerHost = "127.0.0.1"
if ($startWin) { Assert-TcpPortInRange -Name "WinDevServerPortStart" -Port $WinDevServerPortStart }
if ($startMetricsNode) { Assert-TcpPortInRange -Name "MetricsDevServerPortStart" -Port $MetricsDevServerPortStart }
if ($needWails) {
  if ($DevServerPortSearchMax -lt 0) { throw "DevServerPortSearchMax 不能为负数：$DevServerPortSearchMax" }
}

$winDevServerPort = 0
$metricsDevServerPort = 0
if ($startWin) {
  $winDevServerPort = Find-NextAvailableTcpPort -StartPort $WinDevServerPortStart -MaxTry $DevServerPortSearchMax
}
if ($startMetricsNode) {
  $exclude = @()
  if ($winDevServerPort -gt 0) { $exclude += $winDevServerPort }
  $metricsDevServerPort = Find-NextAvailableTcpPort -StartPort $MetricsDevServerPortStart -MaxTry $DevServerPortSearchMax -ExcludePorts $exclude
}

$shellExe = ""
if (Get-Command "pwsh" -ErrorAction SilentlyContinue) {
  $shellExe = "pwsh"
} elseif (Get-Command "powershell" -ErrorAction SilentlyContinue) {
  $shellExe = "powershell"
} else {
  throw "缺少 PowerShell 可执行程序：pwsh 或 powershell。"
}

if ($GoTmpDir.Trim() -eq "") {
  $GoTmpDir = Join-Path $root ".tmp\\gotmp"
}
Ensure-Dir $GoTmpDir

$commonEnv = @(
  "`$env:GOTMPDIR = '$GoTmpDir'",
  "New-Item -ItemType Directory -Force -Path `$env:GOTMPDIR | Out-Null"
)
if ($GoWorkOff) {
  $commonEnv += "`$env:GOWORK = 'off'"
}

if ($startServer) {
  $serverEnv = @(
    "`$host.UI.RawUI.WindowTitle = 'MyFlowHub-Server hub_server'",
    "`$env:HUB_ADDR = '$ServerAddr'",
    "`$env:HUB_NODE_ID = '$ServerNodeId'"
  )
  if ($ParentAddr.Trim() -ne "") {
    $serverEnv += "`$env:HUB_PARENT_ADDR = '$ParentAddr'"
  }

  $serverCmd = (@($commonEnv + $serverEnv) -join "; ") + "; go run ./cmd/hub_server"
  Start-Process -FilePath $shellExe -WorkingDirectory $serverDir -ArgumentList @("-NoExit", "-Command", $serverCmd) | Out-Null
  Write-Host "已启动 Server：$ServerAddr (node-id=$ServerNodeId)" -ForegroundColor Green
}

if ($WaitServer -and ($startWin -or $startMetricsNode)) {
  $target = Parse-ListenAddr $ServerAddr
  Write-Host "等待 Server 就绪：$($target.Host):$($target.Port) ..." -ForegroundColor Yellow
  if (-not (Wait-TcpReady -Host $target.Host -Port $target.Port -TimeoutSec $WaitTimeoutSec)) {
    Write-Warning "等待超时：Server 端口仍不可连接（你仍可手动继续）。"
  } else {
    Write-Host "Server 已就绪。" -ForegroundColor Green
  }
}

if ($startWin) {
  $winEnv = @("`$host.UI.RawUI.WindowTitle = 'MyFlowHub-Win wails dev'")
  $winDevServer = "$devServerHost`:$winDevServerPort"
  if ($winDevServerPort -ne $WinDevServerPortStart) {
    Write-Host "Win DevServer 端口已避让：$WinDevServerPortStart -> $winDevServerPort（$winDevServer）" -ForegroundColor Yellow
  } else {
    Write-Host "Win DevServer：$winDevServer" -ForegroundColor DarkGray
  }
  $winCmd = (@($commonEnv + $winEnv) -join "; ") + "; wails dev -devserver '$winDevServer'"
  Start-Process -FilePath $shellExe -WorkingDirectory $winDir -ArgumentList @("-NoExit", "-Command", $winCmd) | Out-Null
  Write-Host "已启动 Win：wails dev" -ForegroundColor Green
}

if ($startMetricsNode) {
  $metricsNodeEnv = @("`$host.UI.RawUI.WindowTitle = 'MyFlowHub-MetricsNode wails dev'")
  $metricsDevServer = "$devServerHost`:$metricsDevServerPort"
  if ($metricsDevServerPort -ne $MetricsDevServerPortStart) {
    Write-Host "MetricsNode DevServer 端口已避让：$MetricsDevServerPortStart -> $metricsDevServerPort（$metricsDevServer）" -ForegroundColor Yellow
  } else {
    Write-Host "MetricsNode DevServer：$metricsDevServer" -ForegroundColor DarkGray
  }
  $metricsNodeCmd = (@($commonEnv + $metricsNodeEnv) -join "; ") + "; wails dev -devserver '$metricsDevServer'"
  Start-Process -FilePath $shellExe -WorkingDirectory $metricsNodeDir -ArgumentList @("-NoExit", "-Command", $metricsNodeCmd) | Out-Null
  Write-Host "已启动 MetricsNode：wails dev" -ForegroundColor Green
}

Write-Host ""
Write-Host "冒烟验证步骤：" -ForegroundColor Cyan
Write-Host "1) Win 首页 Address 填：127.0.0.1:9000（或你设置的 ServerAddr），Device ID 任意非空（例如 dev-1），点击 Connect"
Write-Host "2) MetricsNode：Connect 到同一 Hub；首次点 Register 或已有 NodeID 点 Login；再点 Start Reporting"
Write-Host "3) Win → VarPool：订阅 MetricsNode 的变量（owner=MetricsNode node_id）：sys_battery_percent / sys_volume_percent / sys_volume_muted，期望自动更新"
Write-Host "4) Win：Presets → Node Echo → Send，期望提示成功，Logs 无明显错误"
Write-Host ""
Write-Host "提示：如端口被占用，可用 -ServerAddr ':9001' 改端口。" -ForegroundColor DarkGray
