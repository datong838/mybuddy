# Compat shim → Check-WindowsDevEnv.ps1
param([switch]$Repair)
& "$PSScriptRoot\Check-WindowsDevEnv.ps1" -Repair:$Repair
exit $LASTEXITCODE
