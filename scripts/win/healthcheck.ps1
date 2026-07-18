# L0 healthcheck (Windows)
$ErrorActionPreference = "Continue"
$fail = 0
function Ok($m){ Write-Host "[OK] $m" -ForegroundColor Green }
function Bad($m){ Write-Host "[MISS] $m" -ForegroundColor Red; $script:fail++ }

$docker = Join-Path $PSScriptRoot "docker.cmd"
if (Test-Path $docker) {
  & $docker info 2>$null | Out-Null
  if ($LASTEXITCODE -eq 0) { Ok "docker engine" } else { Bad "docker engine" }
} else { Bad "docker.cmd" }

try {
  $r = Invoke-WebRequest "http://127.0.0.1/install" -UseBasicParsing -TimeoutSec 5
  if ($r.StatusCode -lt 400) { Ok "dify http $($r.StatusCode)" } else { Bad "dify http $($r.StatusCode)" }
} catch { Bad "dify http" }

try {
  $r = Invoke-WebRequest "http://127.0.0.1:8090/healthz" -UseBasicParsing -TimeoutSec 3
  Ok "adapter healthz"
} catch { Write-Host "[WARN] adapter not up (start: uvicorn on :8090)" -ForegroundColor Yellow }

if ($fail -gt 0) { exit 1 } else { Write-Host "RESULT: PASS"; exit 0 }
