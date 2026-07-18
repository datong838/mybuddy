# Stop buddy adapter on 127.0.0.1:8090
$ErrorActionPreference = "Continue"
$conn = Get-NetTCPConnection -LocalPort 8090 -State Listen -ErrorAction SilentlyContinue | Select-Object -First 1
if ($conn) {
  Stop-Process -Id $conn.OwningProcess -Force -ErrorAction SilentlyContinue
  Write-Host "OK: adapter stopped (pid=$($conn.OwningProcess))"
} else {
  Write-Host "adapter not running on :8090"
}
