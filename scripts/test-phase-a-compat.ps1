[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$BaselineRoot,
    [ValidatePattern('^[0-9a-fA-F]{40}$')][string]$BaselineCommit = 'b1fcd1ac0319ebb349c8289cc6f61ed2d7302dc8'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$env:GOWORK = 'off'
$repositoryRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..')).Path
$baseline = (Resolve-Path -LiteralPath $BaselineRoot).Path
$resolvedBaseline = (& git -C $repositoryRoot rev-parse --verify "$BaselineCommit^{commit}").Trim()
if ($LASTEXITCODE -ne 0 -or $resolvedBaseline -ne $BaselineCommit.ToLowerInvariant()) { throw 'BaselineCommit must identify an available exact commit' }
# Verify the archived Go/module inputs using repository text normalization
# (Windows module tooling may rewrite line endings), before adding the adapter.
$baselineInputs = @(& git -C $repositoryRoot ls-tree -r --name-only $resolvedBaseline | Where-Object { $_ -match '\.go$|(^|/)go\.(mod|sum)$' -and $_ -ne 'tests/fixtures/phase-a-desktop/main.go' })
if ($LASTEXITCODE -ne 0 -or $baselineInputs.Count -eq 0) { throw 'Cannot enumerate baseline inputs' }
foreach ($relative in $baselineInputs) {
    $path = Join-Path $baseline $relative
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) { throw "Missing baseline input: $relative" }
    $expected = (& git -C $repositoryRoot rev-parse "${resolvedBaseline}:$relative").Trim()
    if ($LASTEXITCODE -ne 0) { throw "Cannot resolve baseline input: $relative" }
    $actual = (& git -C $repositoryRoot hash-object "--path=$relative" -- $path).Trim()
    if ($LASTEXITCODE -ne 0 -or $actual -ne $expected) { throw "Baseline input differs from $resolvedBaseline : $relative" }
}
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

# The baseline is a verified git-archive export. Only the shared adapter
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
# Keep three independent real processes alive while replacing only one product.
# The consumer is a Desktop Go binding fixture; this is not native UI evidence.
$lifecycle = @()
if (@($cases | Where-Object { $_.result -ne 'passed' }).Count -eq 0 -and $rollback.result -eq 'passed') {
    $processes = @{}
    $versions = @{ hub=0; desktop=0; metrics=0 }
    $endpoint = ''
    function Start-LifecycleProduct([string]$Product, [int]$Version) {
        $source = $roots[$Version]
        $log = Join-Path $caseRoot ('independent-' + $Product + '-' + [Guid]::NewGuid().ToString('N'))
        if ($Product -eq 'hub') {
            $address = if ($script:lifecycleEndpoint) { $script:lifecycleEndpoint } else { '127.0.0.1:0' }
            $process = Start-CLI (Join-Path $source 'out/bin/mfh-hub.exe') @('-state',$hubState,'-id','1','-listen',$address) $log
            $processes[$Product] = $process
            $ready = Read-Startup "$log.stdout" $process
            $script:lifecycleEndpoint = [string]$ready.endpoints[0]
        } elseif ($Product -eq 'metrics') {
            $process = Start-CLI (Join-Path $source 'out/bin/mfh-metrics.exe') @('-state',$metricsState,'-id','3','-parent-id','1','-parent-key',$hubIdentity.public_key,'-endpoint',$script:lifecycleEndpoint,'-permit',$metricsPermitPath) $log
            $processes[$Product] = $process
            Wait-MetricsReady "$log.stderr" $process
        } else {
            $info = [Diagnostics.ProcessStartInfo]::new()
            $info.FileName = Join-Path $source 'out/bin/phase-a-desktop-bridge.exe'
            $info.UseShellExecute = $false; $info.CreateNoWindow = $true
            $info.RedirectStandardInput = $true; $info.RedirectStandardOutput = $true
            foreach ($argument in @('-state',$desktopState,'-endpoint',$script:lifecycleEndpoint,'-parent-key',$hubIdentity.public_key,'-permit',$desktopPermitPath,'-expect-control-denied','-serve')) {
                $info.ArgumentList.Add($argument)
            }
            $process = [Diagnostics.Process]::new(); $process.StartInfo = $info
            if (-not $process.Start()) { throw 'Cannot start persistent Desktop binding fixture' }
            $processes[$Product] = $process
        }
    }
    function Invoke-LifecycleProbe {
        $bridge = $processes['desktop']
        $bridge.Refresh(); if ($bridge.HasExited) { throw 'Persistent Desktop binding fixture exited' }
        $bridge.StandardInput.WriteLine('probe'); $bridge.StandardInput.Flush()
        $read = $bridge.StandardOutput.ReadLineAsync()
        if (-not $read.Wait(20000)) { throw 'Persistent consumer probe timed out' }
        if (-not $read.Result) { throw 'Persistent consumer closed its response stream' }
        $result = $read.Result | ConvertFrom-Json
        if (-not $result.read -or -not $result.subscribe -or -not $result.configuration_control_denied) { throw 'Latest revocation or read/subscription was not preserved' }
        return $result
    }
    $script:lifecycleEndpoint = ''
    try {
    $expectedConfigHash = (Get-FileHash -LiteralPath (Join-Path $metricsState 'state/metrics.json')).Hash
    $identityHashes = @{}
    foreach ($entry in @(@('hub',$hubState),@('desktop',$desktopState),@('metrics',$metricsState))) {
        $identityHashes[$entry[0]] = (Get-FileHash -LiteralPath (Join-Path $entry[1] 'state/identity.json')).Hash
    }

        foreach ($product in @('hub','metrics','desktop')) { Start-LifecycleProduct $product 0 }
        $null = Invoke-LifecycleProbe
        foreach ($step in @(@('desktop',1),@('metrics',1),@('hub',1),@('desktop',0),@('metrics',0),@('hub',0))) {
            $target = [string]$step[0]; $version = [int]$step[1]
            $survivors = @{}
            foreach ($product in @('hub','desktop','metrics')) {
                if ($product -eq $target) { continue }
                $process = $processes[$product]; $process.Refresh()
                if ($process.HasExited) { throw "Sibling $product exited before replacement" }
                $survivors[$product] = @{ pid=$process.Id; created_ticks=$process.StartTime.ToUniversalTime().Ticks }
            }
            Stop-Owned $processes[$target]
            $started = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
            Start-LifecycleProduct $target $version
            $result = Invoke-LifecycleProbe
            $elapsed = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds() - $started
            foreach ($product in $survivors.Keys) {
                $process = [Diagnostics.Process]::GetProcessById($survivors[$product].pid)
                if ($process.HasExited -or $process.StartTime.ToUniversalTime().Ticks -ne $survivors[$product].created_ticks) { throw "Sibling $product restarted during replacement" }
            }
            foreach ($entry in @(@('hub',$hubState),@('desktop',$desktopState),@('metrics',$metricsState))) {
                if ((Get-FileHash -LiteralPath (Join-Path $entry[1] 'state/identity.json')).Hash -ne $identityHashes[$entry[0]]) { throw 'Product identity changed during replacement' }
            }
            if ((Get-FileHash -LiteralPath (Join-Path $metricsState 'state/metrics.json')).Hash -ne $expectedConfigHash) { throw 'Saved Metrics configuration changed under denied writes' }
            if ($elapsed -gt 15000) { throw "Independent $target startup exceeded 15 seconds: $elapsed ms" }
            $versions[$target] = $version
            $lifecycle += [ordered]@{ target=$target; version=$version; direction=$(if ($version) {'upgrade'} else {'rollback'}); result='passed'; recovery_ms=$elapsed; surviving_processes=$survivors; identities_unchanged=$true; configuration_unchanged=$true; latest_revocation_enforced=$true; read_subscribe=$true; wails_ui=$false }
            Write-Host "Independent $target version $version : passed ($elapsed ms)"
        }
    } catch { $lifecycle += [ordered]@{result='failed';error=$_.Exception.Message} }
    finally { foreach ($process in $processes.Values) { Stop-Owned $process } }
}
$report = [ordered]@{
    version=1; baseline_commit=$resolvedBaseline; baseline_verified_inputs=$baselineInputs.Count; candidate_base_commit=(& git -C $repositoryRoot rev-parse HEAD).Trim(); candidate_working_tree=$true;
    adapter_sha256=(Get-FileHash -LiteralPath $adapterSource).Hash; scope='actual Hub and Metrics console processes, versioned Desktop Go binding adapter; configuration control only';
    wails_ui=$false; physical_audio=$false; formal_AC16='pending'; private_logs=$privateRoot; cases=$cases; rollback_latest_revocation=$rollback; independent_product_lifecycle=$lifecycle
}
Write-JSON (Join-Path $evidence 'compatibility-bridge-matrix.json') $report
if (@($cases | Where-Object { $_.result -ne 'passed' }).Count -or $rollback.result -ne 'passed' -or $lifecycle.Count -ne 6 -or @($lifecycle | Where-Object { $_.result -ne 'passed' }).Count) { throw 'Compatibility adapter matrix has failures; inspect private logs and report' }
