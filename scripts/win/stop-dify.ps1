# Stop only Dify compose (keep dockerd / adapter)
$ErrorActionPreference = "Continue"
$composeDir = "/mnt/c/work/projects/wchat/mybuddy-v01/dify/docker"
wsl -d Ubuntu -u root -- bash -c "cd $composeDir; docker compose -f docker-compose.yaml -f docker-compose.wsl-volumes.yaml down"
Write-Host "OK: Dify compose down"
