# Stop adapter (port 8090) + Dify compose. Does NOT stop WSL dockerd by default.
$ErrorActionPreference = "Continue"
$Root = Resolve-Path (Join-Path $PSScriptRoot "..\..")

Write-Host "=== stop-adapter (:8090) ===" -ForegroundColor Cyan
$conn = Get-NetTCPConnection -LocalPort 8090 -State Listen -ErrorAction SilentlyContinue | Select-Object -First 1
if ($conn) {
  Stop-Process -Id $conn.OwningProcess -Force -ErrorAction SilentlyContinue
  Write-Host "Stopped pid=$($conn.OwningProcess)"
} else {
  Write-Host "adapter not listening on 8090"
}

Write-Host "=== stop-dify (compose down) ===" -ForegroundColor Cyan
$composeDir = "/mnt/c/work/projects/wchat/mybuddy-v01/dify/docker"
wsl -d Ubuntu -u root -- bash -c "cd $composeDir; docker compose -f docker-compose.yaml -f docker-compose.wsl-volumes.yaml down" 2>&1 | Out-Host

# Optional: close desktop exe
Get-Process -Name "AOS-Local-Workstation","desktop" -ErrorAction SilentlyContinue | ForEach-Object {
  Write-Host "Stopping desktop pid=$($_.Id) ($($_.ProcessName))"
  Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
}

Write-Host "OK: stack stopped (dockerd left running; use stop-dockerd.ps1 if needed)" -ForegroundColor Green
