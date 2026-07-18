# One-shot start: dockerd → Dify → adapter
# Desktop: open dist exe or run tauri separately (see docs)
$ErrorActionPreference = "Stop"
$Root = Resolve-Path (Join-Path $PSScriptRoot "..\..")
Set-Location $Root
. "$PSScriptRoot\Load-BuddyDotEnv.ps1"
Import-BuddyDotEnv -Root $Root

Write-Host "=== start-dockerd ===" -ForegroundColor Cyan
& "$PSScriptRoot\start-dockerd.ps1"

Write-Host "=== start-dify ===" -ForegroundColor Cyan
& "$PSScriptRoot\start-dify.ps1"

# Free 8090 then start adapter in background
$conn = Get-NetTCPConnection -LocalPort 8090 -State Listen -ErrorAction SilentlyContinue | Select-Object -First 1
if ($conn) {
  Write-Host "Port 8090 busy (pid=$($conn.OwningProcess)), stopping..." -ForegroundColor Yellow
  Stop-Process -Id $conn.OwningProcess -Force -ErrorAction SilentlyContinue
  Start-Sleep -Seconds 1
}

if (-not $env:BUDDY_DEMO_MODE -and -not $env:DIFY_API_KEY) {
  $env:BUDDY_DEMO_MODE = "1"
  Write-Host "BUDDY_DEMO_MODE=1 (no DIFY_API_KEY)" -ForegroundColor Yellow
}
if (-not $env:DIFY_BASE_URL) { $env:DIFY_BASE_URL = "http://127.0.0.1" }
if ($env:DIFY_API_KEY -match "替换|你的Key|changeme|REPLACE") {
  Write-Host "WARN: edit mybuddy-v01/.env and set real DIFY_API_KEY" -ForegroundColor Yellow
}

$Adapter = Join-Path $Root "adapter"
Write-Host "=== start-adapter (background :8090) ===" -ForegroundColor Cyan
Write-Host "BUDDY_DEMO_MODE=$($env:BUDDY_DEMO_MODE)  DIFY_BASE_URL=$($env:DIFY_BASE_URL)"
$arg = @(
  "-NoProfile", "-Command",
  "Set-Location '$Adapter'; `$env:DIFY_BASE_URL='$($env:DIFY_BASE_URL)'; `$env:BUDDY_DEMO_MODE='$($env:BUDDY_DEMO_MODE)'; if ('$($env:DIFY_API_KEY)') { `$env:DIFY_API_KEY='$($env:DIFY_API_KEY)' }; python -m uvicorn app.main:app --host 127.0.0.1 --port 8090"
)
Start-Process -FilePath "powershell" -ArgumentList $arg -WindowStyle Minimized
Start-Sleep -Seconds 3

& "$PSScriptRoot\healthcheck.ps1"
Write-Host ""
Write-Host "Dify:    http://127.0.0.1/install" -ForegroundColor Green
Write-Host "Adapter: http://127.0.0.1:8090/healthz" -ForegroundColor Green
Write-Host "Desktop: .\dist\windows\AOS-Local-Workstation.exe   OR   cd desktop; npm run tauri dev" -ForegroundColor Green
Write-Host "Stop:    powershell -File .\scripts\win\stop-all.ps1" -ForegroundColor Cyan
Write-Host "Config:  mybuddy-v01\.env  (DIFY_API_KEY / BUDDY_DEMO_MODE)" -ForegroundColor Cyan
