<#
.SYNOPSIS
  Windows 开发环境一条复检（可选 -Repair 自动补齐）。

.DESCRIPTION
  对照 mybuddy-v01/docs/Windows环境安装检查单.md。
  退出码 0 = PASS；非 0 = FAIL。

.PARAMETER Repair
  缺项时尽量自动修复：PATH、rustup default、起 dockerd、起 Dify 等。

.EXAMPLE
  powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\selftest\Check-WindowsDevEnv.ps1
  powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\selftest\Check-WindowsDevEnv.ps1 -Repair
#>
[CmdletBinding()]
param(
  [switch]$Repair
)

$ErrorActionPreference = "Continue"
$fail = 0
$warn = 0

function Ok([string]$m) { Write-Host "[OK]   $m" -ForegroundColor Green }
function Bad([string]$m) { Write-Host "[MISS] $m" -ForegroundColor Red; $script:fail++ }
function Warn([string]$m) { Write-Host "[WARN] $m" -ForegroundColor Yellow; $script:warn++ }
function Info([string]$m) { Write-Host "[..]   $m" -ForegroundColor Cyan }

$Root = Resolve-Path (Join-Path $PSScriptRoot "..\..")
$WinScripts = Join-Path $Root "scripts\win"
$CargoBin = Join-Path $env:USERPROFILE ".cargo\bin"
$NodeDir = "C:\Program Files\nodejs"
$CursorHelper = Join-Path $env:LOCALAPPDATA "Programs\cursor\resources\app\resources\helpers"

function Refresh-Path {
  $machine = [Environment]::GetEnvironmentVariable("Path", "Machine")
  $user = [Environment]::GetEnvironmentVariable("Path", "User")
  $parts = @()
  foreach ($p in @($NodeDir, $CargoBin, $WinScripts)) {
    if ($p -and (Test-Path $p)) { $parts += $p }
  }
  foreach ($chunk in @($user, $machine)) {
    if (-not $chunk) { continue }
    foreach ($p in ($chunk -split ";")) {
      if ($p -and ($parts -notcontains $p) -and ($p -ne $CursorHelper)) { $parts += $p }
    }
  }
  if (Test-Path $CursorHelper) { $parts += $CursorHelper }
  $env:Path = ($parts -join ";")
}

function Ensure-UserPathPrefix([string]$dir) {
  if (-not (Test-Path $dir)) { return }
  $user = [Environment]::GetEnvironmentVariable("Path", "User")
  if (-not $user) { $user = "" }
  $parts = @($user -split ";" | Where-Object { $_ -and ($_ -ne $dir) })
  $new = ($dir + ";" + ($parts -join ";")).TrimEnd(";")
  [Environment]::SetEnvironmentVariable("Path", $new, "User")
  Refresh-Path
}

function Test-HttpOk([string]$url) {
  try {
    $r = Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec 12 -MaximumRedirection 5
    if ($r.StatusCode -ge 200 -and $r.StatusCode -lt 400) { return [int]$r.StatusCode }
  } catch {
    $resp = $_.Exception.Response
    if ($resp) {
      $c = [int]$resp.StatusCode
      if ($c -ge 200 -and $c -lt 500) { return $c }
    }
  }
  return $null
}

function Ensure-Dockerd {
  $start = Join-Path $WinScripts "start-dockerd.ps1"
  if (Test-Path $start) {
    Info "starting dockerd via start-dockerd.ps1"
    & $start 2>&1 | Out-Host
  } else {
    wsl -d Ubuntu -u root -- bash -c "docker info >/dev/null 2>&1 || (update-alternatives --set iptables /usr/sbin/iptables-legacy 2>/dev/null; nohup dockerd >/var/log/dockerd.log 2>&1 & sleep 5); docker version" 2>&1 | Out-Host
  }
}

function Ensure-Dify {
  $start = Join-Path $WinScripts "start-dify.ps1"
  if (Test-Path $start) {
    Info "starting Dify via start-dify.ps1"
    & $start 2>&1 | Out-Host
  } else {
    Bad "start-dify.ps1 missing"
  }
}

function Ensure-Rust {
  $rustup = Join-Path $CargoBin "rustup.exe"
  $init = Join-Path $Root "scripts\_installers\rustup-init.exe"
  $env:RUSTUP_DIST_SERVER = "https://mirrors.ustc.edu.cn/rust-static"
  $env:RUSTUP_UPDATE_ROOT = "https://mirrors.ustc.edu.cn/rust-static/rustup"
  if (-not (Test-Path $rustup)) {
    if (-not (Test-Path $init)) {
      Info "downloading rustup-init.exe"
      $dl = Join-Path $Root "scripts\_installers"
      New-Item -ItemType Directory -Force -Path $dl | Out-Null
      $init = Join-Path $dl "rustup-init.exe"
      curl.exe -L --retry 3 -o $init "https://static.rust-lang.org/rustup/dist/x86_64-pc-windows-msvc/rustup-init.exe"
    }
    if (Test-Path $init) {
      Info "running rustup-init -y"
      $env:RUSTUP_INIT_SKIP_PATH_CHECK = "yes"
      & $init -y --default-toolchain stable 2>&1 | Out-Host
    }
  }
  Ensure-UserPathPrefix $CargoBin
  Refresh-Path
  if (Get-Command rustup -EA SilentlyContinue) {
    & rustup default stable 2>&1 | Out-Host
  }
}

function Ensure-NodePath {
  if (Test-Path $NodeDir) {
    Ensure-UserPathPrefix $NodeDir
    Refresh-Path
  }
}

function Ensure-DockerWrapper {
  $cmd = Join-Path $WinScripts "docker.cmd"
  if (-not (Test-Path $cmd)) {
    New-Item -ItemType Directory -Force -Path $WinScripts | Out-Null
    "@echo off`r`nwsl -d Ubuntu -u root -- docker %*`r`n" | Set-Content -Path $cmd -Encoding ASCII
  }
  Ensure-UserPathPrefix $WinScripts
  Refresh-Path
}

function Ensure-P0Skeleton {
  foreach ($rel in @(
    "docs", "docs\dify-export", "openocta-overlay\skills\SearchKnowledgeDoc",
    "adapter", "desktop", "scripts\win", "scripts\mac", "scripts\linux", "scripts\selftest"
  )) {
    $p = Join-Path $Root $rel
    if (-not (Test-Path $p)) {
      New-Item -ItemType Directory -Force -Path $p | Out-Null
      Info "created $rel"
    }
  }
  $ver = Join-Path $Root "VERSIONS.md"
  if (-not (Test-Path $ver)) {
    $difySha = ""; $octaSha = ""
    if (Test-Path (Join-Path $Root "dify\.git")) {
      Push-Location (Join-Path $Root "dify"); $difySha = (git rev-parse HEAD); Pop-Location
    }
    if (Test-Path (Join-Path $Root "openocta\.git")) {
      Push-Location (Join-Path $Root "openocta"); $octaSha = (git rev-parse HEAD); Pop-Location
    }
    @"
# Upstream version lock
| Component | Full SHA |
| --- | --- |
| Dify | ``$difySha`` |
| OpenOcta | ``$octaSha`` |
"@ | Set-Content -Path $ver -Encoding UTF8
    Info "wrote VERSIONS.md"
  }
}

# ---------- begin ----------
Write-Host "===== Check-WindowsDevEnv  root=$Root  Repair=$Repair =====" -ForegroundColor White
Refresh-Path

# H: WSL (wsl -l is UTF-16; probe distro directly)
Write-Host "`n=== WSL ==="
$ubuntuProbe = & wsl.exe -d Ubuntu -e echo OK 2>&1 | Out-String
if ($ubuntuProbe -match "OK") {
  Ok "WSL Ubuntu present (probe ok)"
} else {
  # Fallback: decode UTF-16 list
  $listRaw = & wsl.exe -l -v 2>&1
  $listText = if ($listRaw -is [System.Array]) { ($listRaw | ForEach-Object { "$_" }) -join "`n" } else { "$listRaw" }
  if ($listText -match "Ubuntu" -or $listText -match "U\0b\0u\0n\0t\0u") {
    Ok "WSL Ubuntu present (list)"
  } else {
    Bad "WSL Ubuntu missing (wsl --install -d Ubuntu); probe=$ubuntuProbe"
    if ($Repair) { Warn "Cannot auto-install WSL distro without reboot/UAC; install Ubuntu manually" }
  }
}

# T2 Node
Write-Host "`n=== Node ==="
if ($Repair) { Ensure-NodePath }
$nodes = @(where.exe node 2>$null)
if ($nodes -and $nodes[0] -like "*Program Files\nodejs*") {
  Ok "node first=$($nodes[0])  ($(& node -v 2>$null))"
} elseif (Test-Path "$NodeDir\node.exe") {
  Bad "system node exists but not first in PATH: $($nodes -join ' | ')"
  if ($Repair) {
    Ensure-NodePath
    $nodes = @(where.exe node 2>$null)
    if ($nodes -and $nodes[0] -like "*Program Files\nodejs*") { Ok "node PATH fixed: $($nodes[0])" }
  }
} else {
  Bad "Node.js not installed at $NodeDir (winget install OpenJS.NodeJS.LTS)"
}

# T3 Rust
Write-Host "`n=== Rust ==="
if ($Repair -and -not (Get-Command rustc -EA SilentlyContinue)) { Ensure-Rust }
Refresh-Path
if (Get-Command rustc -EA SilentlyContinue) {
  Ok "$(rustc -V)"
  if (Get-Command cargo -EA SilentlyContinue) { Ok "$(cargo -V)" } else { Bad "cargo missing" }
} else {
  Bad "rustc/cargo missing"
  if ($Repair) {
    Ensure-Rust
    Refresh-Path
    if (Get-Command rustc -EA SilentlyContinue) { Ok "repaired: $(rustc -V)"; Ok "$(cargo -V)" }
  }
}

# T5 WebView2 / Edge
Write-Host "`n=== WebView2 / Edge ==="
$wv = @(
  "${env:ProgramFiles(x86)}\Microsoft\EdgeWebView\Application",
  "$env:ProgramFiles\Microsoft\EdgeWebView\Application",
  "${env:ProgramFiles(x86)}\Microsoft\Edge\Application"
)
if ($wv | Where-Object { Test-Path $_ }) { Ok "WebView2/Edge runtime found" } else { Warn "WebView2 not found (Tauri Win may need Evergreen Runtime)" }

# T4 MSVC hint
Write-Host "`n=== MSVC ==="
$vswhere = "${env:ProgramFiles(x86)}\Microsoft Visual Studio\Installer\vswhere.exe"
if (Test-Path $vswhere) {
  $vs = & $vswhere -latest -products * -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath 2>$null
  if ($vs) { Ok "MSVC Build Tools: $vs" } else { Warn "vswhere ok but VC Tools component not detected" }
} else { Warn "vswhere missing (Tauri link may need VS Build Tools)" }

# T6 Docker
Write-Host "`n=== Docker ==="
if ($Repair) {
  Ensure-DockerWrapper
  Ensure-Dockerd
}
$dockerCmd = Join-Path $WinScripts "docker.cmd"
$desktop = Test-Path "$env:ProgramFiles\Docker\Docker\Docker Desktop.exe"
if ($desktop) { Ok "Docker Desktop installed" } else { Warn "Docker Desktop not installed (WSL Engine OK for Day1)" }

if (Test-Path $dockerCmd) {
  $dv = & $dockerCmd version 2>&1 | Out-String
  $dv | Write-Host
  if ($LASTEXITCODE -eq 0 -and $dv -match "Server:") { Ok "docker version (Server reachable)" }
  else {
    Bad "docker version failed or no Server"
    if ($Repair) {
      Ensure-Dockerd
      $dv2 = & $dockerCmd version 2>&1 | Out-String
      $dv2 | Write-Host
      if ($LASTEXITCODE -eq 0 -and $dv2 -match "Server:") { Ok "docker Server repaired" }
    }
  }
  $cv = & $dockerCmd compose version 2>&1 | Out-String
  $cv | Write-Host
  if ($LASTEXITCODE -eq 0 -and $cv -match "Compose") { Ok "docker compose version" } else { Bad "docker compose version failed" }
} else {
  Bad "scripts\win\docker.cmd missing"
  if ($Repair) { Ensure-DockerWrapper }
}

# P0
Write-Host "`n=== Project (P0) ==="
if ($Repair) { Ensure-P0Skeleton }
foreach ($d in @("dify", "openocta", "docs", "openocta-overlay", "desktop", "adapter", "VERSIONS.md")) {
  if (Test-Path (Join-Path $Root $d)) { Ok $d } else { Bad "missing $d" }
}

# Day1 Dify HTTP
Write-Host "`n=== Dify HTTP (Day1) ==="
$code = Test-HttpOk "http://127.0.0.1/install"
if (-not $code) { $code = Test-HttpOk "http://127.0.0.1/" }
if ($code) {
  Ok "Dify console HTTP $code (http://127.0.0.1)"
} else {
  Bad "Dify console not reachable"
  if ($Repair) {
    Ensure-Dockerd
    Ensure-Dify
    Start-Sleep -Seconds 8
    $code = Test-HttpOk "http://127.0.0.1/install"
    if (-not $code) { $code = Test-HttpOk "http://127.0.0.1/" }
    if ($code) { Ok "Dify console repaired HTTP $code" } else { Bad "Dify still unreachable after Repair" }
  }
}

# Summary
Write-Host ""
if ($fail -gt 0) {
  Write-Host "RESULT: FAIL ($fail miss, $warn warn)" -ForegroundColor Red
  Write-Host "See: $Root\docs\Windows环境安装检查单.md"
  exit 1
} else {
  Write-Host "RESULT: PASS ($warn warn)" -ForegroundColor Green
  exit 0
}
