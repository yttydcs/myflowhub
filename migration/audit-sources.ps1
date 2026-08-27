[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$LegacyRoot
)

$ErrorActionPreference = 'Stop'
$resolvedRoot = (Resolve-Path -LiteralPath $LegacyRoot).Path
$manifest = Get-Content -Raw -LiteralPath (Join-Path $PSScriptRoot 'source-audit.json') | ConvertFrom-Json

foreach ($repository in $manifest.repositories) {
    $repositoryPath = Join-Path $resolvedRoot $repository.name
    if (-not (Test-Path -LiteralPath (Join-Path $repositoryPath '.git'))) {
        throw "legacy repository is missing: $repositoryPath"
    }

    $commit = (& git -C $repositoryPath rev-parse HEAD).Trim()
    if ($LASTEXITCODE -ne 0 -or $commit -ne $repository.commit) {
        throw "$($repository.name): HEAD $commit does not match $($repository.commit)"
    }
    $tree = (& git -C $repositoryPath show -s '--format=%T' $repository.commit).Trim()
    if ($LASTEXITCODE -ne 0 -or $tree -ne $repository.tree) {
        throw "$($repository.name): tree $tree does not match $($repository.tree)"
    }
    $fileCount = @(& git -C $repositoryPath ls-tree -r --name-only $repository.commit).Count
    if ($LASTEXITCODE -ne 0 -or $fileCount -ne $repository.file_count) {
        throw "$($repository.name): file count $fileCount does not match $($repository.file_count)"
    }

    $actual = @{}
    foreach ($line in @(& git -C $repositoryPath status --porcelain=v1 -uall)) {
        if ($line.Length -lt 4) {
            throw "$($repository.name): invalid porcelain status line: $line"
        }
        $actual[$line.Substring(3).Replace('\', '/')] = $line.Substring(0, 2).Trim()
    }
    if ($actual.Count -ne $repository.dirty.Count) {
        throw "$($repository.name): dirty path count $($actual.Count) does not match $($repository.dirty.Count)"
    }
    foreach ($expected in $repository.dirty) {
        if (-not $actual.ContainsKey($expected.path) -or $actual[$expected.path] -ne $expected.status) {
            throw "$($repository.name): dirty status drift for $($expected.path)"
        }
        $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath (Join-Path $repositoryPath $expected.path)).Hash.ToLowerInvariant()
        if ($hash -ne $expected.sha256) {
            throw "$($repository.name): dirty content drift for $($expected.path)"
        }
    }
    Write-Host "verified $($repository.name)@$commit"
}

Write-Host "verified $($manifest.repositories.Count) read-only migration sources"
