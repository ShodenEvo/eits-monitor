#Requires -RunAsAdministrator
$ErrorActionPreference='SilentlyContinue'
Stop-ScheduledTask -TaskName 'EITS Monitoring Agent'
Unregister-ScheduledTask -TaskName 'EITS Monitoring Agent' -Confirm:$false
Remove-Item 'C:\Program Files\EITS Agent' -Recurse -Force
Write-Host 'EITS Agent program and startup task removed. Data remains in C:\ProgramData\EITS\Agent.'
