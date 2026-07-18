#Requires -Version 5.1
<#
============================================================
  AOS deps - clone into refs/ (SSH)
  Align: docs/palantier/20_tech/22-AOS开源产品维护清单.md
  Root default: C:\work\projects\wchat\mybuddy-v01

  Usage:
    .\clone_aos_deps.ps1
    .\clone_aos_deps.ps1 -Tier P0
    .\clone_aos_deps.ps1 -Tier P0,P1
    .\clone_aos_deps.ps1 -Tier All
    .\clone_aos_deps.ps1 -IncludeOptional
    .\clone_aos_deps.ps1 -Shallow:$false
    .\clone_aos_deps.ps1 -Root "D:\path"

  Tier:
    P0  PaddleOCR, minio, minio-py
    P1  grafana, redis, cosign
    P2  fastmcp-airbyte, fastmcp-extensions, airbyte-agents-benchmark
    Opt Tesseract, SeaweedFS (need -IncludeOptional)

  Skipped by design:
    - helm whole repo (use helm CLI release in CI)
    - kafka / redpanda whole repo for MVP (use release packages when CDC needed)
    - OpenSearch / otel-collector (P2 star; enable later if needed)

  License note:
    MinIO server / Grafana = AGPL -> refs only, do NOT bundle into customer package.
============================================================
#>
param(
    [bool]$Shallow = $true,
    [string]$Root = "C:\work\projects\wchat\mybuddy-v01",
    [string]$Tier = "P0",
    [switch]$IncludeOptional
)

$ErrorActionPreference = "Continue"

function Invoke-Git {
    param(
        [Parameter(Mandatory = $true)][string[]]$GitArgs,
        [string]$ErrorLog = ""
    )
    if ($ErrorLog) {
        $p = Start-Process -FilePath "git" -ArgumentList $GitArgs -NoNewWindow -Wait -PassThru `
            -RedirectStandardOutput $ErrorLog -RedirectStandardError "${ErrorLog}.err"
        if ($p.ExitCode -ne 0 -and (Test-Path "${ErrorLog}.err")) {
            Get-Content "${ErrorLog}.err" -Tail 6 -ErrorAction SilentlyContinue | ForEach-Object {
                Write-Host "    $_" -ForegroundColor DarkRed
            }
        }
        return $p.ExitCode
    }
    $p = Start-Process -FilePath "git" -ArgumentList $GitArgs -NoNewWindow -Wait -PassThru
    return $p.ExitCode
}

$CloneRetries = 3
$CloneRetryDelaySec = 5

# Dir is relative to Root. Optional = only with -IncludeOptional.
$Repos = @(
    @{ Tier = "P0"; Optional = $false; Dir = "refs\ocr\PaddleOCR"; Url = "git@github.com:PaddlePaddle/PaddleOCR.git"; Note = "L1-04 OCR; parser-pdf-ocr sidecar process" },
    @{ Tier = "P0"; Optional = $false; Dir = "refs\objstore\minio"; Url = "git@github.com:minio/minio.git"; Note = "L1-06 MediaSet; AGPL server - install guide only in deliverable" },
    @{ Tier = "P0"; Optional = $false; Dir = "refs\objstore-sdk\minio-py"; Url = "git@github.com:minio/minio-py.git"; Note = "MinIO Python SDK (Apache)" },
    @{ Tier = "P0"; Optional = $true; Dir = "refs\tesseract"; Url = "git@github.com:tesseract-ocr/tesseract.git"; Note = "Optional OCR fallback; already may exist" },
    @{ Tier = "P0"; Optional = $true; Dir = "refs\seaweedfs"; Url = "git@github.com:seaweedfs/seaweedfs.git"; Note = "Optional large-scale object store (Apache); already may exist" },

    @{ Tier = "P1"; Optional = $false; Dir = "refs\obs\grafana"; Url = "git@github.com:grafana/grafana.git"; Note = "AP-07 AGPL - ship Dashboard JSON only" },
    @{ Tier = "P1"; Optional = $false; Dir = "refs\cache\redis"; Url = "git@github.com:redis/redis.git"; Note = "Cache / queue / rate-limit" },
    @{ Tier = "P1"; Optional = $false; Dir = "refs\ferry\cosign"; Url = "git@github.com:sigstore/cosign.git"; Note = "Ferry sign/verify with skopeo" },

    @{ Tier = "P2"; Optional = $false; Dir = "refs\mcp\fastmcp-airbyte"; Url = "git@github.com:airbytehq/fastmcp-airbyte.git"; Note = "L1-12 MCP appendix" },
    @{ Tier = "P2"; Optional = $false; Dir = "refs\mcp\fastmcp-extensions"; Url = "git@github.com:airbytehq/fastmcp-extensions.git"; Note = "MCP extensions paired with fastmcp-airbyte" },
    @{ Tier = "P2"; Optional = $false; Dir = "refs\eval\airbyte-agents-benchmark"; Url = "git@github.com:airbytehq/airbyte-agents-benchmark.git"; Note = "AIP agent eval sidecar tool" }
)

if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Write-Host "git not found. Install Git for Windows first." -ForegroundColor Red
    exit 1
}

Write-Host "==> Checking GitHub SSH ..." -ForegroundColor Cyan
$sshOut = ""
try {
    $sshOut = & ssh -o BatchMode=yes -o StrictHostKeyChecking=accept-new -T git@github.com 2>&1 | Out-String
} catch {
    $sshOut = [string]$_.Exception.Message
}

if ($sshOut -match "successfully authenticated") {
    Write-Host "    SSH OK (no shell access message is normal)" -ForegroundColor Green
} elseif ($sshOut -match "Permission denied|Could not resolve|Connection refused") {
    Write-Host "    SSH failed. Configure a GitHub SSH key, then re-run." -ForegroundColor Red
    exit 1
} else {
    Write-Host "    Unrecognized SSH output; continue and watch clone errors:" -ForegroundColor Yellow
    Write-Host $sshOut -ForegroundColor Gray
}

$want = @{}
if ($Tier -match "(?i)^All$") {
    $want["P0"] = $true
    $want["P1"] = $true
    $want["P2"] = $true
} else {
    foreach ($t in ($Tier -split "[,\s]+")) {
        if ($t) { $want[$t.ToUpper()] = $true }
    }
}

$selected = @($Repos | Where-Object {
        $want.ContainsKey($_.Tier) -and ((-not $_.Optional) -or $IncludeOptional)
    })

if ($selected.Count -eq 0) {
    Write-Host "No repos selected. Use -Tier All or P0,P1,P2 (add -IncludeOptional for Tesseract/SeaweedFS)" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "==> AOS deps clone plan (into refs/)" -ForegroundColor Cyan
Write-Host "    Root : $Root"
Write-Host "    Tier : $($want.Keys -join ',')"
Write-Host "    Opt  : $IncludeOptional"
Write-Host "    Count: $($selected.Count)"
Write-Host "    Mode : $(if ($Shallow) { 'shallow --depth 1' } else { 'full' })"
Write-Host ""
Write-Host "    Skipped by design:" -ForegroundColor DarkYellow
Write-Host "      helm (use CLI release)"
Write-Host "      kafka / redpanda whole repo (use release when CDC needed)"
Write-Host "      OpenSearch / otel-collector (enable later)"
Write-Host ""

if (-not (Test-Path $Root)) {
    New-Item -ItemType Directory -Path $Root -Force | Out-Null
}

$ok = 0
$skip = 0
$fail = 0
$failedList = @()

foreach ($r in $selected) {
    $target = Join-Path $Root $r.Dir
    $name = Split-Path $r.Dir -Leaf
    $parent = Split-Path $target -Parent
    if (-not (Test-Path $parent)) {
        New-Item -ItemType Directory -Path $parent -Force | Out-Null
    }

    Write-Host ("---- [{0}] {1} ----" -f $r.Tier, $name) -ForegroundColor Cyan
    Write-Host ("     {0}" -f $r.Note) -ForegroundColor DarkGray

    if (Test-Path (Join-Path $target ".git")) {
        Write-Host "    SKIP exists: $target" -ForegroundColor Yellow
        $skip++
        continue
    }
    if ((Test-Path $target) -and (Get-ChildItem $target -Force -ErrorAction SilentlyContinue | Measure-Object).Count -gt 0) {
        Write-Host "    SKIP non-empty dir without .git: $target" -ForegroundColor Yellow
        $skip++
        continue
    }

    $gitArgs = @("clone")
    if ($Shallow) { $gitArgs += @("--depth", "1", "--single-branch") }
    $gitArgs += @($r.Url, $target)

    $logBase = Join-Path $env:TEMP ("clone_aos_deps_{0}" -f ($name -replace '[^\w\-]', '_'))
    $code = 1
    for ($i = 1; $i -le $CloneRetries; $i++) {
        if ($i -gt 1) {
            Write-Host "    retry $i/$CloneRetries after ${CloneRetryDelaySec}s ..." -ForegroundColor Yellow
            Start-Sleep -Seconds $CloneRetryDelaySec
            if (Test-Path $target) {
                Remove-Item -LiteralPath $target -Recurse -Force -ErrorAction SilentlyContinue
            }
        }
        Write-Host ("    git {0}" -f ($gitArgs -join " ")) -ForegroundColor DarkGray
        $code = Invoke-Git -GitArgs $gitArgs -ErrorLog $logBase
        if ($code -eq 0) { break }
    }

    if ($code -eq 0) {
        Write-Host "    OK -> $target" -ForegroundColor Green
        $ok++
    } else {
        Write-Host "    FAIL exit=$code" -ForegroundColor Red
        $fail++
        $failedList += $r.Url
        if (Test-Path $target) {
            Remove-Item -LiteralPath $target -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

Write-Host ""
Write-Host "==> Summary" -ForegroundColor Cyan
Write-Host "    OK=$ok  SKIP=$skip  FAIL=$fail"
Write-Host "    See docs/palantier/20_tech/22-AOS开源产品维护清单.md" -ForegroundColor DarkGray
if ($failedList.Count -gt 0) {
    Write-Host "    Failed URLs:" -ForegroundColor Red
    $failedList | ForEach-Object { Write-Host "      $_" -ForegroundColor Red }
    exit 1
}
Write-Host "    Done. refs/ only - do not bundle AGPL servers into customer package." -ForegroundColor Green
exit 0
