<#
.SYNOPSIS
  生成 Android Release 签名 keystore，并输出 GitHub Actions Secrets 配置指引。

.DESCRIPTION
  MyFlowHub-Android 的 Release workflow 需要以下 Secrets：
    - ANDROID_KEYSTORE_BASE64
    - ANDROID_KEYSTORE_PASSWORD
    - ANDROID_KEY_ALIAS
    - ANDROID_KEY_PASSWORD

  本脚本会：
    1) 尝试定位 keytool（PATH/JAVA_HOME/常见安装路径）
    2) 交互式生成 keystore（不会把密码写入文件）
    3) 生成单行 base64 文件（用于粘贴到 GitHub Secrets）
    4) 输出清晰的配置与验证步骤

.PARAMETER OutDir
  输出目录。默认：<workspace>\.tmp\android-signing

.PARAMETER Alias
  keystore 内 key 的 alias。默认：myflowhub

.PARAMETER KeystoreName
  keystore 文件名（相对 OutDir）。默认：myflowhub-release.jks

.PARAMETER Base64Name
  base64 文件名（相对 OutDir）。默认：myflowhub-release.jks.b64

.PARAMETER ValidityDays
  证书有效期（天）。默认：36500（约 100 年）

.PARAMETER Force
  若目标文件已存在，允许覆盖。

.PARAMETER CopyBase64ToClipboard
  生成后把 base64 内容复制到剪贴板（可选）。

.PARAMETER KeytoolPath
  手动指定 keytool.exe 的路径（可选）。

.PARAMETER DName
  证书 DN（可选）。默认提供一个占位 DN 以减少交互问题；如传空字符串将不传 -dname。

.EXAMPLE
  .\scripts\gen-android-release-secrets.ps1

.EXAMPLE
  .\scripts\gen-android-release-secrets.ps1 -CopyBase64ToClipboard

.EXAMPLE
  .\scripts\gen-android-release-secrets.ps1 -KeytoolPath "C:\Program Files\Android\Android Studio\jbr\bin\keytool.exe"
#>

[CmdletBinding()]
param(
  [Parameter()]
  [string]$OutDir = "",

  [Parameter()]
  [string]$Alias = "myflowhub",

  [Parameter()]
  [string]$KeystoreName = "myflowhub-release.jks",

  [Parameter()]
  [string]$Base64Name = "myflowhub-release.jks.b64",

  [Parameter()]
  [int]$ValidityDays = 36500,

  [Parameter()]
  [switch]$Force,

  [Parameter()]
  [switch]$CopyBase64ToClipboard,

  [Parameter()]
  [string]$KeytoolPath = "",

  [Parameter()]
  [string]$DName = "CN=MyFlowHub, OU=Self, O=Self, L=Self, S=Self, C=US"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Resolve-WorkspaceRoot {
  $root = Resolve-Path (Join-Path $PSScriptRoot "..")
  return $root.Path
}

function Ensure-Dir([string]$Path) {
  if (-not (Test-Path -LiteralPath $Path)) {
    New-Item -ItemType Directory -Force -Path $Path | Out-Null
  }
}

function Find-Keytool([string]$ManualPath) {
  if ($ManualPath.Trim() -ne "") {
    $p = $ManualPath.Trim()
    if (-not (Test-Path -LiteralPath $p)) {
      throw "指定的 KeytoolPath 不存在：$p"
    }
    return (Resolve-Path -LiteralPath $p).Path
  }

  $cmd = Get-Command "keytool" -ErrorAction SilentlyContinue
  if ($cmd -and $cmd.Source) {
    return $cmd.Source
  }

  if ($env:JAVA_HOME) {
    $p = Join-Path $env:JAVA_HOME "bin\\keytool.exe"
    if (Test-Path -LiteralPath $p) {
      return (Resolve-Path -LiteralPath $p).Path
    }
  }

  $candidates = @()
  if ($env:ProgramFiles) {
    $studio = Join-Path $env:ProgramFiles "Android\\Android Studio"
    $candidates += (Join-Path $studio "jbr\\bin\\keytool.exe")
    $candidates += (Join-Path $studio "jre\\bin\\keytool.exe")
  }

  foreach ($root in @("$env:ProgramFiles\\Java", "$env:ProgramFiles\\Eclipse Adoptium", "$env:ProgramFiles\\Microsoft")) {
    if ($root -and (Test-Path -LiteralPath $root)) {
      $dirs = Get-ChildItem -LiteralPath $root -Directory -ErrorAction SilentlyContinue | Sort-Object Name -Descending
      foreach ($d in $dirs) {
        $p = Join-Path $d.FullName "bin\\keytool.exe"
        if (Test-Path -LiteralPath $p) {
          $candidates += $p
          break
        }
      }
    }
  }

  foreach ($p in $candidates | Select-Object -Unique) {
    if (Test-Path -LiteralPath $p) {
      return (Resolve-Path -LiteralPath $p).Path
    }
  }

  return $null
}

function Write-Section([string]$Title) {
  Write-Host ""
  Write-Host $Title -ForegroundColor Cyan
  Write-Host ("-" * $Title.Length) -ForegroundColor Cyan
}

$root = Resolve-WorkspaceRoot
if ($OutDir.Trim() -eq "") {
  $OutDir = Join-Path $root ".tmp\\android-signing"
}
Ensure-Dir $OutDir

$keystorePath = Join-Path $OutDir $KeystoreName
$base64Path = Join-Path $OutDir $Base64Name

Write-Section "1) 定位 keytool"
$keytool = Find-Keytool $KeytoolPath
if (-not $keytool) {
  Write-Host "未找到 keytool。原因通常是未安装/未配置 JDK。" -ForegroundColor Red
  Write-Host ""
  Write-Host "可选解决方案：" -ForegroundColor Yellow
  Write-Host "A) 安装 JDK 17+（推荐 Temurin 17）并确保 keytool 在 PATH 中"
  Write-Host "B) 若已安装 Android Studio：尝试使用其自带 keytool："
  if ($env:ProgramFiles) {
    Write-Host ("   " + (Join-Path $env:ProgramFiles "Android\\Android Studio\\jbr\\bin\\keytool.exe"))
  }
  Write-Host ""
  Write-Host "安装完成后可在 PowerShell 运行：where.exe keytool" -ForegroundColor Yellow
  exit 1
}
Write-Host "keytool: $keytool" -ForegroundColor Green

Write-Section "2) 生成 keystore（交互式输入密码）"
if ((Test-Path -LiteralPath $keystorePath) -and -not $Force) {
  throw "目标文件已存在：$keystorePath。若要覆盖，请加 -Force。"
}
if ((Test-Path -LiteralPath $keystorePath) -and $Force) {
  Remove-Item -LiteralPath $keystorePath -Force
}
if ((Test-Path -LiteralPath $base64Path) -and $Force) {
  Remove-Item -LiteralPath $base64Path -Force
}

$args = @(
  "-genkeypair",
  "-v",
  "-keystore", $keystorePath,
  "-storetype", "JKS",
  "-alias", $Alias,
  "-keyalg", "RSA",
  "-keysize", "2048",
  "-validity", [string]$ValidityDays
)
if ($DName.Trim() -ne "") {
  $args += @("-dname", $DName.Trim())
}

Write-Host "将调用 keytool 生成 keystore：$keystorePath" -ForegroundColor Yellow
Write-Host "提示：keytool 会要求你输入 keystore 密码与 key 密码（请妥善保存，丢失将无法覆盖升级）。" -ForegroundColor Yellow
& $keytool @args

if (-not (Test-Path -LiteralPath $keystorePath)) {
  throw "生成失败：未找到 keystore 文件：$keystorePath"
}

Write-Section "3) 生成 base64（用于 GitHub Secrets）"
$bytes = [System.IO.File]::ReadAllBytes($keystorePath)
$b64 = [Convert]::ToBase64String($bytes)
$utf8NoBom = New-Object System.Text.UTF8Encoding($false)
[System.IO.File]::WriteAllText($base64Path, $b64, $utf8NoBom)

Write-Host "keystore: $keystorePath" -ForegroundColor Green
Write-Host "base64   : $base64Path" -ForegroundColor Green

if ($CopyBase64ToClipboard) {
  if (Get-Command "Set-Clipboard" -ErrorAction SilentlyContinue) {
    Set-Clipboard -Value $b64
    Write-Host "已复制 base64 到剪贴板，可直接粘贴到 GitHub Secret：ANDROID_KEYSTORE_BASE64" -ForegroundColor Green
  } else {
    Write-Warning "当前 PowerShell 未提供 Set-Clipboard，已跳过复制。请手动打开 base64 文件复制。"
  }
}

Write-Section "4) 在 GitHub 配置 Secrets（Repository secrets）"
Write-Host "在 GitHub 仓库页面依次进入：" -ForegroundColor Yellow
Write-Host "  Settings -> Secrets and variables -> Actions -> Secrets -> New repository secret"
Write-Host ""
Write-Host "请新增以下 4 个（名称必须完全一致）："
Write-Host "1) ANDROID_KEYSTORE_BASE64     ：复制 $base64Path 的内容粘贴"
Write-Host "2) ANDROID_KEYSTORE_PASSWORD   ：你刚才输入的 keystore 密码"
Write-Host "3) ANDROID_KEY_ALIAS           ：$Alias"
Write-Host "4) ANDROID_KEY_PASSWORD        ：你刚才输入的 key 密码"

Write-Section "5) 验证（触发一次 Release）"
Write-Host "配置完 Secrets 后，在本机执行（示例 v0.1.0）：" -ForegroundColor Yellow
Write-Host "  git tag v0.1.0"
Write-Host "  git push origin v0.1.0"
Write-Host ""
Write-Host "然后到 GitHub 查看：Actions -> release workflow 与 Releases 资产（app-release.apk / myflowhub.aar / build-info.txt）。"

