# Shared: load KEY=VALUE from mybuddy-v01/.env into process env
function Import-BuddyDotEnv {
  param(
    [Parameter(Mandatory = $true)][string]$Root
  )
  $path = Join-Path $Root ".env"
  if (-not (Test-Path $path)) {
    Write-Host "No .env at $path (optional; copy from .env.example)" -ForegroundColor DarkGray
    return
  }
  Get-Content -Path $path -Encoding UTF8 | ForEach-Object {
    $line = $_.Trim()
    if (-not $line -or $line.StartsWith("#")) { return }
    $eq = $line.IndexOf("=")
    if ($eq -lt 1) { return }
    $key = $line.Substring(0, $eq).Trim()
    $val = $line.Substring($eq + 1).Trim()
    if (($val.StartsWith('"') -and $val.EndsWith('"')) -or ($val.StartsWith("'") -and $val.EndsWith("'"))) {
      $val = $val.Substring(1, $val.Length - 2)
    }
    Set-Item -Path "Env:$key" -Value $val
  }
  Write-Host "Loaded .env from $path" -ForegroundColor DarkGray
}
