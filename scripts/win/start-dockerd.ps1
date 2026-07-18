# Start Docker Engine inside WSL (Day1 path when Docker Desktop unavailable)
$ErrorActionPreference = "Stop"
wsl -d Ubuntu -u root -- bash -c "docker info >/dev/null 2>&1 || (nohup dockerd >/var/log/dockerd.log 2>&1 & sleep 4); docker version"
if ($LASTEXITCODE -ne 0) { throw "dockerd failed; see wsl: /var/log/dockerd.log" }
Write-Host "OK: WSL Docker Engine ready"
