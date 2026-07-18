# L1 unit tests: adapter + overlay + desktop
$ErrorActionPreference = "Stop"
$Root = Resolve-Path (Join-Path $PSScriptRoot "..\..")
$fail = 0

Write-Host "=== adapter pytest ===" -ForegroundColor Cyan
Push-Location (Join-Path $Root "adapter")
Remove-Item Env:BUDDY_DEMO_MODE -ErrorAction SilentlyContinue
python -m pip install -q -r requirements.txt
python -m pytest -q
if ($LASTEXITCODE -ne 0) { $fail++ }
Pop-Location

Write-Host "=== overlay pytest ===" -ForegroundColor Cyan
Push-Location (Join-Path $Root "openocta-overlay\skills\SearchKnowledgeDoc")
$env:PYTHONPATH = "$Root;$PWD"
python -m pytest -q tests
if ($LASTEXITCODE -ne 0) { $fail++ }
Pop-Location

Write-Host "=== desktop vitest ===" -ForegroundColor Cyan
Push-Location (Join-Path $Root "desktop")
npm test
if ($LASTEXITCODE -ne 0) { $fail++ }
Pop-Location

if ($fail -gt 0) {
  Write-Host "RESULT: FAIL ($fail suites)" -ForegroundColor Red
  exit 1
}
Write-Host "RESULT: PASS" -ForegroundColor Green
exit 0
