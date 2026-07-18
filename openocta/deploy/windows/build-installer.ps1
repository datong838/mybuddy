param(
  [Parameter(Mandatory=$true)][string]$OpenOctaExe,
  [Parameter(Mandatory=$true)][string]$LauncherExe,
  [string]$NsisScript = "$(Split-Path -Parent $MyInvocation.MyCommand.Path)\\openocta-installer.nsi",
  [string]$OutDir = "$(Get-Location)"
)

$ErrorActionPreference = "Stop"

if (!(Test-Path $OpenOctaExe)) { throw "openocta.exe not found: $OpenOctaExe" }
if (!(Test-Path $LauncherExe)) { throw "openocta-launcher.exe not found: $LauncherExe" }
if (!(Test-Path $NsisScript)) { throw "NSIS script not found: $NsisScript" }

$work = Join-Path $env:TEMP ("openocta-nsis-" + [Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $work | Out-Null

Copy-Item $OpenOctaExe (Join-Path $work "openocta.exe")
Copy-Item $LauncherExe (Join-Path $work "openocta-launcher.exe")
Copy-Item $NsisScript (Join-Path $work "openocta-installer.nsi")
$scriptDir = Split-Path -Parent $NsisScript
$licenseFile = Join-Path $scriptDir "license.txt"
if (Test-Path $licenseFile) { Copy-Item $licenseFile (Join-Path $work "license.txt") }

Push-Location $work
try {
  & makensis.exe ".\\openocta-installer.nsi"
  $out = Join-Path $OutDir "OpenOcta-Setup.exe"
  Copy-Item ".\\OpenOcta-Setup.exe" $out -Force
  Write-Host "Installer generated: $out"
} finally {
  Pop-Location
  Remove-Item -Recurse -Force $work
}

