# Upstream pull script for mybuddy-v01
# Usage: pwsh -File mybuddy-v01/scripts/pull-upstream.ps1

$ErrorActionPreference = "Stop"
$Base = Split-Path $PSScriptRoot -Parent
Set-Location $Base
Write-Host "Base: $Base"

function Ensure-Clone($Name, $Url) {
  $Path = Join-Path $Base $Name
  if (Test-Path (Join-Path $Path ".git")) {
    Write-Host "[skip] $Name already exists: $Path"
    Push-Location $Path
    git rev-parse --short HEAD
    Pop-Location
    return
  }
  Write-Host "[clone] $Url -> $Path"
  git clone --depth 1 $Url $Path
  if ($LASTEXITCODE -ne 0) { throw "clone failed: $Name" }
}

# Prefer SSH (more stable than HTTPS behind flaky networks)
Ensure-Clone "dify" "git@github.com:langgenius/dify.git"
Ensure-Clone "openocta" "git@github.com:openocta/openocta.git"

$lines = @("# mybuddy-v01 upstream lock", "", "Generated: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')", "")
foreach ($name in @("dify", "openocta")) {
  Push-Location (Join-Path $Base $name)
  $sha = (git rev-parse HEAD).Trim()
  $short = (git rev-parse --short HEAD).Trim()
  $branch = (git rev-parse --abbrev-ref HEAD).Trim()
  $remote = (git remote get-url origin).Trim()
  Pop-Location
  $lines += "## $name"
  $lines += "- remote: $remote"
  $lines += "- branch: $branch"
  $lines += "- commit: ``$short`` ($sha)"
  $lines += ""
}
$lines | Set-Content -Path (Join-Path $Base "VERSIONS.md") -Encoding utf8
Write-Host "Wrote VERSIONS.md"
Write-Host "Done."
