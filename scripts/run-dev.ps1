<#
.SYNOPSIS
  启动 canonical monorepo 中的 Hub、Desktop 与 Metrics Windows 开发进程。

.DESCRIPTION
  只使用当前仓库的 cmd/、apps/ 和 state，不读取冻结的旧仓库或 go.work。
  后台进程默认隐藏并将输出写入 <StateRoot>/logs；显式传入 -VisibleWindows 才显示窗口。
  Hub 权限默认拒绝。首次接入前请停止 Hub，使用 mfh-hub 离线模式签发 permit 和策略。

.EXAMPLE
  ./scripts/run-dev.ps1 -WaitHub

.EXAMPLE
  ./scripts/run-dev.ps1 -DryRun

.EXAMPLE
  ./scripts/run-dev.ps1 -SkipDesktop -SkipMetrics -VisibleWindows
#>

[CmdletBinding()]
param(
    [string]$HubListen = '127.0.0.1:7331',
    [ValidateRange(1, [long]::MaxValue)]
    [long]$HubNodeID = 1,
    [string]$StateRoot = '',
    [ValidateRange(1, 65535)]
    [int]$DesktopDevServerPortStart = 34115,
    [ValidateRange(1, 65535)]
    [int]$MetricsDevServerPortStart = 34116,
    [ValidateRange(0, 1000)]
    [int]$DevServerPortSearchMax = 50,
    [ValidateRange(1, 120)]
    [int]$WaitTimeoutSec = 12,
    [switch]$WaitHub,
    [switch]$DryRun,
    [switch]$VisibleWindows,
    [switch]$SkipHub,
    [switch]$SkipDesktop,
    [switch]$SkipMetrics
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$env:GOWORK = 'off'

$root = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..')).Path
$desktopDirectory = Join-Path $root 'apps/desktop'
$metricsDirectory = Join-Path $root 'apps/nodes/metrics/windows'
$hubEntry = Join-Path $root 'cmd/mfh-hub'
if (-not (Test-Path -LiteralPath (Join-Path $root 'go.mod'))) { throw "canonical go.mod not found under $root" }
if (-not (Test-Path -LiteralPath $hubEntry)) { throw "canonical Hub entry not found: $hubEntry" }
if (-not (Test-Path -LiteralPath $desktopDirectory)) { throw "canonical Desktop app not found: $desktopDirectory" }
if (-not (Test-Path -LiteralPath $metricsDirectory)) { throw "canonical Metrics app not found: $metricsDirectory" }

if ($StateRoot.Trim() -eq '') {
    $StateRoot = Join-Path $root '.tmp/dev-state'
} elseif (-not [IO.Path]::IsPathRooted($StateRoot)) {
    $StateRoot = Join-Path $root $StateRoot
}
$StateRoot = [IO.Path]::GetFullPath($StateRoot)
$hubState = Join-Path $StateRoot 'hub'
$desktopState = Join-Path $StateRoot 'desktop'
$logDirectory = Join-Path $StateRoot 'logs'

function Require-Command([string]$Name, [string]$Purpose) {
    $command = Get-Command $Name -ErrorAction SilentlyContinue
    if (-not $command) { throw "'$Name' is required for $Purpose" }
    return $command.Source
}

function Test-LocalTcpPortAvailable([int]$Port) {
    $listener = $null
    try {
        $listener = [Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback, $Port)
        $listener.Start()
        return $true
    } catch {
        return $false
    } finally {
        if ($null -ne $listener) { $listener.Stop() }
    }
}

function Find-AvailablePort([int]$Start, [int[]]$Excluded) {
    for ($offset = 0; $offset -le $DevServerPortSearchMax; $offset++) {
        $candidate = $Start + $offset
        if ($candidate -gt 65535) { break }
        if ($Excluded -contains $candidate) { continue }
        if (Test-LocalTcpPortAvailable $candidate) { return $candidate }
    }
    throw "no available development port from $Start within $DevServerPortSearchMax attempts"
}

function Parse-ListenAddress([string]$Address) {
    if ($Address -notmatch '^(?<host>[^:]+):(?<port>\d+)$') {
        throw "HubListen must use host:port, got '$Address'"
    }
    $port = [int]$Matches.port
    if ($port -lt 1 -or $port -gt 65535) { throw "HubListen port is invalid: $port" }
    return [pscustomobject]@{ Host = $Matches.host; Port = $port }
}

function Wait-TcpReady([string]$HostName, [int]$Port) {
    $deadline = [DateTimeOffset]::UtcNow.AddSeconds($WaitTimeoutSec)
    while ([DateTimeOffset]::UtcNow -lt $deadline) {
        $client = [Net.Sockets.TcpClient]::new()
        try {
            $pending = $client.ConnectAsync($HostName, $Port)
            if ($pending.Wait(500) -and $client.Connected) { return $true }
        } catch {
            # A refused connection is expected while the Hub is starting.
        } finally {
            $client.Dispose()
        }
        Start-Sleep -Milliseconds 200
    }
    return $false
}

function Quote-ProcessArgument([string]$Value) {
    if ($Value -notmatch '[\s"]') { return $Value }
    return '"' + $Value.Replace('"', '\"') + '"'
}

function Start-CanonicalProcess(
    [string]$Name,
    [string]$Executable,
    [string]$WorkingDirectory,
    [string[]]$Arguments
) {
    $argumentText = ($Arguments | ForEach-Object { Quote-ProcessArgument $_ }) -join ' '
    $stdout = Join-Path $logDirectory "$Name.stdout.log"
    $stderr = Join-Path $logDirectory "$Name.stderr.log"
    $parameters = @{
        FilePath               = $Executable
        ArgumentList           = $argumentText
        WorkingDirectory       = $WorkingDirectory
        PassThru               = $true
        RedirectStandardOutput = $stdout
        RedirectStandardError  = $stderr
    }
    if (-not $VisibleWindows) { $parameters.WindowStyle = 'Hidden' }
    $process = Start-Process @parameters
    Write-Host "$Name started: pid=$($process.Id), stdout=$stdout, stderr=$stderr" -ForegroundColor Green
    return $process
}

$startHub = -not $SkipHub
$startDesktop = -not $SkipDesktop
$startMetrics = -not $SkipMetrics
$listen = Parse-ListenAddress $HubListen
$desktopPort = if ($startDesktop) { Find-AvailablePort $DesktopDevServerPortStart @() } else { 0 }
$metricsPort = if ($startMetrics) { Find-AvailablePort $MetricsDevServerPortStart @($desktopPort) } else { 0 }

Write-Host "canonical root: $root"
Write-Host "GOWORK: off"
Write-Host "Hub: cmd/mfh-hub, listen=$HubListen, state=$hubState, enabled=$startHub"
Write-Host "Desktop: apps/desktop, devserver=127.0.0.1:$desktopPort, state=$desktopState, enabled=$startDesktop"
Write-Host "Metrics: apps/nodes/metrics/windows, devserver=127.0.0.1:$metricsPort, enabled=$startMetrics"

if ($DryRun) {
    Write-Host 'dry run: no directory or process was created' -ForegroundColor Yellow
} else {
    New-Item -ItemType Directory -Path $hubState, $desktopState, $logDirectory -Force | Out-Null
    $go = if ($startHub) { Require-Command 'go' 'Hub development' } else { '' }
    $wails = if ($startDesktop -or $startMetrics) { Require-Command 'wails' 'Wails development' } else { '' }

    if ($startHub) {
        [void](Start-CanonicalProcess 'hub' $go $root @(
            'run', './cmd/mfh-hub',
            '-id', [string]$HubNodeID,
            '-listen', $HubListen,
            '-state', $hubState
        ))
        if ($WaitHub) {
            Write-Host "waiting for Hub at $($listen.Host):$($listen.Port)..." -ForegroundColor Yellow
            if (-not (Wait-TcpReady $listen.Host $listen.Port)) {
                throw "Hub did not accept TCP connections within $WaitTimeoutSec seconds; inspect $logDirectory"
            }
            Write-Host 'Hub is accepting TCP connections' -ForegroundColor Green
        }
    } elseif ($WaitHub) {
        Write-Warning '-WaitHub was ignored because -SkipHub was supplied'
    }

    $previousDesktopConfig = $env:MFH_DESKTOP_CONFIG_DIR
    try {
        if ($startDesktop) {
            $env:MFH_DESKTOP_CONFIG_DIR = $desktopState
            [void](Start-CanonicalProcess 'desktop' $wails $desktopDirectory @(
                'dev', '-devserver', "127.0.0.1:$desktopPort"
            ))
        }
        if ($startMetrics) {
            [void](Start-CanonicalProcess 'metrics' $wails $metricsDirectory @(
                'dev', '-devserver', "127.0.0.1:$metricsPort"
            ))
        }
    } finally {
        $env:MFH_DESKTOP_CONFIG_DIR = $previousDesktopConfig
    }
}

Write-Host ''
Write-Host 'Hub uses default-deny policy. Stop the Hub before any offline bootstrap command:' -ForegroundColor Cyan
Write-Host "go run ./cmd/mfh-hub -id $HubNodeID -state '$hubState' -identity"
Write-Host "go run ./cmd/mfh-hub -id $HubNodeID -state '$hubState' -issue-node-id 2 -issue-public-key '<raw-base64-ed25519-key>' -issue-role device"
Write-Host "go run ./cmd/mfh-hub -id $HubNodeID -state '$hubState' -policy grant -subject 2 -action subscribe -resource-node $HubNodeID -resource system/health"
