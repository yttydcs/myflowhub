[CmdletBinding()]
param([Parameter(Mandatory)][string]$BaselineRoot)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$env:GOWORK = 'off'
$repositoryRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..')).Path
$baseline = (Resolve-Path -LiteralPath $BaselineRoot).Path
$evidence = Join-Path $repositoryRoot 'out/phase-a/evidence'
$privateRoot = Join-Path ([IO.Path]::GetTempPath()) ("mfh-phase-a-compat-" + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $privateRoot | Out-Null
New-Item -ItemType Directory -Path $evidence -Force | Out-Null

function Invoke-CLI([string]$Executable, [string[]]$Arguments) {
    $lines = @(& $Executable @Arguments)
    if ($LASTEXITCODE -ne 0) { throw "$([IO.Path]::GetFileName($Executable)) failed with exit $LASTEXITCODE" }
    return ($lines -join "`n")
}
function Write-JSON([string]$Path, $Value) {
    [IO.File]::WriteAllText($Path, ($Value | ConvertTo-Json -Depth 12), [Text.UTF8Encoding]::new($false))
}
function Start-CLI([string]$Executable, [string[]]$Arguments, [string]$LogPrefix) {
    # Arguments are paths, IDs and pinned public keys from this isolated run.
    $quoted = @($Arguments | ForEach-Object { '"' + $_.Replace('"','\"') + '"' })
    return Start-Process -FilePath $Executable -ArgumentList $quoted -WindowStyle Hidden -PassThru -RedirectStandardOutput "$LogPrefix.stdout" -RedirectStandardError "$LogPrefix.stderr"
}
function Read-Startup([string]$Path, [Diagnostics.Process]$Process) {
    $deadline = [DateTime]::UtcNow.AddSeconds(15)
    do {
        if (Test-Path -LiteralPath $Path) {
            $line = Get-Content -LiteralPath $Path -TotalCount 1
            if ($line) { return ($line | ConvertFrom-Json) }
        }
        $Process.Refresh()
        if ($Process.HasExited) { throw "Process exited before ready: $Path" }
        Start-Sleep -Milliseconds 50
    } while ([DateTime]::UtcNow -lt $deadline)
    throw "Process did not announce readiness: $Path"
}
function Stop-Owned([Diagnostics.Process]$Process) {
    if ($null -eq $Process) { return }
    $Process.Refresh()
    if (-not $Process.HasExited) { $Process.Kill(); if (-not $Process.WaitForExit(5000)) { throw 'Test process did not stop' } }
}
function Wait-MetricsReady([string]$Path, [Diagnostics.Process]$Process) {
    $deadline = [DateTime]::UtcNow.AddSeconds(15)
    do {
        if ((Test-Path -LiteralPath $Path) -and (Select-String -LiteralPath $Path -SimpleMatch '"msg":"MetricsNode connected"' -Quiet)) { return }
        $Process.Refresh()
        if ($Process.HasExited) { throw "Metrics exited before ready: $Path" }
        Start-Sleep -Milliseconds 50
    } while ([DateTime]::UtcNow -lt $deadline)
    throw "Metrics did not announce connection: $Path"
}

# The baseline is the A01 git-archive export of b1fcd1a. Only the shared adapter
# is added here; baseline product/runtime source and its frozen binaries remain.
$adapterSource = Join-Path $repositoryRoot 'tests/fixtures/phase-a-desktop/main.go'
$oldAdapterDirectory = Join-Path $baseline 'tests/fixtures/phase-a-desktop'
New-Item -ItemType Directory -Path $oldAdapterDirectory -Force | Out-Null
Copy-Item -LiteralPath $adapterSource -Destination (Join-Path $oldAdapterDirectory 'main.go')
foreach ($sourceRoot in @($repositoryRoot,$baseline)) {
    Push-Location $sourceRoot
    try {
        & go build -trimpath -o out/bin/phase-a-desktop-bridge.exe ./tests/fixtures/phase-a-desktop
        if ($LASTEXITCODE -ne 0) { throw 'Desktop bridge adapter build failed' }
        & go build -trimpath -o out/bin/mfh-metrics.exe ./cmd/mfh-metrics
        if ($LASTEXITCODE -ne 0) { throw 'Metrics console build failed' }
    } finally { Pop-Location }
}
$roots = @($baseline,$repositoryRoot)
$admin = Join-Path $repositoryRoot 'out/bin/mfh-admin.exe'
$cases = @()
foreach ($hubVersion in 0..1) { foreach ($desktopVersion in 0..1) { foreach ($metricsVersion in 0..1) {
    $caseID = "$hubVersion$desktopVersion$metricsVersion"
    $caseRoot = Join-Path $privateRoot $caseID
    New-Item -ItemType Directory -Path $caseRoot | Out-Null
    $hub = Join-Path $roots[$hubVersion] 'out/bin/mfh-hub.exe'
    $desktop = Join-Path $roots[$desktopVersion] 'out/bin/phase-a-desktop-bridge.exe'
    $metrics = Join-Path $roots[$metricsVersion] 'out/bin/mfh-metrics.exe'
    $hubState = Join-Path $caseRoot 'hub'
    $desktopState = Join-Path $caseRoot 'desktop'
    $metricsState = Join-Path $caseRoot 'metrics'
    $hubProcess = $null; $metricsProcess = $null
    $record = [ordered]@{ combination=$caseID; result='failed'; hub_sha256=(Get-FileHash -LiteralPath $hub).Hash; desktop_bridge_sha256=(Get-FileHash -LiteralPath $desktop).Hash; metrics_console_sha256=(Get-FileHash -LiteralPath $metrics).Hash }
    try {
        $hubIdentity = Invoke-CLI $hub @('-state',$hubState,'-id','1','-identity') | ConvertFrom-Json
        $desktopIdentity = Invoke-CLI $desktop @('-state',$desktopState,'-identity') | ConvertFrom-Json
        $metricsIdentity = Invoke-CLI $metrics @('-state',$metricsState,'-id','3','-identity') | ConvertFrom-Json
        $adminState = Join-Path $caseRoot 'admin'
        $adminIdentity = Invoke-CLI $admin @('-state',$adminState,'-id','4','-op','identity') | ConvertFrom-Json
        $desktopPermit = Invoke-CLI $hub @('-state',$hubState,'-id','1','-issue-node-id','2','-issue-public-key',$desktopIdentity.public_key) | ConvertFrom-Json
        $metricsPermit = Invoke-CLI $hub @('-state',$hubState,'-id','1','-issue-node-id','3','-issue-public-key',$metricsIdentity.public_key) | ConvertFrom-Json
        $adminPermit = Invoke-CLI $hub @('-state',$hubState,'-id','1','-issue-node-id','4','-issue-public-key',$adminIdentity.public_key) | ConvertFrom-Json
        $adminPermitPath = Join-Path $caseRoot 'admin-permit.json'; Write-JSON $adminPermitPath $adminPermit
        $null = Invoke-CLI $hub @('-state',$hubState,'-id','1','-policy','bind','-subject','4','-definition','superadmin','-policy-scope','authority-domain')
        $desktopPermitPath = Join-Path $caseRoot 'desktop-permit.json'; Write-JSON $desktopPermitPath $desktopPermit
        $metricsPermitPath = Join-Path $caseRoot 'metrics-permit.json'; Write-JSON $metricsPermitPath $metricsPermit
        foreach ($grant in @(@('system/catalog','read'),@('metrics/config','read'),@('metrics/config','subscribe'),@('metrics/config/update','invoke'))) {
            $null = Invoke-CLI $hub @('-state',$hubState,'-id','1','-policy','grant','-subject','2','-resource-node','3','-resource',$grant[0],'-action',$grant[1])
        }
        $hubLog = Join-Path $caseRoot 'hub-log'
        $hubProcess = Start-CLI $hub @('-state',$hubState,'-id','1','-listen','127.0.0.1:0') $hubLog
        $hubReady = Read-Startup "$hubLog.stdout" $hubProcess
        $endpoint = [string]$hubReady.endpoints[0]
        $metricsLog = Join-Path $caseRoot 'metrics-log'
        $metricsProcess = Start-CLI $metrics @('-state',$metricsState,'-id','3','-parent-id','1','-parent-key',$hubIdentity.public_key,'-endpoint',$endpoint,'-permit',$metricsPermitPath) $metricsLog
        Wait-MetricsReady "$metricsLog.stderr" $metricsProcess
        $definitionFile = Join-Path $caseRoot 'definition.json'
        Write-JSON $definitionFile @{version=1;id='test.phase-a-cli';label='Compatibility CLI management';rules=@(@{resource=@{kind='exact';value='metrics/config'};capability=@{kind='exact';values=@('read')}})}
        $adminArguments = @('-state',$adminState,'-id','4','-parent-id','1','-parent-key',$hubIdentity.public_key,'-endpoint',$endpoint,'-permit',$adminPermitPath,'-op','operate','-owner-id','1')
        $null = Invoke-CLI $admin ($adminArguments + @('-resource','system/policy/definitions','-capability','create','-schema','mfh.policy.definition-put.v1','-request-file',$definitionFile))
        $bindingID = [Guid]::NewGuid().ToString('N')
        $bindingFile = Join-Path $caseRoot 'binding.json'
        Write-JSON $bindingFile @{version=1;binding_id=$bindingID;subject='2';definition_id='test.phase-a-cli';scope=@{kind='owner';node_id='3'}}
        $bindingResult = Invoke-CLI $admin ($adminArguments + @('-resource','system/policy/bindings','-capability','create','-schema','mfh.policy.binding-create.v1','-request-file',$bindingFile)) | ConvertFrom-Json
        if ($bindingResult.payload.binding_id -ne $bindingID) { throw 'Binding create returned a different ID' }
        $result = Invoke-CLI $desktop @('-state',$desktopState,'-endpoint',$endpoint,'-parent-key',$hubIdentity.public_key,'-permit',$desktopPermitPath) | ConvertFrom-Json
        if (-not $result.read -or -not $result.subscribe -or -not $result.configuration_control) { throw 'Incomplete compatibility result' }
        Write-JSON $bindingFile @{version=1;binding_id=$bindingID}
        $null = Invoke-CLI $admin ($adminArguments + @('-resource','system/policy/bindings','-capability','revoke','-schema','mfh.policy.binding-revoke.v1','-request-file',$bindingFile))
        $record.result='passed'; $record.details=$result; $record.management_json_operate=$true
    } catch { $record.error=$_.Exception.Message }
    finally { Stop-Owned $metricsProcess; Stop-Owned $hubProcess }
    $cases += $record
    Write-Host "Desktop-binding compatibility $caseID : $($record.result)"
} } }
$rollback = [ordered]@{result='failed'}
if (@($cases | Where-Object { $_.result -ne 'passed' }).Count -eq 0) {
    $hubProcess=$null; $metricsProcess=$null
    try {
        # Keep the exact N-saved identity/config/policy files, including a new
        # revocation. Do not resurrect an earlier policy backup for rollback.
        $null = Invoke-CLI (Join-Path $repositoryRoot 'out/bin/mfh-hub.exe') @('-state',$hubState,'-id','1','-policy','revoke','-subject','2','-resource-node','3','-resource','metrics/config/update','-action','invoke')
        $hubLog = Join-Path $caseRoot 'rollback-hub-log'
        $hubProcess = Start-CLI (Join-Path $baseline 'out/bin/mfh-hub.exe') @('-state',$hubState,'-id','1','-listen','127.0.0.1:0') $hubLog
        $ready = Read-Startup "$hubLog.stdout" $hubProcess
        $endpoint = [string]$ready.endpoints[0]
        $metricsLog = Join-Path $caseRoot 'rollback-metrics-log'
        $metricsProcess = Start-CLI (Join-Path $baseline 'out/bin/mfh-metrics.exe') @('-state',$metricsState,'-id','3','-parent-id','1','-parent-key',$hubIdentity.public_key,'-endpoint',$endpoint,'-permit',$metricsPermitPath) $metricsLog
        Wait-MetricsReady "$metricsLog.stderr" $metricsProcess
        $result = Invoke-CLI (Join-Path $baseline 'out/bin/phase-a-desktop-bridge.exe') @('-state',$desktopState,'-endpoint',$endpoint,'-parent-key',$hubIdentity.public_key,'-permit',$desktopPermitPath,'-expect-control-denied') | ConvertFrom-Json
        if (-not $result.configuration_control_denied) { throw 'Rollback did not enforce latest revocation' }
        $rollback.result='passed';$rollback.details=$result
    } catch { $rollback.error=$_.Exception.Message }
    finally { Stop-Owned $metricsProcess; Stop-Owned $hubProcess }
}
$report = [ordered]@{
    version=1; baseline_commit='b1fcd1ac0319ebb349c8289cc6f61ed2d7302dc8'; candidate_base_commit=(& git -C $repositoryRoot rev-parse HEAD).Trim(); candidate_working_tree=$true;
    adapter_sha256=(Get-FileHash -LiteralPath $adapterSource).Hash; scope='actual Hub and Metrics console processes, versioned Desktop Go binding adapter; configuration control only';
    wails_ui=$false; physical_audio=$false; formal_AC16='pending'; private_logs=$privateRoot; cases=$cases; rollback_latest_revocation=$rollback
}
Write-JSON (Join-Path $evidence 'compatibility-bridge-matrix.json') $report
if (@($cases | Where-Object { $_.result -ne 'passed' }).Count -or $rollback.result -ne 'passed') { throw 'Compatibility adapter matrix has failures; inspect private logs and report' }
