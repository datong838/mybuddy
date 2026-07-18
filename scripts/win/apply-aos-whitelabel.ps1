# Apply AOS white-label to running Dify (console logo + hide Help ?)
# Requires: Docker running, containers docker-api-1 / docker-web-1
# See: docs/palantier/10_v01/10g-交付面去Dify品牌说明.md

$ErrorActionPreference = "Stop"
$Root = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent
$WhitelabelDir = Join-Path $PSScriptRoot "whitelabel"
$DockerDir = Join-Path $Root "dify\docker"
$FeatureSrc = Join-Path $Root "dify\api\services\feature_service.py"
$VolumeName = "docker_dify_wsl_app"

function Ensure-EnvLine([string]$Path, [string]$Key, [string]$Value) {
  $utf8 = New-Object System.Text.UTF8Encoding $false
  $raw = [System.IO.File]::ReadAllText($Path)
  $lines = $raw -split "`r?`n", -1
  $found = $false
  $out = foreach ($line in $lines) {
    if ($line -match "^\s*$([regex]::Escape($Key))\s*=") {
      $found = $true
      "$Key=$Value"
    } else {
      $line
    }
  }
  if (-not $found) {
    if ($out.Count -gt 0 -and $out[-1] -ne "") { $out += "" }
    $out += "$Key=$Value"
  }
  [System.IO.File]::WriteAllText($Path, ($out -join "`n") + "`n", $utf8)
}

function Push-Utf8FileViaBase64([string]$LocalPath, [string]$Container, [string]$RemotePath) {
  # Prefer stdin redirect (fast); fallback to chunked base64
  cmd /c "docker exec -i $Container sh -c `"cat > $RemotePath`" < `"$LocalPath`""
  if ($LASTEXITCODE -ne 0) {
    $b64 = [Convert]::ToBase64String([System.IO.File]::ReadAllBytes($LocalPath))
    $chunkSize = 6000
    $tmp = "/tmp/_aos_b64.txt"
    docker exec $Container sh -c "rm -f $tmp"
    for ($i = 0; $i -lt $b64.Length; $i += $chunkSize) {
      $len = [Math]::Min($chunkSize, $b64.Length - $i)
      $part = $b64.Substring($i, $len)
      docker exec $Container sh -c "printf %s '$part' >> $tmp"
    }
    docker exec $Container sh -c "base64 -d $tmp > '$RemotePath' && rm -f $tmp"
  }
}

$envFile = Join-Path $DockerDir ".env"
Ensure-EnvLine $envFile "CAN_REPLACE_LOGO" "true"
Ensure-EnvLine $envFile "AOS_WHITE_LABEL" "true"
# 中文品牌名以 api feature_service 源码默认为准（知识顾问），勿写入 .env 避免 Windows 乱码
Ensure-EnvLine $envFile "AOS_BRAND_TITLE" ""

Write-Host "==> Inject API feature_service white-label patch (chunked base64)"
Push-Utf8FileViaBase64 $FeatureSrc "docker-api-1" "/app/api/services/feature_service.py"
if (docker ps -q -f name=docker-worker-1) {
  Push-Utf8FileViaBase64 $FeatureSrc "docker-worker-1" "/app/api/services/feature_service.py"
}

Write-Host "==> Recreate api/worker to load AOS_WHITE_LABEL env"
Push-Location $DockerDir
try {
  docker compose -f docker-compose.yaml -f docker-compose.wsl-volumes.yaml up -d api worker | Out-Host
} finally {
  Pop-Location
}
Start-Sleep -Seconds 4

Write-Host "==> Re-inject API patch after recreate"
Push-Utf8FileViaBase64 $FeatureSrc "docker-api-1" "/app/api/services/feature_service.py"
if (docker ps -q -f name=docker-worker-1) {
  Push-Utf8FileViaBase64 $FeatureSrc "docker-worker-1" "/app/api/services/feature_service.py"
}

Write-Host "==> Replace console logo (知识引擎 PNG embedded in SVG)"
# Prefer prebuilt assets under scripts/win/whitelabel/ (logo.png → logo.svg)
$logoPath = Join-Path $WhitelabelDir "logo.svg"
$logoWPath = Join-Path $WhitelabelDir "logo-monochrome-white.svg"
if (-not (Test-Path $logoPath) -or -not (Test-Path $logoWPath)) {
  throw "Missing logo.svg / logo-monochrome-white.svg under $WhitelabelDir"
}

$logoTargets = @(
  "/app/targets/next/web/public/logo",
  "/app/targets/vinext/public/logo",
  "/app/targets/vinext/dist/client/logo"
)
foreach ($dir in $logoTargets) {
  docker exec docker-web-1 sh -c "test -d $dir"
  if ($LASTEXITCODE -eq 0) {
    Push-Utf8FileViaBase64 $logoPath "docker-web-1" "$dir/logo.svg"
    Push-Utf8FileViaBase64 $logoWPath "docker-web-1" "$dir/logo-monochrome-white.svg"
  }
}

docker restart docker-api-1 docker-web-1 | Out-Null
Write-Host "Done. Hard-refresh http://127.0.0.1/ — expect 知识顾问 title and no Help (?)."
