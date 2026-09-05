[CmdletBinding()]
param(
    [ValidateSet('list', 'test', 'build', 'generate', 'check', 'manifest')]
    [string]$Action = 'list',

    [ValidateSet('all', 'core', 'hub', 'desktop', 'metrics', 'clipboard', 'generated')]
    [string]$Target = 'all',

    [switch]$AllowUnavailable
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$env:GOWORK = 'off'

$root = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..')).Path
$out = Join-Path $root 'out'
$isWindowsHost = $env:OS -eq 'Windows_NT'
if ($env:MFH_FLUTTER_HOME) {
    if (-not [IO.Path]::IsPathRooted($env:MFH_FLUTTER_HOME)) { throw 'MFH_FLUTTER_HOME must be an absolute path' }
    $flutterRelativePath = 'bin/flutter'
    if ($isWindowsHost) { $flutterRelativePath = 'bin/flutter.bat' }
    $flutterExecutable = Join-Path $env:MFH_FLUTTER_HOME $flutterRelativePath
    if (-not (Test-Path -LiteralPath $flutterExecutable)) { throw "MFH_FLUTTER_HOME does not contain Flutter: $env:MFH_FLUTTER_HOME" }
    $env:PATH = "$(Join-Path $env:MFH_FLUTTER_HOME 'bin')$([IO.Path]::PathSeparator)$env:PATH"
}
function Ensure-Directory([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
    }
}

function Get-RepositoryRelativePath([string]$Base, [string]$Path) {
    $basePath = [IO.Path]::GetFullPath($Base).TrimEnd([IO.Path]::DirectorySeparatorChar, [IO.Path]::AltDirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
    $fullPath = [IO.Path]::GetFullPath($Path)
    if (-not $fullPath.StartsWith($basePath, [StringComparison]::OrdinalIgnoreCase)) {
        throw "artifact is outside the canonical repository: $fullPath"
    }
    return $fullPath.Substring($basePath.Length).Replace('\', '/')
}

function Require-Tool([string]$Name, [string]$Purpose) {
    if (Get-Command $Name -ErrorAction SilentlyContinue) { return $true }
    $message = "UNAVAILABLE: '$Name' is required for $Purpose"
    if ($AllowUnavailable) { Write-Warning $message; return $false }
    throw $message
}

function Invoke-Native([string]$WorkingDirectory, [string]$Command, [string[]]$Arguments) {
    Push-Location $WorkingDirectory
    try {
        Write-Host "> $Command $($Arguments -join ' ')" -ForegroundColor DarkGray
        & $Command @Arguments | Out-Host
        if ($LASTEXITCODE -ne 0) { throw "$Command failed with exit code $LASTEXITCODE" }
    } finally {
        Pop-Location
    }
}

function Test-Core {
    if (-not (Require-Tool 'go' 'Go tests')) { return }
    Invoke-Native $root 'go' @('test', './...', '-count=1')
    Invoke-Native $root 'go' @('vet', './...')
}

function Build-Core {
    if (-not (Require-Tool 'go' 'Go builds')) { return }
    $bin = Join-Path $out 'bin'; Ensure-Directory $bin
    foreach ($name in @('mfh-hub', 'mfh-admin', 'mfh-node')) {
        $suffix = if ($isWindowsHost) { '.exe' } else { '' }
        Invoke-Native $root 'go' @('build', '-trimpath', '-o', (Join-Path $bin "$name$suffix"), "./cmd/$name")
    }
}

function Test-Hub {
    Invoke-Native $root 'go' @('test', './host/...', './feature/...', './runtime/auth', './runtime/node', './runtime/tree', '-count=1')
}

function Build-Hub {
    $bin = Join-Path $out 'bin'; Ensure-Directory $bin
    $suffix = if ($isWindowsHost) { '.exe' } else { '' }
    foreach ($name in @('mfh-hub', 'mfh-admin')) { Invoke-Native $root 'go' @('build', '-trimpath', '-o', (Join-Path $bin "$name$suffix"), "./cmd/$name") }
}

function Test-Desktop {
    Invoke-Native $root 'go' @('test', './apps/desktop/...', '-count=1')
    if (-not (Require-Tool 'npm' 'Desktop frontend tests')) { return }
    $frontend = Join-Path $root 'apps/desktop/frontend'
    Invoke-Native $frontend 'npm' @('ci')
    Invoke-Native $frontend 'npm' @('test')
    Invoke-Native $frontend 'npm' @('run', 'build')
}

function Generate-Desktop {
    if (-not (Require-Tool 'wails' 'Desktop Wails bindings')) { return }
    Invoke-Native (Join-Path $root 'apps/desktop') 'wails' @('generate', 'module')
}

function Build-Desktop {
    if (-not $isWindowsHost) {
        if ($AllowUnavailable) { Write-Warning 'UNAVAILABLE: Desktop production build requires Windows'; return }
        throw 'Desktop production build requires Windows'
    }
    if (-not (Require-Tool 'wails' 'Desktop production build')) { return }
    Invoke-Native (Join-Path $root 'apps/desktop') 'wails' @('build', '-clean', '-trimpath', '-platform', 'windows/amd64', '-o', 'mfh-desktop.exe')
}

function Test-Metrics {
    Invoke-Native $root 'go' @('test', './apps/nodes/metrics/...', '-count=1')
    if (Require-Tool 'npm' 'Metrics Windows frontend tests') {
        $frontend = Join-Path $root 'apps/nodes/metrics/windows/frontend'
        Invoke-Native $frontend 'npm' @('ci'); Invoke-Native $frontend 'npm' @('test'); Invoke-Native $frontend 'npm' @('run', 'build')
    }
}

function Generate-Metrics {
    if (-not (Require-Tool 'wails' 'Metrics Wails bindings')) { return }
    Invoke-Native (Join-Path $root 'apps/nodes/metrics/windows') 'wails' @('generate', 'module')
}

function Build-Metrics {
    if ($isWindowsHost -and (Require-Tool 'wails' 'Metrics Windows production build')) {
        Invoke-Native (Join-Path $root 'apps/nodes/metrics/windows') 'wails' @('build', '-clean', '-trimpath', '-platform', 'windows/amd64', '-o', 'mfh-metrics.exe')
    } elseif (-not $isWindowsHost -and -not $AllowUnavailable) { throw 'Metrics Windows production build requires Windows' }
}

function Test-Clipboard {
    Invoke-Native $root 'go' @('test', './apps/nodes/clipboard/...', '-count=1')
    if (-not (Require-Tool 'flutter' 'Clipboard Flutter tests')) { return }
    $app = Join-Path $root 'apps/nodes/clipboard/app'
    Invoke-Native $app 'flutter' @('pub', 'get')
    Invoke-Native $app 'flutter' @('analyze')
    Invoke-Native $app 'flutter' @('test')
}

function Build-Clipboard {
    if (-not (Require-Tool 'flutter' 'Clipboard application builds')) { return }
    $app = Join-Path $root 'apps/nodes/clipboard/app'
    Invoke-Native $app 'flutter' @('build', 'web', '--release')
    if ($isWindowsHost) {
        Invoke-Native $app 'flutter' @('build', 'windows', '--release')
        $release = Join-Path $app 'build/windows/x64/runner/Release'
        Invoke-Native $root 'go' @('build', '-trimpath', '-o', (Join-Path $release 'mfh-clipboard.exe'), './cmd/mfh-clipboard')
    } elseif (-not $AllowUnavailable) { throw 'Clipboard Windows production build requires Windows' }
}

function Generate-Contracts { Invoke-Native $root 'go' @('generate', './sdk/bindings') }

function Check-Generated {
    Generate-Contracts; Generate-Desktop; Generate-Metrics
    Invoke-Native $root 'go' @('test', './sdk/bindings', './internal/protocoltest', '-count=1')
    $tracked = @('sdk/bindings/generated', 'apps/desktop/frontend/src/generated', 'apps/desktop/frontend/wailsjs', 'apps/nodes/metrics/windows/frontend/wailsjs')
    & git -C $root diff --exit-code -- @tracked
    if ($LASTEXITCODE -ne 0) { throw 'generated source differs from the repository' }
}

function Check-Core {
    $unformatted = & gofmt -l (Get-ChildItem -Path $root -Recurse -Filter '*.go' -File | Where-Object { $_.FullName -notmatch '[\\/]build[\\/]' } | ForEach-Object FullName)
    if ($LASTEXITCODE -ne 0) { throw 'gofmt inspection failed' }
    if ($unformatted) { throw "unformatted Go files:`n$($unformatted -join "`n")" }
    Invoke-Native $root 'go' @('test', './internal/archtest', './internal/migrationtest', './internal/protocoltest', './protocol', '-count=1')
    Invoke-Native $root 'go' @('vet', './...')
}

function Write-Manifest {
    Ensure-Directory $out
    $artifactSets = @(
        @{ root = 'out/bin'; filter = '*'; recurse = $false },
        @{ root = 'apps/desktop/build/bin'; filter = '*'; recurse = $true },
        @{ root = 'apps/nodes/metrics/windows/build/bin'; filter = '*'; recurse = $true },
        @{ root = 'apps/nodes/clipboard/app/build/windows/x64/runner/Release'; filter = '*'; recurse = $true },
        @{ root = 'apps/nodes/clipboard/app/build/web'; filter = '*'; recurse = $true }
    )
    $items = foreach ($set in $artifactSets) {
        $artifactRoot = Join-Path $root $set.root
        if (-not (Test-Path -LiteralPath $artifactRoot)) { continue }
        $files = if ($set.recurse) {
            Get-ChildItem -LiteralPath $artifactRoot -Filter $set.filter -File -Recurse
        } else {
            Get-ChildItem -LiteralPath $artifactRoot -Filter $set.filter -File
        }
        $files | ForEach-Object {
            [ordered]@{ path = Get-RepositoryRelativePath $root $_.FullName; bytes = $_.Length; sha256 = (Get-FileHash -Algorithm SHA256 -LiteralPath $_.FullName).Hash.ToLowerInvariant() }
        }
    }
    if (-not $items) { throw 'no build artifacts found; run build targets first' }
    [ordered]@{ version = 1; generated_at_utc = [DateTimeOffset]::UtcNow.ToString('O'); artifacts = @($items | Sort-Object path) } |
        ConvertTo-Json -Depth 5 | Set-Content -LiteralPath (Join-Path $out 'artifact-manifest.json') -Encoding utf8
}

$targetOrder = @('core', 'hub', 'desktop', 'metrics', 'clipboard')
if ($Action -eq 'list') {
    Write-Output 'actions: test, build, generate, check, manifest'
    Write-Output "targets: $($targetOrder -join ', '), generated, all"
    Write-Output 'all is deterministic and follows the target order above; -AllowUnavailable records missing platform tools without claiming a pass.'
    exit 0
}

if ($Action -eq 'manifest') { Write-Manifest; exit 0 }
if ($Target -eq 'generated') {
    if ($Action -eq 'generate') { Generate-Contracts; Generate-Desktop; Generate-Metrics }
    elseif ($Action -eq 'check' -or $Action -eq 'test') { Check-Generated }
    else { throw "action '$Action' is not valid for target generated" }
    exit 0
}

$targets = if ($Target -eq 'all') { $targetOrder } else { @($Target) }
foreach ($current in $targets) {
    Write-Host "[$Action/$current]" -ForegroundColor Cyan
    switch ("$Action/$current") {
        'test/core' { Test-Core }
        'test/hub' { Test-Hub }
        'test/desktop' { Test-Desktop }
        'test/metrics' { Test-Metrics }
        'test/clipboard' { Test-Clipboard }
        'build/core' { Build-Core }
        'build/hub' { Build-Hub }
        'build/desktop' { Build-Desktop }
        'build/metrics' { Build-Metrics }
        'build/clipboard' { Build-Clipboard }
        'generate/core' { Generate-Contracts }
        'generate/hub' { }
        'generate/desktop' { Generate-Desktop }
        'generate/metrics' { Generate-Metrics }
        'generate/clipboard' { }
        'check/core' { Check-Core }
        'check/hub' { Test-Hub }
        'check/desktop' { Test-Desktop }
        'check/metrics' { Test-Metrics }
        'check/clipboard' { Test-Clipboard }
        default { throw "unsupported action/target: $Action/$current" }
    }
}
