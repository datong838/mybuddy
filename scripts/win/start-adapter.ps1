# Start buddy adapter on 127.0.0.1:8090
$ErrorActionPreference = "Stop"
$Root = Resolve-Path (Join-Path $PSScriptRoot "..\..")
. "$PSScriptRoot\Load-BuddyDotEnv.ps1"
Import-BuddyDotEnv -Root $Root

$Adapter = Join-Path $Root "adapter"
if (-not $env:DIFY_BASE_URL) { $env:DIFY_BASE_URL = "http://127.0.0.1" }
Write-Host "DIFY_BASE_URL=$($env:DIFY_BASE_URL)"
Write-Host "BUDDY_DEMO_MODE=$($env:BUDDY_DEMO_MODE)"
if ($env:DIFY_API_KEY) {
  $preview = if ($env:DIFY_API_KEY.Length -gt 10) { $env:DIFY_API_KEY.Substring(0, 10) + "..." } else { "(set)" }
  Write-Host "DIFY_API_KEY=$preview"
} else {
  Write-Host "DIFY_API_KEY=(empty)"
}
if (-not $env:BUDDY_DEMO_MODE -and -not $env:DIFY_API_KEY) {
  Write-Host "HINT: enabling BUDDY_DEMO_MODE=1 for local demo" -ForegroundColor Yellow
  $env:BUDDY_DEMO_MODE = "1"
}
if ($env:DIFY_API_KEY -match "替换|你的Key|changeme|REPLACE") {
  Write-Host "WARN: DIFY_API_KEY still looks like a placeholder — edit mybuddy-v01/.env" -ForegroundColor Yellow
}
Set-Location $Adapter
python -m uvicorn app.main:app --host 127.0.0.1 --port 8090
