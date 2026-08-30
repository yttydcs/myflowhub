[CmdletBinding()]
param(
    [ValidateSet('list', 'test', 'build', 'generate', 'check', 'manifest')]
    [string]$Action = 'list',

    [ValidateSet('all', 'core', 'hub', 'desktop', 'android', 'metrics', 'clipboard', 'embedded', 'generated')]
    [string]$Target = 'all',

    [switch]$AllowUnavailable
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$env:GOWORK = 'off'

$root = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..')).Path
$out = Join-Path $root 'out'
$isWindowsHost = $env:OS -eq 'Windows_NT'
if ($env:MFH_JAVA_HOME) {
    $javaRelativePath = 'bin/java'
    if ($isWindowsHost) { $javaRelativePath = 'bin/java.exe' }
    $javaExecutable = Join-Path $env:MFH_JAVA_HOME $javaRelativePath
    if (-not (Test-Path -LiteralPath $javaExecutable)) { throw "MFH_JAVA_HOME does not contain a Java runtime: $env:MFH_JAVA_HOME" }
    $env:JAVA_HOME = $env:MFH_JAVA_HOME
    $env:PATH = "$(Join-Path $env:JAVA_HOME 'bin')$([IO.Path]::PathSeparator)$env:PATH"
}
if ($env:MFH_FLUTTER_HOME) {
    if (-not [IO.Path]::IsPathRooted($env:MFH_FLUTTER_HOME)) { throw 'MFH_FLUTTER_HOME must be an absolute path' }
    $flutterRelativePath = 'bin/flutter'
    if ($isWindowsHost) { $flutterRelativePath = 'bin/flutter.bat' }
    $flutterExecutable = Join-Path $env:MFH_FLUTTER_HOME $flutterRelativePath
    if (-not (Test-Path -LiteralPath $flutterExecutable)) { throw "MFH_FLUTTER_HOME does not contain Flutter: $env:MFH_FLUTTER_HOME" }
    $env:PATH = "$(Join-Path $env:MFH_FLUTTER_HOME 'bin')$([IO.Path]::PathSeparator)$env:PATH"
}
if ($env:MFH_SHORT_TEMP) {
    if (-not [IO.Path]::IsPathRooted($env:MFH_SHORT_TEMP)) { throw 'MFH_SHORT_TEMP must be an absolute path' }
    if (-not (Test-Path -LiteralPath $env:MFH_SHORT_TEMP)) { New-Item -ItemType Directory -Path $env:MFH_SHORT_TEMP -Force | Out-Null }
    $env:TEMP = (Resolve-Path -LiteralPath $env:MFH_SHORT_TEMP).Path
    $env:TMP = $env:TEMP
}
if (-not $env:ANDROID_HOME -and $env:ANDROID_SDK_ROOT) { $env:ANDROID_HOME = $env:ANDROID_SDK_ROOT }
if (-not $env:ANDROID_HOME -and $isWindowsHost -and $env:LOCALAPPDATA) {
    $standardAndroidSDK = Join-Path $env:LOCALAPPDATA 'Android\Sdk'
    if (Test-Path -LiteralPath $standardAndroidSDK) { $env:ANDROID_HOME = $standardAndroidSDK }
}
if ($env:ANDROID_HOME -and -not $env:ANDROID_SDK_ROOT) { $env:ANDROID_SDK_ROOT = $env:ANDROID_HOME }

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

function Gradle-Command([string]$Directory, [string[]]$Tasks) {
    $command = if ($isWindowsHost) { '.\gradlew.bat' } else { './gradlew' }
    Invoke-Native $Directory $command (@('--no-daemon', '--stacktrace') + $Tasks)
}

function Build-MobileBinding([string]$Package, [string]$JavaPackage, [string]$Output) {
    if (-not (Require-Tool 'gomobile' "generating $Output")) { return $false }
    Ensure-Directory (Split-Path -Parent $Output)
    Invoke-Native $root 'gomobile' @('bind', '-target', 'android/arm64,android/amd64', '-androidapi', '26', '-javapkg', $JavaPackage, '-o', $Output, $Package)
    return $true
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

function Test-Android {
    Invoke-Native $root 'go' @('test', './sdk/bindings/android/...', '-count=1')
    $aar = Join-Path $root 'apps/android/app/libs/myflowhub.aar'
    if (Build-MobileBinding './sdk/bindings/android' 'com.myflowhub.mobile' $aar) { Gradle-Command (Join-Path $root 'apps/android') @('testDebugUnitTest', 'lintDebug') }
}

function Build-Android {
    $aar = Join-Path $root 'apps/android/app/libs/myflowhub.aar'
    if (Build-MobileBinding './sdk/bindings/android' 'com.myflowhub.mobile' $aar) { Gradle-Command (Join-Path $root 'apps/android') @('assembleDebug') }
}

function Test-Metrics {
    Invoke-Native $root 'go' @('test', './apps/nodes/metrics/...', '-count=1')
    if (Require-Tool 'npm' 'Metrics Windows frontend tests') {
        $frontend = Join-Path $root 'apps/nodes/metrics/windows/frontend'
        Invoke-Native $frontend 'npm' @('ci'); Invoke-Native $frontend 'npm' @('test'); Invoke-Native $frontend 'npm' @('run', 'build')
    }
    $aar = Join-Path $root 'apps/nodes/metrics/android/app/libs/metricsmobile.aar'
    if (Build-MobileBinding './apps/nodes/metrics/android/mobile' 'com.myflowhub.metrics' $aar) { Gradle-Command (Join-Path $root 'apps/nodes/metrics/android') @('testDebugUnitTest', 'lintDebug') }
}

function Generate-Metrics {
    if (-not (Require-Tool 'wails' 'Metrics Wails bindings')) { return }
    Invoke-Native (Join-Path $root 'apps/nodes/metrics/windows') 'wails' @('generate', 'module')
}

function Build-Metrics {
    $aar = Join-Path $root 'apps/nodes/metrics/android/app/libs/metricsmobile.aar'
    if (Build-MobileBinding './apps/nodes/metrics/android/mobile' 'com.myflowhub.metrics' $aar) { Gradle-Command (Join-Path $root 'apps/nodes/metrics/android') @('assembleDebug') }
    if ($isWindowsHost -and (Require-Tool 'wails' 'Metrics Windows production build')) {
        Invoke-Native (Join-Path $root 'apps/nodes/metrics/windows') 'wails' @('build', '-clean', '-trimpath', '-platform', 'windows/amd64', '-o', 'mfh-metrics.exe')
    } elseif (-not $isWindowsHost -and -not $AllowUnavailable) { throw 'Metrics Windows production build requires Windows' }
}

function Test-Clipboard {
    Invoke-Native $root 'go' @('test', './apps/nodes/clipboard/...', './sdk/bindings/clipboard', '-count=1')
    if (-not (Require-Tool 'flutter' 'Clipboard Flutter tests')) { return }
    $app = Join-Path $root 'apps/nodes/clipboard/app'
    Invoke-Native $app 'flutter' @('pub', 'get')
    Invoke-Native $app 'flutter' @('analyze')
    Invoke-Native $app 'flutter' @('test')
}

function Build-Clipboard {
    if (-not (Require-Tool 'flutter' 'Clipboard application builds')) { return }
    $aar = Join-Path $root 'apps/nodes/clipboard/app/android/app/libs/clipboardmobile.aar'
    if (-not (Build-MobileBinding './sdk/bindings/clipboard' 'com.myflowhub.gomobile' $aar)) { return }
    $app = Join-Path $root 'apps/nodes/clipboard/app'
    Invoke-Native $app 'flutter' @('build', 'web', '--release')
    Invoke-Native $app 'flutter' @('build', 'apk', '--debug', '--target-platform', 'android-arm64,android-x64', '--split-per-abi')
    if ($isWindowsHost) {
        Invoke-Native $app 'flutter' @('build', 'windows', '--release')
        $release = Join-Path $app 'build/windows/x64/runner/Release'
        Invoke-Native $root 'go' @('build', '-trimpath', '-o', (Join-Path $release 'mfh-clipboard.exe'), './cmd/mfh-clipboard')
    } elseif (-not $AllowUnavailable) { throw 'Clipboard Windows production build requires Windows' }
}

function Test-Embedded {
    if (Require-Tool 'cmake' 'Embedded C tests') {
        $build = Join-Path $out 'cmake/embedded-c'; Ensure-Directory (Split-Path -Parent $build)
        $configure = @('-S', 'embedded/c', '-B', $build, '-DCMAKE_BUILD_TYPE=Release')
        if ($isWindowsHost -and (Get-Command 'mingw32-make' -ErrorAction SilentlyContinue)) { $configure += @('-G', 'MinGW Makefiles') }
        Invoke-Native $root 'cmake' $configure
        Invoke-Native $root 'cmake' @('--build', $build)
        Invoke-Native $root 'ctest' @('--test-dir', $build, '--output-on-failure')
    }
    if (Require-Tool 'python' 'MicroPython-compatible host tests') { Invoke-Native $root 'python' @('-m', 'unittest', 'discover', '-s', 'embedded/micropython/tests', '-v') }
    Invoke-Native $root 'go' @('test', './internal/protocoltest', './protocol', '-count=1')
}

function Build-Embedded {
    Test-Embedded
    if (Require-Tool 'idf.py' 'ESP-IDF v6 build') {
        Invoke-Native (Join-Path $root 'embedded/esp32') 'idf.py' @('set-target', 'esp32s3')
        Invoke-Native (Join-Path $root 'embedded/esp32') 'idf.py' @('build')
    }
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
        @{ root = 'apps/android/app/build/outputs/apk'; filter = '*.apk'; recurse = $true },
        @{ root = 'apps/android/app/libs'; filter = '*.aar'; recurse = $false },
        @{ root = 'apps/nodes/metrics/windows/build/bin'; filter = '*'; recurse = $true },
        @{ root = 'apps/nodes/metrics/android/app/build/outputs/apk'; filter = '*.apk'; recurse = $true },
        @{ root = 'apps/nodes/metrics/android/app/libs'; filter = '*.aar'; recurse = $false },
        @{ root = 'apps/nodes/clipboard/app/build/windows/x64/runner/Release'; filter = '*'; recurse = $true },
        @{ root = 'apps/nodes/clipboard/app/build/app/outputs/flutter-apk'; filter = '*.apk'; recurse = $true },
        @{ root = 'apps/nodes/clipboard/app/build/web'; filter = '*'; recurse = $true },
        @{ root = 'apps/nodes/clipboard/app/android/app/libs'; filter = '*.aar'; recurse = $false },
        @{ root = 'embedded/esp32/build'; filter = '*.bin'; recurse = $false }
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

$targetOrder = @('core', 'hub', 'desktop', 'android', 'metrics', 'clipboard', 'embedded')
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
        'test/android' { Test-Android }
        'test/metrics' { Test-Metrics }
        'test/clipboard' { Test-Clipboard }
        'test/embedded' { Test-Embedded }
        'build/core' { Build-Core }
        'build/hub' { Build-Hub }
        'build/desktop' { Build-Desktop }
        'build/android' { Build-Android }
        'build/metrics' { Build-Metrics }
        'build/clipboard' { Build-Clipboard }
        'build/embedded' { Build-Embedded }
        'generate/core' { Generate-Contracts }
        'generate/hub' { }
        'generate/desktop' { Generate-Desktop }
        'generate/android' { [void](Build-MobileBinding './sdk/bindings/android' 'com.myflowhub.mobile' (Join-Path $root 'apps/android/app/libs/myflowhub.aar')) }
        'generate/metrics' { Generate-Metrics; [void](Build-MobileBinding './apps/nodes/metrics/android/mobile' 'com.myflowhub.metrics' (Join-Path $root 'apps/nodes/metrics/android/app/libs/metricsmobile.aar')) }
        'generate/clipboard' { [void](Build-MobileBinding './sdk/bindings/clipboard' 'com.myflowhub.gomobile' (Join-Path $root 'apps/nodes/clipboard/app/android/app/libs/clipboardmobile.aar')) }
        'generate/embedded' { }
        'check/core' { Check-Core }
        'check/hub' { Test-Hub }
        'check/desktop' { Test-Desktop }
        'check/android' { Test-Android }
        'check/metrics' { Test-Metrics }
        'check/clipboard' { Test-Clipboard }
        'check/embedded' { Test-Embedded }
        default { throw "unsupported action/target: $Action/$current" }
    }
}
