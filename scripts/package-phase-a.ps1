[CmdletBinding()]
param([Parameter(Mandatory)][string]$OutputDirectory)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$repositoryRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..')).Path
$packageRoot = [IO.Path]::GetFullPath($OutputDirectory)
if (-not [IO.Path]::IsPathRooted($OutputDirectory)) { throw 'OutputDirectory must be absolute' }
$products = @(
    @{ name='hub'; files=@('out/bin/mfh-hub.exe','out/bin/mfh-admin.exe') },
    @{ name='desktop'; files=@('apps/desktop/build/bin/mfh-desktop.exe') },
    @{ name='metrics'; files=@('apps/nodes/metrics/windows/build/bin/mfh-metrics.exe','out/bin/mfh-metrics.exe') },
    @{ name='variable-example'; files=@('out/bin/mfh-variable-example.exe') }
)
foreach ($product in $products) {
    foreach ($relative in $product.files) {
        $binaryPath = Join-Path $repositoryRoot $relative
        if (-not (Test-Path -LiteralPath $binaryPath -PathType Leaf)) { throw "Missing $relative; build through scripts/mfh.ps1 first" }
        $binary = [IO.File]::OpenRead($binaryPath)
        $reader = [IO.BinaryReader]::new($binary)
        try {
            if ($reader.ReadUInt16() -ne 0x5A4D) { throw "Not a Windows executable: $relative" }
            $null = $binary.Seek(0x3C, [IO.SeekOrigin]::Begin)
            $peOffset = $reader.ReadUInt32()
            $null = $binary.Seek($peOffset, [IO.SeekOrigin]::Begin)
            if ($reader.ReadUInt32() -ne 0x00004550 -or $reader.ReadUInt16() -ne 0x8664) { throw "Not a Windows x64 executable: $relative" }
        } finally { $reader.Dispose(); $binary.Dispose() }
    }
}
Push-Location $repositoryRoot
try {
    $baseCommit = (& git rev-parse HEAD).Trim()
    if ($LASTEXITCODE -ne 0) { throw 'Cannot resolve source commit' }
    $dirty = [bool](& git status --porcelain)
    $sourcePaths = @(& git -c core.quotepath=false ls-files --cached --others --exclude-standard | Sort-Object -Unique)
    if ($LASTEXITCODE -ne 0) { throw 'Cannot enumerate candidate source' }
    $sources = @($sourcePaths | ForEach-Object {
        $absolute = Join-Path $repositoryRoot $_
        if (Test-Path -LiteralPath $absolute -PathType Leaf) {
            [ordered]@{ path=$_; sha256=(Get-FileHash -LiteralPath $absolute -Algorithm SHA256).Hash.ToLowerInvariant() }
        } else { [ordered]@{ path=$_; deleted=$true } }
    })
    $sourceJSON = $sources | ConvertTo-Json -Depth 4 -Compress
    $hasher = [Security.Cryptography.SHA256]::Create()
    try { $sourceHash = ([BitConverter]::ToString($hasher.ComputeHash([Text.Encoding]::UTF8.GetBytes($sourceJSON)))).Replace('-', '').ToLowerInvariant() }
    finally { $hasher.Dispose() }
    # A working tree build is a preview, never relabelled as the old HEAD's N.
    $identity = if ($dirty) { "worktree-$($sourceHash.Substring(0,12))" } else { $baseCommit.Substring(0,12) }
    $destination = Join-Path $packageRoot "phase-a-$identity"
    if (Test-Path -LiteralPath $destination) { throw "Package destination already exists: $destination" }
    New-Item -ItemType Directory -Path $destination -Force | Out-Null
    $sources | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath (Join-Path $destination 'source-files.json') -Encoding utf8
    $artifacts = @()
    foreach ($product in $products) {
        $directory = Join-Path $destination $product.name
        New-Item -ItemType Directory -Path $directory | Out-Null
        foreach ($relative in $product.files) {
            $fileName = [IO.Path]::GetFileName($relative)
            if ($relative -eq 'out/bin/mfh-metrics.exe') { $fileName = 'mfh-metrics-console.exe' }
            $target = Join-Path $directory $fileName
            Copy-Item -LiteralPath (Join-Path $repositoryRoot $relative) -Destination $target
            $artifacts += [ordered]@{ product=$product.name; file=[IO.Path]::GetFileName($target); bytes=(Get-Item -LiteralPath $target).Length; sha256=(Get-FileHash -LiteralPath $target -Algorithm SHA256).Hash.ToLowerInvariant() }
        }
        Copy-Item -LiteralPath (Join-Path $repositoryRoot 'docs/specs/phase-a-deployment.md') -Destination (Join-Path $directory 'DEPLOYMENT.zh-CN.md')
        Copy-Item -LiteralPath (Join-Path $destination 'source-files.json') -Destination $directory
        if ($product.name -eq 'variable-example') {
            Copy-Item -LiteralPath (Join-Path $repositoryRoot 'examples/variables') -Destination (Join-Path $directory 'provider-source') -Recurse
            Copy-Item -LiteralPath (Join-Path $repositoryRoot 'cmd/mfh-variable-example') -Destination (Join-Path $directory 'command-source') -Recurse
        }
        [ordered]@{ version=1; product=$product.name; base_commit=$baseCommit; working_tree=$dirty; source_sha256=$sourceHash; platform='windows/amd64'; acceptance_status='pending'; artifacts=@($artifacts | Where-Object { $_.product -eq $product.name }) } |
            ConvertTo-Json -Depth 5 | Set-Content -LiteralPath (Join-Path $directory 'manifest.json') -Encoding utf8
        $archive = Join-Path $destination "$($product.name)-windows-x64-$identity.zip"
        Compress-Archive -LiteralPath $directory -DestinationPath $archive -CompressionLevel Optimal
    }
    Add-Type -AssemblyName System.IO.Compression
    $sourceArchive = Join-Path $destination "source-$identity.zip"
    $stream = [IO.File]::Open($sourceArchive, [IO.FileMode]::CreateNew)
    $zip = [IO.Compression.ZipArchive]::new($stream, [IO.Compression.ZipArchiveMode]::Create)
    try {
        foreach ($relative in $sourcePaths) {
            $source = Join-Path $repositoryRoot $relative
            if (-not (Test-Path -LiteralPath $source -PathType Leaf)) { continue }
            $entry = $zip.CreateEntry($relative.Replace('\', '/'), [IO.Compression.CompressionLevel]::Optimal)
            $entryStream = $entry.Open()
            $inputStream = [IO.File]::OpenRead($source)
            try { $inputStream.CopyTo($entryStream) } finally { $inputStream.Dispose(); $entryStream.Dispose() }
        }
    } finally { $zip.Dispose(); $stream.Dispose() }
    $archives = @(Get-ChildItem -LiteralPath $destination -Filter '*.zip' -File | ForEach-Object { [ordered]@{ file=$_.Name; bytes=$_.Length; sha256=(Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant() } })
    [ordered]@{ version=1; created_at_utc=[DateTimeOffset]::UtcNow.ToString('O'); base_commit=$baseCommit; working_tree=$dirty; source_sha256=$sourceHash; acceptance_status='pending'; artifacts=$artifacts; archives=$archives } |
        ConvertTo-Json -Depth 6 | Set-Content -LiteralPath (Join-Path $destination 'manifest.json') -Encoding utf8
    Write-Output $destination
} finally { Pop-Location }
