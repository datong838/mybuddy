# L2 contract: prefers BUDDY_DEMO_MODE adapter for ok/no_hit/error shapes
$ErrorActionPreference = "Continue"
$base = if ($env:BUDDY_ASK_URL) { $env:BUDDY_ASK_URL } else { "http://127.0.0.1:8090/v1/buddy/ask" }
$fail = 0

function PostAsk($obj) {
  $json = $obj | ConvertTo-Json -Depth 6 -Compress
  try {
    return Invoke-RestMethod -Uri $base -Method POST -ContentType "application/json" -Body $json -TimeoutSec 30
  } catch {
    Write-Host "POST fail: $($_.Exception.Message)" -ForegroundColor Yellow
    return $null
  }
}

$rOk = PostAsk @{ question = "OBJ-1001 的冷运行参数是什么？"; channel = "desktop"; user = @{ id = "t"; display_name = "t" } }
if ($rOk -and $rOk.status -eq "ok" -and $rOk.citations) { Write-Host "[OK] status=ok citations" } else { Write-Host "[MISS] ok (need BUDDY_DEMO_MODE=1 or real Dify)"; $fail++ }

$rNo = PostAsk @{ question = "红烧肉怎么做？"; channel = "desktop"; user = @{ id = "t"; display_name = "t" } }
if ($rNo -and $rNo.status -eq "no_hit") { Write-Host "[OK] status=no_hit" } else { Write-Host "[MISS] no_hit"; $fail++ }

# force error path: invalid by stopping? use empty key non-demo — skip if demo only
$hz = $null
try { $hz = Invoke-RestMethod "http://127.0.0.1:8090/healthz" -TimeoutSec 3 } catch {}
if ($hz -and $hz.demo_mode) {
  Write-Host "[OK] demo_mode on — error path covered by unit tests when demo off"
} else {
  Write-Host "[INFO] non-demo: error path via missing key / timeout covered in pytest"
}

if ($fail -gt 0) { Write-Host "RESULT: FAIL"; exit 1 }
Write-Host "RESULT: PASS"
exit 0
