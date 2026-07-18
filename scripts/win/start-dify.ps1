# Bring up Dify Compose via WSL Docker
$ErrorActionPreference = "Stop"
& "$PSScriptRoot\start-dockerd.ps1"
$composeDir = "/mnt/c/work/projects/wchat/mybuddy-v01/dify/docker"
$fixSh = "/mnt/c/work/projects/wchat/mybuddy-v01/scripts/win/fix-dify-storage.sh"
wsl -d Ubuntu -u root -- bash -c "cd $composeDir; [ -f .env ] || cp .env.example .env; docker compose -f docker-compose.yaml -f docker-compose.wsl-volumes.yaml up -d"
# Named volume may be root-owned; API uid 1001 must write privkeys
wsl -d Ubuntu -u root -- bash "$fixSh"
wsl -d Ubuntu -u root -- bash -c "cd $composeDir; docker compose -f docker-compose.yaml -f docker-compose.wsl-volumes.yaml ps"
Write-Host "Open http://127.0.0.1/install (nginx) after healthy"
Write-Host "If setup fails with page error: storage must be owned by uid 1001 (start-dify now fixes this)."
