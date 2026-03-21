<#
.SYNOPSIS
  Preview or clean up MCP runtimes accumulated by long-running Codex sessions.

.DESCRIPTION
  The script only matches known MCP-related `node.exe` / `cmd.exe` processes:
  - chrome-devtools-mcp (including watchdog)
  - ace-tool
  - context7-mcp

  It is designed for temporary mitigation:
  - Supports Preview / Kill
  - Filters by process age to reduce the chance of killing active sessions
  - Supports precise filtering by Kind

.EXAMPLE
  .\cleanup-codex-mcp.ps1

.EXAMPLE
  .\cleanup-codex-mcp.ps1 -Mode Kill -Kinds chrome -MinAgeMinutes 45

.EXAMPLE
  .\cleanup-codex-mcp.ps1 -Mode Kill -Kinds chrome,ace,context7 -MinAgeMinutes 90 -WhatIf
#>

[CmdletBinding()]
param(
  [Parameter()]
  [ValidateSet("Preview", "Kill")]
  [string]$Mode = "Preview",

  [Parameter()]
  [string[]]$Kinds = @("chrome"),

  [Parameter()]
  [ValidateRange(0, 10080)]
  [int]$MinAgeMinutes = 45
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Get-KindRuleMap {
  return [ordered]@{
    chrome = "chrome-devtools-mcp|telemetry[\\/]+watchdog[\\/]+main\.js"
    ace = "(^|[\\/\s])ace-tool(@latest)?($|[\\/\s])|ace-tool[\\/]+dist[\\/]+index\.js"
    context7 = "context7-mcp|@upstash[\\/]+context7-mcp"
  }
}

function Normalize-Kinds {
  param(
    [Parameter(Mandatory = $true)]
    [string[]]$Kinds
  )

  $allowed = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::OrdinalIgnoreCase)
  foreach ($value in @("chrome", "ace", "context7")) {
    [void]$allowed.Add($value)
  }

  $normalized = New-Object System.Collections.Generic.List[string]
  foreach ($rawKind in $Kinds) {
    if ($null -eq $rawKind) {
      continue
    }

    foreach ($part in ($rawKind -split ",")) {
      $kind = $part.Trim().ToLowerInvariant()
      if ($kind -eq "") {
        continue
      }
      if (-not $allowed.Contains($kind)) {
        throw "Invalid Kinds value: '$kind'. Allowed values: chrome, ace, context7."
      }
      if (-not $normalized.Contains($kind)) {
        [void]$normalized.Add($kind)
      }
    }
  }

  if ($normalized.Count -eq 0) {
    throw "Kinds cannot be empty."
  }

  return @($normalized)
}

function Get-KindFromCommandLine {
  param(
    [Parameter(Mandatory = $true)]
    [string]$CommandLine,

    [Parameter(Mandatory = $true)]
    [hashtable]$RuleMap
  )

  foreach ($key in $RuleMap.Keys) {
    if ($CommandLine -match $RuleMap[$key]) {
      return $key
    }
  }

  return $null
}

function Get-MatchingProcesses {
  param(
    [Parameter(Mandatory = $true)]
    [string[]]$Kinds,

    [Parameter(Mandatory = $true)]
    [hashtable]$RuleMap
  )

  $allowedKinds = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::OrdinalIgnoreCase)
  foreach ($kind in $Kinds) {
    [void]$allowedKinds.Add($kind)
  }

  $now = Get-Date
  $procs = Get-CimInstance Win32_Process | Where-Object {
    $_.Name -in @("node.exe", "cmd.exe") -and
    $_.CommandLine
  }

  $rows = foreach ($proc in $procs) {
    $kind = Get-KindFromCommandLine -CommandLine $proc.CommandLine -RuleMap $RuleMap
    if (-not $kind) {
      continue
    }
    if (-not $allowedKinds.Contains($kind)) {
      continue
    }

    $created = [DateTime]$proc.CreationDate
    [pscustomobject]@{
      Kind = $kind
      PID = [int]$proc.ProcessId
      PPID = [int]$proc.ParentProcessId
      Name = [string]$proc.Name
      Created = $created
      AgeMin = [math]::Round(($now - $created).TotalMinutes, 1)
      WS_MB = [math]::Round(([double]$proc.WorkingSetSize) / 1MB, 1)
      CommandLine = [string]$proc.CommandLine
    }
  }

  return $rows
}

function Show-Summary {
  param(
    [Parameter(Mandatory = $true)]
    [object[]]$Rows
  )

  if (-not $Rows -or $Rows.Count -eq 0) {
    Write-Host "Summary: no matching MCP runtime found." -ForegroundColor Yellow
    return
  }

  $now = Get-Date
  $summary = $Rows |
    Group-Object Kind |
    Sort-Object Count -Descending |
    ForEach-Object {
      $group = $_.Group
      [pscustomobject]@{
        Kind = $_.Name
        Count = $_.Count
        WorkingSetMB = [math]::Round((($group | Measure-Object -Property WS_MB -Sum).Sum), 1)
        NewLast30m = ($group | Where-Object { $_.Created -gt $now.AddMinutes(-30) }).Count
        NewLast120m = ($group | Where-Object { $_.Created -gt $now.AddMinutes(-120) }).Count
      }
    }

  Write-Host "Summary:" -ForegroundColor Cyan
  $summary | Format-Table -AutoSize | Out-Host
}

function Show-Targets {
  param(
    [Parameter(Mandatory = $true)]
    [object[]]$Rows,

    [Parameter(Mandatory = $true)]
    [int]$MinAgeMinutes
  )

  $targets = $Rows |
    Where-Object { $_.AgeMin -ge $MinAgeMinutes } |
    Sort-Object @{
      Expression = "Kind"
      Descending = $false
    }, @{
      Expression = "AgeMin"
      Descending = $true
    }, @{
      Expression = "PID"
      Descending = $false
    }

  Write-Host ""
  Write-Host "Eligible targets (older than $MinAgeMinutes min):" -ForegroundColor Cyan
  if (-not $targets -or $targets.Count -eq 0) {
    Write-Host "No eligible target found." -ForegroundColor Yellow
    return @()
  }

  $targets | Select-Object Kind, PID, PPID, Name, AgeMin, WS_MB | Format-Table -AutoSize | Out-Host
  return $targets
}

function Stop-Targets {
  param(
    [Parameter(Mandatory = $true)]
    [object[]]$Targets
  )

  if (-not $Targets -or $Targets.Count -eq 0) {
    Write-Host ""
    Write-Host "Nothing to kill." -ForegroundColor Yellow
    return
  }

  $ordered = $Targets | Sort-Object @{
      Expression = {
        if ($_.Name -eq "node.exe") { 0 } else { 1 }
      }
    }, @{
      Expression = "AgeMin"
      Descending = $true
    }, @{
      Expression = "PID"
      Descending = $false
    }

  Write-Host ""
  Write-Host "Killing $($ordered.Count) process(es)..." -ForegroundColor Cyan
  foreach ($target in $ordered) {
    $label = "$($target.Kind) $($target.Name) PID=$($target.PID) Age=$($target.AgeMin)m"
    try {
      Stop-Process -Id $target.PID -Force -ErrorAction Stop
      Write-Host "Killed $label" -ForegroundColor Green
    }
    catch {
      Write-Warning "Failed to kill PID $($target.PID): $($_.Exception.Message)"
    }
  }
}

$rules = Get-KindRuleMap
$Kinds = Normalize-Kinds -Kinds $Kinds
$rows = @(Get-MatchingProcesses -Kinds $Kinds -RuleMap $rules)

Write-Host ("Kinds: {0}" -f ($Kinds -join ", ")) -ForegroundColor Cyan
Write-Host ("Mode: {0}" -f $Mode) -ForegroundColor Cyan
Write-Host ("MinAgeMinutes: {0}" -f $MinAgeMinutes) -ForegroundColor Cyan
Write-Host ""

Show-Summary -Rows $rows
$targets = @(Show-Targets -Rows $rows -MinAgeMinutes $MinAgeMinutes)

if ($Mode -eq "Preview") {
  Write-Host ""
  Write-Host "Preview only. Re-run with -Mode Kill to terminate the targets above." -ForegroundColor Yellow
  return
}

if (-not $targets -or $targets.Count -eq 0) {
  Write-Host ""
  Write-Host "Nothing to kill." -ForegroundColor Yellow
  return
}

Stop-Targets -Targets $targets
