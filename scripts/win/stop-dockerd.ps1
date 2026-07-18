# Optional: stop WSL dockerd (affects all containers)
$ErrorActionPreference = "Continue"
wsl -d Ubuntu -u root -- bash -c "pkill dockerd; pkill containerd; echo dockerd_stopped"
Write-Host "OK: requested dockerd stop in WSL Ubuntu"
